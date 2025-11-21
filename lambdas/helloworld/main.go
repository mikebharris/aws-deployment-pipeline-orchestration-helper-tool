package main

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(func() (events.LambdaFunctionURLResponse, error) {
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusOK, Headers: map[string]string{
			"Content-Type": "application/json",
		}, Body: `{"message":"Hello World!"}`}, nil
	})
}
