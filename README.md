# Twitter Clone

A full-stack Twitter/X clone: a Next.js frontend and a Go API, PostgreSQL with pgvector, real-time notifications and direct messages, and an AI assistant that answers from recent posts (RAG with Gemini). AWS infrastructure is defined in Terraform, with a self-hosted monitoring stack (Prometheus, Loki, Grafana) on the same host.

<p align="center">
  <img src=".github/assets/app-preview.png" alt="App Preview" width="100%" style="border-radius: 8px;">
</p>

## Features

- **Authentication** — Google sign-in; the session lives in httpOnly cookies (short-lived JWT + rotating refresh token), renewed before server rendering
- **Tweets** — Create, delete, like, retweet, and threaded replies
- **Feeds** — "For You" global feed with **time-decay gravity algorithm** (weighing likes, retweets, and replies), following-only feed, and per-user profile feed
- **Follow / Unfollow** — Follow users with follower & following lists
- **Media Uploads** — S3 presigned-URL uploads served via CloudFront CDN
- **Real-time Updates** — one Server-Sent Events (SSE) stream for notifications and direct messages (fanned out across instances via Redis when several run)
- **Direct Messages** — one-to-one conversations, delivered in real time over the same SSE stream
- **Search** — Search users, tweets, and hashtags
- **Trending & Discovery** — Trending hashtags and suggested users
- **AI Assistant (RAG)** — streaming chat with Gemini that answers using related tweets: new tweets are queued in Postgres (same transaction as the tweet), embedded in the background, and retrieved with pgvector
- **Observability** — Metrics (Prometheus), logs (Loki), a Grafana dashboard and SLO-based alerts, all self-hosted and defined as code

## Tech Stack

| Layer | Technologies |
|---|---|
| Frontend | Next.js (App Router), TypeScript, Tailwind CSS, TanStack Query, Zustand |
| Backend | Go, Gin, PostgreSQL + sqlc + pgvector, Redis (optional) |
| Infrastructure | AWS (EC2 Graviton + Docker Compose, S3, CloudFront, ECR, SSM, Route 53, CloudWatch), Caddy (automatic HTTPS), Terraform, GitHub Actions |
| Observability | Prometheus, Alertmanager, Loki, Grafana Alloy, Grafana, Node Exporter |

## Architecture

How a request is served:

```mermaid
flowchart TB
  user(["Browser"])
  cf["CloudFront"]
  s3[("S3 media<br/>private, OAC")]

  subgraph ec2["EC2 t4g · Docker Compose"]
    caddy["Caddy · twitter.&lt;domain&gt;<br/>HTTPS, routing"]
    web["Next.js<br/>server rendering"]
    api["Go API<br/>+ embedding worker"]
    pg[("PostgreSQL 16<br/>+ pgvector")]
    mon["Prometheus · Loki · Grafana<br/>Alertmanager · Alloy"]
    caddy -- "/*" --> web
    caddy -- "/api/*" --> api
    caddy -- "grafana.&lt;domain&gt;" --> mon
    web -- "server render,<br/>internal network" --> api
    api -- SQL --> pg
    mon -. "scrape, collect logs" .-> api
  end
  ext["Google OAuth · Gemini"]

  user -- "pages, REST, SSE<br/>httpOnly cookies" --> caddy
  user -- "media GET" --> cf --> s3
  user -. "presigned PUT" .-> s3
  api -- HTTPS --> ext
```

How code ships:

```mermaid
flowchart LR
  push(["git push main"]) --> ci["CI<br/>tests, lint, terraform, configs"]
  ci -- "passed" --> deploy["infra/scripts/deploy.sh<br/>(CI or your machine)"]
  deploy -- "push :sha" --> ecr[("ECR")]
  deploy -- "stack files" --> s3[("S3 releases")]
  deploy -- "SSM Run Command<br/>twitter-deploy &lt;sha&gt;" --> ec2["EC2"]
  ec2 -- "sync stack, pull images" --> s3
  ec2 --> ecr
  ec2 -- "render env files" --> ssm[("SSM Parameter Store")]
  deploy -- "GET /readyz" --> ec2
```

