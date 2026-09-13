# Development Contract

## Prerequisites

- Go 1.22 or newer.
- A POSIX shell for the helper scripts.
- No cloud credentials are needed for local development.

## Commands

```sh
go test ./...
go run .
go build -o bin/order-operations-portal .
```

The service is a synthetic, stateless fixture. Use `/healthz` and `/api/orders` for local smoke checks.

## Verification

The candidate must provide test output, build output, independent review evidence and a staging smoke result tied to the same commit/image identity.
