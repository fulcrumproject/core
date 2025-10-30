# Fulcrum Core Design

## Introduction

Fulcrum Core is a comprehensive cloud infrastructure management system designed to orchestrate and monitor distributed cloud resources across multiple providers. It serves as a centralized control plane for managing cloud service participants, their deployed agents, and the various services these agents provision and maintain.

The system is built as a RESTful API and enables organizations to:

- Manage multiple cloud service participants through a unified interface
- Track and control agents deployed across different cloud environments
- Provision and monitor various service types (VMs, containers, Kubernetes clusters, etc.)
- Organize services into logical groups for easier management
- Collect and analyze metrics from agents and services
- Maintain a detailed event log of all system activities
- Coordinate service operations with agents through a robust job queue system
- Securely store and manage sensitive service properties through encrypted vault storage

## Documentation Structure

This document provides a high-level overview of the Fulcrum Core system design. For more detailed information, please refer to:

- [ARCHITECTURE.md](ARCHITECTURE.md): Detailed description of the system's layered architecture, package structure, and implementation patterns
- [AUTHORIZATION.md](AUTHORIZATION.md): Comprehensive authorization rules and role-based permissions
- [SERVICE_TYPE.md](SERVICE_TYPE.md): ServiceType documentation including property schema validation and lifecycle schema
- [openapi.yaml](openapi.yaml): Complete API specification in OpenAPI format

## Context

Fulcrum Core serves as a central management plane for cloud infrastructure, interacting with various actors in the ecosystem. The following diagram illustrates the key actors and their relationships with the Fulcrum Core API:

```mermaid
graph TB
    FC((Fulcrum Core API))
    UI[Fulcrum Core UI]
    FA[Fulcrum Administrators]
    PA[Participant Administrators]
    
    %% Participants containing Agents and Services
    subgraph PART[Cloud Service Participants]
        AG[Agents]
        SVC[Cloud Services]
    end
    
    %% Relationships
    FA -->|Manage & Monitor| UI
    PA -->|Register & Configure| UI
    UI <-->|Interact| FC
    PA -->|Provision Services| FC
    FC -->|Deploy & Control| AG
    AG -->|Report Status & Metrics| FC
    AG -->|Provision & Manage| SVC
```
### Actors and Their Roles

#### Fulcrum Core UI

Fulcrum Core UI is the web interface that facilitates interaction between administrators and the Fulcrum Core API. It:

- Provides a graphical interface for system management and monitoring
- Translates user actions into API calls to the Fulcrum Core
- Displays system status, metrics, and analytics in an intuitive dashboard
- Offers role-based access control for different types of administrators

#### Agents

Agents are software components installed on Cloud Service Participants that act as Fulcrum's local representatives. They:

- Execute service provisioning and management commands from Fulcrum Core
- Report status, health metrics, and operational data back to Fulcrum
- Manage the lifecycle of deployed services (creation, updates, deletion)
- Handle local resource allocation and optimization
- Implement participant-specific operations and API interactions
- Maintain secure communications with the Fulcrum Core through token-based authentication
- Store and utilize instance-specific configuration parameters for customized behavior
- Poll for jobs from the job queue and process them
- Update job status upon completion or failure

#### Fulcrum Administrators

Fulcrum Administrators are responsible for the overall management of the Fulcrum ecosystem. They:

- Configure global system settings and policies
- Monitor the health and performance of the entire system
- Manage user access and permissions
- Review event logs and system metrics
- Orchestrate service groups across multiple participants
- Define service types and their resource requirements
- Oversee agent deployments and their operational status
- Monitor job queue health and processing

#### Participant Administrators

Participant Administrators manage specific participant instances within the Fulcrum system. They:

- Register and configure participant details in Fulcrum
- Deploy and initialize agents on their cloud infrastructure
- Manage participant-specific attributes and capabilities
- Monitor services running on their infrastructure
- Handle participant-specific authentication and access controls
- Coordinate with Fulcrum Administrators on cross-participant operations
- May act as both service providers and consumers depending on context

## Model

This section outlines the service entities and their relationships.

### Class Diagram

