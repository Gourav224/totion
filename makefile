.PHONY: build run fmt clean

build:
	@go build -o totion .

run: build
	@./totion

fmt:
	@gofmt -w .

clean:
	@rm -f totion
