# Docker file for playing with Promtail's client
## By default, will run all available tests (unit + integration)

FROM golang:1.14.2-alpine3.11

WORKDIR /go/src/github.com/ic2hrmk/promtail

COPY go.mod go.sum ./
RUN apk add git && \
    go mod download
COPY . ./

ENV CGO_ENABLED=0
ENV TEST_LOKI_ADDRESS="loki:3100"
ENV TEST_REQUESTS_NUMBER="20"

CMD ["go", "test", "-v", "--tags", "unit", "./..."]
