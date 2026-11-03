COMPOSE_FILE ?= ../compose.yaml
DB_SERVICE ?= database
DB_URL ?= postgres://root:rootpass@localhost:5432/twitter_db?sslmode=disable

.PHONY: db-up db-down db-restart db-logs db-ps db-shell db-wait migrateup migratedown sqlc swagger lint test run

db-up:
	docker compose -f $(COMPOSE_FILE) up -d $(DB_SERVICE)

db-down:
	docker compose -f $(COMPOSE_FILE) down

db-restart: db-down db-up db-wait

db-logs:
	docker compose -f $(COMPOSE_FILE) logs -f $(DB_SERVICE)

db-ps:
	docker compose -f $(COMPOSE_FILE) ps

db-shell:
	docker compose -f $(COMPOSE_FILE) exec $(DB_SERVICE) psql -U root -d twitter_db

db-wait:
	@until docker compose -f $(COMPOSE_FILE) exec -T $(DB_SERVICE) pg_isready -U root -d twitter_db >/dev/null 2>&1; do \
		echo "waiting for postgres..."; \
		sleep 1; \
	done

migrateup:
	migrate -path internal/db/migrations -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path internal/db/migrations -database "$(DB_URL)" -verbose down

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

# The API applies embedded migrations on startup; migrateup/migratedown are for manual use.
run: swagger
	go run ./cmd/api
