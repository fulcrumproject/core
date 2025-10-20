# ServiceType Documentation

## Overview

This document provides comprehensive documentation for ServiceType schemas in Fulcrum Core. ServiceTypes define both the configuration structure (Property Schema) and lifecycle behavior (Lifecycle Schema) for services.

- **Property Schema**: A flexible JSON-based validation system that defines custom validation rules for service properties, ensuring data integrity and consistency for service configurations
- **Lifecycle Schema**: A schema-driven system that defines custom service lifecycles with states, actions, and transitions, enabling different service types to have completely different lifecycles

Both schemas are optional and can be defined independently or together to create rich, validated service definitions without requiring application recompilation.

## Property Schema

### Overview

The Property Schema is a flexible JSON-based validation system that allows administrators to define custom validation rules for service properties. This schema ensures data integrity and consistency for service configurations while providing dynamic validation without requiring application recompilation.

### Schema Structure

Each ServiceType can have an optional `propertySchema` field that defines validation rules for service properties. The schema is a properties.JSON object where each key represents a property name, and its value defines the validation rules for that property.

#### Basic Property Definition

```json
{
  "propertyName": {
    "type": "string|integer|number|boolean|object|array",
    "label": "Human-readable label (optional)",
    "required": true|false,
    "default": "default value (optional)",
    "validators": [...],
    "properties": {...},  // for object types
    "items": {...}        // for array types
  }
}
```

### Property Types

#### Primitive Types

##### String
```json
{
  "name": {
    "type": "string",
    "label": "Service Name",
    "required": true
  }
}
```

##### Integer
```json
{
  "cpu": {
    "type": "integer",
    "label": "CPU Cores",
    "required": true
  }
}
```

##### Number (Float)
```json
{
  "price": {
    "type": "number",
    "label": "Price per Hour",
    "default": 0.0
  }
}
```

##### Boolean
```json
{
  "enabled": {
    "type": "boolean",
    "label": "Service Enabled",
    "default": true
  }
}
```

#### Complex Types

##### Object
For nested object properties, use the `properties` field to define the schema for nested fields:

```json
{
  "metadata": {
    "type": "object",
    "label": "Service Metadata",
    "properties": {
      "owner": {
        "type": "string",
        "label": "Owner",
        "required": true
      },
      "version": {
        "type": "number",
        "label": "Version",
        "default": 1.0
      },
      "tags": {
        "type": "array",
        "items": {
          "type": "string"
        }
      }
    }
  }
}
```

##### Array
For array properties, use the `items` field to define the schema for array elements:

```json
{
  "ports": {
    "type": "array",
    "label": "Port Configuration",
    "items": {
      "type": "integer",
      "validators": [
        { "type": "min", "value": 1 },
        { "type": "max", "value": 65535 }
      ]
    },
    "validators": [
      { "type": "minItems", "value": 1 },
      { "type": "maxItems", "value": 10 }
    ]
  }
}
```

### Validators

Validators provide additional constraints beyond basic type checking. Each validator is an object with a `type` and `value` field.

#### String Validators

##### minLength
Minimum string length:
```json
{
  "validators": [
    { "type": "minLength", "value": 3 }
  ]
}
```

##### maxLength
Maximum string length:
```json
{
  "validators": [
    { "type": "maxLength", "value": 50 }
  ]
}
```

##### pattern
Regular expression pattern:
```json
{
  "validators": [
    { "type": "pattern", "value": "^[a-zA-Z0-9_-]+$" }
  ]
}
```

##### enum
Allowed values from a predefined list:
```json
{
  "validators": [
    { "type": "enum", "value": ["development", "staging", "production"] }
  ]
}
```

#### Numeric Validators (Integer/Number)

##### min
Minimum value:
```json
{
  "validators": [
    { "type": "min", "value": 1 }
  ]
}
```

##### max
Maximum value:
```json
{
  "validators": [
    { "type": "max", "value": 100 }
  ]
}
```

##### enum
Allowed values from a predefined list:
```json
{
  "validators": [
    { "type": "enum", "value": [1, 2, 4, 8, 16, 32] }
  ]
}
```

#### Array Validators

##### minItems
Minimum number of items:
```json
{
  "validators": [
    { "type": "minItems", "value": 1 }
  ]
}
```

##### maxItems
Maximum number of items:
```json
{
  "validators": [
    { "type": "maxItems", "value": 10 }
  ]
}
```

##### uniqueItems
Ensure all items are unique:
```json
{
  "validators": [
    { "type": "uniqueItems", "value": true }
  ]
}
```

#### Reference Validators (for type "reference")

##### serviceType
Validates that a referenced service is of a specific service type or one of multiple allowed types:

Single service type:
```json
{
  "database_service": {
    "type": "reference",
    "label": "Database Service",
    "required": true,
    "validators": [
      { "type": "serviceType", "value": "MySQL" }
    ]
  }
}
```

Multiple allowed service types:
```json
{
  "storage_service": {
    "type": "reference", 
    "label": "Storage Service",
    "required": true,
    "validators": [
      { "type": "serviceType", "value": ["MySQL", "PostgreSQL", "MongoDB"] }
    ]
  }
}
```

##### sameOrigin
Validates that a referenced service belongs to the same consumer or service group:

Same consumer constraint:
```json
{
  "related_service": {
    "type": "reference",
    "validators": [
      { "type": "sameOrigin", "value": "consumer" }
    ]
  }
}
```

Same service group constraint:
```json
{
  "dependent_service": {
    "type": "reference", 
    "validators": [
      { "type": "sameOrigin", "value": "group" }
    ]
  }
}
```

