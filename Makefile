BINARY ?= k8s-mcp-server
IMAGE ?= ghcr.io/vk7416/generic-k8s-mcp:dev

.PHONY: build run test tidy docker-build

build:
	go build -o bin/$(BINARY) ./cmd/k8s-mcp-server

run:
	go run ./cmd/k8s-mcp-server --mode=local --readonly=true

test:
	go test ./...

tidy:
	go mod tidy

docker-build:
	docker build -t $(IMAGE) .
