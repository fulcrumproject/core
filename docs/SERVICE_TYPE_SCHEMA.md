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

## Complete Example

Here's a comprehensive example for a VM service type:

```json
{
  "cpu": {
    "type": "integer",
    "label": "CPU Cores",
    "required": true,
    "validators": [
      { "type": "enum", "value": [1, 2, 4, 8, 16, 32] }
    ]
  },
  "memory": {
    "type": "integer",
    "label": "Memory (MB)",
    "required": true,
    "validators": [
      { "type": "enum", "value": [512, 1024, 2048, 4096, 8192, 16384, 32768, 65536] }
    ]
  },
  "image_name": {
    "type": "string",
    "label": "Container Image",
    "required": true,
    "validators": [
      { "type": "minLength", "value": 5 },
      { "type": "pattern", "value": "^[a-z0-9-]+$" }
    ]
  },
  "environment": {
    "type": "string",
    "label": "Environment",
    "validators": [
      { "type": "enum", "value": ["development", "staging", "production"] }
    ]
  },
  "enable_monitoring": {
    "type": "boolean",
    "label": "Enable Monitoring",
    "default": true
  },
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
      }
    }
  },
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

## Migration and Updates

When updating property schemas:

1. **Backward Compatibility**: Ensure existing services remain valid
2. **Gradual Migration**: Add new optional properties before making them required
3. **Validation Testing**: Use the validation endpoint to test schema changes
4. **Documentation**: Update service documentation when schemas change