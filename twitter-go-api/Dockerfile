# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Download modules first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /out/api ./cmd/api

# Runtime stage: distroless includes CA certs, tzdata and a non-root user, and no shell.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/api ./api
COPY --from=builder /app/db/migration ./db/migration

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["./api"]
