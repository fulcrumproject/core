# Makefile for Fulcrum Core project
# Provides targets for building, testing, and code generation

.PHONY: help
help:
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the application
	go build -o fulcrum ./cmd/fulcrum

.PHONY: run
run: ## Run the application
	go run ./cmd/fulcrum

.PHONY: dev
dev: ## Run the application with Air (live reload)
	air

.PHONY: test
test: ## Run tests
	go test -v ./...

.PHONY: test-db
test-db: ## Run database tests
	go test -v ./pkg/database/...

.PHONY: generate
generate: gen-query gen-mocks ## Generate all code

.PHONY: gen-query
gen-query: ## Generate GORM Gen queries
	go run ./cmd/gormgen

.PHONY: gen-mocks
gen-mocks: ## Generate mocks
	mockery

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: tidy
tidy: ## Tidy dependencies
	go mod tidy

.PHONY: clean
clean: ## Clean generated files
	rm -f fulcrum coverage.out

.DEFAULT_GOAL := help

