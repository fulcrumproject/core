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
    "type": "string|integer|number|boolean|object|array|json",
    "label": "Human-readable label (optional)",
    "required": true|false,
    "default": "default value (optional)",
    "immutable": true|false,                 // optional, if true property cannot be changed after creation
    "secret": {                              // optional, for sensitive values
      "type": "persistent|ephemeral"
    },
    "generator": {                           // optional, for automatic value generation
      "type": "pool|custom",
      "config": {...}                        // generator-specific configuration
    },
    "validators": [...],                     // property-level validators (source, mutable, etc.)
    "properties": {...},                     // for object types
    "items": {...}                           // for array types
  }
}
```

**Field Descriptions:**
- **type**: Data type of the property (primitive or complex)
- **label**: Human-readable label for UI display
- **required**: Whether the property must be provided
- **default**: Default value if not provided
- **immutable**: If `true`, property cannot be changed after creation (defaults to `false`)
- **secret**: Configuration for secure vault storage (persistent or ephemeral secrets)
- **generator**: Configuration for automatic value generation (e.g., pool allocation)
- **validators**: Array of validation rules to apply to the property value (includes `source`, `mutable`, etc.)
- **properties**: Schema for nested object properties (only for `type: "object"`)
- **items**: Schema for array elements (only for `type: "array"`)

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

##### JSON
For properties that accept any valid JSON value without schema validation:

```json
{
  "customData": {
    "type": "json",
    "label": "Custom Configuration",
    "required": false
  }
}
```

**Note:** The `json` type accepts any valid JSON value (strings, numbers, objects, arrays, booleans, null) without schema validation. It's useful for:
- Service pool values that can be strings, objects, or arrays
- Service options with flexible value structures
- Properties with dynamic, unstructured content

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
        {
          "type": "min",
          "config": {
            "value": 1
          }
        },
        {
          "type": "max",
          "config": {
            "value": 65535
          }
        }
      ]
    },
    "validators": [
      {
        "type": "minItems",
        "config": {
          "value": 1
        }
      },
      {
        "type": "maxItems",
        "config": {
          "value": 10
        }
      }
    ]
  }
}
```

### Validators

Validators provide additional constraints beyond basic type checking. Fulcrum Core supports two types of validators:

1. **Property Validators**: Applied to individual properties to validate their values
2. **Schema Validators**: Applied at the schema level to validate relationships between properties

Each validator is configured as an object with a `type` field and validator-specific configuration (commonly a `value` or `config` field).

#### Property Validators

Property validators are specified in the `validators` array within a property definition. They validate the property's value against specific rules.

#### String Validators

##### minLength
Minimum string length:
```json
{
  "validators": [
    {
      "type": "minLength",
      "config": {
        "value": 3
      }
    }
  ]
}
```

##### maxLength
Maximum string length:
```json
{
  "validators": [
    {
      "type": "maxLength",
      "config": {
        "value": 50
      }
    }
  ]
}
```

##### pattern
Regular expression pattern:
```json
{
  "validators": [
    {
      "type": "pattern",
      "config": {
        "pattern": "^[a-zA-Z0-9_-]+$"
      }
    }
  ]
}
```

##### enum
Allowed values from a predefined list:
```json
{
  "validators": [
    {
      "type": "enum",
      "config": {
        "values": ["development", "staging", "production"]
      }
    }
  ]
}
```

#### Numeric Validators (Integer/Number)

##### min
Minimum value:
```json
{
  "validators": [
    {
      "type": "min",
      "config": {
        "value": 1
      }
    }
  ]
}
```

##### max
Maximum value:
```json
{
  "validators": [
    {
      "type": "max",
      "config": {
        "value": 100
      }
    }
  ]
}
```

##### enum
Allowed values from a predefined list:
```json
{
  "validators": [
    {
      "type": "enum",
      "config": {
        "values": [1, 2, 4, 8, 16, 32]
      }
    }
  ]
}
```

#### Array Validators

