# SA3D Modernized - 3D Software Architecture Visualization Platform

SA3D Modernized is a modern, cloud-native platform for visualizing and analyzing software architectures in 3D. It provides real-time collaborative features, comprehensive code analysis, and interactive 3D visualizations to help teams understand and improve their software systems.

## Features

- **3D Architecture Visualization**: Interactive 3D representations of software components and their relationships
- **Real-time Code Analysis**: Automated analysis of code structure, dependencies, and quality metrics
- **Collaborative Features**: Real-time collaboration with team members, annotations, and shared sessions
- **Multi-language Support**: Analyze projects written in Go, Java, Python, JavaScript/TypeScript, C#, and more
- **Quality Metrics**: Comprehensive code quality metrics including complexity, maintainability, and technical debt
- **Modern Architecture**: Microservices-based architecture with Go backend and React frontend

## Architecture

The platform consists of several microservices:

- **API Gateway**: Central entry point handling authentication, routing, and rate limiting
- **Analysis Service**: Performs code analysis and extracts architectural information
- **Visualization Service**: Generates and manages 3D visualizations
- **Collaboration Service**: Handles real-time collaboration features
- **Metrics Service**: Calculates and stores code quality metrics

## Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose
- Node.js 18+ (for frontend development)
- PostgreSQL 16 (via Docker)
- Redis 7 (via Docker)
- Apache Kafka (via Docker)

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/yourusername/sa3d-modernized.git
cd sa3d-modernized
```

2. Copy the environment file:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Start the infrastructure:
```bash
# On Linux/Mac
./scripts/dev.sh dev

# On Windows
scripts\dev.bat dev
```

4. Run the API Gateway:
```bash
# On Linux/Mac
./scripts/dev.sh run api-gateway

# On Windows
scripts\dev.bat run api-gateway
```

5. Run the Analysis Service:
```bash
# In a new terminal
# On Linux/Mac
./scripts/dev.sh run analysis

# On Windows
scripts\dev.bat run analysis
```

## Development

### Project Structure

```
sa3d-modernized/
├── services/              # Microservices
│   ├── api-gateway/       # API Gateway service
│   ├── analysis/          # Code analysis service
│   ├── visualization/     # 3D visualization service
│   ├── collaboration/     # Real-time collaboration service
│   └── metrics/           # Metrics calculation service
├── shared/                # Shared libraries and utilities
├── frontend/              # React frontend application
├── deployments/           # Kubernetes manifests and Helm charts
├── scripts/               # Development and deployment scripts
├── docs/                  # Documentation
└── tests/                 # Integration tests
```

### Running Tests

Run all tests:
```bash
make test
```

Run tests for a specific service:
```bash
cd services/api-gateway
go test ./...
```

### Building Services

Build all services:
```bash
make build
```

Build a specific service:
```bash
docker-compose build api-gateway
```

### Code Style

We use the following tools for code quality:
- `gofmt` and `golangci-lint` for Go code
- `prettier` and `eslint` for JavaScript/TypeScript

Run linters:
```bash
make lint
```

## API Documentation

The API Gateway exposes the following endpoints:

### Authentication
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/logout` - User logout
- `POST /api/v1/auth/refresh` - Refresh access token
- `GET /api/v1/auth/validate` - Validate token

### Projects
- `GET /api/v1/projects` - List projects
- `POST /api/v1/projects` - Create project
- `GET /api/v1/projects/:id` - Get project details
- `PUT /api/v1/projects/:id` - Update project
- `DELETE /api/v1/projects/:id` - Delete project

### Analysis
- `POST /api/v1/analysis/start/:projectId` - Start analysis
- `GET /api/v1/analysis/status/:analysisId` - Get analysis status
- `DELETE /api/v1/analysis/cancel/:analysisId` - Cancel analysis
- `GET /api/v1/analysis/results/:analysisId` - Get analysis results

### Visualization
- `GET /api/v1/visualization/project/:projectId` - Get project visualization
- `POST /api/v1/visualization/render` - Render visualization
- `GET /api/v1/visualization/layouts` - Get available layouts
- `PUT /api/v1/visualization/layout/:projectId` - Update layout

### Metrics
- `GET /api/v1/metrics/project/:projectId` - Get project metrics
- `GET /api/v1/metrics/file/:projectId/:filePath` - Get file metrics
- `GET /api/v1/metrics/trends/:projectId` - Get metric trends
- `GET /api/v1/metrics/compare` - Compare metrics

## Configuration

Each service can be configured through environment variables or configuration files. See the `.env.example` file for available options.

### Key Configuration Options

- `JWT_SECRET`: Secret key for JWT token signing
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string
- `KAFKA_BROKERS`: Kafka broker addresses

## Deployment

### Docker Compose (Development)

```bash
docker-compose up -d
```

### Kubernetes (Production)

```bash
# Apply Kubernetes manifests
kubectl apply -f deployments/k8s/

# Or use Helm
helm install sa3d-modernized deployments/helm/sa3d-modernized
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Based on the original SA3D concept
- Built with modern cloud-native technologies
- Inspired by software architecture visualization best practices