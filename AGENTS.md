# AI Agent Guidelines for Fulcrum Core

This document contains project-specific guidelines and technical context for working on the Fulcrum Core codebase.

---

## Project Status

- This project is NOT in production yet
- Breaking changes are acceptable and often preferred for better design
- Do NOT implement backward compatibility unless explicitly requested
- We do NOT need migrations, release plans, retrocompatibility, deprecation notices, or other production-related overhead
- Focus on building the right solution, not on managing transitions from old solutions

---

## Code Standards

### Go Language

- Use short and clear names in the code
- Use `any` and not `interface{}` in function signatures
- Unused imports are removed automatically by the IDE

### API Conventions

- JSON field names use camelCase (e.g., `providerId`, `serviceType`, `agentInstanceId`)

### Documentation

- Mermaid diagrams should not contain styles
- All code files MUST start with a brief 2-line comment explaining what the file does

### OpenAPI Specification

- API specification is in `docs/openapi/` using OpenAPI 3.1.0
- Split into multiple files for maintainability:
  - `openapi.yaml` - Main entry point
  - `components/schemas/*.yaml` - Schema definitions grouped by domain
  - `components/responses.yaml` - Reusable response definitions
  - `paths/*.yaml` - Path definitions (one file per endpoint)
- Validate changes: `npx @redocly/cli lint docs/openapi/openapi.yaml`
- See `docs/openapi/README.md` for details on structure and workflow

### Maintaining AGENTS.md

- This file is for **AI agents** working on the codebase
- Keep updates **concise** - this is a guidelines document, not detailed API documentation
- Provide high-level overviews and reference detailed docs

---

## Database Management

- We don't need database migrations - we use GORM migration

---

## Testing

### Mock Generation

- We use **mockery** to generate mocks for interfaces
- Configuration is in `.mockery.yml`
- To regenerate all mocks after interface changes, run: `mockery`
- Mocks are generated in separate `mocks/` packages:
  - `pkg/domain/mocks/` - Domain interface mocks
  - `pkg/auth/mocks/` - Auth interface mocks
- Import in tests: `import "github.com/fulcrumproject/core/pkg/domain/mocks"`
- Use alias for auth mocks: `import authmocks "github.com/fulcrumproject/core/pkg/auth/mocks"`
- Generated mocks support testify EXPECT() pattern for type-safe test expectations
- Always regenerate mocks after changing interface signatures

---

## System Architecture

### Overview
This system follows a clean architecture approach with clearly defined layers (API, Domain, Database) that maintain strict dependency rules. Dependencies point inward toward the domain layer, which contains business logic independent of external frameworks.

### Key Design Principles
- Separation of Concerns: Each layer has a specific responsibility
- Dependency Inversion: Dependencies point inward toward the domain core
- Interface Segregation: Small, focused interfaces for different concerns
- Single Responsibility: Each component has one reason to change
- Clean Boundaries: Layers communicate through well-defined interfaces

### Layer Structure

#### API Layer
- Handles HTTP requests through RESTful endpoints
- Converts between JSON/HTTP and domain objects
- Implements authentication and authorization through middleware chain
- Manages pagination and response formatting
- Uses handlers organized by domain entity
- No direct database access; works through domain interfaces

##### Middleware Architecture
- Auth middleware validates tokens and adds identity to context
- Authorization uses AuthzFromExtractor base pattern with specialized extractors:
  - AuthzSimple: No resource scope needed
  - AuthzFromID: Extracts scope from resource ID
  - AuthzFromBody: Extracts scope from request body
- DecodeBody[T] provides type-safe request body handling
- ID middleware extracts and validates UUIDs from URL paths
- RequireAgentIdentity ensures agent-specific authentication

##### Handler Patterns
- Routes use middleware chains for cross-cutting concerns
- Request types implement AuthTargetScopeProvider interface
- Handler methods focus on pure business logic
- Authentication/authorization handled entirely by middleware
- Use MustGetBody[T] and MustGetID for type-safe context access

