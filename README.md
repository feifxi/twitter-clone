# Twitter Clone

A full-stack Twitter/X clone: a Next.js frontend and a Go API (with an optional Spring Boot API that serves the same contract), PostgreSQL with pgvector, real-time notifications and direct messages, and an AI assistant that answers from recent posts (RAG with Gemini). AWS infrastructure is defined in Terraform, with monitoring through Grafana Cloud.

<p align="center">
  <img src="docs/assets/app-preview.png" alt="App Preview" width="100%" style="border-radius: 8px;">
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
- **Observability** — Real-time metrics (Prometheus) and log aggregation (Loki) via Grafana Cloud

## Tech Stack

| Layer | Technologies |
|---|---|
| Frontend | Next.js (App Router), TypeScript, Tailwind CSS, TanStack Query, Zustand |
| Backend | Go, Gin, PostgreSQL + sqlc + pgvector, Redis (optional); optional Spring Boot API |
| Infrastructure | AWS (EC2 + Docker Compose + nginx, RDS, S3, CloudFront, ECR, SSM, CloudWatch), Let's Encrypt (certbot), Terraform, GitHub Actions |
| Observability | Grafana Cloud (Loki, Prometheus), Grafana Alloy, Node Exporter |

## Architecture

How a request is served:

```mermaid
flowchart TB
  user(["Browser"])
  cf["CloudFront"]
  s3[("S3 media<br/>private, OAC")]

  subgraph ec2["EC2 · public subnet · Docker Compose"]
    nginx["nginx · twitter.&lt;domain&gt;<br/>TLS, routing"]
    certbot["certbot<br/>Let's Encrypt"]
    web["Next.js<br/>server rendering"]
    api["Go API<br/>+ embedding worker"]
    nginx -- "/*" --> web
    nginx -- "/api/*" --> api
    certbot -. "certificate" .-> nginx
    web -- "server render,<br/>internal network" --> api
  end

  subgraph priv["Private subnets · no internet route"]
    rds[("RDS PostgreSQL 16<br/>+ pgvector")]
  end
  ext["Google OAuth · Gemini"]

  user -- "pages, REST, SSE<br/>httpOnly cookies" --> nginx
  user -- "media GET" --> cf --> s3
  user -. "presigned PUT" .-> s3
  api -- SQL --> rds
  api -- HTTPS --> ext
```

How code ships:

```mermaid
flowchart LR
  push(["git push main"]) --> ci["CI<br/>tests, lint, terraform"]
  ci -- "passed" --> gha["Deploy workflow<br/>build web + API images"]
  gha -- "push :sha" --> ecr[("ECR")]
  gha -- "infra/ec2 files" --> s3[("S3 releases")]
  gha -- "SSM Run Command<br/>twitter-deploy &lt;sha&gt;" --> ec2["EC2"]
  ec2 -- "sync stack, pull images" --> s3
  ec2 --> ecr
  ec2 -- "render env files,<br/>record release" --> ssm[("SSM Parameter Store")]
  gha -- "GET /readyz" --> ec2
```

- **One host, one domain:** nginx terminates TLS and routes `/api/*` to the API and everything else to Next.js. The web app and the API share an origin, so there is no CORS and the session cookies are plain first-party cookies. A certbot container issues and renews the Let's Encrypt certificate; nginx reloads itself when it changes.
- **A release is one commit:** both images tagged with its SHA plus its `infra/ec2` files. Changing `compose.yml` or the nginx config is an ordinary deploy, not a new instance.
- **Deploys check themselves:** every container has a health check; if the new release doesn't come up healthy within 3 minutes, the instance rolls back to the previous one and the workflow fails. A replaced instance redeploys the last good release on its own at first boot.
- **Sessions in httpOnly cookies:** the browser sends them on every request and the Next.js server forwards them when it renders a page; JavaScript never sees a token.
- **Async work without queues:** tweet embeddings are jobs in a Postgres table (`embedding_jobs`), enqueued in the same transaction as the tweet and processed by a worker inside the API (`FOR UPDATE SKIP LOCKED`). No SQS, Lambda or NAT.
- **No public database:** RDS lives in private subnets with no internet route; only the app host's security group can reach it. Backups are kept for 7 days and the instance is protected from deletion.
- **Scaling out later:** one instance keeps rate limits and live updates in memory. To run several instances behind a load balancer, add a shared Redis (ElastiCache) and set `REDIS_URL`; both APIs already support it.

## Observability

Everything is code and every alert arrives by email.

