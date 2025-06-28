#!/bin/sh
# Test runner script

set -e

echo "Running tests in Docker container..."

# Run shared library tests
echo "Testing shared library..."
docker run --rm -v "$(pwd):/app" -w /app/shared golang:1.23-alpine sh -c "go mod download && go test ./... -v"

# Run analysis service tests
echo "Testing analysis service..."
docker run --rm -v "$(pwd):/app" -w /app/services/analysis golang:1.23-alpine sh -c "go mod download && go test ./... -v"

# Run API gateway tests
echo "Testing API gateway..."
docker run --rm -v "$(pwd):/app" -w /app/services/api-gateway golang:1.23-alpine sh -c "go mod download && go test ./... -v"

echo "All tests completed!"