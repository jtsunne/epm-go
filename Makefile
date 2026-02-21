.PHONY: build test lint run integration

build:
	go build -o bin/epm ./cmd/epm

test:
	go test -race -count=1 ./...

lint:
	go vet ./...
	staticcheck ./...

run:
	go run ./cmd/epm $(ARGS)

integration:
	go test -race -count=1 -tags integration ./... -args -es-uri=$(ES_URI)