| What | How |
|---|---|
| **Metrics** | Grafana Alloy on the instance sends to Grafana Cloud: the API's metrics (by route), host metrics (Node Exporter), the embedding queue, and site-wide request rate, status codes and latency derived from nginx's JSON access logs |
| **Logs** | Every container's logs go to Grafana Cloud Loki, labelled by service; nginx and the API share a request ID per request |
| **Dashboard + alerts** | [`infra/grafana`](infra/grafana) (Terraform): the "Twitter overview" dashboard and alert rules for error rate, slow responses, API down / no metrics, disk, memory and the embedding queue |
| **Uptime** | A Route 53 health check calls `https://twitter.<domain>/readyz` from several regions; a CloudWatch alarm emails you after 2 minutes of failures |
| **Self-healing** | CloudWatch reboots the instance if its OS stops responding and moves it to healthy hardware if AWS's host fails; containers restart on their own; OS patches install weekly (SSM Patch Manager) |
| **Database** | CloudWatch alarms for low storage, high CPU and low memory on RDS |

![Grafana Dashboard](docs/assets/grafana-dashboard.png)

## Prerequisites

| Tool | Needed for |
|---|---|
| [Docker](https://www.docker.com/) | Local Postgres + Redis; the Java tests; building images |
| [Go](https://go.dev/dl/) 1.26+ | The Go API |
| [Node.js](https://nodejs.org/) 22+ | The web app |
| [Make](https://www.gnu.org/software/make/) | Go API tasks (`make run`, `make test`, `make sqlc`) |
| [Google Cloud Console](https://console.cloud.google.com/) | A Google OAuth Client ID for sign-in |
| [Google AI Studio](https://aistudio.google.com/) (optional) | A Gemini API key for the assistant |
| JDK 25 (optional) | Only for the Java API |
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
5.  **Copy Client ID**: Save the **Client ID**. You will need it in `twitter-go-api/app.env`, `twitter-next-web/.env.local` and the Terraform variables.

### 2. Google Gemini Setup (AI & RAG)

This project uses Google Gemini for the AI Assistant and tweet embeddings.

1.  Go to [Google AI Studio](https://aistudio.google.com/).
2.  Click **Get API key** and create a new API key.
3.  Add the key to your `twitter-go-api/app.env` (`GEMINI_API_KEY`) and `infra/terraform/terraform.tfvars` (`gemini_api_key`).
4.  **Verify Model Availability**: You can check which models are currently available for your key by visiting this URL:
    `https://generativelanguage.googleapis.com/v1beta/models?key=YOUR_API_KEY`
5.  (Optional) Customize the models used via `GEMINI_CHAT_MODEL` and `GEMINI_EMBEDDING_MODEL` (defaults to `gemini-2.5-flash` and `gemini-embedding-2-preview`).

### 3. Grafana Cloud Setup (Monitoring)

This project supports centralized logging and metrics via [Grafana Cloud](https://grafana.com/products/cloud/) (Free Tier).

1.  Sign up for a free account at [Grafana Cloud](https://grafana.com/).
2.  In the Cloud Portal, find the **Prometheus** and **Loki** tiles.
3.  Click **Send Logs** for Loki and **Send Metrics** for Prometheus to find your:
    - **URL**
    - **User ID**
4.  Create an **Access Policy** (or API Token) with `metrics:write` and `logs:write` scopes.
5.  Add these values to your `infra/terraform/terraform.tfvars` to enable automatic monitoring.

## Local Development

### 1. Start Postgres + Redis

```bash
docker compose up -d
```

### 2. Run the Go API

```bash
cd twitter-go-api
cp app.env.example app.env   # set TOKEN_SYMMETRIC_KEY (openssl rand -base64 48) and GOOGLE_CLIENT_ID
make run                     # http://localhost:8080; migrations run at startup
```

Swagger UI: http://localhost:8080/swagger/index.html (outside production). The optional Java API runs from the same `app.env` instead (see [`twitter-java-api/README.md`](twitter-java-api/README.md)); run one backend at a time.

### 3. Run the web app

```bash
cd twitter-next-web
cp .example.env .env.local   # set NEXT_PUBLIC_GOOGLE_CLIENT_ID
npm install
npm run dev                  # http://localhost:3000
```

Photo/video uploads also need an S3 bucket (`S3_BUCKET_NAME` + AWS credentials); everything else works without AWS. The assistant needs `GEMINI_API_KEY`.

## Contributing

- Enable the pre-commit hook (gofmt, `terraform fmt`, ESLint on staged files): `git config core.hooksPath .githooks`
- Conventions for the Go API, web app and infra are in [`CLAUDE.md`](CLAUDE.md).

## Validation Commands

The same checks CI runs ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)):

```bash
# Go API: integration tests need a throwaway Postgres (skipped without TEST_DATABASE_URL)
docker run -d --name twitter-test-pg -p 55432:5432 -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=twitter_test pgvector/pgvector:pg16
cd twitter-go-api && go vet ./... && \
  TEST_DATABASE_URL='postgres://test:test@localhost:55432/twitter_test?sslmode=disable' go test -race ./...

# Web
cd twitter-next-web && npm run lint && npx tsc --noEmit && npm test && npm run build

# Java API (starts its own containers; needs Docker)
cd twitter-java-api && ./mvnw verify

# Terraform: all three modules, against mocked providers (no AWS or Grafana account)
cd infra && terraform fmt -check -recursive && \
  for d in bootstrap terraform grafana; do terraform -chdir=$d init -backend=false && terraform -chdir=$d validate; done && \
  terraform -chdir=terraform test && terraform -chdir=grafana test
```

CI also lints the EC2 stack (`shellcheck`, `nginx -t`, `docker compose config`, Alloy config) and checks that generated code (sqlc, test fakes) is up to date.

## AWS Infrastructure

| Service | Purpose |
|---|---|
| **EC2** (t3.small + Elastic IP) | Docker Compose stack: nginx (TLS + routing), certbot, Next.js, Go API, Grafana Alloy, Node Exporter |
| **RDS** | PostgreSQL 16 + pgvector, encrypted, private subnets, 7-day backups, deletion protection |
| **S3 + CloudFront** | Media: presigned uploads to a private bucket, served through CloudFront (OAC) |
| **ECR** | Web and API images, tagged by commit SHA (immutable) |
| **S3 (releases)** | Each commit's `infra/ec2` files, fetched by the instance at deploy time |
| **SSM** | Parameter Store for runtime config, secrets and the current release; Session Manager for shell access and DB tunnels (no SSH port); Patch Manager for weekly OS updates |
| **CloudWatch + SNS** | Uptime check (Route 53 health check), instance auto-recovery and reboot, RDS alarms; alerts by email |
| **IAM OIDC** | GitHub Actions deploys with short-lived credentials (no stored AWS keys) |
| **Route 53** (optional) | DNS for `twitter.<domain>`; any DNS provider works |

Roughly $35–45/month: EC2 t3.small, RDS db.t4g.micro and the public IPv4 address make up nearly all of it.

Terraform lives in [`infra/terraform`](infra/terraform) (one file per concern: `network.tf`, `ec2.tf`, `rds.tf`, `storage.tf`, `deploy.tf`, `alerts.tf`, `ssm.tf`, `ecr.tf`, `github.tf`, `dns.tf`), the Grafana dashboard and alerts in [`infra/grafana`](infra/grafana), and the files that run on the host in [`infra/ec2`](infra/ec2).

## AWS Setup

> You need an [AWS account](https://aws.amazon.com/free/) and a domain you control.

### 1. AWS CLI credentials

Use an admin identity for Terraform, preferably IAM Identity Center:

```bash
aws configure sso   # or: aws configure (IAM user access key)
```

For shell access to the instance, also install the [Session Manager plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) (`brew install --cask session-manager-plugin`).

### 2. State bucket (once per account)

```bash
cd infra/bootstrap
terraform init && terraform apply
terraform output state_bucket
```

### 3. Provision infrastructure

```bash
cd infra/terraform
cp backend.hcl.example backend.hcl            # set bucket = <state_bucket>
cp terraform.tfvars.example terraform.tfvars  # fill in your values
terraform init -backend-config=backend.hcl
terraform plan
terraform apply                               # ~10 min (RDS is the slow part)
```

### 4. DNS

Without `route53_zone_id`, add the record from `terraform output dns_records` at your DNS provider: an `A` record for `twitter.<domain>` → the Elastic IP. certbot keeps retrying every 15 minutes and gets the certificate once the record resolves; until then nginx serves a temporary self-signed one.

Also confirm the two "AWS Notification - Subscription Confirmation" emails (one per region) so alerts can reach you.

### 5. GitHub Actions variables

In **Settings → Secrets and variables → Actions → Variables**, add:

| Variable | Value |
|---|---|
| `AWS_REGION` | e.g. `ap-southeast-1` |
| `AWS_DEPLOY_ROLE_ARN` | `terraform output github_deploy_role_arn` |
| `ECR_API_REPOSITORY` | `terraform output ecr_api_repository` |
| `ECR_WEB_REPOSITORY` | `terraform output ecr_web_repository` |
| `DEPLOY_BUCKET` | `terraform output deploy_bucket` |
| `EC2_INSTANCE_ID` | `terraform output ec2_instance_id` |
| `SITE_DOMAIN` | e.g. `twitter.chanombude.me` |
| `GOOGLE_CLIENT_ID` | your OAuth client ID (compiled into the web app) |

None of these are secrets. Then run the **Deploy** workflow once (Actions → Run workflow) to ship the first release.

### 6. Grafana dashboard and alerts (optional)

Create a free [Grafana Cloud](https://grafana.com/) stack, put its Prometheus/Loki push details in `terraform.tfvars` (the `grafana_cloud_*` variables) and `terraform apply` so Alloy starts shipping. Then, with a service account token (Administration → Service accounts, role Editor):

```bash
cd infra/grafana
cp terraform.tfvars.example terraform.tfvars   # stack URL, token, data source name, alert email
terraform init -backend-config=../terraform/backend.hcl -backend-config="key=grafana.tfstate"
terraform apply
```

### Tear down

RDS has deletion protection. Set `deletion_protection = false` in `rds.tf`, `terraform apply`, then:

```bash
cd infra/terraform && terraform destroy   # keeps a final DB snapshot; stops all other charges
```

Terraform checks run in CI without AWS credentials: `terraform fmt -check`, `validate`, and `terraform test` (plans the whole stack against a mocked AWS provider, see [`tests/`](infra/terraform/tests)).

## Deployment

[`deploy.yml`](.github/workflows/deploy.yml) runs when CI passes on `main` (and is skipped until the `AWS_DEPLOY_ROLE_ARN` variable exists, i.e. before the infrastructure is provisioned):

1. **Build & push** the web and API images to ECR, both tagged with the commit SHA
2. **Upload** that commit's `infra/ec2` files to the releases bucket
3. **Deploy**: SSM Run Command runs [`twitter-deploy <sha>`](infra/ec2/twitter-deploy) on the instance. It fetches the stack files, and the commit's [`deploy.sh`](infra/ec2/deploy.sh) renders the env files from SSM Parameter Store, pulls both images, runs `docker compose up --wait` and records the release. If the health checks fail, it redeploys the previous release and the workflow fails
4. **Verify**: polls `https://twitter.<domain>/readyz` and the home page

**Roll back by hand:** Actions → Deploy → Run workflow with `image_tag` set to an earlier commit SHA (skips the build, reuses that release's files).

**Change a setting:** edit `terraform.tfvars`, `terraform apply` (updates SSM), then re-run the deploy (or `sudo twitter-deploy <sha>` on the host) so the containers pick it up.

**Certificates:** nothing to do. certbot checks twice a day and renews 30 days before expiry; nginx reloads within a minute.

## AWS Access

### Shell on the instance

Through **Session Manager** (no SSH port, no key pairs):

```bash
aws ssm start-session --target $(terraform output -raw ec2_instance_id)
```

Or in the console: **EC2 → Instances → Connect → Session Manager**.

On the host:

```bash
cd /opt/twitter/stack                         # the current release's files
sudo docker compose ps
sudo docker compose logs -f api web nginx
sudo tail -f /var/log/cloud-init-output.log   # first-boot setup progress (~2-3 min)
```

### Connect to RDS

RDS has no public access. Forward a local port through the instance:

```bash
aws ssm start-session \
    --target $(terraform output -raw ec2_instance_id) \
    --document-name AWS-StartPortForwardingSessionToRemoteHost \
    --parameters "host=$(terraform output -raw rds_endpoint | cut -d: -f1),portNumber=5432,localPortNumber=5433"
```

Then connect your DB tool to `127.0.0.1:5433`, database `twitter_db`, with the credentials from `terraform.tfvars` (port `5433` avoids clashing with a local Postgres).

## Project Structure

```
├── twitter-next-web/    # Next.js frontend
├── twitter-go-api/      # Go backend (primary)
├── twitter-java-api/    # Optional Spring Boot backend: same API and schema as the Go one
├── infra/
│   ├── bootstrap/       # One-time: S3 bucket for Terraform state
│   ├── terraform/       # AWS infrastructure (+ tests/ with a mocked-provider plan)
│   ├── grafana/         # Grafana Cloud dashboard and alert rules (Terraform)
│   └── ec2/             # What runs on the host: compose.yml, nginx config, certbot, Alloy config, deploy scripts
├── docs/assets/         # Screenshots used in this README
├── .github/workflows/   # CI (tests, lint, Terraform checks) and the deploy pipeline
├── .githooks/           # pre-commit: gofmt, terraform fmt, ESLint on staged files
└── docker-compose.yml   # Local development (Postgres + pgvector, Redis)
```
