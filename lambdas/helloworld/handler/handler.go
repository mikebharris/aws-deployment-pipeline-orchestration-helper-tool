package handler

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

type Handler struct {
}

func (h Handler) HandleRequest() (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{StatusCode: http.StatusOK, Headers: map[string]string{
		"Content-Type": "application/json",
	}, Body: `{"message":"Hello World!"}`}, nil

}
