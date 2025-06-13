# Keycloak Configuration Notes

## Custom Attributes Configuration

The automatic configuration of custom attributes (participant_id and agent_id) through the realm JSON configuration is not working as expected. These attributes need to be created manually in the Keycloak admin console.

### Steps to Create Custom Attributes:

1. Log into the Keycloak Admin Console
2. Select the "fulcrum" realm
3. Go to "User Federation" → "User Profile"
4. Add the following attributes:
   - `participant_id`: String type
   - `agent_id`: String type

These attributes are required for proper user identification and authorization in the Fulcrum system. The attributes are used in the protocol mappers to include the values in the JWT tokens.

Note: While the protocol mappers are correctly configured in the realm JSON, the underlying attributes must exist before the mappers can function properly.

## Not working configuration

```json
  "attributes": {
    "userProfileEnabled": true
  },
  "userProfile": {
    "attributes": [
      {
        "name": "username",
        "displayName": "${username}",
        "validations": {
          "length": {
            "min": 1,
            "max": 20
          },
          "username-prohibited-characters": {}
        }
      },
      {
        "name": "email",
        "displayName": "${email}",
        "validations": {
          "email": {},
          "length": {
            "max": 255
          }
        }
      },
      {
        "name": "firstName",
        "displayName": "${firstName}",
        "required": {
          "roles": ["user"]
        },
        "permissions": {
          "view": ["admin", "user"],
          "edit": ["admin"]
        },
        "validations": {
          "length": {
            "max": 169
          },
          "person-name-prohibited-characters": {}
        }
      },
      {
        "name": "lastName",
        "displayName": "${lastName}",
        "required": {
          "roles": ["user"]
        },
        "permissions": {
          "view": ["user"],
          "edit": ["admin"]
        },
        "validations": {
          "length": {
            "max": 238
          },
          "person-name-prohibited-characters": {}
        }
      },
      {
        "name": "participant_id",
        "displayName": "Participant ID",
        "required": {
          "roles": ["participant", "agent"],
          "scopes": []
        },
        "validations": {
          "length": {
            "min": 1,
            "max": 64
          }
        }
      },
      {
        "name": "agent_id",
        "displayName": "Agent ID",
        "required": {
          "roles": ["agent"],
          "scopes": []
        },
        "validations": {
          "length": {
            "min": 1,
            "max": 64
          }
        }
      }
    ]
  },
```