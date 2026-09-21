# EB-008 — VS-006 Staging promotion (bundle gate → build digest → approval → deployment verification → Release Receipt)

- Checked at: 2026-09-20 Asia/Taipei
- Base source commit: `02e695d` (feat/vs-005-candidate-gate)
- Decision: `decisions/DR-008-staging-promotion.json` — `ACCEPTED_FOR_DEVELOPMENT`
- Scope: local development/testing with declared deployment fixtures; no Cloud Run deployment executed, no production action
- Review: AI-assisted implementation with automated evidence; **human review `ACCEPTED_FOR_LOCAL_DEVELOPMENT`** by `founder-001` on 2026-09-21 (single-operator; see [HUMAN_REVIEW_PACKET.zh-TW.md](HUMAN_REVIEW_PACKET.zh-TW.md))

## What the slice shows

1. **UI-13** — a release bundling a promotable Change and a Change without an accepted candidate is **GATE_BLOCKED**; the gate table shows `change CHG-A PASS` and `change CHG-B BLOCKED — no candidate accepted for promotion`; the approval control is withheld ("No partial promotion").
2. **UI-12** — the promotable Change alone: `AWAITING_BUILD` → build recorded (digest, source commit, build id, evidence ref) → `READY_FOR_APPROVAL` → `approver-1` approves (bound to the manifest hash, 72h) → deployment recorded (revision, digest, target, smoke PASS, idempotency key) → **PROMOTED** with **RR-001** linking transition, environment config hash, change → decision → work order → candidate → commit, image digest, approver and revision.
3. **UI-14** — a second release for the same Change is approved; then the staging `target_ref` is edited in the topology: the release shows **STALE** with `target_config_drift`, the approval badge turns **STALE**, and the deployment form disappears until a new release is cut against the new topology.
4. Domain tests additionally cover: build from a commit that does not include an accepted candidate → `BUILD_SOURCE_MISMATCH`; mutable tag refused; deployment before approval → `PROMOTION_FAILED` attempt; digest drift → failed attempt but approval kept; failed smoke → failed attempt; missing observed digest → failed attempt (found by the browser run, fixed, now tested); same idempotency key → replay with no new attempt; second promotion → `ALREADY_PROMOTED`; approver = run starter → `APPROVER_SEPARATION` unless single-operator policy (waiver recorded); new build after approval drops the approval; production target → blocked.

## Result

| Check | Result | File |
| --- | --- | --- |
| `go vet ./...`, `go test ./...` — 7 release domain tests + 1 HTTP journey, all earlier suites | PASS | `go-test-output.txt` |
| Frontend typecheck / build / unit tests (22) | PASS | `frontend-quality-output.txt` |
| Browser: promotion journey (UI-13, UI-12, UI-14) plus VS-005a, VS-004, VS-003 and VS-002 journeys | 9 PASS | `browser-output.txt` |
| Consumer fixture trees byte-identical; ledger shows REL-001 GATE_BLOCKED, REL-002 PROMOTED with RR-001, REL-003 APPROVED (then STALE live) | PASS | `mutation-check.txt` |
| Visual: promoted release with receipt; drift-stale release | — | `release-receipt.png`, `release-drift.png` |

## Boundaries and known gaps

- **No real deployment.** Every deployment record in evidence is declared. `scripts/record-promotion.sh` (gcloud → POST) is UNVERIFIED until a Cloud Run service exists.
- Prod-demo promotion with the same digest is not modelled as a distinct step; it needs a prod-demo environment in the topology and a second release.
- The service does not read Cloud Run itself; the operator script is the read-back path.
- Same persistence and actor caveats as DR-005/006/007.
