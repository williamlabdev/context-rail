# EB-010 — UI-20 Documents / AI Context: document baseline, missing-document readiness, Context Pack rebuild (PARTIAL, never synthesized)

- Checked at: 2026-09-21 Asia/Taipei
- Base source commit: `297a641` (feat/vs-007-prod-demo)
- Decision: `decisions/DR-010-context-pack-rebuild.json` — `ACCEPTED_FOR_DEVELOPMENT`
- Scope: local development/testing; consumer fixtures read-only (stat + hash), governance state only
- Review: AI-assisted implementation with automated evidence; human review of DR-010 pending

## What the slice shows

1. **Baseline** — the manifest's document declarations (with `requiredFor`) become baseline v1; every document is verified against the repository now (path / version / content hash / status). The consumer's own `docs/ai/context-pack.json` stays visible as *declared context* (support-insights: STALE, stale source README.md).
2. **A gap, without mutating anything** — the operator declares `docs/operations/runbook.md` (runbook, required for decision + staging) as baseline v2. The file is not created: the row is **MISSING · no content — not synthesized**, document readiness shows **decision NEEDS_INPUT** and **staging NEEDS_INPUT** naming the file, **development READY**.
3. **Rebuild** — the Context Pack `CP-001` is **PARTIAL**: the missing source is listed with the stages it blocks, present sources carry version `v1` and their sha256, readiness is recorded in the pack, limitations state *nothing was synthesized in their place*, decision refs list `DR-001`, the pack has `source_snapshot_hash` and `pack_hash`.
4. **The AI does not fill the gap** — a new Change opened now is **NEEDS_INPUT** with the missing input `decision_documents: … runbook.md is MISSING (runbook, declared by operator)` and owner `project_owner`.
5. **Withdraw** — needs a reason; baseline v3 restores READY, the PARTIAL pack is reported **STALE** (sources changed since generation), a rebuild yields `CP-002` **DERIVED**. Audit lists `baseline.declare`, `context_pack.rebuild`, `baseline.withdraw`.
6. Domain and HTTP tests additionally cover: manifest documents cannot be withdrawn (`MANIFEST_DOCUMENT_PROTECTED`), stale `expected_version` (409), path escaping the root, unknown stages, already-declared paths, unknown project / pack, the consumer adding the file later (declaration satisfied, pack drift → rebuild DERIVED), and the change ledger leaving NEEDS_INPUT once the document is no longer required.

## Result

| Check | Result | File |
| --- | --- | --- |
| `go vet ./...`, `go test ./...` — 3 document domain tests, 1 HTTP journey, every earlier suite | PASS | `go-test-output.txt` |
| Frontend typecheck / build / unit tests (22) | PASS | `frontend-quality-output.txt` |
| Browser: UI-20 journey plus VS-007, VS-006, VS-005a, VS-004, VS-003 and VS-002 journeys, **two consecutive runs on one state dir** | 11 + 11 PASS | `browser-output.txt` |
| Consumer fixture trees byte-identical after both runs; only `documents.json` / `topology.json` / `changes.json` / `releases.json` written under the state dir | PASS | `mutation-check.txt` |
| Visual: PARTIAL pack with missing runbook, readiness per stage, sources with hashes | — | `documents-partial-pack.png` |

## Findings while verifying

- `tests/browser/project-context.spec.ts` asserted the old read-only document row id; updated to the new panel's row and additionally asserts the stale source name. No product change.

## Boundaries and known gaps

- Declaring a required document is how a gap is demonstrated on read-only fixtures; a real consumer repository shows gaps by lacking files its manifest declares.
- Declaring a new required document after decisions were made marks those decisions STALE (the snapshot hash binds the declared set) — by design; the demo declares first or withdraws before opening changes.
- Same persistence and actor caveats as DR-005…DR-009.