```mermaid
classDiagram
    Participant "1" --> "0..N" Agent : has many
    AgentType "0..N" <--> "1..N" ServiceType : can provide
    Agent "0..N" --> "1" AgentType : is of type
    Agent "1" --> "0..N" Service : handles
    Agent "1" --> "0..N" Job : processes
    Service "0..1" --> "1" ServiceType : is of type
    Service "1" --> "0..N" Job : related to
    ServiceGroup "1" --> "0..N" Service : groups
    MetricType "1" --> "0..N" MetricEntry : categorizes
    Participant "1" --> "0..N" Token : has many
    Participant "1" --> "0..N" ServiceOption : provides
    ServiceOptionType "1" --> "0..N" ServiceOption : categorizes
    Participant "1" --> "0..N" ServicePoolSet : provides
    ServicePoolSet "1" --> "0..N" ServicePool : contains
    ServicePool "1" --> "0..N" ServicePoolValue : contains
    ServicePoolValue "0..1" --> "1" Service : allocated to
    Agent "0..1" --> "1" ServicePoolSet : references
    Agent "1" --> "0..N" Token : has many
    Agent "1" --> "0..N" MetricEntry : generates
    Service "1" --> "0..N" MetricEntry : monitored via
    Participant "1" --> "0..N" Service : consumes (via ConsumerParticipantID)
    Participant "1" --> "0..N" ServiceGroup : owns
    Participant "1" --> "0..N" Service : hosts (via Agent)
    Event "0..N" <-- "1" Agent : created via
    Event "0..N" <-- "1" Service : created via
    Event "0..N" <-- "1" Participant : created via
    EventSubscription "1" --> "0..N" Event : tracks processing of

    namespace Participants {
        class Participant {
            id : properties.UUID
            name : string
            status : enum[Enabled|Disabled]
            createdAt : datetime
            updatedAt : datetime
        }

        class ServiceType {
            id : properties.UUID
            name : string
            propertySchema : CustomSchema
            lifecycleSchema : LifecycleSchema
            createdAt : datetime
            updatedAt : datetime
        }

        class AgentType {
            id : properties.UUID
            name : string
            createdAt : datetime
            updatedAt : datetime
        }

        class Agent {
            id : properties.UUID
            name : string
            status : enum[New|Connected|Disconnected|Error|Disabled]
            lastStatusUpdate : datetime
            tags : string[]
            configuration : json
            servicePoolSetId : properties.UUID
            createdAt : datetime
            updatedAt : datetime
        }
    }

    namespace Services {
        class Service {
            id : properties.UUID
            agentInstanceId : string
            name : string
            status : string
            properties : json
            agentInstanceData : json
            consumerParticipantID : properties.UUID
            createdAt : datetime
            updatedAt : datetime
        }

        class ServiceGroup {
            id : properties.UUID
            name : string
            participantID : properties.UUID
            createdAt : datetime
            updatedAt : datetime
        }

        class ServiceOptionType {
            id : properties.UUID
            name : string
            type : string
            description : string
            createdAt : datetime
            updatedAt : datetime
        }

        class ServiceOption {
            id : properties.UUID
            providerId : properties.UUID
            serviceOptionTypeId : properties.UUID
            name : string
            value : jsonb
            enabled : bool
            displayOrder : int
            createdAt : datetime
            updatedAt : datetime
        }

        class ServicePoolSet {
            id : properties.UUID
            name : string
            providerId : properties.UUID
            createdAt : datetime
            updatedAt : datetime
        }

        class ServicePool {
            id : properties.UUID
            name : string
            type : string
            propertyType : enum[string|integer|number|boolean|json]
            generatorType : enum[list|subnet]
            generatorConfig : jsonb
            servicePoolSetId : properties.UUID
            createdAt : datetime
            updatedAt : datetime
        }

        class ServicePoolValue {
            id : properties.UUID
            name : string
            value : jsonb
            servicePoolId : properties.UUID
            serviceId : properties.UUID
            propertyName : string
            allocatedAt : datetime
            createdAt : datetime
            updatedAt : datetime
        }

        class Job {
            id : properties.UUID
            providerId : properties.UUID
            consumerId : properties.UUID
            agentId : properties.UUID
            serviceId : properties.UUID
            action : string
            params : json
            status : enum[Pending,Processing,Completed,Failed]
            priority : int
            errorMessage : string
            claimedAt : datetime
            completedAt : datetime
            createdAt : datetime
            updatedAt : datetime
        }
    }

    namespace Security {
        class Token {
            id : properties.UUID
            name : string
            role : enum[admin|participant|agent]
            hashedValue : string
            expireAt : datetime
            participantID : properties.UUID
            agentID : properties.UUID
            createdAt : datetime
            updatedAt : datetime
        }
        
        class VaultSecret {
            id : properties.UUID
            reference : string
            encryptedValue : bytes
            createdAt : datetime
            updatedAt : datetime
        }
    }
    
    Service "1" --> "0..N" VaultSecret : may reference secrets

    namespace Metrics {
        class MetricEntry {
            id : properties.UUID
            createdAt : datetime
            agentId : properties.UUID        
            serviceId : properties.UUID        
            resourceId : string
            value : number
        }

        class MetricType {
            id : properties.UUID
            entityType : enum[Agent,Service,Resource] 
            name : string
            createdAt : datetime
            updatedAt : datetime
        }
    }

    namespace Domain Events {
        class Event {
            id : properties.UUID
            sequenceNumber : int64
            createdAt : datetime
            initiatorType : string
            initiatorId : string
            type : string
            payload : json
        }
        
        class EventSubscription {
            id : properties.UUID
            subscriberId : string
            instanceId : string
            lastProcessedSequence : int64
            leaseExpiresAt : datetime
            createdAt : datetime
            updatedAt : datetime
        }
    }

    note for Service "Service lifecycle is schema-driven:
    - Status field is a string (not hardcoded enum)
    - States and transitions defined by ServiceType's lifecycleSchema
    - Each service type can have custom lifecycle
    - Properties field for configuration
    - Jobs handle status transitions based on lifecycle schema"

    note for ServiceType "Service types include:
    - VM
    - Kubernetes Node
    - MicroK8s application
    - Kubernetes Cluster
    - Container Runtime services
    - Kubernetes Application controller
    
    Agents can provide specific service types
    based on their AgentType capabilities and
    tags for specialized requirements
    
    Property schemas can use serviceOption validator
    to validate against provider-specific options"
    
    note for ServiceOption "Service options provide dynamic validation
    lists for service properties:
    - ServiceOptionType defines global categories (os, region, etc.)
    - ServiceOption provides provider-specific values
    - Used via serviceOption validator in property schemas
    - Values are JSONB for flexibility (strings, objects, arrays)
    - Can be enabled/disabled without deletion
    - Display order controls presentation order"
    
    note for ServicePoolSet "Service pool sets organize resource pools:
    - Container for related pools belonging to a provider
    - Agents reference a pool set for automatic allocation
    - Supports organizing pools by environment, region, etc.
    - One pool set can contain multiple pools of different types"
    
    note for ServicePool "Service pools manage allocatable resources:
    - Each pool has a type (identifies what property it provides)
    - PropertyType defines the data type: string, integer, number, boolean, json
    - Must match the type of properties that reference it
    - Generator type determines allocation strategy:
      * list: Pre-configured values (manually added)
      * subnet: IP ranges with automatic CIDR allocation
    - Generator config stores type-specific settings (e.g., CIDR)
    - Referenced via pool generator in property definitions
    - Properties using pools require actor: [system] authorizer
    - Values stored as individual ServicePoolValue records"
    
    note for ServicePoolValue "Pool values are allocatable resources:
    - Values can be any JSON type (strings, objects, arrays)
    - Tracks allocation: serviceId, propertyName, allocatedAt
    - Created manually for list pools, automatically for subnet pools
    - Actual values copied directly into service properties
    - Released and marked available when service is deleted
    - No dereferencing - agents receive concrete values"
    
    note for Job "Jobs represent operations that agents
    perform on services. Actions are defined by the
    service type's lifecycle schema (e.g., create, start,
    stop, restart, update, delete).
    
    Jobs include provider/consumer context and parameters.
    
    Failed jobs use error message regexp matching to
    determine the next service state via lifecycle schema
    transitions. This enables custom error handling and
    state routing (e.g., quota errors vs network errors).
    
    Each job transitions service status based on the
    lifecycle schema definition"
    
    note for VaultSecret "Vault secrets provide secure storage
    for sensitive service properties:
    - AES-256-GCM encrypted storage
    - Properties marked as secret are stored as vault://reference
    - Two types: persistent (until deletion) and ephemeral (per job)
    - Only agents can resolve secrets via API endpoint
    - Automatic cleanup based on secret type
    - Only primitive types can be secrets
    - Nested secrets in objects/arrays supported"
    
```

