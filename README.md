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
# Main Database (for application data)
FULCRUM_DB_DSN=host=localhost user=fulcrum password=your_secure_password dbname=fulcrum_db port=5432 sslmode=disable
FULCRUM_DB_LOG_LEVEL=warn
FULCRUM_DB_LOG_FORMAT=text

# Metrics Database (for metrics and monitoring data)
FULCRUM_METRIC_DB_DSN=host=localhost user=fulcrum password=your_secure_password dbname=fulcrum_metrics_db port=5432 sslmode=disable
FULCRUM_METRIC_DB_LOG_LEVEL=warn
FULCRUM_METRIC_DB_LOG_FORMAT=text

# Server Configuration
FULCRUM_PORT=3000
FULCRUM_HEALTH_PORT=3001

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

## Health Endpoints

The application provides health and readiness endpoints on a separate port for monitoring and orchestration purposes.

### Configuration

The health endpoints run on a configurable port (default: 8081):

```bash
# Health endpoints port
FULCRUM_HEALTH_PORT=8081
```

### Endpoints

#### Health Check - `/healthz`

Returns the overall health status of the application and its primary dependencies.

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

The health endpoints check the following primary dependencies:

1. **Database Connectivity**: PostgreSQL database connection and ping
2. **Authentication Services**: 
   - Token authenticator (database-based)
   - OAuth/Keycloak authenticator (if configured)

When any primary dependency is unavailable, the API is considered unable to respond to the majority of requests, resulting in a `DOWN` status.

### Usage Examples

```bash
# Check application health
curl http://localhost:8081/healthz

# Check application readiness
curl http://localhost:8081/ready
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

### API Usage Examples

The Fulcrum Core API provides a comprehensive RESTful interface for managing cloud infrastructure. All API endpoints require authentication using bearer tokens.

#### Authentication

Fulcrum supports two authentication methods:

1. **Token-based Authentication**: Use API tokens for programmatic access
2. **OAuth/Keycloak**: Use OAuth2 flows for user authentication

```bash
# Using token authentication
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://api.fulcrum.example.com/api/v1/participants
```

#### Working with Participants

Participants represent cloud service providers or consumers in the Fulcrum ecosystem.

**Create a Participant:**

```bash
curl -X POST https://api.fulcrum.example.com/api/v1/participants \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Cloud Provider",
    "type": "provider",
    "description": "Enterprise cloud services provider"
  }'
```

**List Participants:**

```bash
curl https://api.fulcrum.example.com/api/v1/participants?page=1&pageSize=20 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**Get a Specific Participant:**

```bash
curl https://api.fulcrum.example.com/api/v1/participants/{participant_id} \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### Managing Agents

Agents are deployed on cloud providers to manage services and collect metrics.

**Register an Agent:**

```bash
curl -X POST https://api.fulcrum.example.com/api/v1/agents \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "agent-01",
    "agent_type_id": "550e8400-e29b-41d4-a716-446655440000",
    "participant_id": "650e8400-e29b-41d4-a716-446655440000",
    "host": "agent-01.example.com",
    "port": 8080,
    "properties": {
      "region": "us-east-1",
      "availability_zone": "us-east-1a"
    }
  }'
```

**Agent Health Check:**

Agents automatically send heartbeats. If an agent doesn't send a heartbeat within the configured timeout period, it's automatically marked as disconnected.

```bash
# Agent sends heartbeat
curl -X POST https://api.fulcrum.example.com/api/v1/agents/{agent_id}/heartbeat \
  -H "Authorization: Bearer YOUR_AGENT_TOKEN"
```

#### Service Management

Services represent cloud resources (VMs, containers, Kubernetes clusters, etc.) managed by agents.

**Create a Service:**

```bash
curl -X POST https://api.fulcrum.example.com/api/v1/services \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-server-01",
    "service_type_id": "750e8400-e29b-41d4-a716-446655440000",
    "agent_id": "850e8400-e29b-41d4-a716-446655440000",
    "properties": {
      "cpu": "4",
      "memory": "16GB",
      "disk": "100GB"
    }
  }'
```

**Update Service Status:**

```bash
curl -X PATCH https://api.fulcrum.example.com/api/v1/services/{service_id} \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "running"
  }'
```

**List Services with Filters:**

```bash
# Filter by agent
curl "https://api.fulcrum.example.com/api/v1/services?agent_id={agent_id}" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Filter by service type
curl "https://api.fulcrum.example.com/api/v1/services?service_type_id={type_id}" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Filter by status
curl "https://api.fulcrum.example.com/api/v1/services?status=running" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### Job Queue Management

