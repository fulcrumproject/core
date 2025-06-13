# Authorization Rules for Fulcrum Core

This document defines the authorization rules for the Fulcrum Core API. It specifies which roles are allowed to perform specific actions on different resource types.

## Authorization Model

Fulcrum Core uses a role-based authorization system where permissions are defined by:
- The user's role (admin, participant, agent)
- The resource being accessed
- The action being performed
- The context (ownership and relationships between resources)

## Roles
- **admin**: System administrator with unrestricted access
- **participant**: Participant administrator that can act as both provider and consumer
- **agent**: Agent role

## Authorization Rules by Resource Type

### Token
- **create**:
  - admin: always
  - participant: for itself and for its agents
  - agent: none (not authorized)
- **get**:
  - admin: all tokens
  - participant: its own tokens and those of its agents
  - agent: none (not authorized)
- **list**:
  - admin: all tokens
  - participant: tokens for its participant and associated agents
  - agent: none (not authorized)
- **update**:
  - admin: always
  - participant: its own tokens and those of its agents
  - agent: none (not authorized)
- **delete**:
  - admin: always
  - participant: its own tokens and those of its agents
  - agent: none (not authorized)
- **regenerate**:
  - admin: always
  - participant: its own tokens and those of its agents
  - agent: none (not authorized)

### Participant
- **create**:
  - admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)
- **get**:
  - admin: all participants
  - participant: its own participant
  - agent: its associated participant
- **list**:
  - admin: all participants
  - participant: only its own participant
  - agent: only its associated participant
- **update**:
  - admin: always
  - participant: its own participant
  - agent: none (not authorized)
- **delete**:
  - admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)

### Agent
- **create**:
  - admin: always
  - participant: for its participant (when acting as provider)
  - agent: none (not authorized)
- **get**:
  - admin: all agents
  - participant: agents belonging to its participant
  - agent: itself only
- **list**:
  - admin: all agents
  - participant: agents belonging to its participant
  - agent: itself only
- **update**:
  - admin: always
  - participant: agents belonging to its participant
  - agent: update its own status only
- **delete**:
  - admin: always
  - participant: agents belonging to its participant
  - agent: none (not authorized)

### AgentType
- **get**:
  - admin: all agent types
  - participant: all agent types
  - agent: all agent types
- **list**:
  - admin: all agent types
  - participant: all agent types
  - agent: all agent types

### Service
- **create**:
  - admin: always
  - participant: when acting as consumer
  - agent: none (not authorized)
- **get**:
  - admin: all services
  - participant: services associated with its participant (as provider or consumer)
  - agent: services assigned to the agent
- **list**:
  - admin: all services
  - participant: services associated with its participant (as provider or consumer)
  - agent: services assigned to the agent
- **update**:
  - admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)
- **delete**:
  - admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)
- **start**:
  - admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)
- **stop**:
  - admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)
- **retry**:
  - admin: always
  - participant: services where it is the consumer participant
  - agent: none (not authorized)

### ServiceType
- **get**:
  - admin: all service types
  - participant: all service types
  - agent: all service types
- **list**:
  - admin: all service types
  - participant: all service types
  - agent: all service types

### ServiceGroup
- **create**:
  - admin: always
  - participant: for its participant
  - agent: none (not authorized)
- **get**:
  - admin: all service groups
  - participant: service groups associated with its participant
  - agent: service groups associated with its assigned services
- **list**:
  - admin: all service groups
  - participant: service groups associated with its participant
  - agent: service groups associated with its assigned services
- **update**:
  - admin: always
  - participant: service groups owned by its participant
  - agent: none (not authorized)
- **delete**:
  - admin: always
  - participant: service groups owned by its participant
  - agent: none (not authorized)

### Job
- **get**:
  - admin: all jobs
  - participant: jobs related to its participant (as provider via agents or as consumer via services)
  - agent: jobs assigned to the agent
- **list**:
  - admin: all jobs
  - participant: jobs related to its participant (as provider via agents or as consumer via services)
  - agent: jobs assigned to the agent
- **get_pending**:
  - admin: none (not authorized)
  - participant: none (not authorized)
  - agent: pending jobs assigned to the agent
- **claim**:
  - admin: none (not authorized)
  - participant: none (not authorized)
  - agent: jobs assigned to the agent
- **complete**:
  - admin: none (not authorized)
  - participant: none (not authorized)
  - agent: jobs claimed by the agent
- **fail**:
  - admin: none (not authorized)
  - participant: none (not authorized)
  - agent: jobs claimed by the agent

### MetricType
- **create**:
  - admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)
- **get**:
  - admin: all metric types
  - participant: all metric types
  - agent: all metric types
- **list**:
  - admin: all metric types
  - participant: all metric types
  - agent: all metric types
- **update**:
  - admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)
- **delete**:
  - admin: always
  - participant: none (not authorized)
  - agent: none (not authorized)

### MetricEntry
- **create**:
  - admin: always
  - participant: none (not authorized)
  - agent: metric entries for services assigned to the agent
- **list**:
  - admin: all metric entries
  - participant: metric entries for its participant (as provider or consumer)
  - agent: metric entries it created

### AuditEntry
- **list**:
  - admin: all audit entries
  - participant: audit entries related to its participant
  - agent: none (not authorized)

## Notes
- Creation of audit entries is handled automatically by the backend and is not exposed as a user action.
- Agent types and service types are pre-provisioned in the system. While create/update/delete operations exist for administrators, these operations are primarily intended for system initialization and maintenance rather than regular use.