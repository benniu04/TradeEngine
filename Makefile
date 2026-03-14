.PHONY: build run-gateway run-processor test docker-up docker-down lint clean

build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/processor ./cmd/processor

run-gateway:
	go run ./cmd/gateway

run-processor:
	go run ./cmd/processor

test:
	go test ./... -v -race -count=1

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down -v

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
