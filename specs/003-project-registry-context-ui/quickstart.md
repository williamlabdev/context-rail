# Quickstart: Project Registry／Documents & AI Context 唯讀工作區

## Prerequisites

- Go 1.22 or newer
- Node.js and npm available locally
- The VS-001 registry contract and the checked-in fixtures:
  - `demo/order-operations-portal`
  - `examples/support-insights`
- `DR-003` accepted and `AWO-003` issued before implementation work begins

## Expected local flow

Terminal A 啟動 local service：

```bash
go run ./cmd/context-rail \
  --fixture-root demo/order-operations-portal \
  --fixture-root examples/support-insights
```

Terminal B 執行驗證：

```bash
go test ./...
go vet ./...
npm --prefix frontend ci
npm --prefix frontend run typecheck
npm --prefix frontend run build
npm --prefix frontend run test
npm --prefix frontend run test:e2e
go build ./...
```

The browser test must start the local Go service with both fixture roots explicitly configured, then open the served workspace. During implementation, the exact host/port flag may be added, but the fixture roots and read-only behavior must remain the same.

## Positive journey

1. Open the local Project Registry.
2. Confirm both fixture-backed Projects are listed.
3. Select `order-operations-portal` and verify identity, repositories, services, environments, documents, decisions, context and readiness.
4. Select `support-insights` and verify that its own records replace the previous detail state.
5. Confirm the page shows source/derived distinction and provenance fields where available.

## Uncertainty journey

1. Open the Project containing stale Context Pack or source drift.
2. Confirm `STALE` remains visible and is not shown as `CURRENT` or `PASS`.
3. Open a record with undeclared runtime or unavailable observation.
4. Confirm `UNDECLARED` or `UNKNOWN` remains visible with a readiness reason.

## Empty and error journeys

- Start with no configured fixture roots and verify an explicit empty state with no create action.
- Request a missing or malformed Project root and verify an explicit error state with no source mutation.
- Delay the read response and verify loading state does not present old data as current.

## Evidence required

Record exact commands, source commit, changed paths, browser test result and before/after hashes in `evidence/EB-003/`. Do not record Cloud Run, IAM, staging or production evidence for this local slice.
