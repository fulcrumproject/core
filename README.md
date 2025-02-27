# Fulcrum Core

Fulcrum Core is a comprehensive cloud infrastructure management system designed to orchestrate and monitor distributed cloud resources across multiple providers. It serves as a centralized control plane for managing cloud service providers, their deployed agents, and the various services these agents provision and maintain.

## Features

- Manage multiple cloud service providers through a unified interface
- Track and control agents deployed across different cloud environments
- Provision and monitor various service types (VMs, containers, Kubernetes clusters, etc.)
- Organize services into logical groups for easier management
- Collect and analyze metrics from agents and services
- Maintain a comprehensive audit trail of all system operations
- Coordinate service operations with agents through a robust job queue system

## Getting Started

### Prerequisites

- Go 1.24 or higher
- Docker and Docker Compose
- PostgreSQL (for local development without Docker)

### Configuration

1. Clone the repository
2. Copy `.env.example` to `.env` and adjust the values as needed:

```
# Database Configuration
DB_HOST=postgres
DB_USER=fulcrum
DB_PASSWORD=fulcrum_password
DB_NAME=fulcrum_db
DB_PORT=5432

# Server Configuration
PORT=3000
```

### Running with Docker

To start the entire application stack:

```bash
docker compose up --build
```

To restart with a clean database:

```bash
docker compose down -v && docker compose up --build
```

To run only the database:

```bash
docker compose up postgres
```

### Running Locally

1. Make sure your `.env` file is configured
2. Start the application:

```bash
go run cmd/fulcrum/main.go
```

For development with hot-reload:

```bash
# Install air if you haven't already
go install github.com/cosmtrek/air@latest

# Run with hot-reload
air
```

## API Documentation

Fulcrum Core's API is documented using the OpenAPI 3.0 specification. The specification is available in the [openapi.yaml](docs/openapi.yaml) file in the project root. This file can be imported into tools like Swagger UI, Postman, or other OpenAPI compatible tools to explore and test the API.

An online version of the API documentation is also available at: TBD

## Project Structure

- `cmd/fulcrum/`: Application entry point
- `internal/api/`: HTTP handlers and routes
- `internal/database/`: Database implementations of repositories
- `internal/domain/`: Domain models and repository interfaces
- `internal/service/`: Business logic services
- `rest-tests/`: HTTP test files for API testing

## Detailed Design Documentation

For a comprehensive overview of Fulcrum Core's architecture, data model, and component interactions, please refer to the [DESIGN.md](docs/DESIGN.md) document.

## License

Apache License 2.0