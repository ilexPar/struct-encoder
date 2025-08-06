.ONESHELL:

SHELL := /bin/bash
STATIC_ANALYSIS_FILE := analysis-report.html

###### Development ######

test::
	@go test ./tests/...

test-cov::
	@gotestsum -f dots-v2 -- -coverprofile cover.out -coverpkg=./pkg/... ./tests/... 

test-html:: test-cov
	@go tool cover -html=cover.out

test-cov-print:: test-cov
	go tool cover -func=cover.out

analyse::
	@echo "Performing Static Analysis with golangci-lint"
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.57 golangci-lint run --out-format html --tests=false --timeout 5m0s > ${STATIC_ANALYSIS_FILE}
	firefox ${STATIC_ANALYSIS_FILE}