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
  "agentName": "test-vm-agent",
  "agentTypeId": "00000000-0000-0000-0000-000000000001",
  "providerId": "00000000-0000-0000-0000-000000000002",
  "fulcrumApiUrl": "http://localhost:3000",
  "vmCount": 10,
  "vmOperationInterval": "30s",
  "jobPollInterval": "5s",
  "metricReportInterval": "60s",
  "operationDelayMin": "2s",
  "operationDelayMax": "10s",
  "errorRate": 0.05
}
```

### Environment Variables

You can also use environment variables to override the configuration:

- `TESTAGENT_NAME`: Agent name
- `TESTAGENT_AGENT_TYPE_ID`: UUID of the agent type
- `TESTAGENT_PROVIDER_ID`: UUID of the provider
- `TESTAGENT_FULCRUM_API_URL`: URL of the Fulcrum Core API
- `TESTAGENT_VM_COUNT`: Number of VMs to simulate
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

## VM Lifecycle

The test agent simulates VMs with the following state transitions:

```
NONE → CREATING → CREATED → STARTING → RUNNING → STOPPING → STOPPED → DELETING → DELETED
```

At any point, operations may fail and transition the VM to an ERROR state.

## Metrics Generated

The test agent generates the following metrics:

### VM-related Metrics

- `vm.count`: Total number of VMs managed
- `vm.state.count`: Number of VMs in each state
- `vm.create.duration`: Time taken to create a VM
- `vm.start.duration`: Time taken to start a VM
- `vm.stop.duration`: Time taken to stop a VM
- `vm.delete.duration`: Time taken to delete a VM
- `vm.operation.failure`: Count of operation failures

### Resource Utilization Metrics

- `vm.cpu.usage`: Simulated CPU usage percentage
- `vm.memory.usage`: Simulated memory usage percentage
- `vm.disk.usage`: Simulated disk usage percentage
- `vm.network.throughput`: Simulated network throughput (Mbps)

### Agent Metrics

- `agent.jobs.processed`: Number of jobs processed
- `agent.jobs.success`: Number of successfully completed jobs
- `agent.jobs.failed`: Number of failed jobs
- `agent.uptime`: Agent uptime in seconds

## Job Processing

The test agent can process the following job types from Fulcrum Core:

- `ServiceCreate`: Creates a new VM
- `ServiceUpdate`: Updates a VM (start/stop operations)
- `ServiceDelete`: Deletes a VM

## Automatic Mode

When `vmCount` is set greater than 0, the agent will automatically:

1. Create the specified number of VMs at startup
2. Periodically perform random operations on these VMs
3. Generate metrics about the operations and VM states
4. Report these metrics back to Fulcrum Core

This mode is useful for demonstrations and testing the Fulcrum Core dashboard.

## Simulation Parameters

- `operationDelayMin` and `operationDelayMax`: Control how long VM operations take
- `errorRate`: Controls how often operations fail (0.0 = never, 1.0 = always)
- `vmOperationInterval`: Controls how often automatic operations occur

## Troubleshooting

### Agent Won't Register

Make sure:
- Fulcrum Core API is running and accessible
- The provided agent type ID exists in Fulcrum Core
- The provided provider ID exists in Fulcrum Core

### No Metrics Appearing

Check:
- The metric type IDs are registered in Fulcrum Core
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