- **Up when you need it:** `infra/scripts/up.sh` creates the whole stack and deploys; `down.sh` removes it. About $1.3 a day while up, nearly nothing while down. See [`infra/README.md`](infra/README.md).
- **One host, one domain:** Caddy terminates TLS (certificates are automatic) and routes `/api/*` to the API and everything else to Next.js. The web app and the API share an origin, so there is no CORS and the session cookies are plain first-party cookies.
- **A release is one commit:** both images tagged with its SHA plus its `infra/compose` and `infra/observability` files. Changing the compose file, Caddyfile or an alert rule is an ordinary deploy.
- **Deploys check themselves:** every container has a health check; if the new release doesn't come up healthy within 3 minutes, the host rolls back to the previous one and the deploy fails.
- **Sessions in httpOnly cookies:** the browser sends them on every request and the Next.js server forwards them when it renders a page; JavaScript never sees a token.
- **Async work without queues:** tweet embeddings are jobs in a Postgres table (`embedding_jobs`), enqueued in the same transaction as the tweet and processed by a worker inside the API (`FOR UPDATE SKIP LOCKED`). No SQS, Lambda or NAT.
- **Disposable data:** Postgres runs next to the app and its data goes away with `down`. That keeps the stack cheap and quick to create; a managed database (RDS) is the step up if the data ever has to last.
- **Scaling out later:** one instance keeps rate limits and live updates in memory. To run several instances behind a load balancer, add a shared Redis and set `REDIS_URL`; the API already supports it.

## Observability

Self-hosted next to the app, with the same configuration locally (`docker compose --profile observability up -d`) and in production. Details, alerts and what to check for each: [`infra/observability/README.md`](infra/observability/README.md).

| What | How |
|---|---|
| **Metrics** | Prometheus scrapes the API (requests by route, latency, the embedding queue, Go runtime), Caddy (every request, pages included) and the host (Node Exporter) |
| **Logs** | Alloy ships every container's logs to Loki, labelled by service and level; Caddy and the API share a request ID per request |
| **Dashboard** | "Twitter overview" in Grafana (`https://grafana.<domain>`): SLOs and error budget, site and API traffic, slowest routes, embedding queue, host, recent errors |
| **Alerts** | SLO burn-rate alerts for API availability and latency, plus API down, site 5xx, disk, memory and the embedding queue; Alertmanager posts them to Discord |
| **AWS safety nets** | CloudWatch recovers or reboots the instance if it fails and emails on high CPU; an AWS Budget emails before the bill surprises you |

## Prerequisites

