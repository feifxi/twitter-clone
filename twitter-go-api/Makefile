COMPOSE_FILE ?= ../compose.yaml
DB_SERVICE ?= database
SQLC_VERSION := 1.30.0

.PHONY: db-wait sqlc swagger lint test run check check-generated vuln

# Used by the root `make dev` before starting the API.
db-wait:
	@until docker compose -f $(COMPOSE_FILE) exec -T $(DB_SERVICE) pg_isready -U root -d twitter_db >/dev/null 2>&1; do \
		echo "waiting for postgres..."; \
		sleep 1; \
	done

# Regenerates query code and the dbtest.MockStore built from it.
sqlc:
	sqlc generate
	go generate ./internal/db/...

# The Swagger spec (internal/swagger, not committed) is generated from the handler annotations;
# run, lint and test regenerate it first. The command is the go:generate line in internal/api/routes.go.
swagger:
	go generate ./internal/api

lint: swagger
	go vet ./...

# What CI runs. Integration tests need TEST_DATABASE_URL (skipped without it).
check: check-generated lint vuln test

# Committed generated code must match its sources: sqlc's query code and dbtest.MockStore.
check-generated:
	@if command -v sqlc >/dev/null; then sqlc diff; \
	else docker run --rm -v "$$PWD:/src" -w /src sqlc/sqlc:$(SQLC_VERSION) diff; fi
	@before=$$(shasum internal/db/dbtest/mock_store.go); go generate ./internal/db/...; \
	if [ "$$before" != "$$(shasum internal/db/dbtest/mock_store.go)" ]; then \
		echo "internal/db/dbtest/mock_store.go was out of date: regenerated it, commit the change"; exit 1; fi

# Known vulnerabilities in code the API actually calls.
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...

test: swagger
	go test -race -cover ./...

# Migrations are embedded and applied on startup.
run: swagger
	go run ./cmd/api
