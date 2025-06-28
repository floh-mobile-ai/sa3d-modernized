# SA3D Modernized

A modern, cloud-native platform for 3D visualization of software architectures.

## Overview

SA3D Modernized analyzes source code statically and dynamically, extracts software metrics, and presents them in interactive 3D visualizations. It supports real-time collaboration and WebXR/VR for immersive code reviews.

## Features

- 🎨 **3D Visualizations**: Treemap, Sphere, Package-Relations, City metaphor
- 🌐 **Multi-Language Support**: Java, C#, Python, JavaScript, TypeScript, Go
- 👥 **Real-time Collaboration**: Multiple users can explore code together
- 🥽 **WebXR/VR Support**: Immersive code reviews in virtual reality
- 🔌 **CI/CD Integration**: RESTful APIs and webhooks
- 📊 **Behavioral Analysis**: Git history-based insights

## Architecture

The system follows a microservices architecture with the following core services:

- **API Gateway**: Request routing and authentication
- **Analysis Service**: Code parsing and metrics extraction
- **Visualization Service**: 3D scene generation
- **Collaboration Service**: Real-time state synchronization
- **Metrics Service**: Time-series data management

## Technology Stack

- **Backend**: Go 1.21+
- **Frontend**: React 18 + TypeScript + Three.js
- **Database**: PostgreSQL 15+ with TimescaleDB
- **Cache**: Redis 7+
- **Message Queue**: Apache Kafka
- **Container**: Docker + Kubernetes

## Getting Started

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Node.js 18+ (for frontend development)
- PostgreSQL 15+
- Redis 7+

### Development Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/sa3d-modernized.git
cd sa3d-modernized
```

2. Copy environment variables:
```bash
cp .env.example .env
```

3. Start infrastructure services:
```bash
docker-compose up -d postgres redis kafka
```

4. Run database migrations:
```bash
make migrate
```

5. Start the services:
```bash
make run-all
```

## Project Structure

```
sa3d-modernized/
├── services/           # Microservices
│   ├── api-gateway/    # API Gateway service
│   ├── analysis/       # Code analysis service
│   ├── visualization/  # 3D visualization service
│   ├── collaboration/  # Real-time collaboration
│   └── metrics/        # Metrics processing
├── frontend/           # React frontend application
├── shared/             # Shared libraries and utilities
├── deployments/        # Kubernetes manifests and Helm charts
├── scripts/            # Build and deployment scripts
└── docs/               # Documentation
```

## Performance

- Analyze 1M+ LOC in under 5 minutes
- Render 10,000+ code elements at 60fps
- Support 1000+ concurrent users
- <100ms latency for real-time collaboration

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.