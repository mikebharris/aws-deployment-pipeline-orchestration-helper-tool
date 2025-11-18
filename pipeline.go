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
var awsRegion = flag.String("region", "us-east-1", "The target AWS region for the deployment")
var appName = flag.String("app-name", "", "Microservices cluster application name (e.g. example-service, hello-world)")
var environment = flag.String("environment", "", "Target environment = prod, nonprod, preprod, staging, dev, test, etc")
var lambdasToBuildAndTest = flag.String("lambda", "all", "Which Lambda functions to test and/or build: <name-of-lambda> or all")
var buildLambdasInDirectory = flag.String("lambdas-dir", "lambdas", "Build all Lambdas in the specified directory")
var stage = flag.String("stage", "", "Deployment stage: unit-test, build, int-test, init, plan, apply, destroy")
var opConfirmed = flag.Bool("confirm", false, "For destructive operations this should be set to true rather than false")

var lambdas []string

func main() {
	flag.Parse()

	lambdas = getListOfLambdaFunctions()

	switch *stage {
	case "unit-test":
		runUnitTestsFor(lambdas)
	case "build":
		build(lambdas)
	case "int-test":
		runIntegrationTestsFor(lambdas)
	case "init":
		fallthrough
	case "plan":
		fallthrough
	case "apply":
		fallthrough
	case "destroy":
		if *appName == "" {
			log.Fatalf("error: --app-name is required to perform Terraform operations")
		}
		runTerraformCommandForRegion(*stage)
	}
}

func getListOfLambdaFunctions() []string {
	var lambdas []string
	if *lambdasToBuildAndTest != "all" {
		lambdas = append(lambdas, *lambdasToBuildAndTest)
	} else {
		lambdas = lambdaServicesInLambdasDirectory()
	}
	return lambdas
}

func runTerraformCommandForRegion(tfOp string) {
	tf := setupTerraformExec(context.Background())

	var stdout bytes.Buffer
	tf.SetStdout(&stdout)

	var tfLog strings.Builder
	tf.SetLogger(log.New(&tfLog, "log: ", log.LstdFlags))

	tfWorkingBucket := fmt.Sprintf("%d-%s-terraform-deployments", *awsAccountNumber, *awsRegion)
	switch tfOp {
	case "init":
		terraformInit(tf, tfWorkingBucket, *awsRegion)
	case "plan":
		terraformPlan(tf, tfWorkingBucket, *awsAccountNumber, *awsRegion, *environment, false)
	case "apply":
		if *opConfirmed {
			terraformApply(tf, tfWorkingBucket, *awsAccountNumber, *awsRegion, *environment)
		} else {
			log.Println("destructive apply not confirmed running plan instead...")
			terraformPlan(tf, tfWorkingBucket, *awsAccountNumber, *awsRegion, *environment, false)
		}
	case "destroy":
		if *opConfirmed {
			terraformDestroy(tf, tfWorkingBucket, *awsAccountNumber, *awsRegion, *environment)
		} else {
			log.Println("destructive destroy not confirmed running plan destroy instead...")
			terraformPlan(tf, tfWorkingBucket, *awsAccountNumber, *awsRegion, *environment, true)
		}
	case "skip":
	default:
		log.Fatalf("Bad operation: --tfop should be one of init, plan, apply, skip, or destroy")
	}

	fmt.Println("\nterraform log: \n******************\n", tfLog.String())
	fmt.Println("\nterraform stdout: \n******************\n", stdout.String())
}

func setupTerraformExec(ctx context.Context) *tfexec.Terraform {
	log.Println("installing Terraform...")
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion("1.6")),
	}

	execPath, err := installer.Install(ctx)
	if err != nil {
		log.Fatalf("error installing Terraform: %s", err)
	}

	workingDir := "terraform"
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		log.Fatalf("error running NewTerraform: %s", err)
	}
	return tf
}

func terraformInit(tf *tfexec.Terraform, tfWorkingBucket string, awsRegion string) {
	remoteStateFile := fmt.Sprintf("tfstate/%s/%s.json", *environment, *appName)
	log.Println("initialising Terraform using remote state file ", remoteStateFile, " ...")
	if err := tf.Init(context.Background(),
		tfexec.Upgrade(true),
		tfexec.BackendConfig(fmt.Sprintf("key=%s", remoteStateFile)),
		tfexec.BackendConfig(fmt.Sprintf("bucket=%s", tfWorkingBucket)),
		tfexec.BackendConfig(fmt.Sprintf("region=%s", awsRegion))); err != nil {
		log.Fatalf("error running Init: %s", err)
	}
}

