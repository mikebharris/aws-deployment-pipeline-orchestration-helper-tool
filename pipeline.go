package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
)

var awsAccountNumber = flag.Uint("account-number", 0, "Account number of AWS deployment target")
var awsRegion = flag.String("region", "", "The target AWS region for the deployment")
var appName = flag.String("app-name", "", "Microservices cluster application name (e.g. example-service, hello-world)")
var environment = flag.String("environment", "", "Target environment = prod, nonprod, preprod, staging, dev, test, etc")
var tfStateBucket = flag.String("tf-state-bucket", "", "Overrides default S3 bucket to use for Terraform remote state storage (optional)")
var specificLambda = flag.String("lambda", "", "Build and test a specific Lambda function rather than all: <name-of-lambda>")
var lambdasDirectory = flag.String("lambdas-dir", "lambdas", "Overrides default directory holding Lambda functions to build and test")
var lambdaCommonCodeDirectory = flag.String("lambda-common-code-dir", "common", "Overrides default directory holding common code shared between Lambda functions")
var tfWorkingDir = flag.String("tf-working-dir", "terraform", "Overrides default Terraform working directory")
var tfVersion = flag.String("tf-version", "1.14", "Version of Terraform to use")
var stage = flag.String("stage", "", "Deployment stage: unit-test, build, int-test, init, plan, apply, destroy")
var opConfirmed = flag.Bool("confirm", false, "For destructive operations this should be set to true rather than false")

func main() {
	flag.Parse()

	var lambdas []string
	if *stage == "unit-test" || *stage == "build" || *stage == "int-test" {
		if *specificLambda == "" {
			var err error
			if lambdas, err = listDirectoriesInDirectory(*lambdasDirectory, *lambdaCommonCodeDirectory); err != nil {
				fmt.Println(fmt.Errorf("error listing lambdas in %s: %s", *lambdasDirectory, err))
				os.Exit(1)
			}
			if len(lambdas) == 0 {
				fmt.Println(fmt.Errorf("no lambdas found in %s", *lambdasDirectory))
				os.Exit(1)
			}
		} else {
			lambdas = append(lambdas, *specificLambda)
		}
	}

	var err error
	switch *stage {
	case "unit-test":
		err = runUnitTestsFor(lambdas)
	case "build":
		err = build(lambdas)
	case "int-test":
		err = runIntegrationTestsFor(lambdas)
	case "init":
		fallthrough
	case "plan":
		fallthrough
	case "apply":
		fallthrough
	case "destroy":
		if *appName == "" {
			fmt.Println(fmt.Errorf("error: --app-name is required to perform Terraform operations"))
			os.Exit(1)
		}
		if *environment == "" {
			fmt.Println(fmt.Errorf("error: --environment is required to perform Terraform operations"))
			os.Exit(1)
		}
		if *awsAccountNumber == 0 {
			fmt.Println(fmt.Errorf("error: --account-number is required to perform Terraform operations"))
			os.Exit(1)
		}
		if *awsRegion == "" {
			fmt.Println(fmt.Errorf("error: --region is required to perform Terraform operations"))
			os.Exit(1)
		}
		err = runTerraformCommandForRegion(*stage, *awsRegion, *tfWorkingDir, *tfVersion, *environment, *appName)
	default:
		fmt.Printf("Bad stage: --stage should be one of unit-test, build, int-test, init, plan, apply, or destroy")
		os.Exit(1)
	}

	if err != nil {
		fmt.Println(fmt.Errorf("error running stage %s: %s", *stage, err))
		os.Exit(1)
	}
}

