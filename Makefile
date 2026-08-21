NAME		:= termdict
OUTPUT_BIN	?= bin/${NAME}

.PHONY: run build clean test vet fmt lint tidy

run:
	go run .

build:
	@echo "building termdict..."
	go build -o ${OUTPUT_BIN} -ldflags="-w -s"
	@echo "output generated to" ${OUTPUT_BIN}

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