#### Entities

##### Core

1. **Participant**
   - Unified entity replacing the separate Provider and Consumer entities
   - Represents an entity that can act as both a service provider and consumer
   - Has name and operational status (Enabled/Disabled)
   - Has many agents deployed within its infrastructure (when acting as a provider)
   - Can consume services (via Service.ConsumerParticipantID)
   - The functional role (provider/consumer) is determined by context and relationships

2. **Agent**
   - Deployed software component that manages services
   - Belongs to a specific Participant (acting as provider) and AgentType
   - Tracks connectivity status (New, Connected, Disconnected, Error, Disabled)
   - Uses secure token-based authentication (via Token entity)
   - Tracks last status update timestamp
   - Contains tags for capabilities and specializations
   - Stores instance-specific configuration parameters as JSON data
   - Processes jobs from the job queue to perform service operations
   - Selected for service provisioning based on service type and tag matching
   
   **Configuration Field:**
   - Optional JSON field that stores agent-specific configuration parameters
   - Allows agents to maintain instance-specific settings (timeouts, retries, environment variables, etc.)
   - Can be set during agent creation and updated via the REST API
   - Enables flexible agent behavior customization without code changes
   - Examples: connection timeouts, retry policies, environment-specific settings, feature flags

3. **Service**
   - Cloud resource managed by an agent
   - Schema-driven lifecycle with status field as a string
   - States and transitions defined by the ServiceType's lifecycleSchema
   - Each service type can define completely custom lifecycles
   - Supports custom actions (e.g., start, stop, restart, backup, maintenance)
   - Error handling uses regexp matching on error messages to determine next state
   - Stores service configuration in a single properties field
   - Stores service-specific resource configuration
   - Can be linked to a consumer participant via ConsumerParticipantID (optional)

   Properties:
   - Properties: properties.JSON data representing the service configuration that can be updated during the service lifecycle. Updates to properties trigger job creation for update operations.
   - Status: String field that must match a state defined in the ServiceType's lifecycleSchema

