# Quickstart: Manual Order Review

Run these commands from `demo/order-operations-portal/`.

## Prerequisites

- Go 1.22 or newer.
- POSIX shell for the helper scripts.
- No cloud credentials are needed for local validation.

## Automated checks

```sh
go test ./...
go vet ./...
go build ./...
```

Expected result: tests, vet and build complete successfully.

## Local smoke path

Start the service in one terminal:

```sh
go run .
```

In another terminal:

```sh
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/api/orders
curl -fsS -X POST http://localhost:8080/api/orders/ORD-1001/review \
  -H 'Content-Type: application/json' \
  -d '{"decision":"APPROVED","note":"Address confirmed"}'
```

Expected result: health is `ok`, the seeded orders are listed, and `ORD-1001` is returned with status `APPROVED`.

Negative checks:

```sh
curl -sS -o /dev/null -w '%{http_code}\n' -X POST http://localhost:8080/api/orders/ORD-1001/review \
  -H 'Content-Type: application/json' \
  -d '{"decision":"APPROVED","note":"   "}'
curl -sS -o /dev/null -w '%{http_code}\n' -X POST http://localhost:8080/api/orders/ORD-9999/review \
  -H 'Content-Type: application/json' \
  -d '{"decision":"REJECTED","note":"Not eligible"}'
```

Expected status codes: `400` for the empty note and `404` for the unknown order.

## Governance check

Local success does not authorize staging. Before deployment, reconcile the test and build outputs to a committed source snapshot, obtain the required review, and satisfy the staging environment gate described in `docs/operations/environments.md`.