#### Domain Layer
- Contains core business logic and entities with behavior
- Defines repository interfaces for data access
- Implements domain services through Commanders
- Uses value objects for domain concepts
- Has no external dependencies

#### Transaction Management
- Store interface provides Atomic method
- Commands use Store.Atomic for transaction boundaries
- Multiple repository operations execute within single transaction
- Ensures data consistency, audit trail, and proper error handling

#### Database Layer
- Implements repository interfaces defined in domain
- Uses Command-Query separation pattern
- Handles database operations and transaction management
- Maps between domain entities and database models
- Optimizes database queries and performance

### Package Structure
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

### Repository Pattern
- EntityRepository interfaces handle write operations
- EntityQuerier interfaces handle read-only operations
- Repositories embed querriers (CQRS-inspired)
- Store interface manages repositories and transactions

### Command Pattern
- Commander interfaces define complex operations
- Commands handle validation, entity creation, and business logic
- Use Store.Atomic to manage transaction boundaries
- Create audit entries within transaction boundaries

### Testing Strategies
- Unit tests for domain entities and business rules
- Repository tests with database test helpers
- Handler tests focus on business logic with simulated middleware context
- Middleware tests verify authorization logic in isolation
- Integration tests verify complete request flow with middleware chain
- End-to-end tests across layers

#### Handler Test Patterns
- Simulate middleware context: decoded bodies, extracted IDs, auth identity
- Test pure business logic without authorization concerns
- Use MustGetBody[T] and MustGetID with mocked context values
- Focus on domain errors and validation scenarios

---

## Domain Model & Business Rules

### System Overview

#### Purpose
Fulcrum Core is a comprehensive cloud infrastructure management system designed to orchestrate and monitor distributed cloud resources across multiple participants. It serves as a centralized control plane for managing cloud service participants, their deployed agents, and the various services these agents provision and maintain.

#### Key Capabilities
- Manage multiple cloud service participants through a unified interface
- Track and control agents deployed across different cloud environments
- Provision and monitor various service types (VMs, containers, Kubernetes clusters, etc.)
- Organize services into logical groups for easier management
- Collect and analyze metrics from agents and services
- Maintain a comprehensive audit trail of all system operations
- Coordinate service operations with agents through a robust job queue system
- Securely store and manage sensitive service properties through encrypted vault storage

### Core Entities

#### Participant
- Represents an entity that can act as both a service provider and consumer
- Has name and operational state (Enabled/Disabled)
- Contains geographical information via country code
- Stores flexible metadata through custom attributes
- Has many agents deployed within its infrastructure (when acting as a provider)
- Can consume services (via Service.ConsumerParticipantID)
- The functional role (provider/consumer) is determined by context and relationships

#### Agent
- Deployed software component that manages services
- Belongs to a specific Participant (acting as provider) and AgentType
- Tracks connectivity state (New, Connected, Disconnected, Error, Disabled)
- Uses secure token-based authentication
- Processes jobs from the job queue to perform service operations

#### Service
- Cloud resource managed by an agent
- State machine driven by ServiceType's lifecycle schema (not hardcoded enums)
- Status field is a string that matches a state in the lifecycle schema
- State transitions determined by lifecycle schema actions and transitions
- Supports error-driven state transitions based on error message regexp matching
- Has properties (configuration that can be updated) and attributes (static metadata)
- Can be linked to a consumer participant via ConsumerParticipantID

#### ServiceGroup
- Organizes related services into logical groups
- Belongs to a specific Participant
- Enables collective management of related services

#### Job
- Represents a discrete operation to be performed by an agent
- Actions are strings defined by the ServiceType's lifecycle schema (e.g., "create", "start", "stop", "delete")
- States include: Pending, Processing, Completed, Failed
- Prioritizes operations for execution order
- Tracks execution timing and error messages
- Error messages are used for lifecycle transition regexp matching (not stored separately as error codes)

