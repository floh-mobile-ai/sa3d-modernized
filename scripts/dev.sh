#!/bin/bash

# Development helper script for SA3D Modernized

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    local missing=()
    
    if ! command_exists docker; then
        missing+=("docker")
    fi
    
    if ! command_exists docker-compose; then
        missing+=("docker-compose")
    fi
    
    if ! command_exists go; then
        missing+=("go")
    fi
    
    if [ ${#missing[@]} -ne 0 ]; then
        print_error "Missing required tools: ${missing[*]}"
        print_info "Please install the missing tools and try again."
        exit 1
    fi
    
    print_info "All prerequisites are installed."
}

# Function to start infrastructure
start_infrastructure() {
    print_info "Starting infrastructure services..."
    docker-compose up -d postgres redis kafka zookeeper
    
    print_info "Waiting for services to be ready..."
    sleep 10
    
    # Check if services are healthy
    if docker-compose ps | grep -E "(postgres|redis|kafka)" | grep -v "Up"; then
        print_error "Some infrastructure services failed to start"
        docker-compose logs postgres redis kafka
        exit 1
    fi
    
    print_info "Infrastructure services are ready."
}

# Function to stop all services
stop_all() {
    print_info "Stopping all services..."
    docker-compose down
    print_info "All services stopped."
}

# Function to build services
build_services() {
    print_info "Building services..."
    
    # Build Go services
    for service in api-gateway analysis visualization collaboration metrics; do
        if [ -d "services/$service" ]; then
            print_info "Building $service..."
            docker-compose build $service-service
        fi
    done
    
    print_info "All services built successfully."
}

# Function to run tests
run_tests() {
    print_info "Running tests..."
    
    # Run Go tests
    for service in api-gateway analysis visualization collaboration metrics; do
        if [ -d "services/$service" ]; then
            print_info "Testing $service..."
            (cd services/$service && go test ./... -v)
        fi
    done
    
    print_info "All tests completed."
}

# Function to run a specific service locally
run_service() {
    local service=$1
    
    if [ -z "$service" ]; then
        print_error "Service name required"
        echo "Usage: $0 run <service-name>"
        exit 1
    fi
    
    if [ ! -d "services/$service" ]; then
        print_error "Service '$service' not found"
        exit 1
    fi
    
    print_info "Running $service locally..."
    
    # Ensure infrastructure is running
    start_infrastructure
    
    # Run the service
    cd services/$service
    go run cmd/server/main.go
}

# Function to show logs
show_logs() {
    local service=$1
    
    if [ -z "$service" ]; then
        docker-compose logs -f
    else
        docker-compose logs -f $service
    fi
}

# Function to clean up
cleanup() {
    print_info "Cleaning up..."
    docker-compose down -v
    
    # Clean Go cache
    go clean -cache -modcache -testcache
    
    print_info "Cleanup completed."
}

# Function to initialize database
init_db() {
    print_info "Initializing database..."
    
    # Start postgres if not running
    docker-compose up -d postgres
    sleep 5
    
    # Run migrations (placeholder - implement actual migrations)
    print_warning "Database migrations not yet implemented"
    
    print_info "Database initialization completed."
}

# Main script logic
case "$1" in
    start)
        check_prerequisites
        start_infrastructure
        ;;
    stop)
        stop_all
        ;;
    build)
        check_prerequisites
        build_services
        ;;
    test)
        check_prerequisites
        run_tests
        ;;
    run)
        check_prerequisites
        run_service "$2"
        ;;
    logs)
        show_logs "$2"
        ;;
    clean)
        cleanup
        ;;
    init-db)
        init_db
        ;;
    dev)
        check_prerequisites
        start_infrastructure
        print_info "Infrastructure is ready. You can now run services locally."
        print_info "Use '$0 run <service-name>' to run a specific service."
        ;;
    *)
        echo "SA3D Modernized Development Script"
        echo ""
        echo "Usage: $0 {start|stop|build|test|run|logs|clean|init-db|dev}"
        echo ""
        echo "Commands:"
        echo "  start     - Start infrastructure services"
        echo "  stop      - Stop all services"
        echo "  build     - Build all services"
        echo "  test      - Run all tests"
        echo "  run       - Run a specific service locally"
        echo "  logs      - Show logs (optionally for specific service)"
        echo "  clean     - Clean up everything"
        echo "  init-db   - Initialize database"
        echo "  dev       - Start infrastructure for local development"
        echo ""
        echo "Examples:"
        echo "  $0 dev                    # Start infrastructure for development"
        echo "  $0 run api-gateway        # Run API Gateway locally"
        echo "  $0 logs analysis-service  # Show logs for analysis service"
        exit 1
        ;;
esac