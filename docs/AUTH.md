# Authorization Rules for Fulcrum Core

This document defines the authorization rules for the Fulcrum Core API. It specifies which roles are allowed to perform specific actions on different resource types.

## Authorization Model

Fulcrum Core uses a role-based authorization system where permissions are defined by:
- The user's role (fulcrum_admin, provider_admin, broker, agent)
- The resource being accessed
- The action being performed
- The context (ownership and relationships between resources)

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
- **get**:
  - fulcrum_admin: all tokens
  - provider_admin: its own tokens and those of its agents
  - broker: its own tokens
  - agent: none (not authorized)
- **list**:
  - fulcrum_admin: all tokens
  - provider_admin: tokens for its provider and associated agents
  - broker: tokens for its broker
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
- **get**:
  - fulcrum_admin: all providers
  - provider_admin: its own provider
  - broker: its associated provider
  - agent: its associated provider
- **list**:
  - fulcrum_admin: all providers
  - provider_admin: only its own provider
  - broker: only its associated provider
  - agent: only its associated provider
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
- **get**:
  - fulcrum_admin: all brokers
  - provider_admin: brokers associated with its provider
  - broker: its own broker
  - agent: none (not authorized)
- **list**:
  - fulcrum_admin: all brokers
  - provider_admin: brokers associated with its provider
  - broker: only its own broker
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
- **get**:
  - fulcrum_admin: all agents
  - provider_admin: agents belonging to its provider
  - broker: agents associated with its services
  - agent: itself only
- **list**:
  - fulcrum_admin: all agents
  - provider_admin: agents belonging to its provider
  - broker: agents associated with its services
  - agent: itself only
- **update**:
  - fulcrum_admin: always
  - provider_admin: agents belonging to its provider
  - broker: none (not authorized)
  - agent: update its own status only
- **delete**:
  - fulcrum_admin: always
  - provider_admin: agents belonging to its provider
  - broker: none (not authorized)
  - agent: none (not authorized)

### AgentType
- **get**:
  - fulcrum_admin: all agent types
  - provider_admin: all agent types
  - broker: all agent types
  - agent: all agent types
- **list**:
  - fulcrum_admin: all agent types
  - provider_admin: all agent types
  - broker: all agent types
  - agent: all agent types

### Service
- **create**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: for its broker
  - agent: none (not authorized)
- **get**:
  - fulcrum_admin: all services
  - provider_admin: services associated with its provider
  - broker: services belonging to its broker
  - agent: services assigned to the agent
- **list**:
  - fulcrum_admin: all services
  - provider_admin: services associated with its provider
  - broker: services belonging to its broker
  - agent: services assigned to the agent
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
- **retry**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: services belonging to its broker
  - agent: none (not authorized)

### ServiceType
- **get**:
  - fulcrum_admin: all service types
  - provider_admin: all service types
  - broker: all service types
  - agent: all service types
- **list**:
  - fulcrum_admin: all service types
  - provider_admin: all service types
  - broker: all service types
  - agent: all service types

### ServiceGroup
- **create**:
  - fulcrum_admin: always
  - provider_admin: none (not authorized)
  - broker: for its broker
  - agent: none (not authorized)
- **get**:
  - fulcrum_admin: all service groups
  - provider_admin: service groups associated with its provider
  - broker: service groups belonging to its broker
  - agent: service groups associated with its assigned services
- **list**:
  - fulcrum_admin: all service groups
  - provider_admin: service groups associated with its provider
  - broker: service groups belonging to its broker
  - agent: service groups associated with its assigned services
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
- **get**:
  - fulcrum_admin: all jobs
  - provider_admin: jobs related to its provider
  - broker: jobs for its services
  - agent: jobs assigned to the agent
- **list**:
  - fulcrum_admin: all jobs
  - provider_admin: jobs related to its provider
  - broker: jobs for its services
  - agent: jobs assigned to the agent
- **get_pending**:
  - fulcrum_admin: none (not authorized)
  - provider_admin: none (not authorized)
  - broker: none (not authorized)
  - agent: pending jobs assigned to the agent
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
- **get**:
  - fulcrum_admin: all metric types
  - provider_admin: all metric types
  - broker: all metric types
  - agent: all metric types
- **list**:
  - fulcrum_admin: all metric types
  - provider_admin: all metric types
  - broker: all metric types
  - agent: all metric types
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
- **list**:
  - fulcrum_admin: all metric entries
  - provider_admin: metric entries for its provider
  - broker: metric entries for its services
  - agent: metric entries it created

### AuditEntry
- **list**:
  - fulcrum_admin: all audit entries
  - provider_admin: audit entries related to its provider
  - broker: audit entries related to its broker
  - agent: none (not authorized)

## Notes
- Creation of audit entries is handled automatically by the backend and is not exposed as a user action.
- Agent types and service types are pre-provisioned in the system. While create/update/delete operations exist for administrators, these operations are primarily intended for system initialization and maintenance rather than regular use.