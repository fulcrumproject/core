# Fulcrum Core

Fulcrum Core is a comprehensive cloud infrastructure management system designed to orchestrate and monitor distributed cloud resources across multiple providers. It serves as a centralized control plane for managing cloud service participants, their deployed agents, and the various services these agents provision and maintain.

**Fulcrum project is proudly supported by [CISPE.cloud](https://www.cispe.cloud), which has contributed financial resources, expertise, and strategic guidance to the project.**  
This support reflects CISPE's commitment to enabling open, sovereign, and interoperable cloud solutions across Europe.

Fulcrum is currently under active development. New agents and features are being actively developed and will soon be available under this organization's repositories. We welcome contributions from the community – please see our [CONTRIBUTING.md](CONTRIBUTING.md) guide for more details on how to get involved.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Configuration](#configuration)
  - [Running with Docker](#running-with-docker)
  - [Running Locally](#running-locally)
- [Usage](#usage)
  - [API Examples](#api-examples)
  - [Building an Agent](#building-an-agent)
- [Health Endpoints](#health-endpoints)
- [Testing](#testing)
- [API Documentation](#api-documentation)
- [Project Structure](#project-structure)
- [Design Documentation](#design-documentation)
- [Contributing](#contributing)
- [License](#license)
- [Troubleshooting](#troubleshooting)

## Overview

Fulcrum Core provides a RESTful API for managing cloud infrastructure across multiple providers through a unified interface. The system implements a job queue mechanism where agents poll for pending operations, execute them on their respective cloud platforms, and report back results.

### Key Concepts

- **Participants**: Entities that can act as both service providers and consumers
- **Agents**: Software components deployed on participants that execute operations and manage services
- **Services**: Cloud resources (VMs, containers, Kubernetes clusters, etc.) managed by agents
- **Jobs**: Operations (create, start, stop, update, delete) that agents perform on services
- **Service Types**: Definitions of different service categories with configurable property schemas
- **Agent Types**: Classifications that determine which service types an agent can manage
- **Tokens**: Authentication mechanism for secure API access with role-based permissions

## Features

- **Multi-Cloud Management**: Manage multiple cloud service providers through a unified interface
- **Agent Orchestration**: Track and control agents deployed across different cloud environments
- **Service Lifecycle Management**: Provision, monitor, and manage various service types (VMs, containers, Kubernetes clusters, etc.)
- **Job Queue System**: Robust queue-based coordination of service operations between Fulcrum Core and agents
- **Service Grouping**: Organize services into logical groups for easier management
- **Metrics Collection**: Collect and analyze metrics from agents and services
- **Event Logging**: Maintain a comprehensive audit log of all system events and operations
- **Flexible Authentication**: Support for both token-based and OAuth/OIDC authentication
- **Property Schema Validation**: Dynamic validation of service configurations without application recompilation
- **Role-Based Access Control**: Fine-grained permissions for admin, participant, and agent roles

## Architecture

Fulcrum Core follows a clean architecture approach with clearly defined layers:

```
┌─────────────────────────────────────────────────────────┐
│                     API Layer (HTTP)                    │
│         Handlers, Routes, Middleware, Auth              │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                    Domain Layer                         │
│      Business Logic, Entities, Commanders, Rules        │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                   Database Layer                        │
│      Repository Implementations, GORM, PostgreSQL       │
└─────────────────────────────────────────────────────────┘
```

For detailed architecture information, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Getting Started

### Prerequisites

- **Go**: 1.24 or higher
- **Docker & Docker Compose**: For containerized deployment
- **PostgreSQL**: 13 or higher (included in Docker Compose)

### Installation

1. Clone the repository:

```bash
git clone https://github.com/fulcrumproject/core.git
cd core
```

2. Create a `.env` file (optional, defaults will work with Docker):

```bash
# Copy the example configuration
cat > .env << EOF
# Database Configuration
FULCRUM_DB_DSN=host=postgres user=fulcrum password=fulcrum dbname=fulcrum_db port=5432 sslmode=disable
FULCRUM_DB_LOG_LEVEL=warn
FULCRUM_DB_LOG_FORMAT=text

# Metrics Database
FULCRUM_METRIC_DB_DSN=host=postgres user=fulcrum password=fulcrum dbname=fulcrum_metrics_db port=5432 sslmode=disable
FULCRUM_METRIC_DB_LOG_LEVEL=warn
FULCRUM_METRIC_DB_LOG_FORMAT=text

# Server Configuration
FULCRUM_PORT=3000
FULCRUM_HEALTH_PORT=8081

# Authentication Configuration
FULCRUM_AUTHENTICATORS=token

# Logging Configuration
FULCRUM_LOG_FORMAT=json
FULCRUM_LOG_LEVEL=info

# Job Configuration
FULCRUM_JOB_MAINTENANCE_INTERVAL=3m
FULCRUM_JOB_RETENTION_INTERVAL=72h
FULCRUM_JOB_TIMEOUT_INTERVAL=5m

# Agent Configuration
FULCRUM_AGENT_HEALTH_TIMEOUT=5m
EOF
```

### Configuration

The application can be configured through environment variables with the prefix `FULCRUM_`. All configuration options:

#### Database Configuration

- `FULCRUM_DB_DSN`: PostgreSQL connection string for main database
- `FULCRUM_DB_LOG_LEVEL`: Database log level (debug, info, warn, error)
- `FULCRUM_DB_LOG_FORMAT`: Log format (text, json)
- `FULCRUM_METRIC_DB_DSN`: PostgreSQL connection string for metrics database
- `FULCRUM_METRIC_DB_LOG_LEVEL`: Metrics database log level
- `FULCRUM_METRIC_DB_LOG_FORMAT`: Metrics log format

#### Server Configuration

- `FULCRUM_PORT`: Main API server port (default: 3000)
- `FULCRUM_HEALTH_PORT`: Health check endpoints port (default: 8081)

#### Authentication Configuration

- `FULCRUM_AUTHENTICATORS`: Comma-separated list of enabled authenticators (e.g., "token", "oauth", "token,oauth")

#### OAuth/Keycloak Configuration (optional, only if OAuth authenticator is enabled)

- `FULCRUM_OAUTH_KEYCLOAK_URL`: Keycloak server URL
- `FULCRUM_OAUTH_REALM`: Keycloak realm name
- `FULCRUM_OAUTH_CLIENT_ID`: OAuth client ID
- `FULCRUM_OAUTH_CLIENT_SECRET`: OAuth client secret
- `FULCRUM_OAUTH_JWKS_CACHE_TTL`: JWKS cache TTL in seconds
- `FULCRUM_OAUTH_VALIDATE_ISSUER`: Validate token issuer (true/false)

#### Logging Configuration

- `FULCRUM_LOG_FORMAT`: Log format (text, json)
- `FULCRUM_LOG_LEVEL`: Log level (debug, info, warn, error)

#### Job Configuration

- `FULCRUM_JOB_MAINTENANCE_INTERVAL`: How often to run job maintenance (e.g., "3m")
- `FULCRUM_JOB_RETENTION_INTERVAL`: How long to keep completed jobs (e.g., "72h")
- `FULCRUM_JOB_TIMEOUT_INTERVAL`: When to mark jobs as timed out (e.g., "5m")

#### Agent Configuration

- `FULCRUM_AGENT_HEALTH_TIMEOUT`: Agent health check timeout (e.g., "5m")

### Running with Docker

The easiest way to get started is using Docker Compose:

```bash
# Start the entire stack (API + PostgreSQL)
docker compose up --build

# The API will be available at http://localhost:3000
# Health endpoints at http://localhost:8081
```

To restart with a clean database:

```bash
docker compose down -v && docker compose up --build
```

### Running Locally

For development without Docker:

1. Start PostgreSQL (or use Docker for just the database):

```bash
docker compose up postgres
```

2. Run the application:

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

## Usage

### API Examples

Once Fulcrum Core is running, you can interact with it via the RESTful API. Here are some common operations:

#### 1. Get the Admin Token (created during database seeding)

The default admin token is `change-me`. In production, you should create a new admin token and delete the default one.

#### 2. Create a Participant

```bash
curl -X POST http://localhost:3000/api/v1/participants \
  -H "Authorization: Bearer change-me" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Cloud Provider",
    "status": "Enabled"
  }'
```

Response:
```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "name": "My Cloud Provider",
  "status": "Enabled",
  "createdAt": "2025-09-30T10:00:00Z",
  "updatedAt": "2025-09-30T10:00:00Z"
}
```

#### 3. Create an Agent

```bash
curl -X POST http://localhost:3000/api/v1/agents \
  -H "Authorization: Bearer change-me" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Agent",
    "providerId": "01234567-89ab-cdef-0123-456789abcdef",
    "agentTypeId": "0195c3c6-4c7d-7e3c-b481-f276e17a7bec",
    "tags": ["vm", "compute"],
    "configuration": {
      "timeout": 30,
      "retries": 3
    }
  }'
```

Response (includes a one-time token):
```json
{
  "id": "abcdef12-3456-7890-abcd-ef1234567890",
  "name": "My Agent",
  "providerId": "01234567-89ab-cdef-0123-456789abcdef",
  "agentTypeId": "0195c3c6-4c7d-7e3c-b481-f276e17a7bec",
  "status": "New",
  "tags": ["vm", "compute"],
  "configuration": {
    "timeout": 30,
    "retries": 3
  },
  "token": {
    "id": "token-id-here",
    "value": "eyJhbGc...", 
    "name": "Agent Token",
    "role": "agent"
  },
  "createdAt": "2025-09-30T10:05:00Z",
  "updatedAt": "2025-09-30T10:05:00Z"
}
```

**Important**: Save the token value securely! It's only shown once during agent creation.

#### 4. Agent Updates Status (using agent token)

```bash
curl -X PUT http://localhost:3000/api/v1/agents/me/status \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "Content-Type: application/json" \
  -d '{
    "status": "Connected"
  }'
```

#### 5. Create a Service

```bash
curl -X POST http://localhost:3000/api/v1/services \
  -H "Authorization: Bearer change-me" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My VM",
    "serviceTypeId": "service-type-id-here",
    "agentId": "abcdef12-3456-7890-abcd-ef1234567890",
    "properties": {
      "vcpus": 2,
      "memory": 4096,
      "disk": 50
    }
  }'
```

This creates a service and automatically creates a "Create" job for the agent.

#### 6. Agent Polls for Jobs

```bash
curl -X GET http://localhost:3000/api/v1/jobs/pending \
  -H "Authorization: Bearer eyJhbGc..."
```

Response:
```json
[
  {
    "id": "job-id-here",
    "action": "Create",
    "serviceId": "service-id-here",
    "agentId": "abcdef12-3456-7890-abcd-ef1234567890",
    "status": "Pending",
    "params": {
      "properties": {
        "vcpus": 2,
        "memory": 4096,
        "disk": 50
      }
    },
    "priority": 1,
    "createdAt": "2025-09-30T10:10:00Z"
  }
]
```

#### 7. Agent Claims a Job

```bash
curl -X POST http://localhost:3000/api/v1/jobs/{job-id}/claim \
  -H "Authorization: Bearer eyJhbGc..."
```

#### 8. Agent Completes a Job

```bash
curl -X POST http://localhost:3000/api/v1/jobs/{job-id}/complete \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "Content-Type: application/json" \
  -d '{
    "externalId": "vm-123",
    "resources": {
      "ipAddress": "192.168.1.100",
      "hostname": "myvm.example.com"
    }
  }'
```

#### 9. Agent Reports Metrics

```bash
curl -X POST http://localhost:3000/api/v1/metric-entries \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "Content-Type: application/json" \
  -d '{
    "metricTypeId": "metric-type-id-here",
    "agentId": "abcdef12-3456-7890-abcd-ef1234567890",
    "serviceId": "service-id-here",
    "value": 75.5
  }'
```

### Building an Agent

To build your own agent that works with Fulcrum Core, you need to implement the agent protocol:

1. **Authentication**: Use the token provided during agent creation
2. **Health Updates**: Periodically update agent status to "Connected"
3. **Job Polling**: Poll `/api/v1/jobs/pending` for new jobs
4. **Job Execution**: Claim, execute, and complete/fail jobs
5. **Metrics Reporting**: Send metrics to Fulcrum Core

#### Example Agent Implementation

Check out the [Fulcrum Test Agent](https://github.com/fulcrumproject/agent-test) for a complete example implementation. This simulated agent demonstrates:

- Agent registration and authentication
- Job queue polling and processing
- VM lifecycle management (create, start, stop, delete)
- Metrics generation and reporting
- Error handling and job failure scenarios

To run the test agent:

```bash
# Clone the agent-test repository
git clone https://github.com/fulcrumproject/agent-test.git
cd agent-test

# Configure environment
cp .env.example .env
# Edit .env with your Fulcrum Core URL and credentials

# Install dependencies
go mod tidy

# Run the agent
go run main.go
```

The test agent provides a great starting point for building your own production agents for real cloud providers.

## Health Endpoints

The application provides health and readiness endpoints on a separate port for monitoring and orchestration purposes.

### Configuration

The health endpoints run on a configurable port (default: 8081):

```bash
FULCRUM_HEALTH_PORT=8081
```

### Endpoints

#### Health Check - `/healthz`

Returns the overall health status of the application and its primary dependencies.

```bash
curl http://localhost:8081/healthz
```

**Success Response (HTTP 200):**
```json
{
  "status": "UP"
}
```

**Failure Response (HTTP 503):**
```json
{
  "status": "DOWN"
}
```

#### Readiness Check - `/ready`

Returns the readiness status of the application to handle requests.

```bash
curl http://localhost:8081/ready
```

**Success Response (HTTP 200):**
```json
{
  "status": "UP"
}
```

**Failure Response (HTTP 503):**
```json
{
  "status": "DOWN"
}
```

### Primary Dependencies Checked

The health endpoints check:

1. **Database Connectivity**: PostgreSQL database connection and ping
2. **Authentication Services**: Token authenticator and OAuth/Keycloak (if configured)

## Testing

To run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage report in browser
go tool cover -html=coverage.out
```

### Integration Testing

The `test/rest/` directory contains HTTP files for manual API testing. These can be used with REST clients like:

- VS Code REST Client extension
- IntelliJ IDEA HTTP Client
- Postman (import as cURL)

Example test files:
- `test/rest/participant.http` - Participant CRUD operations
- `test/rest/agent.http` - Agent management and authentication
- `test/rest/job.http` - Job queue operations
- `test/rest/service.http` - Service lifecycle management

## API Documentation

Fulcrum Core's API is fully documented using the OpenAPI 3.0 specification. The specification is available in [docs/openapi.yaml](docs/openapi.yaml).

### Exploring the API

You can import the OpenAPI specification into:

- **Swagger UI**: [editor.swagger.io](https://editor.swagger.io)
- **Postman**: Import as OpenAPI 3.0
- **Insomnia**: Import as OpenAPI specification
- **Stoplight**: For interactive documentation

### API Endpoints Overview

- **Participants**: `/api/v1/participants` - Manage cloud service participants
- **Agents**: `/api/v1/agents` - Agent registration and management
- **Agent Types**: `/api/v1/agent-types` - Agent type definitions
- **Services**: `/api/v1/services` - Service lifecycle management
- **Service Types**: `/api/v1/service-types` - Service type definitions with schemas
- **Service Groups**: `/api/v1/service-groups` - Logical service grouping
- **Jobs**: `/api/v1/jobs` - Job queue operations
- **Metrics**: `/api/v1/metric-entries` - Metrics collection
- **Metric Types**: `/api/v1/metric-types` - Metric type definitions
- **Events**: `/api/v1/events` - Event logging and consumption
- **Tokens**: `/api/v1/tokens` - Authentication token management

## Project Structure

```
fulcrum-core/
├── cmd/                    # Application entry points
│   └── fulcrum/           # Main application
│       └── main.go
├── pkg/                    # Private application code
│   ├── api/               # HTTP handlers, routes, middleware
│   ├── auth/              # Authentication implementations
│   ├── authz/             # Authorization rules
│   ├── config/            # Configuration management
│   ├── database/          # Repository implementations
│   ├── domain/            # Domain entities and business logic
│   ├── health/            # Health check implementation
│   ├── helpers/           # Utility functions
│   ├── keycloak/          # Keycloak/OAuth integration
│   ├── middlewares/       # HTTP middlewares
│   ├── properties/        # Property schema validation
│   └── response/          # HTTP response utilities
├── docs/                  # Documentation
│   ├── ARCHITECTURE.md    # Architecture details
│   ├── AUTHORIZATION.md   # Authorization rules
│   ├── DESIGN.md          # Design documentation
│   ├── openapi.yaml       # OpenAPI specification
│   └── SERVICE_TYPE_SCHEMA.md  # Schema validation guide
├── test/                  # Test files
│   ├── keycloak/          # Keycloak test configuration
│   └── rest/              # HTTP test files
├── docker-compose.yml     # Docker Compose configuration
├── Dockerfile             # Docker image definition
├── go.mod                 # Go module definition
├── LICENSE.md             # Apache License 2.0
├── README.md              # This file
└── CONTRIBUTING.md        # Contribution guidelines
```

## Design Documentation

For comprehensive information about Fulcrum Core's design and architecture:

- **[DESIGN.md](docs/DESIGN.md)**: System design, data models, and workflows
- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)**: Layer architecture and implementation patterns
- **[AUTHORIZATION.md](docs/AUTHORIZATION.md)**: Authorization rules and role-based access control
- **[SERVICE_TYPE_SCHEMA.md](docs/SERVICE_TYPE_SCHEMA.md)**: Property schema validation syntax

## Contributing

We welcome contributions from the community! Please see our [CONTRIBUTING.md](CONTRIBUTING.md) guide for:

- Development setup instructions
- Code conventions and best practices
- Pull request process
- How to report bugs and request features

## License

Fulcrum Core is licensed under the [Apache License 2.0](LICENSE.md).

## Troubleshooting

### Common Issues

#### Database Connection Failures

Ensure PostgreSQL is running and connection details are correct:

```bash
# Check if PostgreSQL container is running
docker compose ps

# View PostgreSQL logs
docker compose logs postgres

# Test database connection
psql -h localhost -U fulcrum -d fulcrum_db
```

#### Permission Issues in Docker

If you encounter permission issues with Docker volumes:

```bash
# Remove volumes and restart
docker compose down -v
docker compose up --build
```

#### Hot Reload Not Working

Make sure Air is correctly installed and configured:

```bash
# Reinstall Air
go install github.com/cosmtrek/air@latest

# Check .air.toml configuration
cat .air.toml
```

#### Authentication Failures

1. Check that your token is valid and not expired
2. Ensure the `Authorization: Bearer <token>` header is included
3. Verify the token role has permission for the operation

```bash
# List tokens (admin only)
curl http://localhost:3000/api/v1/tokens \
  -H "Authorization: Bearer change-me"
```

#### Agent Connection Issues

1. Verify agent token is correct
2. Check agent health timeout configuration
3. Ensure agent is updating status regularly

```bash
# Check agent status
curl http://localhost:3000/api/v1/agents/me \
  -H "Authorization: Bearer <agent-token>"
```

### Getting Help

For additional support:

- **Issues**: [Open an issue](https://github.com/fulcrumproject/core/issues) on GitHub
- **Discussions**: Join our community discussions
- **Documentation**: Check the [docs/](docs/) directory for detailed guides

### Debug Mode

Enable debug logging for troubleshooting:

```bash
# Set log level to debug
export FULCRUM_LOG_LEVEL=debug

# Run the application
go run cmd/fulcrum/main.go
```

---

**Built with ❤️ by the Fulcrum Project community**