#### Token
- Provides secure authentication mechanism for system access
- Supports different roles: fulcrum_admin, participant, agent
- Contains hashed value stored in database to verify authentication
- Has expiration date for enhanced security
- Scoped to specific Participant or Agent based on role

#### MetricEntry & MetricType
- Record and categorize performance metrics for agents and services
- Track numerical measurements with timestamps
- Associate measurements with specific resources

#### AuditEntry
- Tracks system events and changes for audit purposes
- Records the authority (type and ID) that initiated the action
- Categorizes events by type
- Stores detailed event information in properties

#### ServiceOptionType
- Global category defining a type of service option (e.g., "operating_system", "machine_type", "region")
- Has a name (display), type (unique identifier), and description
- Admin-managed resource (not provider-specific)
- Used as validation categories in service type property schemas

#### ServiceOption
- Provider-specific option value for a ServiceOptionType
- Contains name, value (JSONB), enabled flag, and display order
- Provider-scoped authorization (admin, participant for own provider, agent for own provider)
- Values can be any JSON structure (strings, objects, arrays)
- Used in `serviceOption` validator in service type property schemas

#### ServicePoolSet
- Container for related service pools belonging to a provider
- Agents reference a pool set to enable automatic resource allocation
- Contains name and provider reference
- Supports organizing pools by environment, region, or other criteria

#### ServicePool
- Defines a pool of allocatable resources (IPs, ports, hostnames, etc.)
- Belongs to a ServicePoolSet
- Has type (identifies what property it provides), name, and propertyType
- PropertyType defines the data type provided: string, integer, number, boolean, or json (must match property definitions)
- Generator type determines allocation strategy: `list` (pre-configured values) or `subnet` (IP ranges)
- Generator config stores type-specific configuration (e.g., CIDR for subnets)
- Referenced in property definitions via `servicePoolType` field

#### ServicePoolValue
- Individual allocatable value within a ServicePool
- Can be any JSON type (string, number, object, array)
- Tracks allocation status: serviceId, propertyName, allocatedAt
- Created manually for `list` pools, automatically for `subnet` pools
- Released when service is deleted (marked available for reuse)

#### Vault Secrets System
- Securely stores sensitive service property values using AES-256-GCM encryption
- Properties can be marked as secrets in the service type property schema
- When a property is marked as secret, user provides the actual value, system stores it in vault and replaces it with `vault://reference` in the service properties
- Two secret types:
  - **Persistent**: Cleaned up when service reaches terminal state (e.g., API keys, credentials)
  - **Ephemeral**: Cleaned up after each job completion (e.g., temporary passwords, one-time tokens)
