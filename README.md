# Microservice deployment tool for building and deploying AWS Lambdas and other services with Terraform

This is a [Go program](pipeline.go) that _helps_ you to build and test any AWS Lambda functions and then run the Terraform commands necessary to deploy them and any other request AWS services to AWS.
It takes the following parameters:

```shell
go run pipeline.go --help
Usage of pipeline:
  -account-number uint
    	Account number of AWS deployment target
  -app-name string
    	Microservices cluster application name (e.g. example-service, hello-world)
  -confirm
    	For destructive operations this should be set to true rather than false
  -environment string
    	Target environment = prod, nonprod, preprod, staging, dev, test, etc
  -lambda string
    	Which Lambda functions to test and/or build: <name-of-lambda> or all (default "all")
  -lambdas-dir string
    	Overrides default directory holding Lambda functions to build and test (default "lambdas")
  -region string
    	The target AWS region for the deployment
  -stage string
    	Deployment stage: unit-test, build, int-test, init, plan, apply, destroy
  -tf-state-bucket string
    	Overrides default S3 bucket to use for Terraform remote state storage (optional)
```

The _--stage_ parameter allows you to control each distinct stage of a pipeline deployment process, locally on your development machine, or in each stage of the pipeline:

* unit-test - Run suite of unit tests for all Lambdas
* build - Build all Lambdas for target environment
* int-test - Run suite of integration tests for all Lambdas

Note that the above operations require a Makefile in the Lambda's directory to define how to build and test the Lambda.  
The following operations require AWS credentials to be set in the environment to allow Terraform to deploy to the target AWS account:

* init - Initialise Terraform
* plan - Run Terraform plan
* apply - Run Terraform apply
* destroy - Run Terraform destroy

## Prerequisites

To run the commands to deploy to AWS, an S3 bucket must exist to hold the Terraform state files.  Create this bucket before running the init stage.  The bucket name must be in the format:

```
<account-number>-<region>-<environment>-terraform-deployments
```

You can override this default bucket name for remote state by using the --tf-state-bucket command line parameter in the init stage.

### Running the Go pipeline deployment helper program

Each deployment stage is now described in detail

#### unit-test

This runs all the unit tests for all the Lambdas:
```shell
go run pipeline.go --stage=unit-test
```

Optionally you can unit test just a single Lambda by using the _--lambda__ flag on the command line:
```shell
go run pipeline.go --stage=unit-test --lambda=helloworld
```

#### build

This builds all the Lambdas and runs their unit and integration tests:
```shell
go run pipeline.go --stage=build
```

Optionally you can build just a single Lambda by using the _--lambda__ flag on the command line:
```shell
go run pipeline.go --stage=build --lambda=helloworld
```

#### int-test

This runs all the integration tests for all the Lambdas:
```shell
go run pipeline.go --stage=int-test
```

Optionally you can run integration tests for just a single Lambda by using the _--lambda__ flag on the command line:
```shell
go run pipeline.go --stage=int-test --lambda=helloworld
```

#### init

Run Terraform init:
```shell
AWS_ACCESS_KEY_ID=XXXX AWS_SECRET_ACCESS_KEY=YYYY go run pipeline.go --stage=init --account-number=123456789012 --environment=nonprod --region=eu-west-1
```

#### plan

Run Terraform plan:
```shell
AWS_ACCESS_KEY_ID=XXXX AWS_SECRET_ACCESS_KEY=YYYY go run pipeline.go --stage=plan --app-name=myawsappname --account-number=123456789012 --environment=nonprod 
```

In the above and subsequent, I suggest ''--app-name'' is used to specify the name of your application representing the collection of AWS services that comprise an atomic project/product.

#### apply

Run Terraform apply:
```shell
AWS_ACCESS_KEY_ID=XXXX AWS_SECRET_ACCESS_KEY=YYYY go run pipeline.go --stage=apply --account-number=123456789012 --environment=nonprod --confirm=true
```

#### destroy

Run Terraform destroy:
```shell
AWS_ACCESS_KEY_ID=XXXX AWS_SECRET_ACCESS_KEY=YYYY go run pipeline.go --stage=destroy --account-number=123456789012 --environment=nonprod --confirm=true
```

### FAQ

#### I get "Backend configuration changed" whilst initialising or another operation

You get an error similar to the following:

```shell
$ go run pipeline.go --stage=init --account-number=123456789012 --environment=prod
2024/01/16 16:30:06 error running Init: exit status 1

Error: Backend configuration changed
```

This is normally due to switching between environments and caused by your local Terraform tfstate file being out-of-sync
with the remote tfstate file in S3. You can resolve it by removing the directory `terraform/.terraform` and re-running
the _init_ process.