func runTerraformCommandForRegion(operation string, region string, tfWorkingDir string, tfVersion string, environment string, appName string) error {
	tf, err := setupTerraformExec(context.Background(), tfWorkingDir, tfVersion)
	if err != nil {
		return err
	}

	var stdout bytes.Buffer
	tf.SetStdout(&stdout)

	var tfLog strings.Builder
	tf.SetLogger(log.New(&tfLog, "log: ", log.LstdFlags))

	var tfWorkingBucket string
	if *tfStateBucket != "" {
		tfWorkingBucket = *tfStateBucket
	} else {
		tfWorkingBucket = fmt.Sprintf("%d-%s-terraform-deployments", *awsAccountNumber, *awsRegion)
	}
	log.Println("using tf state bucket ", tfWorkingBucket)

	switch operation {
	case "init":
		err = terraformInit(tf, tfWorkingBucket, region, environment, appName)
	case "plan":
		err = terraformPlan(tf, tfWorkingBucket, region, environment, appName, false)
	case "apply":
		if *opConfirmed {
			err = terraformApply(tf, tfWorkingBucket, region, environment, appName)
		} else {
			log.Println("destructive apply not confirmed running plan instead...")
			err = terraformPlan(tf, tfWorkingBucket, region, environment, appName, false)
		}
	case "destroy":
		if *opConfirmed {
			err = terraformDestroy(tf, tfWorkingBucket, region, environment, appName)
		} else {
			log.Println("destructive destroy not confirmed running plan destroy instead...")
			err = terraformPlan(tf, tfWorkingBucket, region, environment, appName, true)
		}
	}

	if err != nil {
		return fmt.Errorf("error during terraform %s: %s", operation, err)
	}

	fmt.Println("\nterraform log: \n******************\n", tfLog.String())
	fmt.Println("\nterraform stdout: \n******************\n", stdout.String())

	return nil
}

func setupTerraformExec(ctx context.Context, workingDir string, tfVersion string) (*tfexec.Terraform, error) {
	log.Println("installing Terraform...")
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion(tfVersion)),
	}

	execPath, err := installer.Install(ctx)
	if err != nil {
		return nil, fmt.Errorf("error installing Terraform: %s", err)
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, fmt.Errorf("error running NewTerraform: %s", err)
	}
	return tf, nil
}

func terraformInit(tf *tfexec.Terraform, tfWorkingBucket string, region string, environment string, appName string) error {
	remoteStateFile := fmt.Sprintf("tfstate/%s/%s.json", environment, appName)
	log.Println("initialising Terraform using remote state file ", remoteStateFile, " ...")
	if err := tf.Init(context.Background(),
		tfexec.Upgrade(true),
		tfexec.BackendConfig(fmt.Sprintf("key=%s", remoteStateFile)),
		tfexec.BackendConfig(fmt.Sprintf("bucket=%s", tfWorkingBucket)),
		tfexec.BackendConfig(fmt.Sprintf("region=%s", region))); err != nil {
		return fmt.Errorf("error running Init: %s", err)
	}
	return nil
}

func terraformPlan(tf *tfexec.Terraform, tfWorkingBucket string, awsRegion string, environment string, appName string, destroyFlag bool) error {
	if destroyFlag {
		log.Println("planning Terraform destroy...")
	} else {
		log.Println("planning Terraform apply...")
	}
	_, err := tf.Plan(context.Background(),
		tfexec.Refresh(true),
		tfexec.Destroy(destroyFlag),
		tfexec.Var(fmt.Sprintf("distribution_bucket=%s", tfWorkingBucket)),
		tfexec.Var(fmt.Sprintf("region=%s", awsRegion)),
		tfexec.Var(fmt.Sprintf("environment=%s", environment)),
		tfexec.Var(fmt.Sprintf("product=%s", appName)),
		tfexec.VarFile(fmt.Sprintf("environments/%s.tfvars", environment)),
	)
	if err != nil {
		return fmt.Errorf("error running Plan: %s", err)
	}
	return nil
}

