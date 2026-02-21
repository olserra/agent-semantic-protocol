.PHONY: all build test proto fmt lint clean run-handshake run-negotiation deps

all: build

build:
	go build ./...

test:
	go test -v -race -count=1 ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

proto:
	@which protoc > /dev/null || (echo "Install protoc: https://grpc.io/docs/protoc-installation/" && exit 1)
	@which protoc-gen-go > /dev/null || go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	protoc \
		--go_out=proto/gen \
		--go_opt=paths=source_relative \
		--proto_path=proto \
		proto/symplex.proto

fmt:
	gofmt -s -w .
	goimports -w . 2>/dev/null || true

lint:
	@which golangci-lint > /dev/null || (echo "Install golangci-lint: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

clean:
	rm -rf dist/ bin/ coverage.out coverage.html

deps:
	go mod download
	go mod tidy

run-handshake:
	go run ./examples/simple-handshake/main.go

run-negotiation:
	go run ./examples/negotiation-demo/main.go
