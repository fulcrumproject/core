# OpenAPI Specification

This directory contains the OpenAPI 3.1.0 specification for the Fulcrum Core API, split into multiple files for better organization and maintainability.

## Structure

```
docs/openapi/
├── openapi.yaml              # Main entry point (root document)
├── components/
│   ├── schemas/              # Schema definitions grouped by domain
│   │   ├── common.yaml           # Common schemas (ErrorRes, PageRes, UUID, JSONObject)
│   │   ├── participants.yaml     
│   │   ├── tokens.yaml           
│   │   └── ...
│   └── responses.yaml        # Reusable response definitions
└── paths/                    # Path definitions (one file per path)
    ├── participants.yaml
    ├── participants@{id}.yaml
    ├── tokens.yaml
    └── ...

```

## Working with the Split Specification

### Validation

Validate with Redocly CLI:

```bash
npx @redocly/cli lint docs/openapi/openapi.yaml
```

Then validate with swagger-cli:

```bash
npx swagger-cli validate docs/openapi-bundled.yaml
```

### Bundling (Optional)

If you need a single bundled file for tools that don't support `$ref`:

```bash
npx @redocly/cli bundle docs/openapi/openapi.yaml -o docs/openapi-bundled.yaml
```

## Development Workflow

1. Edit the appropriate file (schema, path, or response)
2. Validate with `npx @redocly/cli lint docs/openapi/openapi.yaml`
3. Commit the split files
4. If needed, bundle the specification with `npx @redocly/cli bundle docs/openapi/openapi.yaml -o docs/openapi-bundled.yaml`

