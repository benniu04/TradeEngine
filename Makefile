.PHONY: build run-gateway run-processor test test-unit test-integration test-all docker-up docker-down lint clean

build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/processor ./cmd/processor

run-gateway:
	go run ./cmd/gateway

run-processor:
	go run ./cmd/processor

test:
	go test ./... -v -race -count=1

test-unit:
	go test ./internal/matching/... -v -count=1

test-integration:
	go test -tags integration ./internal/db/... -v -race -count=1

test-all: test-unit test-integration

docker-up:
	docker compose up -d

docker-down:
	docker compose down -v

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
