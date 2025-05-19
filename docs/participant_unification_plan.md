# Plan: Unifying Broker and Provider into Participant Entity

**Objective:** Merge the `Broker` and `Provider` entities into a new, unified entity called `Participant`. A `Participant` can inherently act as both a provider and a consumer of services; its specific role in an interaction is determined by its relationships and context.

**I. Core Principles & Definitions**

1.  **`Participant` Entity:**
    *   A single entity replacing `Broker` and `Provider`.
    *   It will combine attributes from both.
    *   Its functional role (provider/consumer) is determined by context and relationships, not a fixed type field on the entity itself.

2.  **Token Scoping & Roles:**
    *   `AuthRole`s `RoleProviderAdmin` and `RoleBroker` will be consolidated into a new role, e.g., `RoleParticipant`.
    *   Tokens with this new role will be scoped to a `ParticipantID`.
    *   `AuthSubject`s `SubjectProvider` and `SubjectBroker` will be consolidated into `SubjectParticipant`.
    *   `AuthScope` will be updated to use `ParticipantID` instead of `ProviderID` and `BrokerID`.

3.  **Agent Relationship:**
    *   An `Agent` will now belong to a `Participant` (acting as the provider).

4.  **Service Relationship:**
    *   A `Service` is managed by an `Agent` (which belongs to a "providing" `Participant`).
    *   To represent a "consuming" or "brokering" relationship, an optional `BrokerParticipantID` will be added to the `Service` entity.

**II. Detailed Plan**

**Phase 1: Domain Layer Refactoring**

1.  **Create `Participant` Entity (`internal/domain/participant.go`):**
    *   **Struct `Participant`:**
        *   `BaseEntity`
        *   `Name`: `string` (not null)
        *   `CountryCode`: `CountryCode` (optional, from `Provider`)
        *   `Attributes`: `Attributes` (optional, type: jsonb, from `Provider`)
        *   `State`: `ParticipantState` (not null, e.g., "Enabled", "Disabled", from `Provider`)
        *   `Agents`: `[]Agent` (gorm relationship, `foreignKey:ParticipantID`)
    *   **Enum `ParticipantState`:** `"Enabled"`, `"Disabled"` with validation.
    *   **Functions:** `NewParticipant()`, `TableName() string`, `Validate() error`, `Update(...)`.

2.  **Update `Token` Entity (`internal/domain/token.go`):**
    *   **Fields:** Replace `ProviderID`, `Provider`, `BrokerID`, `Broker` with:
        *   `ParticipantID *UUID`
        *   `Participant *Participant` (gorm foreignKey: `ParticipantID`)
    *   **`NewToken()` Function:**
        *   For the new `RoleParticipant`, `scopeID` will refer to a `ParticipantID`. Validate `Participant` existence and set `token.ParticipantID`.
        *   For `RoleAgent`, `agent.ProviderID` (to be `agent.ParticipantID`) is copied to `token.ParticipantID`.
    *   **`Validate()` Method:**
        *   For `RoleParticipant`: Require `ParticipantID`.
        *   Remove cases for `RoleProviderAdmin` and `RoleBroker`.
        *   Adjust logic for `RoleAgent` to use `ParticipantID` from the agent.

3.  **Update `Agent` Entity (`internal/domain/agent.go`):**
    *   **Fields:** Replace `ProviderID`, `Provider` with:
        *   `ParticipantID UUID` (not null)
        *   `Participant *Participant` (gorm foreignKey: `ParticipantID`)
    *   Update related validation and logic.

4.  **Update `Service` Entity (`internal/domain/service.go`):**
    *   **Fields:** Add:
        *   `BrokerParticipantID *UUID` (optional, nullable)
        *   `BrokerParticipant *Participant` (gorm foreignKey: `BrokerParticipantID`)
    *   This field links a service to a participant acting as its broker/consumer.

5.  **Create `ParticipantCommander` (`internal/domain/participant_commander.go`):**
    *   Define interface and implementation (`participantCommander`).
    *   Adapt CRUD logic from `brokerCommander` and `providerCommander`.
    *   **Deletion Logic:**
        *   Check for dependent `Agent`s.
        *   Delete associated `Token`s via `TokenRepository.DeleteByParticipantID()`.

6.  **Create `ParticipantRepository` & `ParticipantQuerier`:**
    *   Define interfaces in `internal/domain/store.go`.
    *   Update `Store` interface: `ParticipantRepo() ParticipantRepository`.

7.  **Update `TokenRepository` Interface (`internal/domain/store.go`):**
    *   Remove `DeleteByProviderID()`, `DeleteByBrokerID()`.
    *   Add `DeleteByParticipantID(ctx context.Context, participantID UUID) error`.

