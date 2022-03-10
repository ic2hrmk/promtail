test: unit-test

# Test on internal methods (cache disabled)
unit-test:
	@go test -v -count=1 --tags="unit" ./...

install-linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.25.0

run-linter:
	golangci-lint run -v

.PHONY: test unit-test
