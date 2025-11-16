export SHELL:=/bin/bash

.PHONY: build
build:
	GOOS=linux CGO_ENABLED=0 go build -o main .

.PHONY: target
target:
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap main.go

.PHONY: int-test
int-test: build
	cd integration-tests && go test

.PHONY: unit-test
unit-test:
	go test . ./... -coverprofile=coverage.out -coverpkg=./...
	go tool cover -html=coverage.out -o ./coverage.html

.PHONY: test
test: unit-test int-test