Combined validators example:
```json
{
  "backend_service": {
    "type": "reference",
    "label": "Backend API Service", 
    "required": true,
    "validators": [
      { "type": "serviceType", "value": ["NodeJS-API", "Python-API"] },
      { "type": "sameOrigin", "value": "consumer" }
    ]
  }
}
```

##### serviceOption
Validates that a value is one of the enabled service options for a specific service option type. Service options are provider-specific, dynamically managed validation lists.

**Basic Usage:**
```json
{
  "operatingSystem": {
    "type": "string",
    "label": "Operating System",
    "required": true,
    "validators": [
      { "type": "serviceOption", "value": "os" }
    ]
  }
}
```

This validates that the `operatingSystem` property value matches an enabled `ServiceOption` with `serviceOptionTypeId` corresponding to the "os" type for the provider.

**How it works:**
1. The validator value references a `ServiceOptionType` (e.g., "os", "machine_type", "region")
2. At validation time, the system looks up all enabled `ServiceOption` entries for the provider
3. The property value must match the `value` field of one of these enabled options
4. Only options with `enabled: true` are considered valid

**Examples:**

Machine type selection:
```json
{
  "machineType": {
    "type": "string",
    "label": "Machine Type",
    "required": true,
    "validators": [
      { "type": "serviceOption", "value": "machine_type" }
    ]
  }
}
```

Region selection:
```json
{
  "region": {
    "type": "string",
    "label": "Region",
    "required": true,
    "validators": [
      { "type": "serviceOption", "value": "region" }
    ]
  }
}
```

Disk type with complex value:
```json
{
  "diskConfig": {
    "type": "object",
    "label": "Disk Configuration",
    "required": true,
    "validators": [
      { "type": "serviceOption", "value": "disk_type" }
    ]
  }
}
```

**Value Matching:**
Service option values can be any JSON type (string, number, object, array). The validator performs exact JSON matching:

- **String values**: `"ubuntu-22.04"` matches option with `value: "ubuntu-22.04"`
- **Object values**: `{"type": "ssd", "size": 100}` matches option with same JSON structure
- **Array values**: `["us-east-1a", "us-east-1b"]` matches option with same array

**Service Option Management:**

Administrators manage service option types:
```http
POST /api/v1/service-option-types
{
  "name": "Operating System",
  "type": "os",
  "description": "Available operating systems for VM instances"
}
```

Providers manage their specific options:
```http
POST /api/v1/service-options
{
  "providerId": "participant-uuid",
  "serviceOptionTypeId": "option-type-uuid",
  "name": "Ubuntu 22.04 LTS",
  "value": "ubuntu-22.04",
  "enabled": true,
  "displayOrder": 1
}
```

**Benefits:**
- **Dynamic**: Options can be added/removed without changing service type schemas
- **Provider-specific**: Each provider can offer different options
- **Flexible**: Values can be simple strings or complex objects
- **Manageable**: Options can be disabled without deletion for controlled rollout

**Error Messages:**
- `"serviceOption validator value must be a string (serviceOptionType)"` - Invalid validator configuration
- `"service option type 'type' not found"` - ServiceOptionType doesn't exist
- `"service option with value 'value' not found or not enabled for provider"` - No matching enabled option found
- `"service option validation requires provider ID in context"` - Missing provider context (internal error)

**Complete Example:**

Service type schema:
```json
{
  "name": "VM Instance",
  "propertySchema": {
    "operatingSystem": {
      "type": "string",
      "label": "Operating System",
      "required": true,
      "validators": [
        { "type": "serviceOption", "value": "os" }
      ]
    },
    "machineType": {
      "type": "string",
      "label": "Machine Type",
      "required": true,
      "validators": [
        { "type": "serviceOption", "value": "machine_type" }
      ]
    },
    "region": {
      "type": "string",
      "label": "Region",
      "required": true,
      "validators": [
        { "type": "serviceOption", "value": "region" }
      ]
    }
  }
}
```

Provider's service options:
```json
[
  {
    "name": "Ubuntu 22.04 LTS",
    "value": "ubuntu-22.04",
    "serviceOptionType": "os",
    "enabled": true,
    "displayOrder": 1
  },
  {
    "name": "Ubuntu 24.04 LTS",
    "value": "ubuntu-24.04",
    "serviceOptionType": "os",
    "enabled": true,
    "displayOrder": 2
  },
  {
    "name": "Standard (2 vCPU, 4GB RAM)",
    "value": "n1-standard-2",
    "serviceOptionType": "machine_type",
    "enabled": true,
    "displayOrder": 1
  }
]
```

Valid service creation:
```json
{
  "name": "web-server-01",
  "serviceTypeId": "vm-type-uuid",
  "properties": {
    "operatingSystem": "ubuntu-22.04",
    "machineType": "n1-standard-2",
    "region": "us-east-1"
  }
}
```

### Property Pool Allocation

Properties with `source: "system"` can use automatic pool allocation via the `servicePoolType` field. Service pools manage finite, exclusive resources (IPs, ports, hostnames) with automatic allocation and lifecycle management.

**Basic Usage:**
```json
{
  "publicIp": {
    "type": "string",
    "label": "Public IP Address",
    "source": "system",
    "updatable": "never",
    "servicePoolType": "public_ip"
  }
}
```

This marks the `publicIp` property for automatic allocation from a pool with type "public_ip" during service creation.

