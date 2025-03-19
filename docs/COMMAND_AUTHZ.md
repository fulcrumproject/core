# Authorization Rules for Fulcrum Core

This document defines the authorization rules for commands in the Fulcrum Core system. It specifies which roles are allowed to perform specific actions on different resource types.

## Roles
- **fulcrum_admin**: System administrator with unrestricted access
- **provider_admin**: Provider administrator
- **broker**: Broker role
- **agent**: Agent role

## Authorization Rules by Resource Type

### Token
- **create**:
  - fulcrum_admin: always
  - provider_admin: for itself and for its agents
  - broker: for itself
  - agent: none (not authorized)
- **update**:
  - fulcrum_admin: always
  - provider_admin: its own tokens and those of its agents
  - broker: its own tokens
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - provider_admin: its own tokens and those of its agents
  - broker: its own tokens
  - agent: none (not authorized)
- **regenerate**:
  - fulcrum_admin: always
  - provider_admin: its own tokens and those of its agents
  - broker: its own tokens
  - agent: none (not authorized)

### Provider
- **create**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: none (not authorized)
- **update**:
  - fulcrum_admin: always
  - provider_admin: its own provider
  - broker: none (not authorized)
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: none (not authorized)

### Broker
- **create**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: none (not authorized)
- **update**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: its own broker
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: none (not authorized)

### Agent
- **create**:
  - fulcrum_admin: always
  - provider_admin: for its provider
  - broker: none (not authorized)
  - agent: none (not authorized)
- **update**:
  - fulcrum_admin: always
  - provider_admin: agents belonging to its provider
  - broker: none (not authorized)
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - provider_admin: agents belonging to its provider
  - broker: none (not authorized)
  - agent: none (not authorized)

### Service
- **create**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: for its broker
  - agent: none (not authorized)
- **update**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: services belonging to its broker
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: services belonging to its broker
  - agent: none (not authorized)
- **start**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: services belonging to its broker
  - agent: none (not authorized)
- **stop**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: services belonging to its broker
  - agent: none (not authorized)

### ServiceGroup
- **create**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: for its broker
  - agent: none (not authorized)
- **update**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: service groups belonging to its broker
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: service groups belonging to its broker
  - agent: none (not authorized)

### Job
- **claim**:
  - fulcrum_admin: none (not authorized)
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: jobs assigned to the agent
- **complete**:
  - fulcrum_admin: none (not authorized)
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: jobs claimed by the agent
- **fail**:
  - fulcrum_admin: none (not authorized)
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: jobs claimed by the agent

### MetricType
- **create**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: none (not authorized)
- **update**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: none (not authorized)

### MetricEntry
- **create**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: metric entries for services assigned to the agent

### AuditEntry
- **create**:
  DONE BY THE BACKEND. NOT A USER ACTION.