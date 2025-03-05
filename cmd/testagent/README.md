# Fulcrum Test Agent

A VM lifecycle management test agent for the Fulcrum Core platform

## Overview

The Fulcrum Test Agent simulates a real Fulcrum agent that manages virtual machine (VM) lifecycles and produces metrics related to these operations. It is designed to help with testing, development, and demonstration of the Fulcrum Core platform without requiring actual cloud providers.

This agent implements the complete Fulcrum agent protocol, including:
- Agent registration and authentication
- VM lifecycle management (create, start, stop, delete)
- Job queue processing
- Metrics generation and reporting
- Realistic operation timing and occasional failures

## Installation

### Prerequisites

- Go 1.24 or higher
- Access to a running Fulcrum Core API
- Agent type and provider registered in Fulcrum Core

### Building

```bash
# Navigate to the test agent directory
cd cmd/testagent

# Build the agent
go build -o testagent
```

## Configuration

The test agent can be configured using a combination of a configuration file and environment variables.

### Configuration File

Create a JSON configuration file (e.g., `config.json`):

```json
{
  "agentToken": "TOKEN",
  "fulcrumApiUrl": "http://localhost:3000",
  "vmUpdateInterval": "30s",
  "jobPollInterval": "5s",
  "metricReportInterval": "60s",
  "operationDelayMin": "2s",
  "operationDelayMax": "10s",
  "errorRate": 0.05
}
```

### Environment Variables

You can also use environment variables to override the configuration:

- `TESTAGENT_AGENT_TOKEN`: Secret token of the agent
- `TESTAGENT_FULCRUM_API_URL`: URL of the Fulcrum Core API
- `TESTAGENT_VM_OPERATION_INTERVAL`: How often to perform VM operations
- `TESTAGENT_JOB_POLL_INTERVAL`: How often to poll for jobs
- `TESTAGENT_METRIC_REPORT_INTERVAL`: How often to report metrics
- `TESTAGENT_OPERATION_DELAY_MIN`: Minimum operation time
- `TESTAGENT_OPERATION_DELAY_MAX`: Maximum operation time
- `TESTAGENT_ERROR_RATE`: Probability of operation failure (0.0-1.0)

## Usage

### Running the Agent

```bash
# Run with default configuration
./testagent

# Run with a configuration file
./testagent -config config.json
```

### Stopping the Agent

The agent handles SIGINT and SIGTERM signals for graceful shutdown. Simply press `Ctrl+C` to stop it cleanly.


## Metrics Generated

The test agent generates the following metrics:

### Resource Utilization Metrics

- `vm.cpu.usage`: Simulated CPU usage percentage
- `vm.memory.usage`: Simulated memory usage percentage
- `vm.disk.usage`: Simulated disk usage percentage
- `vm.network.throughput`: Simulated network throughput (Mbps)

## Job Processing

The test agent can process the following job types from Fulcrum Core:

- `ServiceCreate`: Creates a new VM
- `ServiceUpdate`: Updates a VM (start/stop operations)
- `ServiceDelete`: Deletes a VM

## Simulation Parameters

- `operationDelayMin` and `operationDelayMax`: Control how long VM operations take
- `errorRate`: Controls how often operations fail (0.0 = never, 1.0 = always)
- `vmUpdateInterval`: Controls how often automatic operations occur

## Troubleshooting

### Agent Won't Register

Make sure:
- Fulcrum Core API is running and accessible
- The provided agent token exists in Fulcrum Core

### No Metrics Appearing

Check:
- The metric type names are registered in Fulcrum Core
- The agent is successfully registered
- The metrics reporting interval is not too long

### Operation Failures

The agent simulates occasional failures as controlled by the `errorRate` parameter. This is normal behavior intended to simulate real-world conditions.

## Development and Testing

The test agent includes comprehensive unit tests:

```bash
# Run all tests
go test ./...

# Run specific tests
go test -v ./agent