NAME		:= termdict
OUTPUT_BIN	?= bin/${NAME}

run:
	go mod tidy; \
	go run main.go

build:
	@echo "building termdict..."
	go mod tidy; \
	go build -o ${OUTPUT_BIN} -ldflags="-w -s"
	@echo "output generated to" ${OUTPUT_BIN}

clean:
	rm ${OUTPUT_BIN}