4. **AgentType**
   - Defines the type classification for agents
   - Many-to-many relationship with ServiceTypes
   - Determines which types of services an agent can manage

5. **ServiceType**
   - Defines the type classification for services
   - Includes optional propertySchema for service property validation
   - Includes optional lifecycleSchema defining states, actions, and transitions
   - Enables custom lifecycles per service type without code changes
   - Examples include VM, Container, Kubernetes nodes, Database, etc.

6. **ServiceGroup**
   - Organizes related services into logical groups
   - Belongs to a specific Participant
   - Enables collective management of related services

7. **Job**
   - Represents a discrete operation to be performed by an agent
   - Action field is a string defined by the ServiceType's lifecycleSchema
   - Common actions: create, start, stop, restart, update, delete, backup, etc.
   - Lifecycle statuses: Pending → Processing → Completed/Failed
   - Prioritizes operations for execution order
   - Tracks execution timing through claimedAt and completedAt
   - Records error details for failed operations
   - Error messages are matched against lifecycle regexps to determine next service state
   - Includes provider and consumer participant context
   - Contains parameters for the operation in params field

8. **Token**
   - Provides secure authentication mechanism for system access
   - Supports different roles: admin, participant, agent
   - Contains hashed value stored in database to verify authentication
   - Has expiration date for enhanced security
   - Scoped to specific Participant or Agent based on role
   - Used alongside or instead of OAuth/OIDC authentication depending on system configuration

9. **ServiceOptionType**
   - Global category defining a type of service option (e.g., "operating_system", "machine_type", "region")
   - Has name (display), type (unique identifier), and description
   - Admin-managed resource, not provider-specific
   - Used as validation categories in service type property schemas via `serviceOption` validator
   - Examples: os, machine_type, region, disk_type, network_tier

10. **ServiceOption**
   - Provider-specific option value for a ServiceOptionType
   - Belongs to one Participant (acting as provider) and one ServiceOptionType
   - Contains name, value (JSONB for flexibility), enabled flag, and display order
   - Value can be any JSON structure (strings, objects, arrays)
   - Provider-scoped authorization (admin, participant for own provider, agent for own provider)
   - Used in `serviceOption` validator in service type property schemas
   - Enables dynamic validation lists for service creation without code changes
   - Can be enabled/disabled to control availability without deletion

11. **ServicePoolSet**
   - Container for related service pools belonging to a provider
   - Belongs to one Participant (acting as provider)
   - Agents can reference a pool set to enable automatic resource allocation
   - Contains name for organization (e.g., "Production Pools", "Development Pools")
   - Supports organizing pools by environment, region, or other criteria
   - One pool set can contain multiple pools of different types

