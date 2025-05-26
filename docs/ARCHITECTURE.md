# Layer Architecture

## 1. Introduction

This document describe the layer architectural design of the system.

## 2. Clean Architecture Implementation

The system follows a clean architecture approach with clearly defined layers, each with specific responsibilities. This approach facilitates maintainability, testability, and the ability to adapt to changing requirements.

```mermaid
graph TD
    User[Client Applications] --> API[API Layer]
    API --> Domain[Domain Layer]
    Domain --> DB[Database Layer]
    Domain -.-> External[External Systems]
    
    subgraph "Dependency Direction"
        direction LR
        Outer[Outer Layers] --> Inner[Inner Layers]
    end
```

### 2.1 Design Principles

The architecture adheres to several key design principles:

- **Separation of Concerns**: Each layer has a specific responsibility
- **Dependency Inversion**: Dependencies point inward, with the domain layer at the core
- **Interface Segregation**: Smaller, focused interfaces for different concerns
- **Single Responsibility**: Each component has one reason to change
- **Clean Boundaries**: Each layer communicates through well-defined interfaces

## 3. System Layers

### 3.1 API Layer

The API layer is responsible for handling HTTP requests and providing a RESTful interface to clients. It:

- Defines routes and endpoints for all operations
- Converts between JSON/HTTP and domain objects
- Implements authentication and authorization through middleware
- Manages pagination and filtering of results
- Implements error handling and response formatting

#### 3.1.1 Middleware Architecture

The API layer uses a comprehensive middleware pattern that separates cross-cutting concerns from business logic:

**Authentication Middleware** ([`Auth`](internal/api/middlewares.go:23))
- Extracts and validates Bearer tokens from requests
- Adds authenticated identity to request context
- Rejects unauthenticated requests

**Authorization Middleware Pattern**
The system uses a flexible authorization middleware pattern based on [`AuthzFromExtractor`](internal/api/middlewares.go:120):

```go
// Base authorization middleware using scope extractors
AuthzFromExtractor(subject, action, authorizer, scopeExtractor)

// Specialized authorization middlewares:
AuthzSimple(subject, action, authorizer)              // No resource scope
AuthzFromID(subject, action, authorizer, querier)     // Resource ID-based
AuthzFromBody[T](subject, action, authorizer)         // Request body-based
```

**Request Processing Middleware**
- [`DecodeBody[T]()`](internal/api/middlewares.go:73): Type-safe request body decoding
- [`ID`](internal/api/middlewares.go:44): UUID extraction and validation from URL paths
- [`RequireAgentIdentity()`](internal/api/middlewares.go:238): Agent-specific authentication

#### 3.1.2 Handler Pattern

Each entity has its own handler following a consistent pattern:

```go
// Handler structure
type EntityHandler struct {
    querier   domain.EntityQuerier
    commander domain.EntityCommander
    authz     domain.Authorizer
}

// Routes with middleware chain
func (h *EntityHandler) Routes() func(r chi.Router) {
    return func(r chi.Router) {
        // List with simple authorization
        r.With(
            AuthzSimple(domain.SubjectEntity, domain.ActionRead, h.authz),
        ).Get("/", h.handleList)
        
        // Create with body decoding and authorization
        r.With(
            DecodeBody[CreateEntityRequest](),
            AuthzFromBody[CreateEntityRequest](domain.SubjectEntity, domain.ActionCreate, h.authz),
        ).Post("/", h.handleCreate)
        
        // Resource-specific routes
        r.Group(func(r chi.Router) {
            r.Use(ID) // Extract UUID for all sub-routes
            
            r.With(
                AuthzFromID(domain.SubjectEntity, domain.ActionRead, h.authz, h.querier),
            ).Get("/{id}", h.handleGet)
            
            r.With(
                DecodeBody[UpdateEntityRequest](),
                AuthzFromID(domain.SubjectEntity, domain.ActionUpdate, h.authz, h.querier),
            ).Patch("/{id}", h.handleUpdate)
        })
    }
}
```

#### 3.1.3 Request Types

Request types implement the [`AuthTargetScopeProvider`](internal/api/middlewares.go:203) interface for authorization:

