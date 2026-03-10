FROM golangci/golangci-lint:v2.10.1 AS golangci-lint

FROM golang:1.26.1

COPY --from=golangci-lint /usr/bin/golangci-lint /usr/bin/golangci-lint

RUN go install gotest.tools/gotestsum@latest && \
    go install golang.org/x/tools/cmd/godoc@latest

WORKDIR /workspace