**How it works:**
1. The `servicePoolType` field references a pool type (e.g., "public_ip", "hostname", "port")
2. The property must have `source: "system"` to enable automatic allocation
3. The agent must have a `servicePoolSetId` configured
4. During service creation, the system finds a pool with the matching type in the agent's pool set
5. The property type must match the pool's `propertyType` (e.g., string property → string pool)
6. A value is automatically allocated from the pool and stored directly in the property
7. When the service is deleted, the value is released back to the pool

**Key Features:**
- **Automatic allocation**: No manual value selection required
- **Type validation**: Property type must match pool's propertyType
- **Exclusive access**: Each value can only be allocated to one service at a time
- **Lifecycle management**: Values automatically released on service deletion
- **Direct storage**: Actual values copied into properties (no dereferencing needed)
- **System-source**: Requires `source: "system"` since allocation is automatic

**Pool Types:**

List pools (pre-configured values):
```json
{
  "ipAddress": {
    "type": "string",
    "label": "IP Address",
    "source": "system",
    "updatable": "never",
    "servicePoolType": "public_ip"
  }
}
```

Subnet pools (automatic CIDR allocation):
```json
{
  "privateIp": {
    "type": "string",
    "label": "Private IP Address",
    "source": "system",
    "updatable": "never",
    "servicePoolType": "private_ip"
  }
}
```

JSON type for complex values:
```json
{
  "hostname": {
    "type": "json",
    "label": "Hostname Configuration",
    "source": "system",
    "updatable": "never",
    "servicePoolType": "hostname"
  }
}
```

**Pool Setup:**

1. Create a pool set for the provider:
```http
POST /api/v1/service-pool-sets
{
  "name": "Production Pools",
  "providerId": "participant-uuid"
}
```

2. Create a pool with matching type and propertyType:
```http
POST /api/v1/service-pools
{
  "name": "Public IP Pool",
  "type": "public_ip",
  "propertyType": "string",
  "generatorType": "list",
  "servicePoolSetId": "pool-set-uuid"
}
```

3. Add values (for list generators):
```http
POST /api/v1/service-pool-values
{
  "servicePoolId": "pool-uuid",
  "name": "185.123.45.10",
  "value": "185.123.45.10"
}
```

4. Configure agent with pool set:
```http
PATCH /api/v1/agents/{agent-id}
{
  "servicePoolSetId": "pool-set-uuid"
}
```

**Generator Types:**
- **list**: Pre-configured values stored as individual ServicePoolValue records
- **subnet**: IP addresses automatically generated from CIDR ranges

**Allocation Tracking:**
Each allocated value tracks:
- `serviceId`: Which service owns this value
- `propertyName`: Which property uses this value
- `allocatedAt`: When the allocation occurred

**Value Types:**
Pool values can be any JSON type:
- **String**: `"185.123.45.10"` for simple IPs
- **Object**: `{"ip": "10.0.1.5", "gateway": "10.0.1.1"}` for complex network config
- **Array**: `["dns1.example.com", "dns2.example.com"]` for multiple values

**Benefits:**
- **Resource management**: Prevents conflicts (no two services get the same IP)
- **Automatic**: No manual IP selection by users
- **Trackable**: See which service uses which resource
- **Reusable**: Values automatically returned to pool on deletion
- **Flexible**: Supports simple strings or complex JSON structures

**Error Messages:**
- `"servicePoolType cannot be empty"` - Empty servicePoolType field
- `"servicePoolType requires source to be 'system'"` - Property doesn't have `source: "system"`
- `"property X has type Y but pool Z provides type W"` - Property type doesn't match pool's propertyType
- `"no pool found with type X in pool set"` - Pool type doesn't exist in agent's pool set
- `"failed to allocate from pool X"` - No available values in pool

**Complete Example:**

Service type schema:
```json
{
  "name": "VM Instance",
  "propertySchema": {
    "publicIp": {
      "type": "string",
      "label": "Public IP",
      "source": "system",
      "updatable": "never",
      "servicePoolType": "public_ip"
    },
    "privateIp": {
      "type": "string",
      "label": "Private IP",
      "source": "system",
      "updatable": "never",
      "servicePoolType": "private_ip"
    },
    "hostname": {
      "type": "json",
      "label": "Hostname Config",
      "source": "system",
      "updatable": "never",
      "servicePoolType": "hostname"
    }
  }
}
```

Pool setup (list pool):
```http
POST /api/v1/service-pool-values
{
  "servicePoolId": "public-ip-pool-uuid",
  "name": "185.123.45.10",
  "value": "185.123.45.10"
}
```

Pool setup (subnet pool):
```http
POST /api/v1/service-pools
{
  "name": "Private Network",
  "type": "private_ip",
  "propertyType": "string",
  "generatorType": "subnet",
  "generatorConfig": {
    "cidr": "192.168.1.0/24",
    "excludeFirst": 1,
    "excludeLast": 1
  },
  "servicePoolSetId": "pool-set-uuid"
}
```

Pool setup (complex JSON values):
```http
POST /api/v1/service-pools
{
  "name": "Hostname Pool",
  "type": "hostname",
  "propertyType": "json",
  "generatorType": "list",
  "servicePoolSetId": "pool-set-uuid"
}

POST /api/v1/service-pool-values
{
  "servicePoolId": "hostname-pool-uuid",
  "name": "web01.example.com",
  "value": {
    "hostname": "web01.example.com",
    "internalDns": "web01.internal",
    "zone": "us-east-1a"
  }
}
```

Service creation (automatic allocation):
```json
{
  "name": "web-server-01",
  "serviceTypeId": "vm-type-uuid",
  "agentId": "agent-with-pool-set-uuid",
  "properties": {}
}
```