Jobs represent asynchronous operations that agents need to perform.

**Create a Job:**

```bash
curl -X POST https://api.fulcrum.example.com/api/v1/jobs \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "950e8400-e29b-41d4-a716-446655440000",
    "type": "provision",
    "properties": {
      "action": "create_vm",
      "parameters": {
        "image": "ubuntu-22.04",
        "flavor": "m1.medium"
      }
    }
  }'
```

**Agent Polls for Jobs:**

```bash
curl "https://api.fulcrum.example.com/api/v1/jobs?agent_id={agent_id}&status=pending" \
  -H "Authorization: Bearer YOUR_AGENT_TOKEN"
```

**Update Job Status:**

```bash
curl -X PATCH https://api.fulcrum.example.com/api/v1/jobs/{job_id} \
  -H "Authorization: Bearer YOUR_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "completed",
    "result": {
      "success": true,
      "vm_id": "i-1234567890abcdef"
    }
  }'
```

#### Metrics Collection

Collect and query metrics from agents and services.

**Submit Metrics:**

```bash
curl -X POST https://api.fulcrum.example.com/api/v1/metric-entries \
  -H "Authorization: Bearer YOUR_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "metric_type_id": "a50e8400-e29b-41d4-a716-446655440000",
    "service_id": "b50e8400-e29b-41d4-a716-446655440000",
    "value": 85.5,
    "timestamp": "2025-09-30T10:30:00Z"
  }'
```

**Query Metrics:**

```bash
# Get metrics for a specific service
curl "https://api.fulcrum.example.com/api/v1/metric-entries?service_id={service_id}&from=2025-09-01T00:00:00Z&to=2025-09-30T23:59:59Z" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### Event Subscriptions

Subscribe to events for real-time notifications about system changes.

**Create Event Subscription:**

```bash
curl -X POST https://api.fulcrum.example.com/api/v1/events/subscriptions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://your-app.example.com/webhooks/fulcrum",
    "event_types": ["service.created", "service.updated", "job.completed"],
    "filters": {
      "participant_id": "c50e8400-e29b-41d4-a716-446655440000"
    }
  }'
```

### SDK and Client Libraries

While Fulcrum Core provides a REST API, you can also build client libraries in your preferred programming language. Here's an example in Go:

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type FulcrumClient struct {
    BaseURL string
    Token   string
    Client  *http.Client
}

func NewFulcrumClient(baseURL, token string) *FulcrumClient {
    return &FulcrumClient{
        BaseURL: baseURL,
        Token:   token,
        Client:  &http.Client{},
    }
}

func (c *FulcrumClient) CreateParticipant(name, ptype, description string) error {
    data := map[string]string{
        "name":        name,
        "type":        ptype,
        "description": description,
    }
    
    jsonData, _ := json.Marshal(data)
    req, _ := http.NewRequest("POST", c.BaseURL+"/api/v1/participants", bytes.NewBuffer(jsonData))
    req.Header.Set("Authorization", "Bearer "+c.Token)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.Client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusCreated {
        return fmt.Errorf("failed to create participant: %d", resp.StatusCode)
    }
    
    return nil
}

func main() {
    client := NewFulcrumClient("https://api.fulcrum.example.com", "YOUR_TOKEN")
    err := client.CreateParticipant("My Provider", "provider", "My cloud provider")
    if err != nil {
        fmt.Println("Error:", err)
    }
}
```

### Python Example

```python
import requests
import json

class FulcrumClient:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.token = token
        self.headers = {
            'Authorization': f'Bearer {token}',
            'Content-Type': 'application/json'
        }
    
    def create_participant(self, name, ptype, description):
        url = f"{self.base_url}/api/v1/participants"
        data = {
            "name": name,
            "type": ptype,
            "description": description
        }
        response = requests.post(url, headers=self.headers, json=data)
        response.raise_for_status()
        return response.json()
    
    def list_services(self, agent_id=None, status=None):
        url = f"{self.base_url}/api/v1/services"
        params = {}
        if agent_id:
            params['agent_id'] = agent_id
        if status:
            params['status'] = status
        
        response = requests.get(url, headers=self.headers, params=params)
        response.raise_for_status()
        return response.json()

# Usage
client = FulcrumClient('https://api.fulcrum.example.com', 'YOUR_TOKEN')
participant = client.create_participant('My Provider', 'provider', 'Description')
services = client.list_services(status='running')
```

## Project Structure

```
fulcrum-core/
├── cmd/             # Application entry points
│   └── fulcrum/     # Main application entry point
├── docs/            # Documentation
├── pkg/        # Private application and library code
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

