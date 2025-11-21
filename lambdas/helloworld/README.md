# Hello World Lambda Function

This directory contains a simple "Hello World" AWS Lambda function written in Go.  It only serves to demonstrate the structure of a Lambda function project that can be deployed using the accompanying AWS deployment tool.

Once deployed, you will be presented with the URL as an output from the Terraform.    For example:


```json
{
  "endpoint_url": {
    "sensitive": false,
    "type": "string",
    "value": "https://uarvijxd3rbi5vzzfevubhh2di0xfuqz.lambda-url.us-east-1.on.aws/"
  }
}
```

Visiting this URL will yield the not very informative, yet friendly output in JSON:

```json
{"message":"Hello World!"}
```

## Building

To build the Lambdas, change to the service in the _lambdas_ directory and type:

```shell
make build
```

To build the Lambda for the target AWS environment, which may have a different processor architecture from your local development, type:

```shell
make target
```

This is normally because, for example, you are developing on an Intel Mac but deploying to an ARM64 AWS Lambda environment.

## Running Tests

There are integration tests (aka service tests) that use Gherkin syntax to test integration between the Lambda and other dependent AWS services.  The tests make use of Docker containers to emulate the various services locally, and therefore you need Docker running.

The integration tests use another GitHub project of mine, [Test Container Network for Go](https://github.com/mikebharris/testcontainernetwork-go), which wraps the creation and manipulation of a network of Docker containers for common services in a set of helper routines to reduce the quantity of boilerplate code whilst maintaining the ability to test-drive their development.

To run the integration tests, change to the service in the _lambdas_ directory and type:

```shell
make int-test
```

Alternatively, you can change to the integration-tests directory and type:

```shell
cd lambdas/processor/integration-tests
go test
```

There are unit tests than can be run, again by changing to the service in the _functions_ directory and typing:

```shell
make unit-test
```

You can run both unit and integration tests for a given service with:

```shell
make test
```
