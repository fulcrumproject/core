# TypeScript Client Generation

Questo documento spiega come funziona la generazione automatica del client TypeScript per l'API Fulcrum Core.

## Overview

Il client TypeScript viene generato automaticamente dal file `docs/openapi.yaml` utilizzando OpenAPI Generator e pubblicato su npm come `@fulcrum/core-client`.

## Workflow

### Trigger Automatici

Il client viene rigenerato automaticamente quando:

1. **Push su `main` o `dev`** con modifiche a `docs/openapi.yaml`
2. **Pull Request** che modificano `docs/openapi.yaml`

### Trigger Manuali

Puoi attivare manualmente la generazione:

1. Vai su **Actions** → **Generate and Publish TypeScript Client**
2. Clicca **Run workflow**
3. Inserisci la versione desiderata (es: `1.0.0`)
4. Opzionalmente forza la pubblicazione

## Configurazione

### File di Configurazione

Il file `openapi-generator-config.json` contiene tutte le opzioni per la generazione:

```json
{
  "supportsES6": true,
  "npmName": "@fulcrum/core-client",
  "typescriptThreePlus": true,
  "withInterfaces": true,
  "stringEnums": true,
  "nullSafe": true,
  "useSingleRequestParameter": true,
  "enumPropertyNaming": "UPPERCASE",
  "modelPropertyNaming": "camelCase"
}
```

### Secrets Richiesti

- `NPM_TOKEN`: Token di autenticazione per npm (già configurato)

## Output Generato

Il client generato ha questa struttura:

```
generated-client/
├── api/                    # Classi API
│   ├── agents-api.ts      # API per gli agenti
│   ├── events-api.ts      # API per gli eventi
│   ├── jobs-api.ts        # API per i job
│   ├── metrics-api.ts     # API per le metriche
│   ├── participants-api.ts # API per i partecipanti
│   ├── services-api.ts    # API per i servizi
│   └── tokens-api.ts      # API per i token
├── models/                 # Interfacce TypeScript
│   ├── agent.ts
│   ├── agent-status.ts
│   ├── event.ts
│   ├── job.ts
│   ├── participant.ts
│   ├── service.ts
│   └── token.ts
├── runtime.ts             # Client HTTP
├── index.ts              # Esportazioni principali
├── package.json          # Configurazione npm
└── tsconfig.json         # Configurazione TypeScript
```

## Utilizzo del Client

### Installazione

```bash
npm install @fulcrum/core-client
```

### Configurazione Base

```typescript
import { Configuration, AgentsApi, ParticipantsApi } from '@fulcrum/core-client';

const config = new Configuration({
  basePath: 'https://api.fulcrum.testudosrl.dev/api/v1',
  accessToken: 'your-oauth-token',
});

const agentsApi = new AgentsApi(config);
const participantsApi = new ParticipantsApi(config);
```

### Esempi di Utilizzo

#### Lista Agenti

```typescript
// Lista tutti gli agenti
const agents = await agentsApi.getAgents();

// Con paginazione
const agentsPage = await agentsApi.getAgents(1, 20);
```

#### Creazione di un Agente

```typescript
const newAgent = await agentsApi.createAgent({
  name: "aws-agent-01",
  status: "New",
  participantId: "123e4567-e89b-12d3-a456-426614174000",
  agentTypeId: "456e7890-e89b-12d3-a456-426614174000"
});
```

#### Gestione Servizi

```typescript
import { ServicesApi } from '@fulcrum/core-client';

const servicesApi = new ServicesApi(config);

// Lista servizi
const services = await servicesApi.getServices();

// Avvia un servizio
await servicesApi.startService("service-id");

// Ferma un servizio
await servicesApi.stopService("service-id");
```

#### Gestione Job

```typescript
import { JobsApi } from '@fulcrum/core-client';

const jobsApi = new JobsApi(config);

// Lista job
const jobs = await jobsApi.getJobs();

// Job in attesa per l'agente
const pendingJobs = await jobsApi.getPendingJobs();
```

## Autenticazione

Il client supporta l'autenticazione OAuth2 Bearer token come definito nello spec OpenAPI:

```typescript
const config = new Configuration({
  basePath: 'https://api.fulcrum.testudosrl.dev/api/v1',
  accessToken: 'your-oauth-token',
  headers: {
    'Authorization': 'Bearer your-token'
  }
});
```

## Gestione Errori

```typescript
try {
  const agents = await agentsApi.getAgents();
} catch (error) {
  if (error.response) {
    console.error('API Error:', error.response.status, error.response.data);
  } else {
    console.error('Network Error:', error.message);
  }
}
```

## Type Safety

Il client fornisce type safety completa per:

- **Parametri di input**: Validazione automatica dei tipi
- **Risposte API**: Tipi TypeScript per tutte le risposte
- **Enum**: String enum per stati e tipi
- **Interfacce**: Interfacce complete per tutti i modelli

## Versioning

Il client segue il semantic versioning:

- **Major**: Breaking changes nell'API
- **Minor**: Nuove funzionalità compatibili
- **Patch**: Bug fixes e miglioramenti

## Troubleshooting

### Problemi Comuni

1. **Token scaduto**: Aggiorna il token OAuth2
2. **CORS**: Verifica che il server supporti CORS
3. **Versioni**: Assicurati di usare la versione corretta del client

### Debug

```typescript
const config = new Configuration({
  basePath: 'https://api.fulcrum.testudosrl.dev/api/v1',
  accessToken: 'your-oauth-token',
  // Abilita debug
  middleware: [{
    pre: (context) => {
      console.log('Request:', context.init);
      return context;
    },
    post: (context) => {
      console.log('Response:', context.response);
      return context;
    }
  }]
});
```

## Contribuire

Per modificare la generazione del client:

1. Modifica `openapi-generator-config.json`
2. Aggiorna `docs/openapi.yaml` se necessario
3. Testa localmente con `npm run generate-client`
4. Commit e push per attivare la generazione automatica