# Service Architecture Class Diagram

This document outlines the service architecture class relationships and their key components.

## Class Diagram

```mermaid
classDiagram
    Provider "1" --> "0..N" Agent : has many
    AgentType "0..N" --> "1..N" ServiceType : can provide
    Agent "0..N" --> "1" AgentType : is of type
    Agent "1" --> "0..N" Service : handles
    Service "0..1" --> "1" ServiceType : is of type
    ServiceGroup "1" --> "0..N" Service : groups many
    MetricEntry "1" --> "0..N" MetricType : is of type

    namespace Providers {
        class Provider {
            id : UUID
            name : string
            state : enum[Enabled|Disabled]
            attributes : map[string]string[] 
            createdAt : datetime
            updatedAt : datetime
        }

        class ServiceType {
            id : UUID
            name : string
            resourceDefinitions : json
            createdAt : datetime
            updatedAt : datetime
        }

        class AgentType {
            id : UUID
            name : string
            propertyDefinitions : json
            createdAt : datetime
            updatedAt : datetime
        }

        class Agent {
            id : UUID
            name : string
            state : enum[New|Connected|Disconnected|Error|Disabled]
            tokenHash : string 
            attributes : map[string]string[] 
            properties : json
            createdAt : datetime
            updatedAt : datetime
        }
    }

    namespace Services {
        class Service {
            id : UUID
            name : string
            state : enum[New,Creating,Created,Updating,Updated,Deleting, Deleted, Error]
            attributes : map[string]string[]
            resources : json
            createdAt : datetime
            updatedAt : datetime
        }

        class ServiceGroup {
            id : UUID
            name : string
            createdAt : datetime
            updatedAt : datetime
        }
    }

    namespace Metrics {
        class MetricEntry {
            id : UUID
            createdAt : datetime
            value : number
        }

        class MetricType {
            id : UUID
            entity : string 
            name : string
            createdAt : datetime
            updatedAt : datetime
        }
    }

    namespace Audit {
        class AuditEntry {
            id : UUID
            createdAt : datetime
            authorityType : string
            authorityId : string
            type : string
            properties : json
        }
    }

    note for ServiceType "Resource definitions can be eg.:
    - VM
    - Container
    - Container Image
    - VM Image
    - Kub Control Plane + Kub Worker"

    note for ServiceType "Service types include:
    - VM (VMrunner)
    - K8-Node (node, labels, nodeUsed)
    - MicroK8s application
    - Cluster Kubernetes Autodocs
    - Container Runtime services
    - K8s Application Reconcilier base"
```

## Component Descriptions

### Core Components

1. **Provider (Cloud Service Provider)**
   - Primary identifier for the cloud service provider
   - One-to-many relationship with Service Managers

2. **Agent**
   - Manages service instances and their lifecycle
   - Contains type information
   - Links to both services and versioning

3. **Service**
   - Represents individual service instances
   - Maintains versioning through ServiceVersion
   - Associates with multiple resources

### Resource Management

1. **Resource**
   - Represents infrastructure resources
   - Supports versioning through ResourceVersion
   - Categorized by ResourceType

2. **ResourceType**
   - Categorizes resources into specific types:
     - Virtual Machines (VM)
     - Containers
     - Container Images
     - VM Images
     - Kubernetes Components

### Metrics