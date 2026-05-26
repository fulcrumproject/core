.PHONY: e2e e2e-up e2e-down

# Bring up Postgres + Keycloak + realm provisioning (no api).
# Idempotent — leaves containers running between e2e runs for fast iteration.
e2e-up:
	docker compose up -d postgres keycloak keycloak-provisioning --wait

# Stop and remove the e2e service containers but PRESERVE volumes
# (dev `fulcrum_db` lives in the postgres volume — never wipe it from here).
e2e-down:
	docker compose stop postgres keycloak keycloak-provisioning

# Full e2e run: deps up -> tests. Leaves deps running on purpose; run
# `make e2e-down` (or just `docker compose stop`) when you're done.
e2e: e2e-up
	go test -tags e2e -timeout 5m -count=1 ./test/e2e/...

dev: ## Start development
	docker compose up postgres keycloak keycloak-provisioning --wait
	trap 'kill %1 2>/dev/null; docker compose down' EXIT; \
	air
