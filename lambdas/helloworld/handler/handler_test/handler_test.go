package handler_test

import (
	"helloworld/handler"

	"github.com/stretchr/testify/assert"

	"net/http"
	"testing"
)

func Test_shouldReturnAFriendlyMessageWhenCalled(t *testing.T) {
	// Given
	h := handler.Handler{}

	// When
	response, err := h.HandleRequest()

	// Then
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "application/json", response.Headers["Content-Type"])
	assert.Equal(t, `{"message":"Hello World!"}`, response.Body)
}