func terraformApply(tf *tfexec.Terraform, tfWorkingBucket string, awsRegion string, environment string, appName string) error {
	log.Println("applying Terraform...")
	if err := tf.Apply(context.Background(),
		tfexec.Refresh(true),
		tfexec.Var(fmt.Sprintf("distribution_bucket=%s", tfWorkingBucket)),
		tfexec.Var(fmt.Sprintf("region=%s", awsRegion)),
		tfexec.Var(fmt.Sprintf("environment=%s", environment)),
		tfexec.Var(fmt.Sprintf("product=%s", appName)),
		tfexec.VarFile(fmt.Sprintf("environments/%s.tfvars", environment)),
	); err != nil {
		return fmt.Errorf("error running Apply: %s", err)
	}
	return displayTerraformOutputs(tf)
}

func terraformDestroy(tf *tfexec.Terraform, tfWorkingBucket string, awsRegion string, environment string, appName string) error {
	log.Println("destroying all the things...")
	if err := tf.Destroy(context.Background(),
		tfexec.Refresh(true),
		tfexec.Var(fmt.Sprintf("distribution_bucket=%s", tfWorkingBucket)),
		tfexec.Var(fmt.Sprintf("region=%s", awsRegion)),
		tfexec.Var(fmt.Sprintf("environment=%s", environment)),
		tfexec.Var(fmt.Sprintf("product=%s", appName)),
		tfexec.VarFile(fmt.Sprintf("environments/%s.tfvars", environment)),
	); err != nil {
		return fmt.Errorf("error running Destroy: %s", err)
	}
	return displayTerraformOutputs(tf)
}

func displayTerraformOutputs(tf *tfexec.Terraform) error {
	outputs, err := tf.Output(context.Background())
	if err != nil {
		return fmt.Errorf("error outputting outputs: %v", err)
	}
	if len(outputs) > 0 {
		fmt.Println("Terraform outputs:")
	}
	for key := range outputs {
		if outputs[key].Sensitive {
			continue
		}
		fmt.Println(fmt.Sprintf("%s = %s\n", key, string(outputs[key].Value)))
	}
	return nil
}

func runUnitTestsFor(lambdas []string) error {
	for _, lambda := range lambdas {
		log.Printf("running tests for %s Lambda...\n", lambda)
		stdout, err := runCmdIn(fmt.Sprintf("lambdas/%s", lambda), "make", "unit-test")
		if err != nil {
			return fmt.Errorf("building lambda %s: %s", lambda, err)
		}
		fmt.Println("unit tests passed; stdout = ", stdout)
	}
	return nil
}

func build(lambdas []string) error {
	for _, lambda := range lambdas {
		log.Printf("building %s Lambda...\n", lambda)
		stdout, err := runCmdIn(fmt.Sprintf("lambdas/%s", lambda), "make", "target")
		if err != nil {
			return fmt.Errorf("building lambda %s: %s", lambda, err)
		}
		fmt.Println("build succeeded; stdout = ", stdout)
	}
	return nil
}

func listDirectoriesInDirectory(parentDir string, ignoreChildDir string) ([]string, error) {
	var directories []string
	dirEntries, err := os.ReadDir(parentDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %v", parentDir, err)
	}
	if len(dirEntries) == 0 {
		return nil, nil
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.Name() == ignoreChildDir {
			continue
		}
		directories = append(directories, dirEntry.Name())
	}
	return directories, nil
}

func runIntegrationTestsFor(lambdas []string) error {
	for _, lambda := range lambdas {
		log.Printf("running integration tests for %s Lambda...\n", lambda)
		stdout, err := runCmdIn(fmt.Sprintf("lambdas/%s", lambda), "make", "int-test")
		if err != nil {
			return fmt.Errorf("running integration tests: %s", err)
		}
		fmt.Println("integration tests passed; stdout = ", stdout)
	}
	return nil
}

func runCmdIn(dir string, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), command, args...)
	cmd.Dir = dir
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running %s %v: %s\n\n******************\n\n%s\n******************\n\n", command, strings.Join(args, " "), err, stdout.String())
	}
	return stdout.String(), nil
}