| Tool | Needed for |
|---|---|
| [Docker](https://www.docker.com/) | Local Postgres, Redis and media storage; building images |
| [Go](https://go.dev/dl/) 1.26+ | The Go API |
| [Node.js](https://nodejs.org/) 22+ | The web app |
| [Make](https://www.gnu.org/software/make/) | Shortcuts at the repo root (`make` lists them) and Go API tasks |
| [Google Cloud Console](https://console.cloud.google.com/) | A Google OAuth Client ID for sign-in |
| [Google AI Studio](https://aistudio.google.com/) (optional) | A Gemini API key for the assistant |
| [Terraform](https://developer.hashicorp.com/terraform/install) 1.10+ (optional) | Checking or provisioning the infrastructure |
| [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) (optional) | Only for a real AWS deployment |

**Install via Homebrew (macOS):**

```bash
brew install go node make hashicorp/tap/terraform
brew install --cask docker
```

## Third-Party Setup

### 1. Google OAuth 2.0 Setup

This project uses Google OAuth for authentication. You need to create a Client ID to allow users to sign in.

1.  Go to the [Google Cloud Console](https://console.cloud.google.com/).
2.  **Create a Project**: Click on the project dropdown and select "New Project".
3.  **OAuth Consent Screen**:
    - Go to "APIs & Services" > "OAuth consent screen".
    - Select **External** and click "Create".
    - Fill in the required App Information (App name, support email, developer email).
    - Save and continue until the end.
4.  **Create Credentials**:
    - Go to "APIs & Services" > "Credentials".
    - Click **+ Create Credentials** > **OAuth client ID**.
    - Select **Web application** as the Application type.
    - **Authorized JavaScript origins**:
      - `http://localhost:3000` (Local)
      - `https://twitter.<domain>` (Production)
    - No redirect URI is needed: the app signs in with Google ID tokens.
5.  **Copy Client ID**: Save the **Client ID**. You will need it in `twitter-go-api/app.env`, `twitter-next-web/.env.local` and `infra/terraform/app/terraform.tfvars`.

### 2. Google Gemini Setup (AI & RAG)

This project uses Google Gemini for the AI Assistant and tweet embeddings.

1.  Go to [Google AI Studio](https://aistudio.google.com/).
2.  Click **Get API key** and create a new API key.
3.  Add the key to your `twitter-go-api/app.env` (`GEMINI_API_KEY`) and `infra/terraform/app/terraform.tfvars` (`gemini_api_key`).
4.  **Verify Model Availability**: You can check which models are currently available for your key by visiting this URL:
    `https://generativelanguage.googleapis.com/v1beta/models?key=YOUR_API_KEY`
5.  (Optional) Customize the models used via `GEMINI_CHAT_MODEL` and `GEMINI_EMBEDDING_MODEL` (defaults to `gemini-2.5-flash` and `gemini-embedding-2-preview`).

## Local Development

`make dev` runs steps 1–3 in one terminal (once `app.env` and `.env.local` exist); the root `Makefile` also has a shortcut for each step (`make` lists them).

### 1. Start Postgres, Redis and media storage

```bash
docker compose up -d
```

### 2. Run the Go API

```bash
cd twitter-go-api
cp app.env.example app.env   # set TOKEN_SYMMETRIC_KEY (openssl rand -base64 48) and GOOGLE_CLIENT_ID
make run                     # http://localhost:8080; migrations run at startup
```

Swagger UI: http://localhost:8080/swagger/index.html (outside production).

### 3. Run the web app

```bash
cd twitter-next-web
cp .example.env .env.local   # set NEXT_PUBLIC_GOOGLE_CLIENT_ID
npm install
npm run dev                  # http://localhost:3000
```

### 4. Monitoring (optional)

```bash
docker compose --profile observability up -d   # Grafana http://localhost:3001, Prometheus :9090
```

Photo/video uploads go to a local S3-compatible server (RustFS, started with the database): the same presigned-upload flow as production, no AWS account needed. Browse uploaded files at http://localhost:9001/rustfs/console/ (`devaccess` / `devsecret123`). The assistant needs `GEMINI_API_KEY`.

## Contributing

- Enable the pre-commit hook (gofmt, `terraform fmt`, ESLint on staged files): `git config core.hooksPath .githooks`
- Conventions for the Go API, web app and infra are in [`CLAUDE.md`](CLAUDE.md).

## Validation Commands

The same checks CI runs ([`.github/workflows/ci-cd.yml`](.github/workflows/ci-cd.yml)); `make check` runs them from the root:

```bash
# Go API: integration tests need a throwaway Postgres (skipped without TEST_DATABASE_URL)
docker run -d --name twitter-test-pg -p 55432:5432 -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=twitter_test pgvector/pgvector:pg16
cd twitter-go-api && go vet ./... && \
  TEST_DATABASE_URL='postgres://test:test@localhost:55432/twitter_test?sslmode=disable' go test -race ./...

# Web
cd twitter-next-web && npm run lint && npx tsc --noEmit && npm test && npm run build

# Terraform, against a mocked AWS provider (no AWS account)
cd infra && terraform fmt -check -recursive && \
  for d in terraform/bootstrap terraform/app; do terraform -chdir=$d init -backend=false && terraform -chdir=$d validate; done && \
  terraform -chdir=terraform/app test
```

CI also checks the shell scripts (`shellcheck`), the compose file, the Caddy, Prometheus, Alertmanager and Alloy configs, and that generated files (sqlc, test fakes, SLO rules, the dashboard) are up to date.

## Deploying to AWS

Everything is in [`infra/README.md`](infra/README.md): a one-time setup (bootstrap and DNS), then

```bash
make infra-up     # infra/scripts/up.sh: create the stack and deploy (~5 min)
make deploy       # infra/scripts/deploy.sh: ship code changes (CI does this on every push to main)
make infra-down   # infra/scripts/down.sh: remove the stack
```

## Project Structure

```
├── twitter-next-web/    # Next.js frontend
├── twitter-go-api/      # Go backend
├── infra/               # AWS and monitoring as code (see infra/README.md)
│   ├── terraform/       #   bootstrap/ (once: DNS name servers, GitHub deploy role), app/ (the stack)
│   ├── compose/         #   what runs on the host: compose file, Caddyfile, rollout script
│   ├── observability/   #   Prometheus rules, Alertmanager, Loki, Alloy, Grafana dashboard
│   └── scripts/         #   up.sh, deploy.sh, down.sh
├── .github/             # CI/CD workflow (tests, lint, infra checks, then deploy); README screenshot
├── .githooks/           # pre-commit: gofmt, terraform fmt, ESLint on staged files
├── compose.yaml         # Local development (Postgres + pgvector, Redis, media storage, optional monitoring)
└── Makefile             # Shortcuts for the whole repo (`make` lists them)
```