```go
type CreateEntityRequest struct {
    Name       string      `json:"name"`
    ProviderID domain.UUID `json:"providerId"`
}

func (r CreateEntityRequest) AuthTargetScope() (*domain.AuthTargetScope, error) {
    return &domain.AuthTargetScope{ProviderID: &r.ProviderID}, nil
}
```

#### 3.1.4 Pure Handler Methods

Handler methods focus solely on business logic, with authentication/authorization handled by middleware:

```go
func (h *EntityHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
    // Get pre-validated request body
    req := MustGetBody[CreateEntityRequest](r.Context())
    
    // Execute business logic
    entity, err := h.commander.Create(r.Context(), req.Name, req.ProviderID)
    if err != nil {
        render.Render(w, r, ErrFromDomain(err))
        return
    }
    
    // Return response
    render.Status(r, http.StatusCreated)
    render.JSON(w, r, entity)
}
```

### 3.2 Domain Layer

The domain layer contains the core business logic, entities, and interfaces. It:

- Defines all business entities and their behavior
- Contains pure business logic without external dependencies
- Defines repository interfaces for data access
- Implements domain services for complex operations
- Uses value objects for domain concepts

Domain entities use a rich domain model approach with behaviors encapsulated in the entities themselves:

```go
// Example domain entity with behavior
type Service struct {
    ID           UUID
    Name         string
    CurrentState ServiceState
    TargetState  *ServiceState
    // Other properties...
    
    // Domain behavior
    Transition(targetState ServiceState) (*Action, error)
    Update(name *string, props *Properties) (bool, *Action, error)
}
```

The domain layer also defines `Commander` interfaces that encapsulate complex operations involving multiple entities:

```go
// Service command interface
type ServiceCommander interface {
    Create(ctx context.Context, params...) (*Service, error)
    Update(ctx context.Context, id UUID, params...) (*Service, error)
    Transition(ctx context.Context, id UUID, target ServiceState) (*Service, error)
    // Other operations...
}

// Commander implementation with transaction handling
type serviceCommander struct {
    store Store
    auditCommander AuditCommander
}

func (c *serviceCommander) Create(ctx context.Context, params...) (*Service, error) {
    // Create the entity
    entity := NewService(...)
    
    // Use store to handle transaction
    err := c.store.Atomic(ctx, func(s Store) error {
        // Multiple operations within a single transaction
        if err := s.ServiceRepo().Create(ctx, entity); err != nil {
            return err
        }
        
        // Create a related job
        job := NewJob(...)
        if err := s.JobRepo().Create(ctx, job); err != nil {
            return err
        }
        
        // Create audit entry
        if _, err := c.auditCommander.Create(ctx, ...); err != nil {
            return err
        }
        
        return nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return entity, nil
}
```

### 3.2.1 Transaction Management in Commands

The system implements a robust approach to transaction management through the Store interface:

1. **Store Interface**: Provides an `Atomic` method that executes a function within a transaction
2. **Transaction Boundaries**: Each command operation defines clear transaction boundaries
3. **All-or-Nothing Operations**: Multiple repository operations are executed atomically
4. **Consistent Audit Trail**: Audit entries are created within the same transaction as the data changes

This pattern ensures:
- Data consistency across related entities
- Proper error handling with automatic rollback
- Audit records that perfectly match actual data changes
- Clean, reusable transaction logic that isn't tied to specific repositories

The `Atomic` method abstracts the transaction mechanism, allowing the domain layer to define transaction boundaries without coupling to database-specific transaction implementations.

### 3.3 Database Layer

The database layer implements the repository interfaces defined in the domain layer. It:

- Provides concrete implementations of repositories
- Implements the Command-Query separation pattern
- Handles database-specific concerns and transactions
- Performs ORM mapping between domain entities and database models
- Manages database queries and optimizations

The layer employs a Command-Query Responsibility Separation (CQRS) inspired approach:

- **Querier interfaces** define read-only operations (queries)
- **Repository interfaces** embed queriers and add command (write) operations
- This separation allows for more focused and optimized read operations

The database layer uses a repository pattern with a store interface managing transaction boundaries:

