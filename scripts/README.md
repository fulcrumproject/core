# Script per Generazione Client TypeScript

Questo documento spiega come usare gli script locali per generare e testare il client TypeScript.

## Script Disponibili

### 1. `generate-client.sh` - Script Principale

Script bash completo per generare, testare e pubblicare il client TypeScript.

#### Utilizzo:

```bash
# Genera client con versione di default (0.0.1)
./scripts/generate-client.sh

# Genera client con versione specifica
./scripts/generate-client.sh 1.0.0

# Genera e pubblica su npm
./scripts/generate-client.sh 1.0.0 --publish
```

#### Funzionalità:

- ✅ Verifica prerequisiti (Node.js, npm, file OpenAPI)
- ✅ Installa OpenAPI Generator automaticamente
- ✅ Genera client TypeScript da `docs/openapi.yaml`
- ✅ Aggiunge README e metadata
- ✅ Installa dipendenze e build
- ✅ Test del client generato
- ✅ Pubblicazione su npm (opzionale)
- ✅ Output colorato e dettagliato

### 2. `test-client.js` - Script di Test

Script Node.js per testare il client TypeScript generato.

#### Utilizzo:

```bash
# Testa il client generato
node scripts/test-client.js

# Oppure via npm
npm run test-client
```

#### Test Eseguiti:

- ✅ Struttura file (verifica presenza file richiesti)
- ✅ Package.json (verifica campi obbligatori)
- ✅ TypeScript config (verifica configurazione)
- ✅ File API (verifica classi API)
- ✅ File modelli (verifica interfacce/tipi)
- ✅ Build (verifica compilazione)

## Script NPM

### Comandi Disponibili:

```bash
# Genera client di test (versione 0.0.1)
npm run generate-client:test

# Genera e pubblica client (versione 1.0.0)
npm run generate-client:publish

# Testa client generato
npm run test-client

# Build client generato
npm run client:build

# Installa dipendenze client
npm run client:install
```

## Workflow di Sviluppo

### 1. Sviluppo Locale

```bash
# 1. Modifica docs/openapi.yaml
# 2. Genera client di test
npm run generate-client:test

# 3. Testa il client
npm run test-client

# 4. Se tutto ok, pubblica
npm run generate-client:publish
```

### 2. Debug

```bash
# Genera client con output dettagliato
./scripts/generate-client.sh 0.0.1

# Testa struttura e build
npm run test-client
```

### 3. Pubblicazione

```bash
# Assicurati di essere loggato su npm
npm login

# Genera e pubblica
npm run generate-client:publish
```

## Prerequisiti

### Software Richiesto:

- **Node.js 18+**
- **npm 8+**
- **Git** (per metadata repository)

### File Richiesti:

- `docs/openapi.yaml` - Specifica OpenAPI
- `openapi-generator-config.json` - Configurazione generatore

### Opzionale:

- **jq** - Per aggiornamento automatico versione nel config

## Configurazione

### Variabili d'Ambiente:

```bash
# Per pubblicazione su npm
export NPM_TOKEN="your-npm-token"

# Per debug
export DEBUG="true"
```

### Configurazione OpenAPI Generator:

Il file `openapi-generator-config.json` controlla:

- Nome pacchetto npm
- Versione
- Opzioni TypeScript
- Naming conventions
- Struttura output

## Troubleshooting

### Problemi Comuni:

1. **"Node.js non trovato"**
   ```bash
   # Installa Node.js 18+
   brew install node@18  # macOS
   ```

2. **"npm non trovato"**
   ```bash
   # Installa npm
   npm install -g npm@latest
   ```

3. **"OpenAPI spec non trovato"**
   ```bash
   # Verifica che docs/openapi.yaml esista
   ls -la docs/openapi.yaml
   ```

4. **"Build fallita"**
   ```bash
   # Verifica dipendenze
   cd generated-client && npm install
   ```

5. **"Non sei loggato su npm"**
   ```bash
   # Login su npm
   npm login
   ```

### Debug Avanzato:

```bash
# Genera con output dettagliato
DEBUG=true ./scripts/generate-client.sh 0.0.1

# Testa solo build
cd generated-client && npm run build

# Verifica struttura
tree generated-client/
```

## Output

### Directory Generata:

```
generated-client/
├── api/           # Classi API
├── models/        # Interfacce TypeScript
├── runtime.ts     # Client HTTP
├── index.ts       # Esportazioni
├── package.json   # Configurazione npm
├── tsconfig.json  # Configurazione TypeScript
└── README.md      # Documentazione
```

### File di Test:

- `generated-client/` - Client generato
- `scripts/test-client.js` - Script di test
- Log dettagliati durante la generazione

## Integrazione con CI/CD

Gli script locali sono compatibili con la GitHub Action:

- Stessa configurazione
- Stessi file di output
- Stessi test
- Stessa struttura

Questo permette di testare localmente prima di pushare su GitHub.