- Only primitive types can be secrets (string, integer, number, boolean, json), not objects or arrays
- However, object properties and array items can contain secret properties
- Agents resolve secrets via `GET /api/v1/vault/secrets/{reference}` endpoint
- Vault interface defined in `pkg/schema`, implementation in `pkg/database`
- Encryption key configured via `VAULT_ENCRYPTION_KEY` environment variable
- All cleanup operations are best-effort (errors logged, don't fail operations)

### Entity Relationships
- Participant has many Agents (when acting as provider)
- Participant has many ServiceOptions (when acting as provider)
- Participant has many ServicePoolSets (when acting as provider)
- Agent belongs to one Participant and one AgentType
- Agent may reference a ServicePoolSet for automatic resource allocation
- Agent handles many Services and processes many Jobs
- Service is of one ServiceType and may belong to a ServiceGroup
- Service can be linked to a consumer participant via ConsumerParticipantID
- ServiceGroup belongs to a specific Participant and has many Services
- ServiceOption belongs to one Participant (provider) and one ServiceOptionType
- ServicePoolSet belongs to one Participant (provider) and contains many ServicePools
- ServicePool belongs to one ServicePoolSet and contains many ServicePoolValues
- ServicePoolValue belongs to one ServicePool and may be allocated to one Service
- Jobs are related to specific Agents and Services
- AgentType can provide various ServiceTypes (many-to-many)

### Authorization System

#### Roles
- **fulcrum_admin**: System administrator with unrestricted access
- **participant**: Participant administrator with access to participant-specific resources
- **agent**: Agent role with access to jobs assigned to it

#### Key Authorization Patterns
- fulcrum_admin generally has full access to all resources
- participant can manage its own participant and related agents/services
- agent can only claim and update jobs assigned to it
- Resources are scoped to specific participants or agents
- Participants can act as both providers (hosting agents/services) and consumers (consuming services)

### Service Management

#### Lifecycle Schema
Services use a schema-driven state machine defined in `ServiceType.LifecycleSchema`. Each service type can have a completely custom lifecycle.

**Schema Structure:**
- **States**: List of valid states (e.g., "New", "Started", "Stopped", "Deleted")
- **Actions**: Operations that can be performed (e.g., "create", "start", "stop", "delete")
- **InitialState**: Starting state for new services
- **TerminalStates**: States where no further actions are allowed
- **RunningStates**: States considered "running" for uptime calculation

**Transitions:**
Each action defines transitions between states:
- **from**: Source state
- **to**: Destination state
- **onError**: Whether this transition handles errors (boolean)
- **onErrorRegexp**: Optional regex to match specific error messages

**Error Handling:**
- When a job fails, the error message is matched against transition regexps
- If a transition has `onError: true` and its regexp matches, that transition is used
- If no regexp is specified, the transition matches any error
- This allows routing to different states based on specific error types (e.g., "quota exceeded" vs "network error")

**Example:**
```json
{
  "states": [{"name": "New"}, {"name": "Started"}, {"name": "Failed"}],
  "actions": [
    {
      "name": "start",
      "transitions": [
        {"from": "New", "to": "Started"},
        {"from": "New", "to": "Failed", "onError": true, "onErrorRegexp": "quota.*exceeded"},
        {"from": "New", "to": "Stopped", "onError": true}
      ]
    }
  ],
  "initialState": "New",
  "terminalStates": ["Deleted"],
  "runningStates": ["Started"]
}
```

#### State Transitions
State transitions are driven by the lifecycle schema, not hardcoded. Common patterns:
- Creating → Created: Service is initially created
- Created → Starting: Service begins startup
- Starting → Started: Service is fully running
- Started → Stopping: Service begins shutdown
- Stopping → Stopped: Service is fully stopped
- Failed states: Services transition to error states based on error message patterns

The actual states and transitions depend on the ServiceType's lifecycle schema.

#### Properties vs Attributes vs AgentInstanceData
- **Properties**: Service configuration with authorization and updatability constraints
  - **Actor authorizers**: Control who can set/update (defaults to user, explicit for agent/system)
  - **State authorizers**: Control when properties can be updated (based on service state)
  - **Immutable**: Boolean flag to prevent any updates after creation
- **Attributes**: Static metadata set during creation (for selection/filtering)
- **AgentInstanceData**: Agent-owned runtime data and technical infrastructure info

#### Property Types
Service properties support multiple types including:
- Basic types: `string`, `integer`, `number`, `boolean`
- Complex types: `object`, `array`, `json`
- Special types: `serviceReference`
- **JSON type**: Accepts any valid JSON value without schema validation
  - Used for pool values and options that can be strings, objects, or arrays
  - Backend validation ensures valid JSON structure
  - Example: IP pools may use `{"ip": "192.168.1.10", "gateway": "192.168.1.1"}` or simple strings

#### Property Secrets
Properties can be marked as secrets for secure storage:
- **Secret Field**: Property definition includes `secret` object with `type` field
- **Secret Types**:
  - `persistent`: Stored until service deletion (API keys, long-lived credentials)
  - `ephemeral`: Deleted after each job completion (temporary tokens, one-time passwords)
- **Storage**: User provides actual value → system stores in encrypted vault → property contains `vault://reference`
- **Resolution**: Agents call `GET /api/v1/vault/secrets/{reference}` to retrieve actual value
- **Restrictions**: Only primitive types (string, integer, number, boolean, json) can be secrets
- **Cleanup**: Ephemeral secrets cleaned after every job; persistent secrets cleaned on service deletion

#### Property Validators
Properties can have validators including:
- **serviceOption**: Validates against provider-managed option lists
  - Requires ServiceOptionType in validator value
  - Provides dynamic dropdowns and validation

#### Property Pool Allocation
Properties with `actor: ["system"]` authorizer can use automatic pool allocation via generators:
- **Generator**: Automatic value generation (e.g., pool allocation)
- **Actor authorizer**: Restricts property to system-only (prevents manual setting)
- **servicePoolType** in generator config: Specifies which pool type to allocate from (matches ServicePool.Type)
  - Property type must match pool's propertyType (e.g., string property → string pool)
  - System validates type compatibility during service creation
  - Actual values copied directly into properties during service creation
  - Values released and marked available when service is deleted
  - No dereferencing needed - agents receive concrete values

**Example:**
```json
{
  "publicIp": {
    "type": "string",
    "immutable": true,
    "authorizers": [
      {
        "type": "actor",
        "config": {
          "actors": ["system"]
        }
      }
    ],
    "generator": {
      "type": "pool",
      "config": {
        "poolType": "public_ip"
      }
    }
  }
}
```

### Job Processing

#### Job States
- **Pending**: Job created and waiting for an agent to claim it
- **Processing**: Job claimed by an agent and in progress
- **Completed**: Job successfully finished (terminal, non-active)
- **Failed**: Job encountered an error (terminal, non-active)
  - Error message drives service state transition via lifecycle schema regexp matching
  - Failed jobs do not block new attempts - users can retry by calling the action again

#### Job Processing Flow
1. Service operation requested (create/start/stop/update/delete)
2. Job created in Pending state
3. Agent polls for pending jobs
4. Agent claims job (transitions to Processing)
5. Agent performs the operation
6. Agent updates job to Completed or Failed (optionally including agent properties)
7. Service state updated based on job outcome:
   - **On success**: Service transitions to the success state defined in lifecycle
   - **On failure**: Error message is matched against lifecycle transition regexps to determine next state

#### Agent Property Updates

Agents can update service properties when completing a job by including a `properties` field in the completion request. This allows agents to report discovered values like IP addresses, instance IDs, and other infrastructure details.

**When to Use `properties` vs `agentInstanceData`:**
- **`properties`**: Configuration values that are part of the service schema
  - Validated against the ServiceType's property schema
  - Subject to actor and state authorization constraints
  - Become part of the service's configuration
  - Examples: ipAddress, port, instanceId, hostname
  - Use when: The value is defined in the service type's property schema with `actor: ["agent"]` authorizer

- **`agentInstanceData`**: Technical infrastructure information
  - Not validated against property schema
  - Can be any arbitrary JSON structure
  - Used for monitoring and resource tracking
  - Examples: CPU usage, memory allocation, disk I/O stats
  - Use when: Reporting runtime metrics or technical details not in the schema

**Property Validation:**
- Agents can only set/update properties with `actor: ["agent"]` authorizer
- During initial service creation: Agents can set immutable properties marked for agent access
- During subsequent updates: Both actor and state authorization constraints are validated
- Validation errors return HTTP 400 and roll back the entire job completion
- See `docs/SERVICE_TYPE.md` for detailed examples and error messages

### Monitoring & Audit

#### Metrics Subsystem
- Collects performance data from agents and services
- Tracks resource utilization and health status
- Different metric types for different entity types (Agent, Service, Resource)
- Used for monitoring and reporting

#### Audit Subsystem
- Records all system operations for accountability
- Created automatically by the backend (not a user action)
- Includes authority type, ID, operation type, and properties
- Created within the same transaction as data changes

