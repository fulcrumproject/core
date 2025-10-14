# ServiceType Property Schema Documentation

## Overview

The ServiceType Property Schema is a flexible properties.JSON-based validation system that allows administrators to define custom validation rules for service properties. This schema ensures data integrity and consistency for service configurations while providing dynamic validation without requiring application recompilation.

## Schema Structure

Each ServiceType can have an optional `propertySchema` field that defines validation rules for service properties. The schema is a properties.JSON object where each key represents a property name, and its value defines the validation rules for that property.

### Basic Property Definition

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

## Property Types

### Primitive Types

#### String
```json
{
  "name": {
    "type": "string",
    "label": "Service Name",
    "required": true
  }
}
```

#### Integer
```json
{
  "cpu": {
    "type": "integer",
    "label": "CPU Cores",
    "required": true
  }
}
```

#### Number (Float)
```json
{
  "price": {
    "type": "number",
    "label": "Price per Hour",
    "default": 0.0
  }
}
```

#### Boolean
```json
{
  "enabled": {
    "type": "boolean",
    "label": "Service Enabled",
    "default": true
  }
}
```

### Complex Types

#### Object
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

#### Array
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

## Validators

Validators provide additional constraints beyond basic type checking. Each validator is an object with a `type` and `value` field.

### String Validators

#### minLength
Minimum string length:
```json
{
  "validators": [
    { "type": "minLength", "value": 3 }
  ]
}
```

#### maxLength
Maximum string length:
```json
{
  "validators": [
    { "type": "maxLength", "value": 50 }
  ]
}
```

#### pattern
Regular expression pattern:
```json
{
  "validators": [
    { "type": "pattern", "value": "^[a-zA-Z0-9_-]+$" }
  ]
}
```

#### enum
Allowed values from a predefined list:
```json
{
  "validators": [
    { "type": "enum", "value": ["development", "staging", "production"] }
  ]
}
```

### Numeric Validators (Integer/Number)

#### min
Minimum value:
```json
{
  "validators": [
    { "type": "min", "value": 1 }
  ]
}
```

#### max
Maximum value:
```json
{
  "validators": [
    { "type": "max", "value": 100 }
  ]
}
```

#### enum
Allowed values from a predefined list:
```json
{
  "validators": [
    { "type": "enum", "value": [1, 2, 4, 8, 16, 32] }
  ]
}
```

### Array Validators

#### minItems
Minimum number of items:
```json
{
  "validators": [
    { "type": "minItems", "value": 1 }
  ]
}
```

#### maxItems
Maximum number of items:
```json
{
  "validators": [
    { "type": "maxItems", "value": 10 }
  ]
}
```

#### uniqueItems
Ensure all items are unique:
```json
{
  "validators": [
    { "type": "uniqueItems", "value": true }
  ]
}
```

### Reference Validators (for type "reference")

#### serviceType
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

#### sameOrigin
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

## Property Source

The `source` field controls who can set and update a property value. This enables proper separation between user-provided configuration and agent-discovered information.

### Source Values

#### input (default)
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

#### agent
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

### Source Usage Patterns

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

## Property Updatability

The `updatable` field controls when and if a property can be modified after initial creation. This prevents accidental changes to immutable infrastructure or ensures changes only happen in safe states.

### Updatable Values

#### always (default)
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

#### never
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

#### statuses
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

### Updatability Patterns

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

## Complete Examples

### VM Service Type with Mixed Sources

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

## API Usage

### Retrieving ServiceType with Schema

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

### Validating Properties

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

## Best Practices

1. **Start Simple**: Begin with basic type validation and add validators as needed
2. **Use Descriptive Labels**: Provide clear, human-readable labels for all properties
3. **Set Reasonable Defaults**: Use default values for optional properties when appropriate
4. **Validate Early**: Use the validation endpoint during development to test schemas
5. **Document Constraints**: Use enum validators to clearly define allowed values
6. **Nested Validation**: Leverage object and array types for complex configurations
7. **Error Handling**: Always check validation results before processing service properties

## Error Messages

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

## Migration and Updates

When updating property schemas:

1. **Backward Compatibility**: Ensure existing services remain valid
2. **Gradual Migration**: Add new optional properties before making them required
3. **Validation Testing**: Use the validation endpoint to test schema changes
4. **Documentation**: Update service documentation when schemas change