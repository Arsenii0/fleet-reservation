# fleet-reservation

Reservation microservice — creates, tracks, and releases compute instances via gRPC and Kafka.

## Run

**1. Authenticate with AWS SSO** (required before starting — the deployer mounts `~/.aws` into the container):

```bash
aws sso login --profile <your-profile>
```

Set the profile name in your environment so docker-compose picks it up:

**2. Generate protobuf** (re-run when `protobuf/api.proto` changes):

```bash
./scripts/proto-generate.sh
```

**3. Start all services:**

```bash
docker compose up
```

**4. Open the UI:** [http://localhost:8080](http://localhost:8080)

> **Note:** After a successful reservation, wait ~5 minutes before connecting via VNC — the Ubuntu desktop and VNC server are installed via EC2 user data on first boot.
>
> **TODO ArsenP:** Build an AMI with VNC pre-installed to eliminate the wait.
>
> **TODO ArsenP:** `vnc_allowed_cidr` is hardcoded to a single machine IP in `deployer/terraform/openclaw-guardian/variables.tf` — make it configurable via env/tfvars.

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
