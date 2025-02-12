# Fulcrum Core Implementation Plan

## Phase 1: Domain Model Implementation

### 1. Base Entities and Common Types
```go
// Common types and interfaces
type State string
type UUID string
type Attributes map[string][]string
type JSON map[string]interface{}

// Base entity for common fields
type BaseEntity struct {
    ID        UUID      `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 2. Provider Domain Implementation

#### Step 1: Core Provider Entities
1. Provider Entity
```go
type ProviderState string
const (
    ProviderEnabled  ProviderState = "Enabled"
    ProviderDisabled ProviderState = "Disabled"
)

type Provider struct {
    BaseEntity
    Name        string
    State       ProviderState
    CountryCode string
    Attributes  Attributes
    Agents      []Agent
}
```

2. AgentType Entity
```go
type AgentType struct {
    BaseEntity
    Name          string
    ServiceTypes  []ServiceType `gorm:"many2many:agent_type_service_types"`
}
```

3. ServiceType Entity
```go
type ServiceType struct {
    BaseEntity
    Name                string
    ResourceDefinitions JSON
}
```

4. Agent Entity
```go
type AgentState string
const (
    AgentNew          AgentState = "New"
    AgentConnected    AgentState = "Connected"
    AgentDisconnected AgentState = "Disconnected"
    AgentError        AgentState = "Error"
    AgentDisabled     AgentState = "Disabled"
)

type Agent struct {
    BaseEntity
    Name        string
    State       AgentState
    TokenHash   string
    CountryCode string
    Attributes  Attributes
    Properties  JSON
    ProviderID  UUID
    AgentTypeID UUID
    Services    []Service
}
```

### 3. Services Domain Implementation

#### Step 2: Service Entities
1. Service Entity
```go
type ServiceState string
const (
    ServiceNew      ServiceState = "New"
    ServiceCreating ServiceState = "Creating"
    ServiceCreated  ServiceState = "Created"
    ServiceUpdating ServiceState = "Updating"
    ServiceUpdated  ServiceState = "Updated"
    ServiceDeleting ServiceState = "Deleting"
    ServiceDeleted  ServiceState = "Deleted"
    ServiceError    ServiceState = "Error"
)

type Service struct {
    BaseEntity
    Name          string
    State         ServiceState
    Attributes    Attributes
    Resources     JSON
    AgentID       UUID
    ServiceTypeID UUID
    GroupID       UUID
}
```

2. ServiceGroup Entity
```go
type ServiceGroup struct {
    BaseEntity
    Name     string
    Services []Service
}
```

### 4. Metrics Domain Implementation

#### Step 3: Metrics Entities
1. MetricEntry Entity
```go
type MetricEntry struct {
    BaseEntity
    AgentID    UUID
    ServiceID  UUID
    ResourceID string
    Value      float64
    TypeID     UUID
}
```

2. MetricType Entity
```go
type EntityType string
const (
    EntityTypeAgent    EntityType = "Agent"
    EntityTypeService  EntityType = "Service"
    EntityTypeResource EntityType = "Resource"
)

type MetricType struct {
    BaseEntity
    EntityType EntityType
    Name       string
}
```

### 5. Audit Domain Implementation

#### Step 4: Audit Entities
```go
type AuditEntry struct {
    BaseEntity
    AuthorityType string
    AuthorityID   string
    Type          string
    Properties    JSON
}
```

## Phase 2: Repository Layer Implementation

### Step 5: Repository Interfaces
Create interfaces for each domain entity:

```go
type Repository[T any] interface {
    Create(ctx context.Context, entity *T) error
    Update(ctx context.Context, entity *T) error
    Delete(ctx context.Context, id UUID) error
    FindByID(ctx context.Context, id UUID) (*T, error)
    List(ctx context.Context, filters map[string]interface{}) ([]T, error)
}

// Specific repositories with additional methods as needed
type ProviderRepository interface {
    Repository[Provider]
    FindByCountryCode(ctx context.Context, code string) ([]Provider, error)
}

type AgentRepository interface {
    Repository[Agent]
    FindByProvider(ctx context.Context, providerID UUID) ([]Agent, error)
    UpdateState(ctx context.Context, id UUID, state AgentState) error
}

type ServiceRepository interface {
    Repository[Service]
    FindByAgent(ctx context.Context, agentID UUID) ([]Service, error)
    UpdateState(ctx context.Context, id UUID, state ServiceState) error
}
```

## Phase 3: Service Layer Implementation

### Step 6: Service Layer
Implement business logic services for each domain:

```go
type ProviderService interface {
    CreateProvider(ctx context.Context, provider *Provider) error
    UpdateProviderState(ctx context.Context, id UUID, state ProviderState) error
    AssignAgentToProvider(ctx context.Context, providerID, agentID UUID) error
}

type AgentService interface {
    CreateAgent(ctx context.Context, agent *Agent) error
    UpdateAgentState(ctx context.Context, id UUID, state AgentState) error
    AssignServiceToAgent(ctx context.Context, agentID, serviceID UUID) error
}

type ServiceManager interface {
    CreateService(ctx context.Context, service *Service) error
    UpdateServiceState(ctx context.Context, id UUID, state ServiceState) error
    AssignToGroup(ctx context.Context, serviceID, groupID UUID) error
}

type MetricsService interface {
    RecordMetric(ctx context.Context, entry *MetricEntry) error
    GetMetrics(ctx context.Context, filters map[string]interface{}) ([]MetricEntry, error)
}

type AuditService interface {
    RecordAudit(ctx context.Context, entry *AuditEntry) error
    GetAuditTrail(ctx context.Context, filters map[string]interface{}) ([]AuditEntry, error)
}
```

## Phase 4: API Implementation

### Step 7: REST API Endpoints
Implement RESTful endpoints for each domain:

1. Provider API
   - POST /api/v1/providers
   - GET /api/v1/providers
   - GET /api/v1/providers/{id}
   - PUT /api/v1/providers/{id}
   - DELETE /api/v1/providers/{id}
   - GET /api/v1/providers/{id}/agents

2. Agent API
   - POST /api/v1/agents
   - GET /api/v1/agents
   - GET /api/v1/agents/{id}
   - PUT /api/v1/agents/{id}
   - DELETE /api/v1/agents/{id}
   - PUT /api/v1/agents/{id}/state

3. Service API
   - POST /api/v1/services
   - GET /api/v1/services
   - GET /api/v1/services/{id}
   - PUT /api/v1/services/{id}
   - DELETE /api/v1/services/{id}
   - PUT /api/v1/services/{id}/state

4. Metrics API
   - POST /api/v1/metrics
   - GET /api/v1/metrics
   - GET /api/v1/metrics/{id}

5. Audit API
   - GET /api/v1/audit
   - GET /api/v1/audit/{id}

## Implementation Order

1. **Week 1: Core Domain Implementation**
   - Implement base entities and common types
   - Implement Provider domain entities
   - Implement Service domain entities
   - Set up database migrations

2. **Week 2: Repository Layer**
   - Implement repository interfaces
   - Implement GORM repositories
   - Add database indexes
   - Implement basic CRUD operations

3. **Week 3: Service Layer**
   - Implement business logic services
   - Add validation rules
   - Implement state management
   - Add event handling

4. **Week 4: API Layer**
   - Implement REST API endpoints
   - Add request/response DTOs
   - Add input validation
   - Implement error handling

5. **Week 5: Testing & Documentation**
   - Add unit tests
   - Add integration tests
   - Add API documentation
   - Add monitoring and metrics

## Next Steps

1. Review and approve domain model implementation
2. Set up development environment with PostgreSQL
3. Create initial database migration
4. Begin implementation of core domain entities

Would you like to proceed with this implementation plan? We can then switch to Code mode to start implementing the core domain entities.