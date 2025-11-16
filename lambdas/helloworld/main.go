package main

import (
	"helloworld/handler"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler.Handler{}.HandleRequest)
}
