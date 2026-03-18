# fleet-reservation

Reservation microservice — creates, tracks, and releases compute instances via gRPC and Kafka.

## Run

**1. Generate protobuf** (re-run when `protobuf/api.proto` changes):

```bash
./scripts/proto-generate.sh
```

Builds a container with all proto tools and outputs Go files into `gen/`.

**2. Start all services:**

```bash
docker compose up
```

---

## Code layout

```
cmd/              entrypoint
config/           env config loading
gen/              generated protobuf (git-ignored, auto-generated on startup)
protobuf/         .proto source + buf config
scripts/          entrypoint.sh (proto gen + app start)
internal/
  adapters/
    api/          gRPC server
    db/           PostgreSQL (GORM)
    message/      Kafka consumer & producer
    timer/        cleanup timer
  core/
    application/  use-case logic
    domain/       types and domain structs
    ports/        interfaces
```
