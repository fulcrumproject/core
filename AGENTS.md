# AGENTS.md

This document provides comprehensive guidance for AI agents working on the Fulcrum Core project. It covers development guidelines, system architecture, and domain knowledge essential for effective contribution to this cloud infrastructure management system.

## Table of Contents
- [Development Guidelines](#development-guidelines)
- [System Architecture](#system-architecture)
- [Domain Model](#domain-model)
- [Authorization System](#authorization-system)
- [Service Management](#service-management)
- [Job Processing](#job-processing)
- [Monitoring & Audit](#monitoring--audit)

---

# DEVELOPMENT GUIDELINES

## General Rules

### Role & Context
- You are an architect and senior developer using golang
- Read the md files in the docs folder and the README.md

### Code Style
- Use short and clear names in the code
- Use `any` and not `interface{}` in function signatures
- Unused imports are removed automatically by the IDE

### Documentation & Diagrams
- Mermaid diagrams should not contain styles

### Database Management
- We don't need database migrations - we use GORM migration

## Commenting Guidelines
When generating code, always add comments that **explain why, not what**. Focus on rationale, assumptions, and trade-offs rather than repeating the code.

* **Implementation**: clarify tricky logic or unusual design choices.
  ```ts
  // Using a loop instead of users.length to skip soft-deleted users
  ```
* **Documentation**: describe APIs, parameters, return values, errors.
  ```go
  // GetActiveUsers retrieves all enabled users from the database.
  // Returns an error if the database query fails.
  func GetActiveUsers(ctx context.Context) ([]User, error)
  ```
* **Contextual**: note assumptions, dependencies, or performance/security concerns.
  ```go
  // Requires cache pre-loaded; DB fallback is too slow
  ```

**Checklist**: accurate, up to date, clear, explains magic numbers/flags, understandable by newcomers.
**Do**: explain intent, document APIs, keep comments current.
**Don't**: restate code, leave outdated notes, use vague language.

---

# SYSTEM ARCHITECTURE

## Overview
This system follows a clean architecture approach with clearly defined layers (API, Domain, Database) that maintain strict dependency rules. Dependencies point inward toward the domain layer, which contains business logic independent of external frameworks.

## Key Design Principles
- Separation of Concerns: Each layer has a specific responsibility
- Dependency Inversion: Dependencies point inward toward the domain core
- Interface Segregation: Small, focused interfaces for different concerns
- Single Responsibility: Each component has one reason to change
- Clean Boundaries: Layers communicate through well-defined interfaces

## Layer Structure

### API Layer
- Handles HTTP requests through RESTful endpoints
- Converts between JSON/HTTP and domain objects
- Implements authentication and authorization through middleware chain
- Manages pagination and response formatting
- Uses handlers organized by domain entity
- No direct database access; works through domain interfaces

#### Middleware Architecture
- Auth middleware validates tokens and adds identity to context
- Authorization uses AuthzFromExtractor base pattern with specialized extractors:
  - AuthzSimple: No resource scope needed
  - AuthzFromID: Extracts scope from resource ID
  - AuthzFromBody: Extracts scope from request body
- DecodeBody[T] provides type-safe request body handling
- ID middleware extracts and validates UUIDs from URL paths
- RequireAgentIdentity ensures agent-specific authentication

#### Handler Patterns
- Routes use middleware chains for cross-cutting concerns
- Request types implement AuthTargetScopeProvider interface
- Handler methods focus on pure business logic
- Authentication/authorization handled entirely by middleware
- Use MustGetBody[T] and MustGetID for type-safe context access

### Domain Layer
- Contains core business logic and entities with behavior
- Defines repository interfaces for data access
- Implements domain services through Commanders
- Uses value objects for domain concepts
- Has no external dependencies

### Transaction Management
- Store interface provides Atomic method
- Commands use Store.Atomic for transaction boundaries
- Multiple repository operations execute within single transaction
- Ensures data consistency, audit trail, and proper error handling

### Database Layer
- Implements repository interfaces defined in domain
- Uses Command-Query separation pattern
- Handles database operations and transaction management
- Maps between domain entities and database models
- Optimizes database queries and performance

## Package Structure
```
/
├── cmd/             # Application entry points
├── internal/        # Private application code
│   ├── api/         # HTTP handlers
│   ├── domain/      # Business logic, entities, interfaces
│   ├── database/    # Repository implementations
│   ├── config/      # Configuration
│   └── logging/     # Logging utilities
└── test/            # Test files
```

## Repository Pattern
- EntityRepository interfaces handle write operations
- EntityQuerier interfaces handle read-only operations
- Repositories embed querriers (CQRS-inspired)
- Store interface manages repositories and transactions

## Command Pattern
- Commander interfaces define complex operations
- Commands handle validation, entity creation, and business logic
- Use Store.Atomic to manage transaction boundaries
- Create audit entries within transaction boundaries

## Testing Strategies
- Unit tests for domain entities and business rules
- Repository tests with database test helpers
- Handler tests focus on business logic with simulated middleware context
- Middleware tests verify authorization logic in isolation
- Integration tests verify complete request flow with middleware chain
- End-to-end tests across layers

### Handler Test Patterns
- Simulate middleware context: decoded bodies, extracted IDs, auth identity
- Test pure business logic without authorization concerns
- Use MustGetBody[T] and MustGetID with mocked context values
- Focus on domain errors and validation scenarios

---

# DOMAIN MODEL

## System Overview

### Purpose
Fulcrum Core is a comprehensive cloud infrastructure management system designed to orchestrate and monitor distributed cloud resources across multiple participants. It serves as a centralized control plane for managing cloud service participants, their deployed agents, and the various services these agents provision and maintain.

### Key Capabilities
- Manage multiple cloud service participants through a unified interface
- Track and control agents deployed across different cloud environments
- Provision and monitor various service types (VMs, containers, Kubernetes clusters, etc.)
- Organize services into logical groups for easier management
- Collect and analyze metrics from agents and services
- Maintain a comprehensive audit trail of all system operations
- Coordinate service operations with agents through a robust job queue system

## Core Entities

### Participant
- Represents an entity that can act as both a service provider and consumer
- Has name and operational state (Enabled/Disabled)
- Contains geographical information via country code
- Stores flexible metadata through custom attributes
- Has many agents deployed within its infrastructure (when acting as a provider)
- Can consume services (via Service.ConsumerParticipantID)
- The functional role (provider/consumer) is determined by context and relationships

### Agent
- Deployed software component that manages services
- Belongs to a specific Participant (acting as provider) and AgentType
- Tracks connectivity state (New, Connected, Disconnected, Error, Disabled)
- Uses secure token-based authentication
- Processes jobs from the job queue to perform service operations

### Service
- Cloud resource managed by an agent
- Has sophisticated state management with current and target states
- State transitions: Creating → Created → Starting → Started → Stopping → Stopped → Deleting → Deleted
- Supports both hot updates (while running) and cold updates (while stopped)
- Tracks failed operations with error messages and retry counts
- Has properties (configuration that can be updated) and attributes (static metadata)
- Can be linked to a consumer participant via ConsumerParticipantID

### ServiceGroup
- Organizes related services into logical groups
- Belongs to a specific Participant
- Enables collective management of related services

### Job
- Represents a discrete operation to be performed by an agent
- Actions include: Create, Start, Stop, HotUpdate, ColdUpdate, Delete
- States include: Pending, Processing, Completed, Failed
- Prioritizes operations for execution order
- Tracks execution timing and error details

### Token
- Provides secure authentication mechanism for system access
- Supports different roles: fulcrum_admin, participant, agent
- Contains hashed value stored in database to verify authentication
- Has expiration date for enhanced security
- Scoped to specific Participant or Agent based on role

### MetricEntry & MetricType
- Record and categorize performance metrics for agents and services
- Track numerical measurements with timestamps
- Associate measurements with specific resources

### AuditEntry
- Tracks system events and changes for audit purposes
- Records the authority (type and ID) that initiated the action
- Categorizes events by type
- Stores detailed event information in properties

## Entity Relationships
- Participant has many Agents (when acting as provider)
- Agent belongs to one Participant and one AgentType
- Agent handles many Services and processes many Jobs
- Service is of one ServiceType and may belong to a ServiceGroup
- Service can be linked to a consumer participant via ConsumerParticipantID
- ServiceGroup belongs to a specific Participant and has many Services
- Jobs are related to specific Agents and Services
- AgentType can provide various ServiceTypes (many-to-many)

---

# AUTHORIZATION SYSTEM

## Roles
- **fulcrum_admin**: System administrator with unrestricted access
- **participant**: Participant administrator with access to participant-specific resources
- **agent**: Agent role with access to jobs assigned to it

## Key Authorization Patterns
- fulcrum_admin generally has full access to all resources
- participant can manage its own participant and related agents/services
- agent can only claim and update jobs assigned to it
- Resources are scoped to specific participants or agents
- Participants can act as both providers (hosting agents/services) and consumers (consuming services)

---

# SERVICE MANAGEMENT

## State Transitions
- Creating → Created: Service is initially created
- Created → Starting: Service begins startup
- Starting → Started: Service is fully running
- Started → Stopping: Service begins shutdown
- Stopping → Stopped: Service is fully stopped
- Started → HotUpdating: Service update while running
- HotUpdating → Started: Hot update completed
- Stopped → ColdUpdating: Service update while stopped
- ColdUpdating → Stopped: Cold update completed
- Stopped → Deleting: Service begins deletion
- Deleting → Deleted: Service is fully removed

## Properties vs Attributes
- Properties: JSON data representing service configuration that can be updated (triggers state transitions)
- Attributes: Static metadata about the service set during creation (used for selection, identification, and filtering)

---

# JOB PROCESSING

## Job States
- Pending: Job created and waiting for an agent to claim it
- Processing: Job claimed by an agent and in progress
- Completed: Job successfully finished
- Failed: Job encountered an error
- Failed jobs may auto-retry after timeout

## Job Processing Flow
1. Service operation requested (create/start/stop/update/delete)
2. Job created in Pending state
3. Agent polls for pending jobs
4. Agent claims job (transitions to Processing)
5. Agent performs the operation
6. Agent updates job to Completed or Failed
7. Service state updated based on job outcome

---

# MONITORING & AUDIT

## Metrics Subsystem
- Collects performance data from agents and services
- Tracks resource utilization and health status
- Different metric types for different entity types (Agent, Service, Resource)
- Used for monitoring and reporting

## Audit Subsystem
- Records all system operations for accountability
- Created automatically by the backend (not a user action)
- Includes authority type, ID, operation type, and properties
- Created within the same transaction as data changes