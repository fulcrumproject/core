# Authorization Rules for Fulcrum Core

This document defines the authorization rules for the Fulcrum Core API. It specifies which roles are allowed to perform specific actions on different resource types.

## Authorization Model

Fulcrum Core uses a role-based authorization system where permissions are defined by:
- The user's role (fulcrum_admin, participant, agent)
- The resource being accessed
- The action being performed
- The context (ownership and relationships between resources)

## Roles
- **fulcrum_admin**: System administrator with unrestricted access
- **participant**: Participant administrator that can act as both provider and consumer
- **agent**: Agent role

## Authorization Rules by Resource Type

### Token
- **create**:
  - fulcrum_admin: always
  - participant: for itself and for its agents
  - agent: none (not authorized)
- **get**:
  - fulcrum_admin: all tokens
  - participant: its own tokens and those of its agents
  - agent: none (not authorized)
- **list**:
  - fulcrum_admin: all tokens
  - participant: tokens for its participant and associated agents
  - agent: none (not authorized)
- **update**:
  - fulcrum_admin: always
  - participant: its own tokens and those of its agents
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - participant: its own tokens and those of its agents
  - agent: none (not authorized)
- **regenerate**:
  - fulcrum_admin: always
  - participant: its own tokens and those of its agents
  - agent: none (not authorized)

### Participant
- **create**:
  - fulcrum_admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)
- **get**:
  - fulcrum_admin: all participants
  - participant: its own participant
  - agent: its associated participant
- **list**:
  - fulcrum_admin: all participants
  - participant: only its own participant
  - agent: only its associated participant
- **update**:
  - fulcrum_admin: always
  - participant: its own participant
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)

### Agent
- **create**:
  - fulcrum_admin: always
  - participant: for its participant (when acting as provider)
  - agent: none (not authorized)
- **get**:
  - fulcrum_admin: all agents
  - participant: agents belonging to its participant
  - agent: itself only
- **list**:
  - fulcrum_admin: all agents
  - participant: agents belonging to its participant
  - agent: itself only
- **update**:
  - fulcrum_admin: always
  - participant: agents belonging to its participant
  - agent: update its own status only
- **delete**:
  - fulcrum_admin: always
  - participant: agents belonging to its participant
  - agent: none (not authorized)

### AgentType
- **get**:
  - fulcrum_admin: all agent types
  - participant: all agent types
  - agent: all agent types
- **list**:
  - fulcrum_admin: all agent types
  - participant: all agent types
  - agent: all agent types

### Service
- **create**:
  - fulcrum_admin: always
  - participant: when acting as consumer
  - agent: none (not authorized)
- **get**:
  - fulcrum_admin: all services
  - participant: services associated with its participant (as provider or consumer)
  - agent: services assigned to the agent
- **list**:
  - fulcrum_admin: all services
  - participant: services associated with its participant (as provider or consumer)
  - agent: services assigned to the agent
- **update**:
  - fulcrum_admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)
- **start**:
  - fulcrum_admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)
- **stop**:
  - fulcrum_admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)
- **retry**:
  - fulcrum_admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)

### ServiceType
- **get**:
  - fulcrum_admin: all service types
  - participant: all service types
  - agent: all service types
- **list**:
  - fulcrum_admin: all service types
  - participant: all service types
  - agent: all service types

### ServiceGroup
- **create**:
  - fulcrum_admin: always
  - participant: for its participant
  - agent: none (not authorized)
- **get**:
  - fulcrum_admin: all service groups
  - participant: service groups associated with its participant
  - agent: service groups associated with its assigned services
- **list**:
  - fulcrum_admin: all service groups
  - participant: service groups associated with its participant
  - agent: service groups associated with its assigned services
- **update**:
  - fulcrum_admin: always
  - participant: service groups owned by its participant
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - participant: service groups owned by its participant
  - agent: none (not authorized)

### Job
- **get**:
  - fulcrum_admin: all jobs
  - participant: jobs related to its participant (as provider via agents or as consumer via services)
  - agent: jobs assigned to the agent
- **list**:
  - fulcrum_admin: all jobs
  - participant: jobs related to its participant (as provider via agents or as consumer via services)
  - agent: jobs assigned to the agent
- **get_pending**:
  - fulcrum_admin: none (not authorized)
  - participant: none (not authorized)
  - agent: pending jobs assigned to the agent
- **claim**:
  - fulcrum_admin: none (not authorized)
  - participant: none (not authorized)
  - agent: jobs assigned to the agent
- **complete**:
  - fulcrum_admin: none (not authorized)
  - participant: none (not authorized)
  - agent: jobs claimed by the agent
- **fail**:
  - fulcrum_admin: none (not authorized)
  - participant: none (not authorized)
  - agent: jobs claimed by the agent

### MetricType
- **create**:
  - fulcrum_admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)
- **get**:
  - fulcrum_admin: all metric types
  - participant: all metric types
  - agent: all metric types
- **list**:
  - fulcrum_admin: all metric types
  - participant: all metric types
  - agent: all metric types
- **update**:
  - fulcrum_admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)
- **delete**:
  - fulcrum_admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)

### MetricEntry
- **create**:
  - fulcrum_admin: always
  - participant: none (not authorized)
  - agent: metric entries for services assigned to the agent
- **list**:
  - fulcrum_admin: all metric entries
  - participant: metric entries for its participant (as provider or consumer)
  - agent: metric entries it created

### AuditEntry
- **list**:
  - fulcrum_admin: all audit entries
  - participant: audit entries related to its participant
  - agent: none (not authorized)

## Notes
- Creation of audit entries is handled automatically by the backend and is not exposed as a user action.
- Agent types and service types are pre-provisioned in the system. While create/update/delete operations exist for administrators, these operations are primarily intended for system initialization and maintenance rather than regular use.