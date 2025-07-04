# Fulcrum Core API Client

Auto-generated TypeScript client for the Fulcrum Core API.

## Installation

```bash
pnpm add @fulcrum/core-client
```

## Usage

```typescript
import { Configuration, AgentsApi, ParticipantsApi } from '@fulcrum/core-client';

const config = new Configuration({
  basePath: 'https://api.fulcrum.testudosrl.dev/api/v1',
  accessToken: 'your-oauth-token',
});

const agentsApi = new AgentsApi(config);
const participantsApi = new ParticipantsApi(config);

// Example usage
const agents = await agentsApi.getAgents();
const participants = await participantsApi.getParticipants();
```

## Authentication

This client supports OAuth2 Bearer token authentication as defined in the OpenAPI specification.

## Generated from

Generated from OpenAPI specification version: b36b60a0a53298bf943a4559e46f4f3b64ae84e6
Last updated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
