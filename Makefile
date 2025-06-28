.PHONY: help build test clean run-all docker-build docker-up docker-down migrate lint fmt

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build all services"
	@echo "  test         - Run all tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  run-all      - Run all services locally"
	@echo "  docker-build - Build Docker images"
	@echo "  docker-up    - Start services with Docker Compose"
	@echo "  docker-down  - Stop Docker Compose services"
	@echo "  migrate      - Run database migrations"
	@echo "  lint         - Run linters"
	@echo "  fmt          - Format code"

# Build all services
build:
	@echo "Building API Gateway..."
	cd services/api-gateway && go build -o ../../bin/api-gateway ./cmd/server
	@echo "Building Analysis Service..."
	cd services/analysis && go build -o ../../bin/analysis ./cmd/server
	@echo "Building Visualization Service..."
	cd services/visualization && go build -o ../../bin/visualization ./cmd/server
	@echo "Building Collaboration Service..."
	cd services/collaboration && go build -o ../../bin/collaboration ./cmd/server
	@echo "Building Metrics Service..."
	cd services/metrics && go build -o ../../bin/metrics ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	find . -name "*.test" -delete

# Run all services locally
run-all:
	@echo "Starting all services..."
	goreman start

# Build Docker images
docker-build:
	docker-compose build

# Start services with Docker Compose
docker-up:
	docker-compose up -d

# Stop Docker Compose services
docker-down:
	docker-compose down

# Run database migrations
migrate:
	@echo "Running database migrations..."
	cd scripts && ./migrate.sh up

# Run linters
lint:
	@echo "Running linters..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofumpt -l -w .