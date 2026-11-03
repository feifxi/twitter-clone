COMPOSE_FILE ?= ../compose.yaml
DB_SERVICE ?= database

.PHONY: db-wait sqlc swagger lint test run

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

test: swagger
	go test -race -cover ./...

# Migrations are embedded and applied on startup.
run: swagger
	go run ./cmd/api
