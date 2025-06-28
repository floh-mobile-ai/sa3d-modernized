# Makefile for SA3D Modernized

.PHONY: help build test clean run-local run-docker lint fmt deps docker-build docker-up docker-down docker-test

# Default target
help:
	@echo "Available targets:"
	@echo "  make build          - Build all services"
	@echo "  make test           - Run all tests"
	@echo "  make test-docker    - Run tests in Docker"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make run-local      - Run services locally (requires Go)"
	@echo "  make run-docker     - Run services in Docker"
	@echo "  make lint           - Run linters"
	@echo "  make fmt            - Format code"
	@echo "  make deps           - Download dependencies"
	@echo "  make docker-build   - Build Docker images"
	@echo "  make docker-up      - Start Docker services"
	@echo "  make docker-down    - Stop Docker services"
	@echo "  make docker-logs    - Show Docker logs"

# Build all services
build:
	@echo "Building services..."
	@cd services/analysis && go build -o ../../bin/analysis ./cmd/server
	@cd services/api-gateway && go build -o ../../bin/api-gateway ./cmd/server
	@echo "Build complete!"

# Run tests
test:
	@echo "Running tests..."
	@cd shared && go test ./... -v
	@cd services/analysis && go test ./... -v
	@cd services/api-gateway && go test ./... -v
	@echo "Tests complete!"

# Run tests in Docker
test-docker:
	@echo "Running tests in Docker..."
	@docker-compose -f docker-compose.test.yml run --rm test-integration
	@echo "Docker tests complete!"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean -cache -modcache -testcache
	@echo "Clean complete!"

# Run services locally
run-local:
	@echo "Starting infrastructure..."
	@docker-compose up -d postgres redis kafka zookeeper
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Infrastructure ready. Run services with:"
	@echo "  cd services/api-gateway && go run cmd/server/main.go"
	@echo "  cd services/analysis && go run cmd/server/main.go"

# Run services in Docker
run-docker: docker-up

# Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run ./...
	@echo "Linting complete!"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Formatting complete!"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@cd shared && go mod download
	@cd services/analysis && go mod download
	@cd services/api-gateway && go mod download
	@echo "Dependencies downloaded!"

# Docker commands
docker-build:
	@echo "Building Docker images..."
	@docker-compose build
	@echo "Docker build complete!"

docker-up:
	@echo "Starting Docker services..."
	@docker-compose up -d
	@echo "Docker services started!"
	@echo "API Gateway: http://localhost:8080"
	@echo "Analysis Service: http://localhost:8081"

docker-down:
	@echo "Stopping Docker services..."
	@docker-compose down
	@echo "Docker services stopped!"

docker-logs:
	@docker-compose logs -f

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	@cp .env.example .env
	@echo "Please edit .env with your configuration"
	@echo "Development setup complete!"

# Quick test for a specific service
test-shared:
	@cd shared && go test ./... -v

test-analysis:
	@cd services/analysis && go test ./... -v

test-gateway:
	@cd services/api-gateway && go test ./... -v

# Docker-based test for specific service
test-shared-docker:
	@docker build -f Dockerfile.test --build-arg SERVICE=shared -t test-shared .

test-analysis-docker:
	@docker build -f Dockerfile.test --build-arg SERVICE=analysis -t test-analysis .

test-gateway-docker:
	@docker build -f Dockerfile.test --build-arg SERVICE=api-gateway -t test-gateway .