12. **ServicePool**
   - Defines a pool of allocatable resources (IPs, ports, hostnames, etc.)
   - Belongs to one ServicePoolSet
   - Has type (identifies what property it provides, e.g., "public_ip", "hostname")
   - Has propertyType (data type provided: string, integer, number, boolean, json)
   - PropertyType must match the type of service properties that reference this pool
   - Has name for human-readable identification
   - Generator type determines allocation strategy:
     * `list`: Pre-configured values manually added as ServicePoolValue records
     * `subnet`: IP ranges with automatic CIDR-based allocation
   - Generator config stores type-specific configuration (e.g., CIDR, excludeFirst, excludeLast for subnets)
   - Referenced via pool generator in property definitions
   - Type validation ensures property type matches pool's propertyType
   - Values stored as individual ServicePoolValue records (not JSON arrays)

13. **ServicePoolValue**
   - Individual allocatable value within a ServicePool
   - Value field can be any JSON type (strings, objects, arrays) for flexibility
   - Tracks allocation status:
     * serviceId: Which service this value is allocated to (null if available)
     * propertyName: Which property in the service uses this value
     * allocatedAt: When the allocation occurred
   - Created manually for `list` pools via API
   - Created automatically on-the-fly for `subnet` pools during service creation
   - Actual values are copied directly into service properties (no dereferencing)
   - Released and marked available when service is deleted
   - Cannot be deleted if currently allocated

##### Metrics

1. **MetricEntry**
   - Records individual metric measurements
   - Associated with specific Agent and Service
   - Identifies the measured resource through ResourceID
   - Stores numerical measurement value
   - Links to MetricType for classification

2. **MetricType**
   - Defines categories of metrics that can be collected
   - Specifies the entity type being measured (Agent, Service, or Resource)
   - Provides naming and classification for metrics

##### Events

1. **Event**
   - Tracks system events and changes for audit purposes
   - Records the initiator (type and ID) that initiated the action
   - Categorizes events by type
   - Stores detailed event information in properties
   - Has a sequence number for chronological ordering
   - Provides audit trail for system operations and changes

2. **EventSubscription**
   - Manages external system subscriptions to domain events
   - Tracks processing progress through lastProcessedSequence
   - Implements lease-based exclusive processing to prevent conflicts
   - Supports multiple instances of the same subscriber for high availability
   - Enables ordered event consumption through sequence-based fetching
   - Used by external systems to maintain consistent event processing state

##### Security

1. **Token**
   - Provides secure authentication mechanism for system access
   - Supports different roles: admin, participant, agent
   - Contains hashed value stored in database to verify authentication
   - Has expiration date for enhanced security
   - Scoped to specific Participant or Agent based on role

2. **VaultSecret**
   - Securely stores sensitive service property values using AES-256-GCM encryption
   - Properties marked as secret are stored with vault references (`vault://reference`)
   - Stores encrypted values that are only accessible via secure resolution endpoint
   - Two secret types:
     - **Persistent**: Long-lived secrets cleaned up on service deletion (API keys, credentials)
     - **Ephemeral**: Short-lived secrets cleaned up after each job completion (temporary passwords, tokens)
   - Only agents can resolve vault references to retrieve actual secret values
   - Encryption key configured via `VAULT_ENCRYPTION_KEY` environment variable
   - Automatic cleanup based on secret type and service lifecycle
   - Supports secrets in primitive types and nested within objects/arrays

Fulcrum Core implements a comprehensive authorization system with role-based access control (RBAC):

- Three predefined roles: admin, participant, and agent
- Fine-grained permission control for different resource types and actions
- Context-aware permissions based on resource ownership and relationships

The authentication system supports multiple authenticators that can be enabled via the `FULCRUM_AUTHENTICATORS` configuration:

- **Token Authentication**: Local token-based authentication using secure hashed tokens
- **OAuth/OIDC Authentication**: Integration with external OAuth 2.0/OpenID Connect providers (e.g., Keycloak)

The system can be configured to use one or both authentication methods simultaneously through a composite authenticator pattern. OAuth authentication supports JWT token validation with custom claims for role and scope extraction.

For detailed information about roles, permissions, and authorization rules, refer to [AUTHORIZATION.md](AUTHORIZATION.md).


## Technical Overview

