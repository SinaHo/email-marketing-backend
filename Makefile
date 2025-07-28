
.PHONY: proto build test lint run docker-build docker-up docker-down

GOPATH := $(shell go env GOPATH)
BINARY_NAME := app

# Generate protobuf code
proto:
	@echo "⏳ Generating protobuf code..."
	bash scripts/generate_proto.sh

# Build the Go binary
build: proto
	@echo "⏳ Building binary..."
	go build -o $(BINARY_NAME) ./cmd/server

# Run tests
test:
	@echo "⏳ Running tests..."
	go test ./... -v

# Lint code
lint:
	@echo "⏳ Running linters..."
	golangci-lint run

# Run locally (requires config/config.yaml)
run: build
	@echo "🚀 Starting server..."
	./$(BINARY_NAME)

# Bring up Docker Compose
docker-up:
	docker-compose up --build -d

# Tear down Docker Compose
docker-down:
	docker-compose down