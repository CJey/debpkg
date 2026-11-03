export CGO_ENABLED=0

.PHONY: all build install test lint fmt

all: build

build:
	go build ./...

install:
	go install ./cmd/debpkg

test:
	go test ./...

lint:
	go vet ./...

fmt:
	gofmt -s -w .
