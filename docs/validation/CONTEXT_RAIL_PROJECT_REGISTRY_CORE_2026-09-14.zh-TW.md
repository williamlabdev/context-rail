# ContextRail Project Registry Core

- Date: 2026-09-14 Asia/Taipei
- Implementation commit: `38c965f09e0b3001ec8b1a699a7e42d854c406a3`
- Decision: `DR-002` / `ACCEPTED_FOR_DEVELOPMENT`
- Work Order: `AWO-002` / `ISSUED`
- Scope: local read-only Go Project Registry package and CLI
- Cloud Run, IAM and production: not used

## Comparison

The Go CLI and the existing Python oracle were run against these explicit roots:

- `demo/order-operations-portal`
- `examples/support-insights`

The normalized snapshots are equal after removing `observed_at`, which is intentionally generated independently at runtime. The comparison preserves the source-derived differences: the Order Operations context is `DERIVED`, the Support Insights context is `STALE`, and the Support Insights runtime is `UNDECLARED`.

## Verification

- `go test -count=1 ./...`: PASS
- `go vet ./...`: PASS
- `go build ./...`: PASS
- Source tree hash before/after import: PASS; covered by `tests/registry/mutation_test.go`
- Python oracle comparison: PASS; see [EB-002 oracle comparison](../../evidence/EB-002/oracle-comparison.txt)

## Limitations

This is a local contract/import boundary. It does not provide HTTP, persistence, authentication, provider discovery, Context Pack rebuild, Cloud Run deployment or production promotion. The YAML parser dependency is pinned in `go.mod`/`go.sum`; future manifest shapes require new contract tests before support is claimed.
