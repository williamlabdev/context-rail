# EB-003 — VS-002 Project Registry／Context UI

- Checked at: 2026-09-14 Asia/Taipei
- Current source commit: `25204a5`
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

## Implementation result

`IMPLEMENTATION_COMPLETE_PENDING_HUMAN_REVIEW`: the bounded local slice includes the read-only Go HTTP boundary, React workspace, uncertain-state preservation, explicit loading/empty/error journeys, focused tests and browser evidence. This result is not a staging or production release decision.

`AWO-003` is limited to the VS-002 paths. VS-001's local Work Order does not authorize VS-002 paths.

## Evidence index

- `HUMAN_REVIEW_PACKET.zh-TW.md` — human-readable review guide and current boundary
- `go-test-output.txt` — exact Go test output
- `go-quality-output.txt` — exact `go vet`, build and module verification output
- `frontend-quality-output.txt` — exact frontend audit, typecheck, unit test and build output
- `browser-output.txt` — exact Playwright browser journey output
- `oracle-comparison.txt` — Go/Python normalized snapshot comparison
- `mutation-check.txt` — consumer fixture hash comparison before/after read-only import
- `README.md` — scope, provenance and evidence index

## Verification result

| Layer | Result | Evidence |
| --- | --- | --- |
| Go contract/API | PASS | [go-test-output.txt](go-test-output.txt) |
| Go quality/module integrity | PASS | [go-quality-output.txt](go-quality-output.txt) |
| Frontend quality | PASS | [frontend-quality-output.txt](frontend-quality-output.txt) |
| Browser journeys | PASS: 4 tests | [browser-output.txt](browser-output.txt) |
| Go/Python oracle comparison | PASS with `observed_at` excluded | [oracle-comparison.txt](oracle-comparison.txt) |
| Consumer fixture mutation | PASS: hashes unchanged | [mutation-check.txt](mutation-check.txt) |

The final human review may accept or reject this local implementation. No evidence in EB-003 authorizes Cloud Run, IAM, staging or production.