func terraformPlan(tf *tfexec.Terraform, tfWorkingBucket string, awsAccountNumber uint, awsRegion string, environment string, destroyFlag bool) {
	if destroyFlag {
		log.Println("planning Terraform destroy...")
	} else {
		log.Println("planning Terraform apply...")
	}
	_, err := tf.Plan(context.Background(),
		tfexec.Refresh(true),
		tfexec.Destroy(destroyFlag),
		tfexec.Var(fmt.Sprintf("distribution_bucket=%s", tfWorkingBucket)),
		tfexec.Var(fmt.Sprintf("account_number=%d", awsAccountNumber)),
		tfexec.Var(fmt.Sprintf("region=%s", awsRegion)),
		tfexec.Var(fmt.Sprintf("environment=%s", environment)),
		tfexec.Var(fmt.Sprintf("product=%s", *appName)),
		tfexec.VarFile(fmt.Sprintf("environments/%s.tfvars", environment)),
	)
	if err != nil {
		log.Fatalf("error running Plan: %s", err)
	}
}

func terraformApply(tf *tfexec.Terraform, workingBucket string, awsAccountNumber uint, awsRegion string, environment string) {
	log.Println("applying Terraform...")
	if err := tf.Apply(context.Background(),
		tfexec.Refresh(true),
		tfexec.Var(fmt.Sprintf("distribution_bucket=%s", workingBucket)),
		tfexec.Var(fmt.Sprintf("account_number=%d", awsAccountNumber)),
		tfexec.Var(fmt.Sprintf("region=%s", awsRegion)),
		tfexec.Var(fmt.Sprintf("environment=%s", environment)),
		tfexec.Var(fmt.Sprintf("product=%s", *appName)),
		tfexec.VarFile(fmt.Sprintf("environments/%s.tfvars", environment)),
	); err != nil {
		log.Fatalf("error running Apply: %s", err)
	}
	displayTerraformOutputs(tf)
}

func terraformDestroy(tf *tfexec.Terraform, workingBucket string, awsAccountNumber uint, awsRegion string, environment string) {
	log.Println("destroying all the things...")
	if err := tf.Destroy(context.Background(),
		tfexec.Refresh(true),
		tfexec.Var(fmt.Sprintf("distribution_bucket=%s", workingBucket)),
		tfexec.Var(fmt.Sprintf("account_number=%d", awsAccountNumber)),
		tfexec.Var(fmt.Sprintf("region=%s", awsRegion)),
		tfexec.Var(fmt.Sprintf("environment=%s", environment)),
		tfexec.Var(fmt.Sprintf("product=%s", *appName)),
		tfexec.VarFile(fmt.Sprintf("environments/%s.tfvars", environment)),
	); err != nil {
		log.Fatalf("error running Destroy: %s", err)
	}
	displayTerraformOutputs(tf)
}

func displayTerraformOutputs(tf *tfexec.Terraform) {
	outputs, err := tf.Output(context.Background())
	if err != nil {
		log.Fatalf("Error outputting outputs: %v", err)
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
}

func runUnitTestsFor(lambdas []string) {
	for _, lambda := range lambdas {
		log.Printf("running tests for %s Lambda...\n", lambda)
		stdout := runCmdIn(fmt.Sprintf("lambdas/%s", lambda), "make", "unit-test")
		fmt.Println("unit tests passed; stdout = ", stdout)
	}
}

func build(lambdas []string) {
	for _, lambda := range lambdas {
		log.Printf("building %s Lambda...\n", lambda)
		stdout := runCmdIn(fmt.Sprintf("lambdas/%s", lambda), "make", "target")
		fmt.Println("build succeeded; stdout = ", stdout)
	}
}

func lambdaServicesInLambdasDirectory() []string {
	var lambdas []string
	servicesInLambdasDirectory, err := os.ReadDir(*buildLambdasInDirectory)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	if len(servicesInLambdasDirectory) == 0 {
		fmt.Println("No services found in lambdas directory")
		return nil
	}

	for _, lambda := range servicesInLambdasDirectory {
		if lambda.Name() == "common" { // this allows you to have common code between your Lambdas
			continue
		}
		lambdas = append(lambdas, lambda.Name())
	}
	return lambdas
}

func runIntegrationTestsFor(lambdas []string) {
	for _, lambda := range lambdas {
		log.Printf("running integration tests for %s Lambda...\n", lambda)
		stdout := runCmdIn(fmt.Sprintf("lambdas/%s", lambda), "make", "int-test")
		fmt.Println("integration tests passed; stdout = ", stdout)
	}
}

func runCmdIn(dir string, command string, args ...string) string {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		log.Fatalf("error running %s %s: %s\n\n******************\n\n%s\n******************\n\n", command, args, err, stdout.String())
	}
	return stdout.String()
}
