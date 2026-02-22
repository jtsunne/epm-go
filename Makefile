.PHONY: build test lint clean run integration

build:
	go build -ldflags="-X main.version=$(shell git describe --tags --always --dirty)" -o bin/epm ./cmd/epm

test:
	go test -race -count=1 ./...

lint:
	go vet ./...
	@which staticcheck && staticcheck ./... || echo "staticcheck not installed, skipping"

clean:
	rm -rf bin/

run:
	go run ./cmd/epm $(ARGS)

integration:
	ES_URI=$(ES_URI) go test -tags=integration ./...