After service creation, properties contain allocated values:
```json
{
  "properties": {
    "publicIp": "185.123.45.10",
    "privateIp": "192.168.1.1",
    "hostname": {
      "hostname": "web01.example.com",
      "internalDns": "web01.internal",
      "zone": "us-east-1a"
    }
  }
}
```

View allocation status:
```http
GET /api/v1/service-pool-values?servicePoolId=public-ip-pool-uuid
```

Response shows allocated status:
```json
{
  "items": [
    {
      "id": "value-uuid",
      "name": "185.123.45.10",
      "value": "185.123.45.10",
      "serviceId": "service-uuid",
      "propertyName": "publicIp",
      "allocatedAt": "2025-10-20T12:00:00Z"
    }
  ]
}
```

### Property Source

The `source` field controls who can set and update a property value. This enables proper separation between user-provided configuration and agent-discovered information.

#### Source Values

##### input (default)
Properties set by users through the API. These represent the desired configuration.

```json
{
  "instanceName": {
    "type": "string",
    "label": "Instance Name",
    "source": "input",
    "required": true
  }
}
```

**Behavior:**
- Users can set this property when creating a service
- Users can update this property (subject to updatability rules)
- Agents cannot modify this property
- If `source` is omitted, "input" is the default

##### agent
Properties set by agents after provisioning resources. These represent actual provisioned values.

```json
{
  "ipAddress": {
    "type": "string",
    "label": "Assigned IP Address",
    "source": "agent"
  }
}
```

**Behavior:**
- Users cannot set or update this property
- Agents can set this property when completing a job
- Agents can update this property (subject to updatability rules)
- Typically used for discovered values like IP addresses, ports, UUIDs

#### Source Usage Patterns

**Configuration vs Discovery**
```json
{
  "diskSize": {
    "type": "integer",
    "label": "Disk Size (GB)",
    "source": "input",
    "required": true
  },
  "actualDiskPath": {
    "type": "string",
    "label": "Disk Path",
    "source": "agent"
  }
}
```

User specifies `diskSize`, agent reports back `actualDiskPath` after provisioning.

### Property Updatability

The `updatable` field controls when and if a property can be modified after initial creation. This prevents accidental changes to immutable infrastructure or ensures changes only happen in safe states.

#### Updatable Values

##### always (default)
Property can be updated at any time in any service status.

```json
{
  "description": {
    "type": "string",
    "label": "Description",
    "updatable": "always"
  }
}
```

**Behavior:**
- Can be updated in any service state
- Suitable for metadata and non-critical settings
- If `updatable` is omitted, "always" is the default

##### never
Property cannot be updated after initial creation (immutable).

```json
{
  "uuid": {
    "type": "string",
    "label": "Instance UUID",
    "source": "agent",
    "updatable": "never"
  }
}
```

**Behavior:**
- Can be set during initial service creation
- Cannot be changed after initial creation
- Any attempt to update returns a validation error
- Suitable for identifiers and immutable configuration

**Note:** For agent-source properties, "initial creation" means the first job completion (typically the Create job). Agents can set immutable properties during this first job, but cannot update them in subsequent jobs.

##### statuses
Property can only be updated when service is in specific statuses. Requires `updatableIn` array.

```json
{
  "diskSize": {
    "type": "integer",
    "label": "Disk Size (GB)",
    "source": "input",
    "updatable": "statuses",
    "updatableIn": ["Stopped"]
  }
}
```

**Behavior:**
- Can only be updated when service status is in the `updatableIn` list
- Updates attempted in other statuses return validation errors
- Suitable for properties requiring service to be in a safe state

#### Updatability Patterns

**Immutable Identifiers**
```json
{
  "instanceId": {
    "type": "string",
    "label": "Cloud Instance ID",
    "source": "agent",
    "updatable": "never"
  }
}
```

**State-Conditional Updates**
```json
{
  "cpu": {
    "type": "integer",
    "label": "CPU Cores",
    "source": "input",
    "updatable": "statuses",
    "updatableIn": ["Stopped"],
    "validators": [
      { "type": "enum", "value": [1, 2, 4, 8] }
    ]
  },
  "memory": {
    "type": "integer",
    "label": "Memory (GB)",
    "source": "input",
    "updatable": "statuses",
    "updatableIn": ["Stopped"],
    "validators": [
      { "type": "enum", "value": [1, 2, 4, 8, 16] }
    ]
  }
}
```

**Hot-Updatable Configuration**
```json
{
  "maxConnections": {
    "type": "integer",
    "label": "Max Connections",
    "source": "input",
    "updatable": "statuses",
    "updatableIn": ["Started", "Stopped"]
  }
}
```

### Combined Source and Updatability

**User Configuration (Mutable)**
```json
{
  "environment": {
    "type": "string",
    "label": "Environment",
    "source": "input",
    "updatable": "always",
    "validators": [
      { "type": "enum", "value": ["dev", "staging", "prod"] }
    ]
  }
}
```

**User Configuration (Immutable After Creation)**
```json
{
  "region": {
    "type": "string",
    "label": "Cloud Region",
    "source": "input",
    "updatable": "never",
    "required": true
  }
}
```

**Agent-Discovered (Immutable)**
```json
{
  "ipAddress": {
    "type": "string",
    "label": "IP Address",
    "source": "agent",
    "updatable": "never"
  }
}
```

**Agent-Discovered (Mutable)**
```json
{
  "healthStatus": {
    "type": "string",
    "label": "Health Status",
    "source": "agent",
    "updatable": "always"
  }
}
```