```go
// Store interface
type Store interface {
    Atomic(ctx context.Context, fn func(Store) error) error
    EntityRepo() EntityRepository
    // Other repository getters...
}

// Repository implementation pattern
type EntityRepository interface {
    EntityQuerier // Embeds the querier interface
    
    // Command operations
    Create(ctx context.Context, entity *Entity) error
    Save(ctx context.Context, entity *Entity) error
    Delete(ctx context.Context, id UUID) error
}

// Querier implementation pattern (read-only operations)
type EntityQuerier interface {
    FindByID(ctx context.Context, id UUID) (*Entity, error)
    List(ctx context.Context, filter Filter) ([]*Entity, error)
    Exists(ctx context.Context, id UUID) (bool, error)
    // Other read-only methods...
}
```

### 3.4 Application Layer

The application layer serves as the entry point and wires together all components:

- Initializes and configures the system
- Manages application lifecycle
- Sets up middleware and request processing pipelines
- Establishes connections to databases and external systems
- Launches background workers for maintenance tasks
## 4. Project Structure Layout

The system follows a modular directory structure that reinforces the separation of concerns and clean architecture principles:

```
/
├── cmd/             # Application entry points
│   └── server/      # Main application entry point
├── docs/            # Documentation
├── internal/        # Private application and library code
│   ├── api/         # HTTP handlers and routes
│   ├── config/      # Configuration handling
│   ├── database/    # Database implementations of repositories
│   ├── domain/      # Domain models and repository interfaces
│   └── logging/     # Logging utilities
└── test/            # Test files
    └── rest/        # HTTP test files for API testing
```

### 4.1 Key Directories and Their Purposes

#### cmd/

The `cmd` directory contains the application entry points. Each subdirectory is a separate executable:

- `server/`: The main API server application
- Additional executables might include CLI tools, migration utilities, etc.

#### internal/

The `internal` directory contains all private application code, organized by architectural layer:

- `api/`: Contains all HTTP handlers, route definitions, middleware, and request/response models
  - Handler files are typically organized by domain entity
  - Middleware for authentication, logging, error handling
  - Response formatting utilities

- `domain/`: Contains the core business logic and entities
  - Entity definitions with behavior methods
  - Repository interfaces
  - Service/commander interfaces and implementations
  - Value objects and enums

- `database/`: Contains repository implementations
  - ORM-specific repository implementations
  - Database connection management
  - Migration utilities
  - Query builders

- `config/`: Contains configuration handling
  - Environment variable processing
  - Configuration struct definitions
  - Defaults and validation

- `logging/`: Contains logging utilities
  - Log formatting
  - Log level management
  - Context-aware logging helpers

#### test/

The `test` directory contains test-related files that don't fit within the standard Go package structure:

- `rest/`: Contains HTTP files for API testing
- May include fixtures, test data, and other test support files

### 4.2 Package Dependencies

The package dependencies follow the dependency rule of clean architecture, with outer layers depending on inner layers:

```
api → domain ← database
       ↑
     config
```

- `api` depends on `domain` to access entities and repository interfaces
- `database` depends on `domain` to implement repository interfaces
- `config` is used by multiple packages but doesn't depend on other packages
- `domain` doesn't depend on any other package

## 5. Testing Strategies

The system employs multiple testing approaches:

### 5.1 Unit Testing

- Domain entity tests verify business rules and state transitions
- Mock interfaces enable isolated component testing
- Table-driven tests cover edge cases and validation

### 5.2 Repository Testing

- Database tests using test helpers and utilities
- Transaction-based test cleanup
- Seeding test data for consistent starting points

### 5.3 API Testing

- Handler tests with mocked domain services
- End-to-end API tests using HTTP test files
- Authentication and authorization testing

### 5.4 Integration Testing

- Tests that validate the interaction between layers
- Full-stack tests that exercise the entire system
- Database migrations and schema validation tests

## 6. Conclusion

This layered architecture represents a clean, maintainable design that adheres to solid software engineering principles. The separation of concerns between API, domain, and database layers creates clear boundaries and responsibilities while enabling complex business operations to be executed reliably.

The architecture demonstrates how proper separation of concerns and well-defined interfaces enable each layer to evolve independently while maintaining a cohesive overall system. The command pattern approach provides a clean way to encapsulate business logic and ensure data consistency.

This architecture is designed to be maintainable and extensible while maintaining a consistent programming model that is easy to understand.