Fulcrum Core is built with Go, leveraging its concurrency model and performance characteristics to handle distributed infrastructure management efficiently. The system follows clean architecture principles, with clear separation of domain logic, data access, and API layers. The core technology stack includes:

- **Backend**: Go with Chi router for RESTful API endpoints
- **Database**: PostgreSQL with GORM for object-relational mapping
- **Containerization**: Docker and Docker Compose for deployment

For detailed information about the system's architecture, layers, and implementation patterns, refer to [ARCHITECTURE.md](ARCHITECTURE.md).

### Services, Agents, and Jobs

#### Service Lifecycle Schema

Service lifecycles are now schema-driven rather than hardcoded. Each ServiceType defines its own lifecycle with custom states, actions, and transitions.

**Example: Simple VM Lifecycle**

```mermaid
stateDiagram-v2
    [*] --> New: service created
    New --> Stopped: create action
    Stopped --> Started: start action
    Started --> Stopped: stop action
    Started --> Started: update action (hot update)
    Stopped --> Stopped: update action (cold update)
    Stopped --> Deleted: delete action
    Started --> Deleted: delete action
```

**Note:** This is just one example. ServiceTypes can define completely different lifecycles:
- Different state names (e.g., Provisioning, Maintenance, Backup)
- Different actions (e.g., restart, pause, resume, snapshot)
- Error-driven transitions with regexp matching
- Multi-step actions with intermediate states

See [SERVICE_TYPE.md](SERVICE_TYPE.md) for comprehensive lifecycle schema documentation and examples.

#### Service Property Schema Validation

Fulcrum Core provides a flexible properties.JSON-based validation system for service properties through the Service Property Schema feature. This system ensures data integrity and consistency for service configurations while providing dynamic validation without requiring application recompilation.

See [SERVICE_TYPE.md](SERVICE_TYPE.md) for comprehensive property schema documentation and examples.


For detailed schema syntax and examples, see [SERVICE_TYPE.md](SERVICE_TYPE.md).

#### Agent Authentication Flow

```mermaid
sequenceDiagram
    participant Client
    participant API as Fulcrum Core API
    participant DB as Database
    
    Client->>API: POST /api/v1/agents
    API->>API: Generate secure random token
    API->>API: Hash token
    API->>DB: Create token entity for agent
    API->>DB: Store agent
    API->>Client: Return agent data with token (once only)
    Client->>Client: Securely store token
    
    Note over Client,API: Later API calls
    
    Client->>API: GET /api/v1/jobs/pending
    Note right of Client: With Authorization: Bearer <token>
    API->>API: Validate token by hashing and comparing
    API->>DB: Query jobs for authenticated agent
    API->>Client: Return pending jobs
```

#### Job Management Flow

Job statuses and transitions can be visualized as follows:

```mermaid
stateDiagram-v2
    [*] --> Pending: Job Created
    Pending --> Processing: Agent Claims Job
    Processing --> Completed: Operation Successful
    Processing --> Failed: Operation Error
    Completed --> [*]
    Failed --> [*]
```

**Note on Retrying**: Failed jobs are terminal (non-active). To retry an operation, users simply call the action endpoint again, which creates a new Pending job.

**Note:** When a job fails, the error message is matched against lifecycle transition regexps to determine the next service state. This enables intelligent error handling and state routing based on error types.

The job queue system manages the complete lifecycle of service operations from creation to completion. The following diagram illustrates the job management flow:

```mermaid
sequenceDiagram
    participant Client as Client/Admin UI
    participant API as Fulcrum Core API 
    participant Agent as Agent
    participant MS as Managed System

    %% Job Creation
    Client->>API: Request service operation (create/update/delete)
    API->>API: Update service status (transitioning)
    API->>API: Create pending job
    API-->>Client: Return response with job ID

    %% Job Discovery and Claiming
    Agent->>API: Poll for pending jobs (GET /jobs/pending)
    Note right of Agent: Uses token authentication
    API-->>Agent: Return list of pending jobs
    
    Agent->>API: Claim job (POST /jobs/{id}/claim)
    API->>API: Update job status to Processing
    API-->>Agent: Confirm job claimed

    %% Job Execution
    Agent->>MS: Execute required operation
    Note right of Agent: Create/start/stop/update/delete service

    %% Successful Completion Path
    alt Successful Operation
        MS-->>Agent: Operation succeeded
        Agent->>API: Complete job (POST /jobs/{id}/complete)
        API->>API: Update job status to Completed
        API->>API: Update service status
        API-->>Agent: Confirm job completed

    %% Failed Operation Path
    else Failed Operation
        MS-->>Agent: Operation failed with error
        Agent->>API: Fail job (POST /jobs/{id}/fail)
        API->>API: Update job status to Failed and record error
        API->>API: Update service state based on error message regexp matching
        Note right of API: Lifecycle schema determines next state
        API-->>Agent: Confirm job failed
    end

```