### Complete Examples

#### VM Service Type with Mixed Sources

Here's a comprehensive example for a VM service type with user configuration and agent-discovered properties:

```json
{
  "instanceName": {
    "type": "string",
    "label": "Instance Name",
    "source": "input",
    "updatable": "always",
    "required": true,
    "validators": [
      { "type": "minLength", "value": 3 },
      { "type": "maxLength", "value": 50 },
      { "type": "pattern", "value": "^[a-zA-Z0-9-]+$" }
    ]
  },
  "region": {
    "type": "string",
    "label": "Cloud Region",
    "source": "input",
    "updatable": "never",
    "required": true,
    "validators": [
      { "type": "enum", "value": ["us-east-1", "us-west-2", "eu-west-1"] }
    ]
  },
  "cpu": {
    "type": "integer",
    "label": "CPU Cores",
    "source": "input",
    "updatable": "statuses",
    "updatableIn": ["Stopped"],
    "required": true,
    "validators": [
      { "type": "enum", "value": [1, 2, 4, 8, 16, 32] }
    ]
  },
  "memory": {
    "type": "integer",
    "label": "Memory (GB)",
    "source": "input",
    "updatable": "statuses",
    "updatableIn": ["Stopped"],
    "required": true,
    "validators": [
      { "type": "enum", "value": [1, 2, 4, 8, 16, 32, 64] }
    ]
  },
  "diskSize": {
    "type": "integer",
    "label": "Disk Size (GB)",
    "source": "input",
    "updatable": "statuses",
    "updatableIn": ["Stopped"],
    "required": true,
    "validators": [
      { "type": "min", "value": 10 },
      { "type": "max", "value": 1000 }
    ]
  },
  "imageId": {
    "type": "string",
    "label": "VM Image ID",
    "source": "input",
    "updatable": "never",
    "required": true
  },
  "instanceId": {
    "type": "string",
    "label": "Cloud Instance ID",
    "source": "agent",
    "updatable": "never"
  },
  "ipAddress": {
    "type": "string",
    "label": "IP Address",
    "source": "agent",
    "updatable": "never"
  },
  "privateIpAddress": {
    "type": "string",
    "label": "Private IP Address",
    "source": "agent",
    "updatable": "never"
  },
  "hostname": {
    "type": "string",
    "label": "Hostname",
    "source": "agent",
    "updatable": "never"
  },
  "tags": {
    "type": "object",
    "label": "Resource Tags",
    "source": "input",
    "updatable": "always",
    "properties": {
      "environment": {
        "type": "string",
        "validators": [
          { "type": "enum", "value": ["dev", "staging", "prod"] }
        ]
      },
      "owner": {
        "type": "string"
      }
    }
  }
}
```

### Disk Service Type

Example for a managed disk with state-conditional resizing:

```json
{
  "name": {
    "type": "string",
    "label": "Disk Name",
    "source": "input",
    "updatable": "always",
    "required": true
  },
  "sizeGb": {
    "type": "integer",
    "label": "Size (GB)",
    "source": "input",
    "updatable": "statuses",
    "updatableIn": ["Stopped"],
    "required": true,
    "validators": [
      { "type": "min", "value": 10 },
      { "type": "max", "value": 16384 }
    ]
  },
  "type": {
    "type": "string",
    "label": "Disk Type",
    "source": "input",
    "updatable": "never",
    "required": true,
    "validators": [
      { "type": "enum", "value": ["ssd", "hdd", "nvme"] }
    ]
  },
  "diskId": {
    "type": "string",
    "label": "Cloud Disk ID",
    "source": "agent",
    "updatable": "never"
  },
  "actualSizeGb": {
    "type": "integer",
    "label": "Actual Size (GB)",
    "source": "agent",
    "updatable": "always"
  },
  "devicePath": {
    "type": "string",
    "label": "Device Path",
    "source": "agent",
    "updatable": "never"
  }
}
```

### API Usage

#### Retrieving ServiceType with Schema

```http
GET /api/v1/service-types/{id}
```

Response includes the `propertySchema` field:
```json
{
  "id": "uuid",
  "name": "VM",
  "propertySchema": {
    "cpu": {
      "type": "integer",
      "label": "CPU Cores",
      "required": true,
      "validators": [...]
    }
  },
  "createdAt": "2023-01-01T00:00:00Z",
  "updatedAt": "2023-01-01T00:00:00Z"
}
```

#### Validating Properties

```http
POST /api/v1/service-types/{id}/validate
Content-Type: application/json

{
  "properties": {
    "cpu": 4,
    "memory": 2048,
    "image_name": "nginx",
    "environment": "production"
  }
}
```

Response:
```json
{
  "valid": true,
  "errors": []
}
```

Or with validation errors:
```json
{
  "valid": false,
  "errors": [
    {
      "path": "cpu",
      "message": "required field is missing"
    },
    {
      "path": "memory",
      "message": "value is not in allowed enum values"
    }
  ]
}
```

### Best Practices

1. **Start Simple**: Begin with basic type validation and add validators as needed
2. **Use Descriptive Labels**: Provide clear, human-readable labels for all properties
3. **Set Reasonable Defaults**: Use default values for optional properties when appropriate
4. **Validate Early**: Use the validation endpoint during development to test schemas
5. **Document Constraints**: Use enum validators to clearly define allowed values
6. **Nested Validation**: Leverage object and array types for complex configurations
7. **Error Handling**: Always check validation results before processing service properties

### Error Messages

The validation system provides detailed error messages with path information:

- `"required field is missing"` - A required property is not provided
- `"unknown property"` - A property not defined in the schema is provided
- `"expected {type}, got {actualType}"` - Type mismatch
- `"string length {actual} is less than minimum {min}"` - String too short
- `"string length {actual} exceeds maximum {max}"` - String too long
- `"string does not match pattern {pattern}"` - Regex pattern mismatch
- `"value is not in allowed enum values"` - Value not in enum list
- `"value {actual} is less than minimum {min}"` - Number below minimum
- `"value {actual} exceeds maximum {max}"` - Number above maximum
- `"array length {actual} is less than minimum {min}"` - Array too short
- `"array length {actual} exceeds maximum {max}"` - Array too long
- `"array contains duplicate items"` - Duplicate items when uniqueItems is true

### Reference Type Error Messages

- `"invalid service ID format"` - Invalid UUID format for service reference
- `"referenced service does not exist"` - Referenced service not found in database
- `"referenced service must belong to the same consumer"` - Consumer constraint violation
- `"referenced service must belong to the same service group"` - Group constraint violation
- `"referenced service is not of the allowed service type"` - Service type constraint violation  
- `"serviceType validator value must be a string or array of strings"` - Invalid validator configuration

### Property Source and Updatability Error Messages

- `"property 'propertyName' cannot be updated by user (source: agent)"` - User attempted to update an agent-source property
- `"property 'propertyName' cannot be updated by agent (source: input)"` - Agent attempted to update a user-input property  
- `"property 'propertyName' cannot be updated (updatable: never)"` - Attempted to update an immutable property
- `"property 'propertyName' cannot be updated in status 'StatusName' (allowed statuses: [Status1, Status2])"` - Attempted to update a property in a disallowed status

### Migration and Updates

When updating property schemas:

1. **Backward Compatibility**: Ensure existing services remain valid
2. **Gradual Migration**: Add new optional properties before making them required
3. **Validation Testing**: Use the validation endpoint to test schema changes
4. **Documentation**: Update service documentation when schemas change

---

## Lifecycle Schema

### Overview

The ServiceType Lifecycle Schema is a flexible schema-driven system that allows administrators to define custom service lifecycles with states, actions, and transitions. This enables different service types to have completely different lifecycles without requiring application code changes.

### Why Lifecycle Schemas?

Traditional hardcoded state machines are inflexible and require code changes to support new service types. Lifecycle schemas solve this by:

- **Flexibility**: Each service type can have its own custom lifecycle
- **Dynamic**: Add new service types with custom lifecycles without recompiling
- **Consistency**: Enforce valid state transitions at the domain level
- **Expressiveness**: Support complex workflows with error handling
- **Maintainability**: Lifecycle logic is declarative JSON, not imperative code

### Lifecycle Schema Structure

Each ServiceType can have an optional `lifecycleSchema` field that defines the service lifecycle:

```json
{
  "states": [...],
  "actions": [...],
  "initialState": "New",
  "terminalStates": ["Deleted"],
  "runningStates": ["Started"]
}
```

### Schema Fields

| Field            | Type                     | Required | Description                                        |
| ---------------- | ------------------------ | -------- | -------------------------------------------------- |
| `states`         | Array of LifecycleState  | Yes      | List of all possible states                        |
| `actions`        | Array of LifecycleAction | Yes      | List of actions that can be performed              |
| `initialState`   | String                   | Yes      | Starting state for new services                    |
| `terminalStates` | Array of String          | No       | States where no further actions allowed            |
| `runningStates`  | Array of String          | No       | States considered "running" for uptime calculation |

### States

States represent the possible conditions of a service. Each state has a name:

```json
{
  "name": "Started"
}
```

**Common State Examples:**
- `New` - Service just created
- `Stopped` - Service provisioned but not running
- `Started` - Service is actively running
- `Failed` - Service encountered an error
- `Deleted` - Service has been removed

### Actions

Actions define operations that can be performed on a service. Each action has:

```json
{
  "name": "start",
  "requestSchemaType": "properties",
  "transitions": [...]
}
```

#### Action Fields

| Field               | Type                         | Required | Description                                            |
| ------------------- | ---------------------------- | -------- | ------------------------------------------------------ |
| `name`              | String                       | Yes      | Name of the action (e.g., "start", "stop")             |
| `requestSchemaType` | String                       | No       | Type of request body: "properties" or omit for no body |
| `transitions`       | Array of LifecycleTransition | Yes      | State transitions for this action                      |

#### Request Schema Type

The `requestSchemaType` field determines what kind of request body the action accepts:

- **Omitted or empty**: Action requires no request body (e.g., start, stop, restart)
- **`"properties"`**: Action accepts service properties in request body (e.g., create, update)

### Transitions

Transitions define how services move from one state to another when an action is performed:

```json
{
  "from": "Stopped",
  "to": "Started",
  "onError": false
}
```

#### Transition Fields

| Field           | Type    | Required | Description                                          |
| --------------- | ------- | -------- | ---------------------------------------------------- |
| `from`          | String  | Yes      | Source state                                         |
| `to`            | String  | Yes      | Destination state                                    |
| `onError`       | Boolean | No       | Whether this is an error transition (default: false) |
| `onErrorRegexp` | String  | No       | Regex pattern to match error messages                |

#### Success vs Error Transitions

Each action can have two types of transitions:

**Success Transitions** (`onError: false` or omitted):
- Applied when the job completes successfully
- Only one success transition should exist per (action, from-state) pair

