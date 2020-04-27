test: unit-test external-test

# Test on internal methods (cache disabled)
unit-test:
	@go test -v -count=1 --tags="unit" ./...

# Test inside Docker Compose environment
external-test:
	@docker-compose \
		-f docker-compose.test.yml up \
		--build \
		--abort-on-container-exit \
		--exit-code-from promtail

install-linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.25.0

run-linter:
	golangci-lint run -v

.PHONY: test unit-test external-test
