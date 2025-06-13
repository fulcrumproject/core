# Fulcrum Core

Fulcrum Core is a comprehensive cloud infrastructure management system designed to orchestrate and monitor distributed cloud resources across multiple providers. It serves as a centralized control plane for managing cloud service providers, their deployed agents, and the various services these agents provision and maintain.

**Fulcrum project is proudly supported by [CISPE.cloud](https://www.cispe.cloud), which has contributed financial resources, expertise, and strategic guidance to the project.**  
This support reflects CISPE's commitment to enabling open, sovereign, and interoperable cloud solutions across Europe.

Fulcrum is currently under active development. New agents and features are being actively developed and will soon be available under this organization's repositories. We welcome contributions from the community – please see our [CONTRIBUTING.md](CONTRIBUTING.md) guide for more details on how to get involved.

## Table of Contents

- [Fulcrum Core](#fulcrum-core)
  - [Table of Contents](#table-of-contents)
  - [Features](#features)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Configuration](#configuration)
    - [Running with Docker](#running-with-docker)
    - [Running locally](#running-locally)
  - [Testing](#testing)
  - [API Documentation](#api-documentation)
  - [Project Structure](#project-structure)
  - [Design Documentation](#design-documentation)
  - [Troubleshooting](#troubleshooting)
    - [Common Issues](#common-issues)

## Features

- Manage multiple cloud service providers through a unified interface
- Track and control agents deployed across different cloud environments
- Provision and monitor various service types (VMs, containers, Kubernetes clusters, etc.)
- Organize services into logical groups for easier management
- Collect and analyze metrics from agents and services
- Maintain a comprehensive log of all system events and operations
- Coordinate service operations with agents through a robust job queue system

## Getting Started

### Prerequisites

- Go 1.24 or higher
- Docker and Docker Compose
- PostgreSQL (for local development without Docker)

### Configuration

1. Clone the repository:
```bash
git clone https://github.com/your-organization/fulcrum-core.git
```
2. Copy `.env.example` to `.env` and adjust the values as needed:

```
# Database Configuration
FULCRUM_DB_HOST=localhost
FULCRUM_DB_PORT=5432
FULCRUM_DB_USER=fulcrum
FULCRUM_DB_PASSWORD=your_secure_password
FULCRUM_DB_NAME=fulcrum_db
FULCRUM_DB_SSL_MODE=disable
FULCRUM_DB_LOG_LEVEL=warn
FULCRUM_DB_LOG_FORMAT=text

# Server Configuration
FULCRUM_PORT=3000

# Authentication Configuration
# Comma-separated list of enabled authenticators (e.g., "token", "oauth", "token,oauth")
FULCRUM_AUTHENTICATORS=token,oauth

# OAuth/Keycloak Configuration (only required if "oauth" authenticator is enabled)
FULCRUM_OAUTH_KEYCLOAK_URL=http://localhost:8080
FULCRUM_OAUTH_REALM=fulcrum
FULCRUM_OAUTH_CLIENT_ID=fulcrum-api
FULCRUM_OAUTH_CLIENT_SECRET=your_client_secret
FULCRUM_OAUTH_JWKS_CACHE_TTL=3600
FULCRUM_OAUTH_VALIDATE_ISSUER=true

# Logging Configuration
FULCRUM_LOG_FORMAT=text
FULCRUM_LOG_LEVEL=info

# Job Configuration
FULCRUM_JOB_MAINTENANCE_INTERVAL=3m
FULCRUM_JOB_RETENTION_INTERVAL=72h
FULCRUM_JOB_TIMEOUT_INTERVAL=5m

# Agent Configuration
FULCRUM_AGENT_HEALTH_TIMEOUT=5m
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

### Running locally

1. Make sure your `.env` file is configured
2. Run only the database:
```bash
docker compose up postgres
```
3. Start the application:
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

## Testing

To run the test suite:

```bash
go test ./...
```

For tests with coverage report:

```bash
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out
```

## API Documentation

Fulcrum Core's API is documented using the OpenAPI 3.0 specification. The specification is available in the [openapi.yaml](docs/openapi.yaml) file in the project root. This file can be imported into tools like Swagger UI, Postman, or other OpenAPI compatible tools to explore and test the API.

An online version of the API documentation will be available soon.

## Project Structure

```
fulcrum-core/
├── cmd/             # Application entry points
│   └── fulcrum/     # Main application entry point
├── docs/            # Documentation
├── internal/        # Private application and library code
│   ├── api/         # HTTP handlers and routes
│   ├── config/      # Configuration handling
│   ├── database/    # Database implementations of repositories
│   ├── domain/      # Domain models and repository interfaces
│   └── logging/     # Logging utilities
└── test/            # Test files
    └── rest/        # HTTP test files for API testing

```
## Design Documentation

For a comprehensive overview of Fulcrum Core's architecture, data model, and component interactions, please refer to the [DESIGN.md](docs/DESIGN.md) document.

## Troubleshooting

### Common Issues

- **Database Connection Failures**: Ensure your PostgreSQL server is running and the connection details in `.env` are correct.

- **Permission Issues in Docker**: If you encounter permission issues with Docker volumes, try running `docker compose down -v` to remove volumes and then restart.

- **Hot Reload Not Working**: Make sure you have the latest version of Air installed and your `.air.toml` file is correctly configured.

For more support, please [open an issue](https://github.com/your-organization/fulcrum-core/issues) on our GitHub repository.