**Error Transitions** (`onError: true`):
- Applied when the job fails
- Multiple error transitions can exist per (action, from-state) pair
- Error message is matched against `onErrorRegexp` to select which transition to use
- If no regexp is specified, matches any error

#### Error Regexp Matching

When a job fails, the system uses the error message to determine the next state:

1. Look for error transitions (`onError: true`) for the current action and state
2. If a transition has `onErrorRegexp`, check if the error message matches the regexp
3. Use the first matching transition
4. If no regexps are specified, use the first error transition (catches all errors)

**Example: Quota-specific error handling:**
```json
{
  "name": "start",
  "transitions": [
    {
      "from": "Stopped",
      "to": "Started"
    },
    {
      "from": "Stopped",
      "to": "QuotaExceeded",
      "onError": true,
      "onErrorRegexp": "quota.*exceeded"
    },
    {
      "from": "Stopped",
      "to": "Failed",
      "onError": true
    }
  ]
}
```

This configuration:
- Success: `Stopped → Started`
- Error with "quota exceeded" message: `Stopped → QuotaExceeded`
- Any other error: `Stopped → Failed`

### Terminal States

Terminal states are end states where no further actions can be performed:

```json
{
  "terminalStates": ["Deleted", "Terminated"]
}
```

Services in terminal states:
- Cannot perform any lifecycle actions
- Attempts to perform actions return an error
- Represent final, irreversible states

### Running States

Running states are used for uptime calculation and monitoring:

```json
{
  "runningStates": ["Started", "Running", "Active"]
}
```

Services in running states are considered:
- Actively providing their service
- Counted toward uptime metrics
- Expected to be operational

If `runningStates` is empty or not specified, services are never considered "running" for uptime purposes.

### Complete Examples

#### Example 1: Simple VM Lifecycle

A basic VM lifecycle with start/stop operations:

```json
{
  "states": [
    {"name": "New"},
    {"name": "Stopped"},
    {"name": "Started"},
    {"name": "Deleted"}
  ],
  "actions": [
    {
      "name": "create",
      "requestSchemaType": "properties",
      "transitions": [
        {"from": "New", "to": "Stopped"}
      ]
    },
    {
      "name": "start",
      "transitions": [
        {"from": "Stopped", "to": "Started"}
      ]
    },
    {
      "name": "stop",
      "transitions": [
        {"from": "Started", "to": "Stopped"}
      ]
    },
    {
      "name": "update",
      "requestSchemaType": "properties",
      "transitions": [
        {"from": "Stopped", "to": "Stopped"},
        {"from": "Started", "to": "Started"}
      ]
    },
    {
      "name": "delete",
      "transitions": [
        {"from": "Stopped", "to": "Deleted"},
        {"from": "Started", "to": "Deleted"}
      ]
    }
  ],
  "initialState": "New",
  "terminalStates": ["Deleted"],
  "runningStates": ["Started"]
}
```

#### Example 2: Advanced Lifecycle with Error Handling

A more complex lifecycle with intermediate states and error handling:

```json
{
  "states": [
    {"name": "New"},
    {"name": "Provisioning"},
    {"name": "Stopped"},
    {"name": "Starting"},
    {"name": "Started"},
    {"name": "Stopping"},
    {"name": "Failed"},
    {"name": "Deleted"}
  ],
  "actions": [
    {
      "name": "create",
      "requestSchemaType": "properties",
      "transitions": [
        {"from": "New", "to": "Provisioning"},
        {"from": "Provisioning", "to": "Stopped"},
        {"from": "Provisioning", "to": "Failed", "onError": true, "onErrorRegexp": "quota.*exceeded"},
        {"from": "Provisioning", "to": "Failed", "onError": true}
      ]
    },
    {
      "name": "start",
      "transitions": [
        {"from": "Stopped", "to": "Starting"},
        {"from": "Starting", "to": "Started"},
        {"from": "Starting", "to": "Failed", "onError": true}
      ]
    },
    {
      "name": "stop",
      "transitions": [
        {"from": "Started", "to": "Stopping"},
        {"from": "Stopping", "to": "Stopped"},
        {"from": "Stopping", "to": "Failed", "onError": true}
      ]
    },
    {
      "name": "restart",
      "transitions": [
        {"from": "Started", "to": "Stopping"},
        {"from": "Stopping", "to": "Starting"},
        {"from": "Starting", "to": "Started"}
      ]
    },
    {
      "name": "delete",
      "transitions": [
        {"from": "Stopped", "to": "Deleted"},
        {"from": "Failed", "to": "Deleted"}
      ]
    }
  ],
  "initialState": "New",
  "terminalStates": ["Deleted"],
  "runningStates": ["Started"]
}
```

#### Example 3: Database Lifecycle with Maintenance States

A database service with backup and maintenance operations:

```json
{
  "states": [
    {"name": "New"},
    {"name": "Stopped"},
    {"name": "Started"},
    {"name": "Backup"},
    {"name": "Maintenance"},
    {"name": "Deleted"}
  ],
  "actions": [
    {
      "name": "create",
      "requestSchemaType": "properties",
      "transitions": [
        {"from": "New", "to": "Stopped"}
      ]
    },
    {
      "name": "start",
      "transitions": [
        {"from": "Stopped", "to": "Started"}
      ]
    },
    {
      "name": "stop",
      "transitions": [
        {"from": "Started", "to": "Stopped"}
      ]
    },
    {
      "name": "backup",
      "transitions": [
        {"from": "Started", "to": "Backup"},
        {"from": "Backup", "to": "Started"}
      ]
    },
    {
      "name": "maintenance",
      "transitions": [
        {"from": "Stopped", "to": "Maintenance"},
        {"from": "Maintenance", "to": "Stopped"}
      ]
    },
    {
      "name": "delete",
      "transitions": [
        {"from": "Stopped", "to": "Deleted"}
      ]
    }
  ],
  "initialState": "New",
  "terminalStates": ["Deleted"],
  "runningStates": ["Started", "Backup"]
}
```

