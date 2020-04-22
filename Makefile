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

.PHONY: test unit-test external-test