8.  **Update `AuditEntry` Event Types (`internal/domain/audit_entry.go`):**
    *   Deprecate `Broker` and `Provider` event types.
    *   Add `EventTypeParticipantCreated`, `EventTypeParticipantUpdated`, `EventTypeParticipantDeleted`.
    *   Update `auditCommander` calls to use new types and `ParticipantID`.

9.  **Update Authorization (`internal/domain/auth.go`, `internal/domain/auth_rule.go`):**
    *   Modify `AuthRole` enum: Remove `RoleProviderAdmin`, `RoleBroker`; add `RoleParticipant`.
    *   Modify `AuthSubject` enum: Remove `SubjectProvider`, `SubjectBroker`; add `SubjectParticipant`.
    *   Modify `AuthScope` struct: Replace `ProviderID`, `BrokerID` with `ParticipantID`.
    *   Update `ValidateAuthScope` function.
    *   Update `AuthRule`s: Rules for old provider/broker roles/subjects will be mapped to the new `RoleParticipant` and `SubjectParticipant`.

**Phase 2: Database Layer Refactoring**

10. **GORM Model & Repository for `Participant` (`internal/database/gorm_repo_participant.go`):**
    *   Implement GORM struct and repository methods.

11. **Update GORM Models & Repositories:**
    *   `Token` (`internal/database/gorm_repo_token.go`): Update fields, implement `DeleteByParticipantID`.
    *   `Agent` (`internal/database/gorm_repo_agent.go`): Update fields.
    *   `Service` (`internal/database/gorm_repo_service.go`): Add `BrokerParticipantID`.

12. **Update `GormStore` (`internal/database/gorm_store.go`):**
    *   Add `ParticipantRepo()` implementation.

**Phase 3: API Layer Refactoring**

13. **API Handlers for `Participant` (`internal/api/handlers_participant.go`):**
    *   Create new handlers for `Participant` CRUD, adapting from old broker/provider handlers. Update DTOs.

14. **Update API Handlers:**
    *   `Token` (`internal/api/handlers_token.go`): Adapt to `ParticipantID` scoping.
    *   `Agent` (`internal/api/handlers_agent.go`): Update if creation/listing used provider details.
    *   `Service` (`internal/api/handlers_service.go`): Adapt if creation/update used broker details.

15. **Update API Middlewares (`internal/api/middlewares.go`):**
    *   Ensure auth checks use `Participant` scoping.

**Phase 4: Database Migrations**

16. **Database Schema Changes (No Data Migration Required):**
    *   Develop script(s) (e.g., using GORM migrations or raw SQL) for the following schema modifications:
    1.  **Create `participants` table:** Based on the `Participant` GORM model.
    2.  **Alter `tokens` table:**
        *   Add `participant_id UUID NULL` (or NOT NULL if appropriate for new tokens).
        *   Add Foreign Key constraint: `tokens.participant_id` REFERENCES `participants(id)`.
        *   Drop `provider_id` and `broker_id` columns.
    3.  **Alter `agents` table:**
        *   Rename `provider_id` column to `participant_id` (or add `participant_id`, copy data if any existing dev data needs preserving for FK, then drop `provider_id`). Ensure it's NOT NULL.
        *   Update/Add Foreign Key constraint: `agents.participant_id` REFERENCES `participants(id)`.
    4.  **Alter `services` table:**
        *   Add `broker_participant_id UUID NULL`.
        *   Add Foreign Key constraint: `services.broker_participant_id` REFERENCES `participants(id)`.
    5.  **Drop `brokers` and `providers` tables.**

**Phase 5: Testing & Documentation**

17. **Update/Create Unit & Integration Tests:** For all modified components.
18. **Update Documentation:**
    *   `README.md`, `docs/ARCHITECTURE.MD`, `docs/AUTH.MD`, `docs/DESIGN.MD`, `docs/openapi.yaml`.

**Mermaid Diagram of Key Entity Changes:**

```mermaid
graph TD
    subgraph Entities
        Participant[Participant <br> ID <br> Name <br> CountryCode (opt) <br> Attributes (opt) <br> State]
        Token[Token <br> ID <br> Name <br> Role (AuthRole) <br> HashedValue <br> ExpireAt <br> ParticipantID (fk) <br> AgentID (fk, opt)]
        Agent[Agent <br> ID <br> ... <br> ParticipantID (fk)]
        Service[Service <br> ID <br> ... <br> AgentID (fk) <br> BrokerParticipantID (fk, opt)]
    end

    Participant --o Agent : "has many (if acting as provider)"
    Participant --o Token : "has many (scoped tokens)"
    Agent --o Token : "has one (agent token)"
    Agent --o Service : "manages"
    Participant -- "Brokers/Consumes (optional)" Service : links to Service.BrokerParticipantID

    classDef entity fill:#f9f,stroke:#333,stroke-width:2px;
    class Participant,Token,Agent,Service entity;