The job management process follows these steps:

1. **Job Creation**: 
   - When a service operation (create, update, delete) is initiated via the API
   - The ServiceCommander creates a job with status "Pending"
   - The job is assigned to the appropriate agent
   - Job contains all necessary data to perform the operation

2. **Job Polling and Claiming**:
   - Agents periodically poll `/api/v1/jobs/pending` for new jobs
   - When a job is available, the agent claims it using `/api/v1/jobs/{id}/claim`
   - The job status changes to "Processing"
   - A timestamp is recorded in the `claimedAt` field

3. **Job Processing**:
   - The agent performs the requested operation on the cloud participant
   - The agent maintains a secure connection with the job queue using token-based authentication
   - The service status remains stable during processing (job handles the transition logic)

4. **Job Completion**:
   - On successful completion, the agent calls `/api/v1/jobs/{id}/complete` with result data
   - The job status changes to "Completed"
   - A timestamp is recorded in the `completedAt` field
   - The service status is updated accordingly (Started, Stopped, Deleted)
   - For property updates, the service `properties` field is updated with the new configuration

5. **Job Failure Handling**:
   - If an operation fails, the agent calls `/api/v1/jobs/{id}/fail` with error details
   - The job status changes to "Failed"
   - The error message is matched against lifecycle transition regexps
   - The service transitions to the appropriate error state based on the match
   - This enables intelligent error handling (e.g., quota errors vs network errors)
   - Jobs may be automatically retried based on error type and configured policies

6. **Job Maintenance**:
   - Background workers periodically:
     - Release stuck jobs (processing too long)
     - Clean up old completed/failed jobs after retention period
     - Monitor queue health and performance metrics

### Vault Secrets Management

The vault secrets system provides secure storage and management of sensitive service properties using AES-256-GCM encryption. This system ensures that sensitive data like passwords, API keys, and tokens are never exposed in plain text through the API or database.

```mermaid
sequenceDiagram
    participant User
    participant API as Fulcrum Core API
    participant Vault as Vault Storage
    participant Agent
    participant Service as Cloud Service

    %% Secret Storage Flow
    User->>API: Create service with secret property
    Note right of User: password: "mysecretpass"
    API->>Vault: Store encrypted secret
    Vault-->>API: Return reference (vault://abc123)
    API->>API: Replace value with reference
    Note right of API: password: "vault://abc123"
    API-->>User: Service created

    %% Secret Resolution Flow
    Agent->>API: Get pending job
    API-->>Agent: Job with service properties
    Note right of Agent: password: "vault://abc123"
    Agent->>API: GET /vault/secrets/abc123
    API->>Vault: Retrieve encrypted secret
    Vault-->>API: Decrypted value
    API-->>Agent: Return actual value
    Note right of Agent: password: "mysecretpass"
    Agent->>Service: Use actual password

    %% Secret Cleanup (Ephemeral)
    Agent->>API: Complete job
    API->>Vault: Delete ephemeral secrets
    Note right of API: Cleanup after each job

    %% Secret Cleanup (Persistent)
    User->>API: Delete service
    API->>Vault: Delete all remaining secrets
    Note right of API: Cleanup on service deletion
```

#### Key Features

1. **Secure Storage**:
   - All secrets encrypted with AES-256-GCM before storage
   - Encryption key configured via `VAULT_ENCRYPTION_KEY` environment variable
   - Secrets never exposed in plain text through API responses

2. **Secret Types**:
   - **Persistent**: Long-lived secrets (API keys, credentials) cleaned up on service deletion
   - **Ephemeral**: Short-lived secrets (temporary passwords, one-time tokens) cleaned up after each job

3. **Agent Resolution**:
   - Agents retrieve actual secret values via `GET /api/v1/vault/secrets/{reference}`
   - Only agents can access the vault resolution endpoint (enforced by authentication middleware)
   - Agents receive decrypted values for use in provisioning operations

