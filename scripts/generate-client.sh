#!/bin/bash

# Script per generare e testare localmente il client TypeScript
# Usage: ./scripts/generate-client.sh [version] [--publish]

set -e

# Colori per output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Funzione per log colorati
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configurazione
VERSION=${1:-"0.0.1"}
PUBLISH=${2:-""}
CLIENT_DIR="generated-client"
OPENAPI_SPEC="docs/openapi.yaml"
CONFIG_FILE="openapi-generator-config.json"

# Verifica prerequisiti
check_prerequisites() {
    log_info "Verificando prerequisiti..."

    # Verifica Node.js
    if ! command -v node &> /dev/null; then
        log_error "Node.js non trovato. Installa Node.js 18+"
        exit 1
    fi

    # Verifica npm
    if ! command -v npm &> /dev/null; then
        log_error "npm non trovato. Installa npm"
        exit 1
    fi

    # Verifica OpenAPI spec
    if [ ! -f "$OPENAPI_SPEC" ]; then
        log_error "File OpenAPI spec non trovato: $OPENAPI_SPEC"
        exit 1
    fi

    # Verifica config file
    if [ ! -f "$CONFIG_FILE" ]; then
        log_error "File di configurazione non trovato: $CONFIG_FILE"
        exit 1
    fi

    log_success "Prerequisiti verificati"
}

# Installa OpenAPI Generator
install_openapi_generator() {
    log_info "Installando OpenAPI Generator..."

    if command -v openapi-generator-cli &> /dev/null; then
        log_info "OpenAPI Generator già installato"
    else
        npm install -g @openapitools/openapi-generator-cli@latest
        log_success "OpenAPI Generator installato"
    fi
}

# Aggiorna configurazione con versione
update_config() {
    log_info "Aggiornando configurazione con versione: $VERSION"

    if command -v jq &> /dev/null; then
        jq --arg version "$VERSION" '.npmVersion = $version' "$CONFIG_FILE" > temp-config.json
        mv temp-config.json "$CONFIG_FILE"
        log_success "Configurazione aggiornata"
    else
        log_warning "jq non trovato, versione non aggiornata nel config"
    fi
}

# Genera client
generate_client() {
    log_info "Generando client TypeScript..."

    # Rimuovi directory esistente
    if [ -d "$CLIENT_DIR" ]; then
        rm -rf "$CLIENT_DIR"
        log_info "Directory $CLIENT_DIR rimossa"
    fi

    # Crea directory
    mkdir -p "$CLIENT_DIR"

    # Genera client
    openapi-generator-cli generate \
        -i "$OPENAPI_SPEC" \
        -g typescript-fetch \
        -o "./$CLIENT_DIR" \
        -c "$CONFIG_FILE"

    log_success "Client generato in $CLIENT_DIR"
}

# Aggiungi README
add_readme() {
    log_info "Aggiungendo README..."

    cat > "$CLIENT_DIR/README.md" << EOF
# Fulcrum Core API Client

Auto-generated TypeScript client for the Fulcrum Core API.

## Installation

\`\`\`bash
npm install @fulcrum/core-client
\`\`\`

## Usage

\`\`\`typescript
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
\`\`\`

## Authentication

This client supports OAuth2 Bearer token authentication as defined in the OpenAPI specification.

## Generated from

Generated from OpenAPI specification version: $(git rev-parse HEAD 2>/dev/null || echo "unknown")
Last updated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
EOF

    log_success "README aggiunto"
}

# Aggiorna package.json
update_package_json() {
    log_info "Aggiornando package.json..."

    cd "$CLIENT_DIR"

    npm pkg set description="Auto-generated TypeScript client for Fulcrum Core API"
    npm pkg set keywords="fulcrum,api,client,typescript,openapi"
    npm pkg set author="Fulcrum Project <https://fulcrumproject.org>"
    npm pkg set license="Apache-2.0"
    npm pkg set repository.type="git"
    npm pkg set repository.url="https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/]*\).*/\1/')"
    npm pkg set repository.directory="$CLIENT_DIR"
    npm pkg set bugs.url="https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/]*\).*/\1/')/issues"
    npm pkg set homepage="https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/]*\).*/\1/')#readme"
    npm pkg set engines.node=">=18.0.0"
    npm pkg set engines.npm=">=8.0.0"

    cd ..
    log_success "package.json aggiornato"
}

# Installa dipendenze e build
setup_client() {
    log_info "Configurando client..."

    cd "$CLIENT_DIR"

    # Installa dipendenze
    log_info "Installando dipendenze..."
    npm install

    # Build
    log_info "Building client..."
    npm run build

    cd ..
    log_success "Client configurato e buildato"
}

# Test client
test_client() {
    log_info "Testando client..."

    cd "$CLIENT_DIR"

    # Test se disponibile
    if npm run test 2>/dev/null; then
        log_success "Test passati"
    else
        log_warning "Nessun test configurato o test falliti"
    fi

    cd ..
}

# Pubblica su npm (se richiesto)
publish_to_npm() {
    if [ "$PUBLISH" = "--publish" ]; then
        log_info "Pubblicando su npm..."

        # Verifica se sei loggato su npm
        if ! npm whoami &> /dev/null; then
            log_error "Non sei loggato su npm. Esegui 'npm login'"
            exit 1
        fi

        cd "$CLIENT_DIR"

        # Pubblica
        npm publish --access public

        cd ..
        log_success "Client pubblicato su npm come @fulcrum/core-client@$VERSION"
    else
        log_info "Pubblicazione saltata (usa --publish per pubblicare)"
    fi
}

# Mostra informazioni
show_info() {
    log_info "=== Riepilogo ==="
    log_info "Versione: $VERSION"
    log_info "Directory client: $CLIENT_DIR"
    log_info "OpenAPI spec: $OPENAPI_SPEC"
    log_info "Pubblicazione: $([ "$PUBLISH" = "--publish" ] && echo "Sì" || echo "No")"

    if [ -d "$CLIENT_DIR" ]; then
        log_info "File generati:"
        find "$CLIENT_DIR" -name "*.ts" | head -10 | while read file; do
            echo "  - $file"
        done

        if [ -f "$CLIENT_DIR/package.json" ]; then
            log_info "Package info:"
            echo "  - Nome: $(cd $CLIENT_DIR && npm pkg get name | tr -d '"')"
            echo "  - Versione: $(cd $CLIENT_DIR && npm pkg get version | tr -d '"')"
        fi
    fi
}

# Main
main() {
    log_info "Avvio generazione client TypeScript..."

    check_prerequisites
    install_openapi_generator
    update_config
    generate_client
    add_readme
    update_package_json
    setup_client
    test_client
    publish_to_npm
    show_info

    log_success "Generazione completata!"

    if [ "$PUBLISH" != "--publish" ]; then
        echo ""
        log_info "Per pubblicare su npm, esegui:"
        echo "  ./scripts/generate-client.sh $VERSION --publish"
    fi
}

# Esegui main
main "$@"