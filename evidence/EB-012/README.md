# EB-012 — Real agent run on the real governed repository, observed back from GitHub (VS-004 + VS-005 end to end)

- Checked at: 2026-09-21 Asia/Taipei (all timestamps below UTC)
- Governed Project: `github.com/williamlabdev/order-operations-portal` (public, `main` @ `7b2402e`), cloned to `context-rail/tmp/order-operations-portal` — **not** the fixture copy under `demo/`
- ContextRail: `develop` @ `4c7b057` (binary cross-compiled for the operator's Linux VM; state dir `context-rail/.context-rail-state/real-run/`, git-ignored, portable — the same state loads on the Mac with `CONTEXT_RAIL_STATE_DIR=.context-rail-state/real-run` and reports `stale: false`)
- Decisions referenced: `decisions/DR-006-change-decision-pack.json`, `decisions/DR-007-candidate-gate.json` (unknowns updated by this bundle)
- Replaces: the *declared* candidate fixtures of EB-006/EB-007 as the evidence for VS-004/VS-005 (those journeys stay as regression tests)

## What happened, in order

| # | Step | Actor | Result |
| --- | --- | --- | --- |
| 1 | Open Change from `requests/REQ-002-order-exception-evidence-links.md` (title, objective, scope, AC-001…005, allowed paths `main.go` `main_test.go` `web/index.html` `runs/`, forbidden actions, target `staging`, business constraints) | `claude-cowork` (operator's instruction) | `CHG-001` v1 **DECISION_READY** — no missing inputs; observation "project context is STALE" (the consumer's own `docs/ai/context-pack.json` is behind its sources); rule-advisor options: minimal_reversible_slice (recommended), defer, managed_storage_with_signed_urls |
| 2 | **Human decision 1** — accept `minimal_reversible_slice` | William as `founder-001` (via Cowork prompt) | `DEC-001` ACCEPTED_FOR_DEVELOPMENT, bound to `source_snapshot_hash sha256:436aec8e…`, topology v1; Brief + Agent Context Pack share the lineage (`dec-001-decision-response.json`) |
| 3 | Compile Work Order | `founder-001` | `AWO-001` ISSUED, `work_order_hash sha256:952c8da4…0fe061`, `target_branch change/chg-001` ← `main`, expires 2026-10-05 (`awo-001-work-order.json`, `agent-context-pack.json`) |
| 4 | **Agent run** `ARR-002` — implement the pack inside the allowed paths | Claude (Cowork) as `claude-cowork`, run started by `williamlabdev` | `POST /api/orders/{id}/evidence` + listing + UI card; 5 new tests, 8 existing untouched; `gofmt` / `go vet` / `go test` (13/13) / `go build` green; local smoke incl. UI screenshot. Commits `14554c9` (code) and `8853114` (run record `runs/ARR-002-…json`), pushed as `change/chg-001`; **PR #1** opened (`github-pull-1.json`, `github-compare.json`) |
| 5 | CI on the head commit | GitHub Actions | check runs `test` and `build` **success** on `8853114` (`github-check-runs.json`) |
| 6 | Submit candidate **with GitHub read-back** (declared head commit was a placeholder on purpose) | `claude-cowork`; server started in the VM with `GITHUB_TOKEN` from the operator's `gh` login, token never left the VM | `CAND-001` — observation `source: github`, base `7b2402e`, head `8853114`, PR URL, **4 changed paths from the compare API**, checks `test` PASS / `build` PASS from the check runs (declared `local-smoke` kept). 9 gates PASS/deferred, `independent_review` **NEEDS_REVIEW** → verdict **NEEDS_REVIEW**, acceptance refused (`cand-001-needs-review.png`) |
| 7 | Independent AI technical review (separate agent run, actor `claude-cowork-reviewer`; not a human approval) | subagent | **APPROVED**, no blocking findings; posted on PR #1 as review `5262715677` (`ai-review-claude-cowork-reviewer.md`, `github-pull-1-reviews.json`) |
| 8 | Re-submit with the AI review + declared single-operator controls, read back again | `claude-cowork` | `CAND-002` — same observation, `independent_review` **WAIVED** (policy lists `single-operator-controls` for staging) → verdict **CANDIDATE_ACCEPTABLE** (`cand-002-gates-accepted.png`) |
| 9 | **Human decision 2** — accept the candidate for promotion | William as `founder-001` (via Cowork prompt) | `CAND-002` ACCEPTED_FOR_PROMOTION; `CHG-001` **CANDIDATE_ACCEPTED**, `stale: false` (`chg-001-candidate.json`) |

Release bundle, build digest, staging and prod-demo are not part of this bundle (Cloud Run is scheduled for 2026-10-08; `scripts/record-promotion.sh`).

## What this verifies (DR-007 unknowns closed)

- **GitHub read-back VERIFIED** against a real repository: `compare` (base/head/commits/files, branch name with `/`), `pulls?head=owner:branch`, `pulls/{n}/reviews`, `commits/{sha}/check-runs`. Provider facts replaced the declaration: the placeholder head commit became `8853114`, the changed paths came from the diff, `test`/`build` came from check runs with their run URLs as evidence refs.
- **A real coding-agent run is recorded**: the run record lives in the governed repository (`runs/ARR-002-…json`) and is itself part of the observed diff.
- **UI-11 on real data**: the first candidate was refused for lack of an independent review even though every technical gate passed; the waiver needed both a separate AI review *and* declared single-operator controls, and it is limited to staging.
- **Three separate human decisions** are visible with the actor and rationale (`founder-001` for DEC-001 and CAND-002); the agent and the reviewer are recorded under their own identities.

## Result

| Check | Result | File |
| --- | --- | --- |
| Governed repo: `gofmt -l .`, `go vet ./...`, `go test ./...` (13), `go build ./...` on the candidate | PASS | `go-test-output.txt`, `go-build-output.txt` |
| Governed repo CI check runs on head `8853114` (`test`, `build`) | success | `github-check-runs.json` |
| Independent AI review | APPROVED (0 blocking, 2 non-blocking, 4 notes) | `ai-review-claude-cowork-reviewer.md` |
| Candidate gate, read back from GitHub | CAND-001 NEEDS_REVIEW → CAND-002 CANDIDATE_ACCEPTABLE → ACCEPTED_FOR_PROMOTION | `chg-001-candidate.json` |
| ContextRail fixtures (`demo/`, `examples/`) untouched by the real run | `git status` clean | `mutation-check.txt` |
| ContextRail frontend after the gate-table CSS fix: typecheck / build / unit tests (25) | PASS | `frontend-quality-output.txt` |
| Visual | — | `cand-002-gates-accepted.png`, `cand-001-needs-review.png`, `dec-001-decision.png`, `awo-001-work-order.png`, `order-portal-evidence-links.png` |

## Findings while verifying

- The gate table in the candidate card wrapped gate names character by character and squeezed the detail column (visible in the first screenshots). Fixed in `frontend/src/styles/index.css` (`.gate-table` column widths; badge label and code no longer break mid-word). Display only; no test change.
- Id namespaces: ContextRail issued `AWO-001` / `ARR-002` in its own ledger while the governed repository already carries a brownfield `work-orders/AWO-001-manual-order-review.json` (REQ-001, different hash). The reviewer flagged that the two `AWO-001` do not reconcile. Known limitation: ledger ids are per ContextRail state, not negotiated with the consumer's files — a later slice should either import the consumer's issued ids or prefix ContextRail's.
- `project.yaml` of the governed repository still says `visibility: private`; the repository is public since 2026-09-21. Data drift on the consumer side, not a ContextRail defect; left for the operator (it is outside the Work Order's allowed paths).
- The run record's `declared_commands` does not list `./scripts/validate-demo.sh` (AGENTS.md asks for it). It was not run in the agent workspace (the clone there was partial); CI's `build` job runs it and passed. Recorded as-is rather than claimed.

## Boundaries

- The coding agent was Claude in Cowork (`claude-cowork`), not Claude Code; the playbook's Claude Code prompt remains valid for a rerun by the operator.
- The human decisions were taken by the operator through Cowork prompts and recorded with the `X-ContextRail-Actor` header (`founder-001`); the header is recorded, not authenticated (P0 boundary, DR-006/DR-007).
- The AI review is a compensating control under the single-operator policy for low-risk staging only; production still requires an independent human reviewer.
- No deployment, no Cloud Run, no Gemini in this run (rule-advisor only).