4. **Automatic Cleanup**:
   - Ephemeral secrets: Cleaned up after every job completion (success or failure)
   - Persistent secrets: Cleaned up when service reaches terminal state
   - All cleanup operations are best-effort (errors logged, don't fail operations)

5. **Restrictions**:
   - Only primitive types can be secrets (string, integer, number, boolean, json)
   - Objects and arrays cannot be secrets themselves, but can contain secret properties
   - Nested secrets in objects and arrays are fully supported

### Event Consumption API

The Event Consumption API provides external systems with a reliable mechanism to consume domain events in chronological order. This API implements lease-based exclusive processing to ensure events are processed exactly once, even in distributed environments with multiple consumer instances.

#### Event Lease Management Flow

```mermaid
sequenceDiagram
    participant ES as External System
    participant API as Fulcrum Core API
    participant DB as Database
    
    ES->>API: POST /api/v1/events/lease
    Note right of ES: Acquire lease and fetch events
    
    API->>DB: Find or create EventSubscription
    API->>DB: Check existing lease
    
    alt No active lease
        API->>DB: Acquire lease (update leaseExpiresAt)
        API->>DB: Fetch events from lastProcessedSequence
        API-->>ES: 200 OK with events and lease info
        
        Note over ES: Process events...
        
        ES->>API: POST /api/v1/events/ack
        Note right of ES: Acknowledge processed events
        API->>DB: Validate lease ownership
        API->>DB: Update lastProcessedSequence
        API-->>ES: 200 OK
        
    else Active lease held by different instance
        API-->>ES: 409 Conflict
        Note right of ES: Retry after lease expires
        
    else Active lease held by same instance
        API->>DB: Extend lease
        API->>DB: Fetch new events
        API-->>ES: 200 OK with events
    end
```

#### Key Features

1. **Ordered Processing**: Events are fetched in chronological order using sequence numbers
2. **Exclusive Leases**: Only one instance of a subscriber can hold a lease at a time
3. **Automatic Renewal**: Leases can be renewed by making subsequent lease API calls
4. **Progress Tracking**: Each subscription tracks the last processed sequence number
5. **Conflict Resolution**: Returns HTTP 409 when lease is held by another instance
6. **Configurable Duration**: Lease duration can be customized (default: 5 minutes, max: 1 hour)
7. **Separate Acknowledgement**: Events are acknowledged separately from lease acquisition (Option B)

#### API Endpoints

The Event Consumption API provides two main endpoints:

- `POST /api/v1/events/lease` - Acquire or renew a lease and fetch events
- `POST /api/v1/events/ack` - Acknowledge processed events and update progress

For detailed API specifications, request/response schemas, and authentication requirements, see [openapi.yaml](openapi.yaml).

### High-Availability Deployment

```mermaid
graph TB
    %% Internet entry point
    Internet((Internet Gateway))
    
    %% Region 1
    subgraph Region1[Region 1]
        LB1[Load Balancer]
        
        subgraph APICluster1[API Cluster]
            API1[API Instance 1]
            API2[API Instance 2]
        end
        
        PrimaryDB1[(PostgreSQL Primary<br/>Core Application Data)]
        MetricsDB1[(PostgreSQL/InfluxDB<br/>Metrics & Events)]
    end
    
    %% Region 2
    subgraph Region2[Region 2]
        LB2[Load Balancer]
        
        subgraph APICluster2[API Cluster]
            API3[API Instance 3]
            API4[API Instance 4]
        end
        
        PrimaryDB2[(PostgreSQL Replica<br/>Core Application Data)]
        MetricsDB2[(PostgreSQL/InfluxDB<br/>Metrics & Events)]
    end
    
    %% Agents
    Agent1[Agent 1]
    Agent2[Agent 2]
    Agent3[Agent 3]
    
    %% Connections
    Internet --> LB1 & LB2
    LB1 --> API1 & API2
    LB2 --> API3 & API4
    
    %% Core data connections
    API1 & API2 --> PrimaryDB1
    API3 & API4 -.Reads.-> PrimaryDB2
    API3 & API4 -.Writes.-> PrimaryDB1
    
    %% Metrics data connections
    API1 & API2 --> MetricsDB1
    API3 & API4 --> MetricsDB2
    
    %% Replication
    PrimaryDB1 -.Replication.-> PrimaryDB2
    MetricsDB1 -.Replication.-> MetricsDB2
    
    Agent1 & Agent2 & Agent3 --> Internet
```