##### minItems
Minimum number of items:
```json
{
  "validators": [
    {
      "type": "minItems",
      "config": {
        "value": 1
      }
    }
  ]
}
```

##### maxItems
Maximum number of items:
```json
{
  "validators": [
    {
      "type": "maxItems",
      "config": {
        "value": 10
      }
    }
  ]
}
```

##### uniqueItems
Ensure all items are unique:
```json
{
  "validators": [
    {
      "type": "uniqueItems",
      "config": {
        "value": true
      }
    }
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
      {
        "type": "serviceType",
        "config": {
          "value": "MySQL"
        }
      }
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
      {
        "type": "serviceType",
        "config": {
          "value": ["MySQL", "PostgreSQL", "MongoDB"]
        }
      }
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
      {
        "type": "sameOrigin",
        "config": {
          "value": "consumer"
        }
      }
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
      {
        "type": "sameOrigin",
        "config": {
          "value": "group"
        }
      }
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
      {
        "type": "serviceType",
        "config": {
          "value": ["NodeJS-API", "Python-API"]
        }
      },
      {
        "type": "sameOrigin",
        "config": {
          "value": "consumer"
        }
      }
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
      {
        "type": "serviceOption",
        "config": {
          "value": "os"
        }
      }
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
      {
        "type": "serviceOption",
        "config": {
          "value": "machine_type"
        }
      }
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
      {
        "type": "serviceOption",
        "config": {
          "value": "region"
        }
      }
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
      {
        "type": "serviceOption",
        "config": {
          "value": "disk_type"
        }
      }
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
        {
          "type": "serviceOption",
          "config": {
            "value": "os"
          }
        }
      ]
    },
    "machineType": {
      "type": "string",
      "label": "Machine Type",
      "required": true,
      "validators": [
        {
          "type": "serviceOption",
          "config": {
            "value": "machine_type"
          }
        }
      ]
    },
    "region": {
      "type": "string",
      "label": "Region",
      "required": true,
      "validators": [
        {
          "type": "serviceOption",
          "config": {
            "value": "region"
          }
        }
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

#### Schema Validators

Schema validators operate at the schema level and validate relationships or constraints across multiple properties. They are specified in the root `validators` array of the property schema (not within individual property definitions).

**Schema Structure with Validators:**
```json
{
  "properties": {
    "propertyA": {
      "type": "string"
    },
    "propertyB": {
      "type": "string"
    }
  },
  "validators": [
    {
      "type": "validatorType",
      "config": {...}
    }
  ]
}
```

**Note:** Schema validators are currently supported by the engine infrastructure but no built-in schema validators are provided in the base system. Custom schema validators can be registered by the application to enforce cross-property validation rules such as:

- **Mutual exclusivity**: Ensure only one of several properties is set
- **Conditional requirements**: Require property B when property A has a specific value
- **Cross-property constraints**: Validate relationships between property values
- **Complex business rules**: Enforce domain-specific validation logic

**Example Registration (Application Code):**
```go
// Custom schema validator implementation
type ExactlyOneValidator struct{}

func (v *ExactlyOneValidator) Validate(
    ctx context.Context,
    schemaCtx ServicePropertyContext,
    operation Operation,
    oldProperties map[string]any,
    newProperties map[string]any,
    config map[string]any,
) error {
    // Validation logic that checks multiple properties
    return nil
}

