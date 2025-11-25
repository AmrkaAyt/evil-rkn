# evil-rkn

`evil-rkn` is a Go service for fetching, storing, and serving a registry of blocked resources (RKN-like), exposing both gRPC and HTTP (via gRPC-Gateway) APIs.

The service maintains an in-memory copy of the registry, periodically refreshes it from an external source, and provides simple health and readiness endpoints suitable for containerized deployments and Kubernetes probes.

---

## Features

- Periodic registry updates with configurable interval.
- Exponential backoff with jitter on update failures.
- Thread-safe in-memory holder for the current registry.
- gRPC API with an HTTP/JSON gateway.
- Basic liveness and readiness endpoints:
  - `/healthz` – liveness check.
  - `/readyz` – readiness check.

---

## Project structure

This is an approximate structure; adjust if your layout differs:

- `internal/registry` – registry updater, backoff logic, and in-memory holder.
- `internal/http` – HTTP gateway server and health endpoints.
- `internal/domain` – domain models (registry representation, etc.).
- `proto/` – protobuf definitions and generated code.
- `cmd/` – entrypoints (main packages) for running the service.

---

## Requirements

- Go 1.24.7
- Make, Docker, and docker-compose are optional but useful for local development.

---

## Building

From the repository root:

```bash
go mod tidy
go build ./...
```

If you have a dedicated main package (for example in `cmd/server`), you can build a binary like this:

```bash
go build -o bin/evil-rkn ./cmd/server
```

Adjust the path to the main package to match your project.

---

## Running

If you use a single binary:

```bash
./bin/evil-rkn
```

or, if the main package is at the module root:

```bash
go run ./...
```

The HTTP gateway listens on the configured HTTP address and proxies requests to the gRPC endpoint. Typical configuration would expose:

- HTTP (gRPC-Gateway + health endpoints) on `:8080`
- gRPC on `:9090`

Check your actual flags/environment variables for precise ports.

---

## Configuration

The updater is configured via a `Config` struct:

```go
type Config struct {
    Interval       time.Duration // base update interval
    InitialBackoff time.Duration // initial backoff delay
    MaxBackoff     time.Duration // maximum backoff delay
}
```

Typical values might look like:

```go
Config{
    Interval:       6 * time.Hour,
    InitialBackoff: 30 * time.Second,
    MaxBackoff:     30 * time.Minute,
}
```

The updater:

- Performs an initial update on startup.
- On failures, increases the delay using exponential backoff (with jitter) up to `MaxBackoff`.
- Resets the failure counter after a successful update.

The HTTP gateway:

- Registers the gRPC-Gateway handlers against the gRPC endpoint.
- Exposes `/healthz` and `/readyz`:
  - `/healthz` – returns `200 OK` with `"ok"` if the process is running.
  - `/readyz` – returns `200 OK` with `"ready"`; in production this can be extended to perform a real gRPC health check.

---

## HTTP endpoints

The exact HTTP paths depend on your protobuf definitions and gRPC-Gateway configuration, but in general:

- `GET /healthz` – liveness check.
- `GET /readyz` – readiness check.
- `/*` – proxied to gRPC via gRPC-Gateway (for example, `/v1/...`).

Refer to your generated gRPC-Gateway code (`pb.Register...HandlerFromEndpoint`) and `.proto` files for concrete REST paths.

---

## Development

Run tests:

```bash
go test ./...
```

Typical development workflow:

1. Modify protobuf definitions in `proto/`.
2. Regenerate gRPC and gRPC-Gateway code.
3. Implement or update handlers in the gRPC server.
4. Adjust the HTTP gateway and registry updater if needed.
5. Run tests and linting.
6. Build and run the service locally or in Docker.

---

## Notes

This README is intentionally minimal and focused on practical usage and structure. Extend it with concrete examples of requests/responses once the public API surface is finalized.