### API Usage

#### Creating a ServiceType with Lifecycle

```http
POST /api/v1/service-types
Content-Type: application/json

{
  "name": "VM Instance",
  "propertySchema": {...},
  "lifecycleSchema": {
    "states": [
      {"name": "New"},
      {"name": "Stopped"},
      {"name": "Started"},
      {"name": "Deleted"}
    ],
    "actions": [
      {
        "name": "create",
        "requestSchemaType": "properties",
        "transitions": [
          {"from": "New", "to": "Stopped"}
        ]
      },
      {
        "name": "start",
        "transitions": [
          {"from": "Stopped", "to": "Started"}
        ]
      }
    ],
    "initialState": "New",
    "terminalStates": ["Deleted"],
    "runningStates": ["Started"]
  }
}
```

#### Performing Lifecycle Actions

Use the generic action endpoint to perform any lifecycle action:

```http
POST /api/v1/services/{id}/{action}
Content-Type: application/json

{
  "properties": {
    // Optional properties for actions with requestSchemaType: "properties"
  }
}
```

**Examples:**

Start a service (no properties needed):
```http
POST /api/v1/services/123e4567-e89b-12d3-a456-426614174000/start
```

Restart a service (no properties needed):
```http
POST /api/v1/services/123e4567-e89b-12d3-a456-426614174000/restart
```

Backup with parameters (if action has requestSchemaType):
```http
POST /api/v1/services/123e4567-e89b-12d3-a456-426614174000/backup
Content-Type: application/json

{
  "properties": {
    "backupName": "daily-backup",
    "compressionLevel": 9
  }
}
```

### Validation and Constraints

The system enforces the following lifecycle rules:

1. **Valid States**: All state names in transitions must be defined in `states`
2. **Valid Initial State**: `initialState` must be in the `states` list
3. **Valid Terminal States**: All `terminalStates` must be in the `states` list
4. **Valid Running States**: All `runningStates` must be in the `states` list
5. **Action Availability**: Only actions defined in the lifecycle can be performed
6. **State Transitions**: Actions can only transition from states defined in their transitions
7. **Terminal State Block**: No actions can be performed on services in terminal states
8. **Unique Transitions**: Each (action, from-state) pair should have only one success transition

### Best Practices

1. **Start Simple**: Begin with basic states (New, Stopped, Started, Deleted) and add complexity as needed
2. **Meaningful State Names**: Use clear, descriptive state names that reflect the service condition
3. **Error Handling**: Always provide error transitions for states that can fail
4. **Terminal States**: Mark irreversible end states as terminal
5. **Running States**: Accurately mark which states should count as "running" for metrics
6. **Intermediate States**: Use intermediate states (e.g., Starting, Stopping) for long-running operations
7. **Error Regexps**: Use specific error regexps for known error types, with a catch-all fallback
8. **Multi-step Actions**: Break complex actions into multiple steps with intermediate states
9. **Idempotent Actions**: Design actions to be safely retryable from the same state
10. **Documentation**: Document the lifecycle flow for each service type

### Common Patterns

#### Pattern 1: Immediate vs Progressive Actions

**Immediate** (single-step):
```json
{
  "name": "start",
  "transitions": [
    {"from": "Stopped", "to": "Started"}
  ]
}
```

**Progressive** (multi-step):
```json
{
  "name": "start",
  "transitions": [
    {"from": "Stopped", "to": "Starting"},
    {"from": "Starting", "to": "Started"}
  ]
}
```

Progressive is better for long-running operations where you want to track progress.

#### Pattern 2: Restart as Compound Action

Restart can be modeled as a compound action that stops then starts:

```json
{
  "name": "restart",
  "transitions": [
    {"from": "Started", "to": "Stopping"},
    {"from": "Stopping", "to": "Starting"},
    {"from": "Starting", "to": "Started"}
  ]
}
```

#### Pattern 3: Graceful Degradation

Allow operations to proceed despite non-critical errors:

```json
{
  "name": "upgrade",
  "transitions": [
    {"from": "Started", "to": "Upgrading"},
    {"from": "Upgrading", "to": "Started"},
    {"from": "Upgrading", "to": "Started", "onError": true, "onErrorRegexp": "minor.*error"},
    {"from": "Upgrading", "to": "Failed", "onError": true}
  ]
}
```

#### Pattern 4: Recovery Actions

Provide actions to recover from failed states:

```json
{
  "name": "recover",
  "transitions": [
    {"from": "Failed", "to": "Stopped"}
  ]
}
```

### Lifecycle vs Property Schema Interaction

Lifecycle schemas work together with property schemas:

- **Property Schema**: Defines *what* can be configured (properties, validation)
- **Lifecycle Schema**: Defines *when* and *how* things can change (actions, transitions)

The `updatable` and `updatableIn` fields in property schemas reference lifecycle states:

```json
{
  "propertySchema": {
    "diskSize": {
      "type": "integer",
      "updatable": "statuses",
      "updatableIn": ["Stopped"]
    }
  },
  "lifecycleSchema": {
    "states": [
      {"name": "Stopped"},
      {"name": "Started"}
    ]
  }
}
```

This ensures `diskSize` can only be updated when the service is in the `Stopped` state.