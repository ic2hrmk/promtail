# Test inside Docker Compose environment
test:
	@docker-compose \
		-f docker-compose.test.yml up \
		--build \
		--abort-on-container-exit \
		--exit-code-from promtail

.PHONY: test