// Register with engine
engine := NewServicePropertyEngine(store, vault)
engine.RegisterSchemaValidator("exactlyOne", &ExactlyOneValidator{})
```

Schema validators enable powerful cross-property validation while keeping individual property definitions clean and focused.

### Property Pool Allocation

Properties can use automatic pool allocation via the `pool` generator. Service pools manage finite, exclusive resources (IPs, ports, hostnames) with automatic allocation and lifecycle management.

**Basic Usage:**
```json
{
  "publicIp": {
    "type": "string",
    "label": "Public IP Address",
    "immutable": true,
    "generator": {
      "type": "pool",
      "config": {
        "poolType": "public_ip"
      }
    },
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "system"
        }
      }
    ]
  }
}
```

This configures the `publicIp` property for automatic allocation from a pool with type "public_ip" during service creation.

**How it works:**
1. The generator `config.poolType` field specifies which pool type to allocate from (e.g., "public_ip", "hostname", "port")
2. The `source` validator marks the property as system-generated (preventing manual user/agent input)
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
- **System-source**: Use `source` validator to prevent manual setting

**Pool Types:**

List pools (pre-configured values):
```json
{
  "ipAddress": {
    "type": "string",
    "label": "IP Address",
    "immutable": true,
    "generator": {
      "type": "pool",
      "config": {
        "poolType": "public_ip"
      }
    },
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "system"
        }
      }
    ]
  }
}
```

Subnet pools (automatic CIDR allocation):
```json
{
  "privateIp": {
    "type": "string",
    "label": "Private IP Address",
    "immutable": true,
    "generator": {
      "type": "pool",
      "config": {
        "poolType": "private_ip"
      }
    },
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "system"
        }
      }
    ]
  }
}
```

JSON type for complex values:
```json
{
  "hostname": {
    "type": "json",
    "label": "Hostname Configuration",
    "immutable": true,
    "generator": {
      "type": "pool",
      "config": {
        "poolType": "hostname"
      }
    },
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "system"
        }
      }
    ]
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
- `"pool generator config missing 'poolType'"` - Generator config missing poolType field
- `"pool generator config 'poolType' must be a string"` - Pool type must be a string value
- `"pool generator config 'poolType' cannot be empty"` - Empty poolType value
- `"pool generator requires service context"` - Service context missing (internal error)
- `"agent does not have a pool set configured"` - Agent's servicePoolSetId is not set
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
      "immutable": true,
      "generator": {
        "type": "pool",
        "config": {
          "poolType": "public_ip"
        }
      },
      "validators": [
        {
          "type": "source",
          "config": {
            "source": "system"
          }
        }
      ]
    },
    "privateIp": {
      "type": "string",
      "label": "Private IP",
      "immutable": true,
      "generator": {
        "type": "pool",
        "config": {
          "poolType": "private_ip"
        }
      },
      "validators": [
        {
          "type": "source",
          "config": {
            "source": "system"
          }
        }
      ]
    },
    "hostname": {
      "type": "json",
      "label": "Hostname Config",
      "immutable": true,
      "generator": {
        "type": "pool",
        "config": {
          "poolType": "hostname"
        }
      },
      "validators": [
        {
          "type": "source",
          "config": {
            "source": "system"
          }
        }
      ]
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

### Generators

Generators are components that automatically compute or allocate property values during service creation or updates. Unlike validators that check existing values, generators create values that weren't provided by the user.

**Key Concepts:**
- Generators run during the schema engine's `Apply` operation
- They can generate values based on current context (service state, agent, etc.)
- Multiple generators can be registered with the schema engine
- Generators are optional - properties work without them

**Built-in Generator: Pool Generator**

The `servicePoolType` field uses the pool generator internally to allocate resources from service pools. This generator:
- Allocates values from pools configured in the agent's service pool set
- Ensures exclusive allocation (one value per service)
- Automatically releases values when services are deleted
- Supports multiple pool types (list pools, subnet pools)

**Example: Pool Generator (via servicePoolType)**
```json
{
  "publicIp": {
    "type": "string",
    "source": "system",
    "updatable": "never",
    "servicePoolType": "public_ip"
  }
}
```

This property uses the pool generator to:
1. Find a pool with type "public_ip" in the agent's pool set
2. Allocate an available value from that pool
3. Store the actual value in the property
4. Track the allocation for cleanup on deletion

**Custom Generators**

Applications can register custom generators for domain-specific value generation:

```go
// Custom generator implementation
type HostnameGenerator struct{}

func (g *HostnameGenerator) Generate(
    ctx context.Context,
    schemaCtx ServicePropertyContext,
    propPath string,
    currentValue any,
    config map[string]any,
) (value any, generated bool, err error) {
    // Generate hostname based on service context
    hostname := fmt.Sprintf("%s-%s", config["prefix"], schemaCtx.Service.ID)
    return hostname, true, nil
}

func (g *HostnameGenerator) ValidateConfig(propPath string, config map[string]any) error {
    // Validate generator configuration
    if config["prefix"] == nil {
        return fmt.Errorf("prefix is required")
    }
    return nil
}

// Register with engine
engine.RegisterGenerator("hostname", &HostnameGenerator{})
```

**Generator Configuration:**

Generators would be configured in property definitions (future enhancement):
```json
{
  "hostname": {
    "type": "string",
    "source": "system",
    "generator": {
      "type": "hostname",
      "config": {
        "prefix": "server"
      }
    }
  }
}
```

**Generator vs Default vs Pool:**
- **Default**: Static value used when property not provided
- **Generator**: Dynamic value computed at runtime based on context
- **Pool (servicePoolType)**: Special generator for allocating from finite resource pools

**Return Values:**
- `(value, true, nil)`: Value was generated successfully
- `(nil, false, nil)`: No generation needed (e.g., value already exists)
- `(nil, false, error)`: Generation failed

**Use Cases:**
- **Unique identifiers**: Generate UUIDs, slugs, or sequential IDs
- **Computed values**: Calculate based on other properties
- **Resource allocation**: Assign from pools (IPs, ports, hostnames)
- **Timestamps**: Set creation or modification times
- **Derived properties**: Generate based on service context

Generators make schemas more powerful by automating value creation while keeping property definitions declarative.

### Property Secrets

Properties can be marked as secrets to enable secure storage using encrypted vault storage. When a property is marked as secret, users provide the actual sensitive value, which is stored encrypted in the vault, and the property value is replaced with a `vault://reference` string.

**Basic Usage:**
```json
{
  "apiKey": {
    "type": "string",
    "label": "API Key",
    "source": "input",
    "required": true,
    "secret": {
      "type": "persistent"
    }
  }
}
```

This marks the `apiKey` property as a persistent secret. When creating a service, users provide the actual API key value, which is then stored encrypted and replaced with a vault reference.

**How it works:**
1. User provides actual secret value when creating/updating a service
2. System encrypts the value using AES-256-GCM and stores it in the vault
3. A unique reference (e.g., `vault://abc123def456`) is generated
4. The property value is replaced with this reference in the service properties
5. Agents retrieve the actual value by calling `GET /api/v1/vault/secrets/{reference}`
6. Secrets are automatically cleaned up based on their type

**Secret Types:**

There are two types of secrets:

- **`persistent`**: Long-lived secrets that remain until service deletion
  - Use for: API keys, database passwords, SSL certificates, long-term credentials
  - Cleanup: When service reaches terminal state (e.g., Deleted)
  - Example: API key needed throughout the entire service lifetime

- **`ephemeral`**: Short-lived secrets that are deleted after each job completion
  - Use for: Temporary passwords, one-time tokens, initialization secrets
  - Cleanup: After every job completion (success or failure)
  - Example: Initial setup password that should only exist during first job

**Type Restrictions:**

Only primitive types can be secrets:
- ✅ `string`, `integer`, `number`, `boolean`, `json`
- ❌ `object`, `array` (the container itself)

However, objects can contain properties that are secrets, and arrays of objects can have items with secret properties:

```json
{
  "database": {
    "type": "object",
    "properties": {
      "host": {
        "type": "string"
      },
      "password": {
        "type": "string",
        "secret": {
          "type": "persistent"
        }
      }
    }
  },
  "users": {
    "type": "array",
    "items": {
      "type": "object",
      "properties": {
        "username": {
          "type": "string"
        },
        "password": {
          "type": "string",
          "secret": {
            "type": "ephemeral"
          }
        }
      }
    }
  }
}
```

**Agent Resolution:**

Agents resolve vault references by calling the vault resolution endpoint:

```http
GET /api/v1/vault/secrets/abc123def456
Authorization: Bearer <agent-token>
```

Response:
```json
{
  "value": "actual-secret-value"
}
```

**Security Features:**

1. **Encryption**: All secrets encrypted with AES-256-GCM before storage
2. **Access Control**: Only agents can access the vault resolution endpoint
3. **No Exposure**: Secrets never appear in plain text in API responses or logs
4. **Automatic Cleanup**: Ephemeral secrets cleaned after each job, persistent on deletion
5. **Reference Format**: `vault://` prefix makes secrets identifiable in properties

**Cleanup Behavior:**

**Ephemeral Secrets:**
```
Job 1 (install): Creates vault://secret1 → Job completes → secret1 deleted ✅
Job 2 (configure): Creates vault://secret2 → Job completes → secret2 deleted ✅
Job 3 (start): Creates vault://secret3 → Job completes → secret3 deleted ✅
```

**Persistent Secrets:**
```
Service creation: Creates vault://apikey → Remains throughout service life
Service operations: Agents use vault://apikey for each operation
Service deletion: vault://apikey deleted ✅
```

**Mixed Example:**
```json
{
  "apiKey": {
    "type": "string",
    "secret": {
      "type": "persistent"
    }
  },
  "setupPassword": {
    "type": "string",
    "secret": {
      "type": "ephemeral"
    }
  }
}
```

When service is created:
- Both secrets stored: `vault://key1` and `vault://pass1`

After first job:
- Persistent key remains: `vault://key1` ✅
- Ephemeral password deleted: `vault://pass1` ❌

On service deletion:
- All remaining secrets deleted: `vault://key1` ❌

**Benefits:**

- **Security**: Sensitive data never exposed in API responses or database
- **Flexibility**: Two secret types for different use cases
- **Automatic**: Cleanup handled by system, no manual intervention
- **Nested**: Support for secrets in complex structures
- **Audit**: All secret access is logged and controlled

**Error Messages:**

- `"only primitive types (string, integer, number, boolean, json) can be secrets"` - Attempted to mark object/array as secret
- `"secret type must be 'persistent' or 'ephemeral'"` - Invalid secret type
- `"secret configuration must include 'type' field"` - Missing type in secret config
- `"vault is not configured"` - VAULT_ENCRYPTION_KEY not set in environment
- `"failed to store secret in vault"` - Vault storage error
- `"secret with reference X not found"` - Agent tried to resolve non-existent reference

**Configuration:**

The vault system requires configuration via environment variable:

```bash
# Generate a 32-byte (256-bit) hex key
openssl rand -hex 32

# Set in environment
export VAULT_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

Without this configuration, secret properties will not work and service creation will fail with a validation error.

**Complete Example:**

Service type with secrets:
```json
{
  "name": "Web Application",
  "propertySchema": {
    "appName": {
      "type": "string",
      "label": "Application Name",
      "source": "input",
      "required": true
    },
    "apiKey": {
      "type": "string",
      "label": "API Key",
      "source": "input",
      "required": true,
      "secret": {
        "type": "persistent"
      }
    },
    "initialPassword": {
      "type": "string",
      "label": "Initial Admin Password",
      "source": "input",
      "required": true,
      "secret": {
        "type": "ephemeral"
      }
    },
    "databaseConfig": {
      "type": "object",
      "properties": {
        "host": {
          "type": "string",
          "required": true
        },
        "port": {
          "type": "integer",
          "default": 5432
        },
        "password": {
          "type": "string",
          "required": true,
          "secret": {
            "type": "persistent"
          }
        }
      }
    }
  }
}
```

Service creation request:
```json
{
  "name": "my-web-app",
  "serviceTypeId": "web-app-type-uuid",
  "properties": {
    "appName": "MyApp",
    "apiKey": "sk_live_abc123xyz789",
    "initialPassword": "temp_pass_123",
    "databaseConfig": {
      "host": "db.example.com",
      "port": 5432,
      "password": "db_secret_password"
    }
  }
}
```

Stored service properties (what API returns):
```json
{
  "appName": "MyApp",
  "apiKey": "vault://a1b2c3d4",
  "initialPassword": "vault://e5f6g7h8",
  "databaseConfig": {
    "host": "db.example.com",
    "port": 5432,
    "password": "vault://i9j0k1l2"
  }
}
```

Agent resolves secrets:
```bash
# Resolve API key (persistent)
curl -H "Authorization: Bearer agent-token" \
  https://api.fulcrum.example/api/v1/vault/secrets/a1b2c3d4
# Returns: {"value": "sk_live_abc123xyz789"}

# Resolve initial password (ephemeral)
curl -H "Authorization: Bearer agent-token" \
  https://api.fulcrum.example/api/v1/vault/secrets/e5f6g7h8
# Returns: {"value": "temp_pass_123"}

# Resolve database password (persistent)
curl -H "Authorization: Bearer agent-token" \
  https://api.fulcrum.example/api/v1/vault/secrets/i9j0k1l2
# Returns: {"value": "db_secret_password"}
```

After first job completes:
- `apiKey` (persistent): Still `vault://a1b2c3d4` ✅
- `initialPassword` (ephemeral): Reference deleted from vault ❌
- `databaseConfig.password` (persistent): Still `vault://i9j0k1l2` ✅

### Property Source

The `source` validator controls who can set and update a property value. This enables proper separation between user-provided configuration, agent-discovered information, and system-generated values.

**Implementation:** The source control is enforced through the `source` validator in the property's validators array, not through a dedicated `source` field.

#### Source Values

##### input (default)
Properties set by users through the API. These represent the desired configuration.

```json
{
  "instanceName": {
    "type": "string",
    "label": "Instance Name",
    "required": true,
    "validators": []
  }
}
```

**Behavior:**
- Users can set this property when creating a service
- Users can update this property (subject to immutability and mutable validator rules)
- Agents cannot modify this property
- If no `source` validator is specified, properties default to user input

##### agent
Properties set by agents after provisioning resources. These represent actual provisioned values.

```json
{
  "ipAddress": {
    "type": "string",
    "label": "Assigned IP Address",
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  }
}
```

**Behavior:**
- Users cannot set or update this property
- Agents can set this property when completing a job
- Agents can update this property (subject to immutability and mutable validator rules)
- Typically used for discovered values like IP addresses, ports, UUIDs

##### system
Properties automatically generated by the system through generators. Users and agents cannot set these manually.

```json
{
  "publicIp": {
    "type": "string",
    "label": "Public IP Address",
    "immutable": true,
    "generator": {
      "type": "pool",
      "config": {
        "poolType": "public_ip"
      }
    },
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "system"
        }
      }
    ]
  }
}
```

**Behavior:**
- System generates the value automatically via the configured generator
- Users and agents cannot set or modify this property manually
- Typically used with pool allocation or computed values

#### Source Usage Patterns

**Configuration vs Discovery**
```json
{
  "diskSize": {
    "type": "integer",
    "label": "Disk Size (GB)",
    "required": true
  },
  "actualDiskPath": {
    "type": "string",
    "label": "Disk Path",
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  }
}
```

User specifies `diskSize`, agent reports back `actualDiskPath` after provisioning.

**System-Generated with Pool**
```json
{
  "ipAddress": {
    "type": "string",
    "label": "IP Address",
    "immutable": true,
    "generator": {
      "type": "pool",
      "config": {
        "poolType": "public_ip"
      }
    },
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "system"
        }
      }
    ]
  }
}
```

System automatically allocates from pool, preventing manual setting.

### Property Updatability

Properties can be controlled for mutability in two ways:

1. **`immutable` field**: Simple flag to prevent any updates after creation
2. **`mutable` validator**: State-specific mutability control for complex workflows

#### Immutable Properties

Use the `immutable: true` field to mark properties that cannot be changed after initial creation. This is enforced at the schema engine level.

```json
{
  "uuid": {
    "type": "string",
    "label": "Instance UUID",
    "immutable": true,
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  }
}
```

**Behavior:**
- Can be set during initial service creation (for user properties) or first job completion (for agent properties)
- Cannot be changed after initial creation
- Any attempt to update returns a validation error: `"property is immutable and cannot be changed"`
- Suitable for identifiers, region selection, and other immutable configuration
- Default is `false` (mutable)

**Examples:**

System-generated immutable identifier:
```json
{
  "publicIp": {
    "type": "string",
    "label": "Public IP",
    "immutable": true,
    "generator": {
      "type": "pool",
      "config": {
        "poolType": "public_ip"
      }
    },
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "system"
        }
      }
    ]
  }
}
```

Agent-discovered immutable value:
```json
{
  "instanceId": {
    "type": "string",
    "label": "Cloud Instance ID",
    "immutable": true,
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  }
}
```

User-configured immutable value:
```json
{
  "region": {
    "type": "string",
    "label": "Cloud Region",
    "immutable": true,
    "required": true,
    "validators": [
      {
        "type": "enum",
        "config": {
          "values": ["us-east-1", "us-west-2", "eu-west-1"]
        }
      }
    ]
  }
}
```

#### State-Specific Mutability

Use the `mutable` validator to control when properties can be updated based on service state. This is useful for properties that can only be changed in specific lifecycle states.

```json
{
  "diskSize": {
    "type": "integer",
    "label": "Disk Size (GB)",
    "required": true,
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Stopped"]
        }
      },
      {
        "type": "min",
        "config": {
          "value": 10
        }
      },
      {
        "type": "max",
        "config": {
          "value": 1000
        }
      }
    ]
  }
}
```

**Behavior:**
- Only applies to update operations (not create)
- Checks if service is in one of the allowed states before allowing update
- Error message: `"property cannot be updated in state 'CurrentState'"`
- Suitable for properties that require service to be in a safe state for modification

**Validator Configuration:**
- `updatableIn`: Array of service status strings when updates are allowed
- Must contain at least one status
- Statuses should match those defined in the service type's lifecycle schema

**Examples:**

Hardware changes requiring stopped state:
```json
{
  "cpu": {
    "type": "integer",
    "label": "CPU Cores",
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Stopped"]
        }
      },
      {
        "type": "enum",
        "config": {
          "values": [1, 2, 4, 8, 16, 32]
        }
      }
    ]
  },
  "memory": {
    "type": "integer",
    "label": "Memory (GB)",
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Stopped"]
        }
      },
      {
        "type": "enum",
        "config": {
          "values": [1, 2, 4, 8, 16, 32, 64]
        }
      }
    ]
  }
}
```

Hot-updatable configuration:
```json
{
  "maxConnections": {
    "type": "integer",
    "label": "Max Connections",
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Started", "Stopped"]
        }
      }
    ]
  }
}
```

Configuration updatable only during maintenance:
```json
{
  "databaseConfig": {
    "type": "object",
    "properties": {
      "host": {
        "type": "string"
      },
      "port": {
        "type": "integer"
      }
    },
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Maintenance"]
        }
      }
    ]
  }
}
```

#### Combined Mutability Patterns

**Always Mutable (default)**
```json
{
  "description": {
    "type": "string",
    "label": "Description"
  }
}
```
No immutable field and no mutable validator = always updatable.

**Completely Immutable**
```json
{
  "createdDate": {
    "type": "string",
    "label": "Creation Date",
    "immutable": true
  }
}
```
Cannot be changed after creation in any state.

**State-Conditional Mutability**
```json
{
  "diskSize": {
    "type": "integer",
    "label": "Disk Size (GB)",
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Stopped"]
        }
      }
    ]
  }
}
```
Can be updated, but only when service is in Stopped state.

**Combining with Source Control**
```json
{
  "healthStatus": {
    "type": "string",
    "label": "Health Status",
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  }
}
```
Agent can update in any state (no immutable or mutable validator).

```json
{
  "maintenanceMode": {
    "type": "boolean",
    "label": "Maintenance Mode",
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Stopped", "Started"]
        }
      }
    ]
  }
}
```
User can update, but only in specific states.

### Complete Examples

#### VM Service Type with Mixed Sources

Here's a comprehensive example for a VM service type with user configuration, agent-discovered properties, and system-generated values:

```json
{
  "instanceName": {
    "type": "string",
    "label": "Instance Name",
    "required": true,
    "validators": [
      {
        "type": "minLength",
        "config": {
          "value": 3
        }
      },
      {
        "type": "maxLength",
        "config": {
          "value": 50
        }
      },
      {
        "type": "pattern",
        "config": {
          "pattern": "^[a-zA-Z0-9-]+$"
        }
      }
    ]
  },
  "region": {
    "type": "string",
    "label": "Cloud Region",
    "immutable": true,
    "required": true,
    "validators": [
      {
        "type": "enum",
        "config": {
          "values": ["us-east-1", "us-west-2", "eu-west-1"]
        }
      }
    ]
  },
  "cpu": {
    "type": "integer",
    "label": "CPU Cores",
    "required": true,
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Stopped"]
        }
      },
      {
        "type": "enum",
        "config": {
          "values": [1, 2, 4, 8, 16, 32]
        }
      }
    ]
  },
  "memory": {
    "type": "integer",
    "label": "Memory (GB)",
    "required": true,
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Stopped"]
        }
      },
      {
        "type": "enum",
        "config": {
          "values": [1, 2, 4, 8, 16, 32, 64]
        }
      }
    ]
  },
  "diskSize": {
    "type": "integer",
    "label": "Disk Size (GB)",
    "required": true,
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Stopped"]
        }
      },
      {
        "type": "min",
        "config": {
          "value": 10
        }
      },
      {
        "type": "max",
        "config": {
          "value": 1000
        }
      }
    ]
  },
  "imageId": {
    "type": "string",
    "label": "VM Image ID",
    "immutable": true,
    "required": true
  },
  "publicIp": {
    "type": "string",
    "label": "Public IP Address",
    "immutable": true,
    "generator": {
      "type": "pool",
      "config": {
        "poolType": "public_ip"
      }
    },
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "system"
        }
      }
    ]
  },
  "instanceId": {
    "type": "string",
    "label": "Cloud Instance ID",
    "immutable": true,
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  },
  "ipAddress": {
    "type": "string",
    "label": "Private IP Address",
    "immutable": true,
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  },
  "hostname": {
    "type": "string",
    "label": "Hostname",
    "immutable": true,
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  },
  "tags": {
    "type": "object",
    "label": "Resource Tags",
    "properties": {
      "environment": {
        "type": "string",
        "validators": [
          {
            "type": "enum",
            "config": {
              "values": ["dev", "staging", "prod"]
            }
          }
        ]
      },
      "owner": {
        "type": "string"
      }
    }
  }
}
```

#### Disk Service Type

Example for a managed disk with state-conditional resizing:

```json
{
  "name": {
    "type": "string",
    "label": "Disk Name",
    "required": true
  },
  "sizeGb": {
    "type": "integer",
    "label": "Size (GB)",
    "required": true,
    "validators": [
      {
        "type": "mutable",
        "config": {
          "updatableIn": ["Stopped"]
        }
      },
      {
        "type": "min",
        "config": {
          "value": 10
        }
      },
      {
        "type": "max",
        "config": {
          "value": 16384
        }
      }
    ]
  },
  "type": {
    "type": "string",
    "label": "Disk Type",
    "immutable": true,
    "required": true,
    "validators": [
      {
        "type": "enum",
        "config": {
          "values": ["ssd", "hdd", "nvme"]
        }
      }
    ]
  },
  "diskId": {
    "type": "string",
    "label": "Cloud Disk ID",
    "immutable": true,
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  },
  "actualSizeGb": {
    "type": "integer",
    "label": "Actual Size (GB)",
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
  },
  "devicePath": {
    "type": "string",
    "label": "Device Path",
    "immutable": true,
    "validators": [
      {
        "type": "source",
        "config": {
          "source": "agent"
        }
      }
    ]
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