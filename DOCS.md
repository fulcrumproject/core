# Service Architecture Class Diagram

This document outlines the service architecture class relationships and their key components.

## Class Diagram

```mermaid
classDiagram
    ServiceProvider "1" --> "0..N" ServiceAgent
    ServiceAgentType "0..N" --> "1..N" ServiceType
    ServiceAgent "0..N" --> "1" ServiceAgentType
    ServiceAgent "1" --> "0..N" Service
    Service "1" --> "1..N" Resource
    ServiceGroup "1" --> "0..N" Service
    Resource --> ResourceType
    Resource --> Resource
    MetricEntry "1" --> "0..N" MetricType

    namespace Providers {
        class ServiceProvider {
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
            createdAt : datetime
            updatedAt : datetime
        }

        class ServiceAgentType {
            id : UUID
            name : string
            createdAt : datetime
            updatedAt : datetime
        }

        class ServiceAgent {
            id : UUID
            name : string
            state : enum[New|Connected|Disabled|Error]
            attributes : map[string]string[] 
            createdAt : datetime
            updatedAt : datetime
        }
    }

    namespace Services {
        class Service {
            id : UUID
            name : string
            attributes : map[string]string[] 
            createdAt : datetime
            updatedAt : datetime
        }

        class Resource {
            id : UUID
            name : string
            createdAt : datetime
            updatedAt : datetime
        }

        class ResourceType {
            id : UUID
            name : string
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

    note for ResourceType "Types include:
    - VM
    - Container
    - Container Image
    - VM Image
    - Kub Control Plane + Kub Worker"

    note for ServiceType "Types include:
    - VM (VMrunner)
    - K8-Node (node, labels, nodeUsed)
    - MicroK8s application
    - Cluster Kubernetes Autodocs
    - Container Runtime services
    - K8s Application Reconcilier base"
```

## Component Descriptions

### Core Components

1. **ServiceProvider (Cloud Service Provider)**
   - Primary identifier for the cloud service provider
   - One-to-many relationship with Service Managers

2. **ServiceAgent**
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