.PHONY: build test lint run

build:
	go build -o bin/epm ./cmd/epm

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

run:
	go run ./cmd/epm $(ARGS)
