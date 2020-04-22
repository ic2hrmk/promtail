# Docker file for playing with Promtail's client
## By default, will run all available tests (unit + integration)

FROM golang:1.14.2-alpine3.11

WORKDIR /tests

COPY go.mod go.sum ./
RUN apk add git && \
    go mod download
COPY *.go ./

ENV CGO_ENABLED=0
ENV TEST_LOKI_ADDRESS="loki:3100"
ENV TEST_REQUESTS_NUMBER="20"

ENTRYPOINT ["go", "test", "-v"]
CMD ["--tags", "unit", "./..."]
