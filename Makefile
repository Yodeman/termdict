NAME		:= termdict
OUTPUT_BIN	?= bin/${NAME}

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
LDFLAGS := -s -w \
	-X github.com/yodeman/termdict/internal/config.AppVersion=$(VERSION) \
	-X github.com/yodeman/termdict/internal/config.Commit=$(COMMIT)

# Source-build install destination (make install).
PREFIX ?= $(shell go env GOPATH)/bin

.PHONY: run build clean test vet fmt lint tidy install

run:
	go run .

build:
	@echo "building termdict $(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o ${OUTPUT_BIN} .
	@echo "output generated to" ${OUTPUT_BIN}

install:
	go build -ldflags "$(LDFLAGS)" -o ${PREFIX}/${NAME} .
	@echo "installed to ${PREFIX}/${NAME}"

clean:
	rm -f ${OUTPUT_BIN}

test:
	go test ./... -race

vet:
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run \
		|| echo "golangci-lint not installed; skipping (https://golangci-lint.run/welcome/install/)"
