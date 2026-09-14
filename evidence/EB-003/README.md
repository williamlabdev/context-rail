# EB-003 — VS-002 Project Registry／Context UI

- Checked at: 2026-09-14 Asia/Taipei
- Current source commit: `0ecc1da`
- Scope: local read-only implementation and verification for VS-002
- Cloud Run, IAM and production: not used

## Start here

先讀 [HUMAN_REVIEW_PACKET.zh-TW.md](HUMAN_REVIEW_PACKET.zh-TW.md)。它是給人看的結論與導覽；本目錄的 command output 是可追溯的原始證據。

## Prerequisite result

`ACCEPTED_FOR_LOCAL_DEVELOPMENT`: VS-001 has delivered the reusable Go contract/package and its evidence bundle, and the human review accepted this local scope. This is a prerequisite result, not the final VS-002 review result.

Evidence for this result:

- `decisions/DR-002-project-registry-import.json` is `ACCEPTED_FOR_DEVELOPMENT`.
- `work-orders/AWO-002-project-registry-import.json` is `ISSUED` with a valid canonical hash.
- `specs/002-project-registry-import/tasks.md` records T001–T021 complete for the local slice.
- The reusable Go package, local CLI, contract tests and `evidence/EB-002/` are present and verified.
- `evidence/EB-002/HUMAN_REVIEW_PACKET.zh-TW.md` records the human review result and its local-only boundary.
- The existing Python importer remains the contract oracle/reference; it is not the VS-001 Go runtime package.

## Explicit fixture roots

- `demo/order-operations-portal`
- `examples/support-insights`

## Local tool versions observed

```text
go version go1.26.6 darwin/arm64
v25.8.1
11.11.0
Python 3.9.6
```

The first four lines correspond to Go, Node.js, npm and the project-local Python interpreter used for this check (`template/.venv/bin/python`).

## Current implementation checkpoint

The VS-002 local Work Order has been issued and the current checkpoint includes the read-only Go HTTP boundary, React workspace, focused tests and a positive browser journey. The remaining work is explicit failure-state coverage and final evidence capture.

`AWO-003` is limited to the VS-002 paths. VS-001's local Work Order does not authorize VS-002 paths.

## Evidence index

- `HUMAN_REVIEW_PACKET.zh-TW.md` — human-readable review guide and current boundary
- `go-test-output.txt` — planned final Go test output
- `go-quality-output.txt` — planned final `go vet` and build output
- `frontend-quality-output.txt` — planned final frontend checks
- `browser-output.txt` — planned browser journey output
- `README.md` — scope, provenance and evidence index
