test: unit-test external-test

# Test on internal methods (cache disabled)
unit-test:
	@go test -v -count=1 --tags="unit" ./...

# Test inside Docker Compose environment
external-test:
	@docker-compose \
		-f docker-compose.integration-test.yml up \
		--build \
		--abort-on-container-exit \
		--exit-code-from promtail

install-linter:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(GOPATH)/bin v1.24.0

run-linter:
	golangci-lint run -v


.PHONY: test unit-test external-test
