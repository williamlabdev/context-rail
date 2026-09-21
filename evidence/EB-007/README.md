# EB-007 — VS-005a Candidate gate (agent run → observed evidence → deterministic gates → second human decision)

- Checked at: 2026-09-20 Asia/Taipei
- Base source commit: `7dd463b` (feat/vs-004-decision-pack)
- Decision: `decisions/DR-007-candidate-gate.json` — `ACCEPTED_FOR_DEVELOPMENT`
- Scope: local development/testing with declared (fixture) observations; no real agent run, no GitHub read-back executed, no deployment
- Review: AI-assisted implementation with automated evidence; **human review `ACCEPTED_FOR_LOCAL_DEVELOPMENT`** by `founder-001` on 2026-09-21 (single-operator; see [HUMAN_REVIEW_PACKET.zh-TW.md](HUMAN_REVIEW_PACKET.zh-TW.md))

## What the slice shows

1. **UI-10** — a candidate whose observed diff includes `infra/iam/public-bucket.yaml` is **BLOCKED**: the `allowed_paths` gate names the path, `required_checks` still shows PASS (a green suite does not override policy), the accept control is disabled, and the human records REJECTED.
2. **UI-11** — a candidate reviewed only by the run starter is **NEEDS_REVIEW**: the `independent_review` gate says what is missing (a human other than `dev-1`, or AI review + declared single-operator controls where policy allows); acceptance is refused.
3. A compliant candidate with an independent reviewer is **CANDIDATE_ACCEPTABLE**; `smoke` is `DEFERRED_TO_PROMOTION`; `founder-001` accepts it and the change becomes **CANDIDATE_ACCEPTED**.
4. Domain tests additionally cover: work-order hash mismatch, wrong branch, expired order and failed check → BLOCKED; missing check → NEEDS_EVIDENCE; single-operator WAIVED path; stale decision → candidates shown STALE and acceptance refused; forbidden `modify:` prefix inside allowed paths → BLOCKED.
5. HTTP tests cover the same journey plus a **fake read-back** that observes an IAM file the operator did not declare (candidate blocked), keeps declared local checks, and maps adapter failure / absence to `502 READ_BACK_FAILED` / `422 READ_BACK_UNAVAILABLE`.

## Result

| Check | Result | File |
| --- | --- | --- |
| `go vet ./...`, `go test ./...` — 8 candidate domain tests + 2 candidate HTTP tests, all earlier suites | PASS | `go-test-output.txt` |
| Frontend typecheck / build / unit tests (22) | PASS | `frontend-quality-output.txt` |
| Browser: candidate journey (UI-10, UI-11, accept) plus VS-004, VS-003 and VS-002 journeys | 8 PASS | `browser-output.txt` |
| Consumer fixture trees byte-identical; ledger shows CAND-001 BLOCKED/REJECTED, CAND-002 NEEDS_REVIEW, CAND-003 ACCEPTABLE/ACCEPTED, 8 audit events | PASS | `mutation-check.txt` |
| Visual: candidate review section after the journey (note: CAND-002 shows STALE because a later spec changed the topology materially) | — | `candidate-review.png` |

## Boundaries and known gaps

- **No real agent run.** Every candidate here is a declared fixture. The real Claude Code run on the demo repository (with GitHub read-back) is the remaining half of VS-005 and needs the repository to exist (see `scripts/bootstrap-w1.sh`).
- **GitHubReadBack UNVERIFIED.** Exercised only through a fake adapter in tests.
- The gate enforces forbidden actions by `modify:<prefix>` only; `deploy:production` and similar are promotion-gate concerns (VS-006).
- An already-accepted candidate is not re-labelled when the decision later goes stale; the promotion gate must re-check.
- Same persistence and actor caveats as DR-005/DR-006.
