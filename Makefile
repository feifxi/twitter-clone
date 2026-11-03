# Shortcuts for the whole repo. Each target calls the app's own tooling (twitter-go-api/Makefile,
# npm scripts, infra/scripts), so running those directly works the same.
.DEFAULT_GOAL := help
SCRIPTS := scripts/*.sh compose/*.sh terraform/app/twitter-deploy
.PHONY: help dev db db-reset monitoring stop api web check check-api check-web check-infra gen infra-up deploy infra-down

help: ## List the targets
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  make %-12s %s\n", $$1, $$2}'

# Local development
dev: db ## Start the local services, then the API and the web app together (Ctrl+C stops both)
	@$(MAKE) -s -C twitter-go-api db-wait
	@$(MAKE) -j2 api web

db: ## Start Postgres, Redis and media storage (S3-compatible)
	docker compose up -d

db-reset: ## Delete the local data and start fresh (after editing the schema)
	docker compose --profile observability down -v
	docker compose up -d

monitoring: ## Start the local services and the monitoring stack (Grafana :3001)
	docker compose --profile observability up -d

stop: ## Stop every local container (data is kept)
	docker compose --profile observability stop

api: ## Run the Go API on :8080
	$(MAKE) -C twitter-go-api run

web: ## Run the web app on :3000
	cd twitter-next-web && npm run dev

# Checks (CI runs these and validates the compose, Caddy and monitoring configs)
check: check-api check-web check-infra ## Run every check below

check-api: ## go vet + race tests (integration tests need TEST_DATABASE_URL)
	$(MAKE) -C twitter-go-api lint test

check-web: ## lint, types, tests, build
	cd twitter-next-web && npm run lint && npx tsc --noEmit && npm test && npm run build

check-infra: gen ## terraform fmt/validate/test, shellcheck
	terraform fmt -check -recursive infra/terraform
	for dir in bootstrap app; do \
		terraform -chdir=infra/terraform/$$dir init -backend=false -input=false >/dev/null && \
		terraform -chdir=infra/terraform/$$dir validate || exit 1; \
	done
	terraform -chdir=infra/terraform/app test
	cd infra && if command -v shellcheck >/dev/null; then shellcheck -x $(SCRIPTS); \
		else docker run --rm -v "$$PWD:/mnt" -w /mnt koalaman/shellcheck:v0.11.0 -x $(SCRIPTS); fi

gen: ## Regenerate the SLO rules and the Grafana dashboard
	python3 infra/observability/prometheus/gen_slo_rules.py
	python3 infra/observability/grafana/gen_dashboard.py

# AWS (see infra/README.md)
infra-up: ## Create the AWS stack and deploy this checkout
	bash infra/scripts/up.sh

deploy: ## Deploy this checkout, or an earlier release: make deploy SHA=<commit>
	bash infra/scripts/deploy.sh $(SHA)

infra-down: ## Destroy the AWS stack, data included
	bash infra/scripts/down.sh
