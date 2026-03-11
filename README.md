# fleet-reservation

Reservation microservice for the Fleet VM Manager. Handles lifecycle of resource reservations — creating, tracking, and releasing compute instances via gRPC and Kafka.

## Quick start

```bash
# 1. Generate protobuf (requires buf)
buf generate

# 2. Tidy dependencies
go mod tidy

# 3. Run all services
docker compose up --build
```

## Code layout

```
cmd/          — entrypoint
config/       — configuration loading
internal/
  adapters/
    api/      — gRPC server
    db/       — PostgreSQL adapter (GORM)
    message/  — Kafka consumer & producer
    timer/    — cleanup timer
  core/
    application/ — use-case logic
    domain/      — types, enums, message structs
    ports/       — interface definitions
```
