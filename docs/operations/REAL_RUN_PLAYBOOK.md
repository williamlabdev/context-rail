# Real run playbook — Claude Code + GitHub read-back (no GCP needed)

Goal: replace the *declared* fixtures behind VS-004/VS-005 with a real coding-agent run on the real demo repository, observed back from GitHub. Everything here runs on the operator's Mac; the only cloud service touched is GitHub.

Repositories:

- ContextRail — `github.com/williamlabdev/context-rail` (branch `develop`)
- Governed Project — `github.com/williamlabdev/order-operations-portal` (public; split from `demo/order-operations-portal` with its history; `main`; CI check runs `test` and `build` on pull requests and pushes)

> **Executed 2026-09-21 — see [`evidence/EB-012/README.md`](../../evidence/EB-012/README.md).** The run below was carried out once with Claude in Cowork as the coding agent (`claude-cowork`, not Claude Code): CHG-001 → DEC-001 (accepted by the operator as `founder-001`) → AWO-001 → PR #1 on the governed repository → CAND-001 NEEDS_REVIEW → independent AI review → CAND-002 CANDIDATE_ACCEPTABLE → accepted for promotion. The governance state of that run is in `context-rail/.context-rail-state/real-run/` (git-ignored) and the clone in `context-rail/tmp/order-operations-portal` (git-ignored); start the server on the Mac with `CONTEXT_RAIL_STATE_DIR=.context-rail-state/real-run CONTEXT_RAIL_FIXTURE_ROOTS=tmp/order-operations-portal` to see it. Rerunning the steps below with Claude Code produces a second run record (`ARR-003`) and a new Change; nothing here is single-use.

## 0. Prerequisites (Mac)

Go 1.22+, Node 22, `gh` logged in as `williamlabdev`, Claude Code. The fixture copy inside ContextRail stays untouched; the real Project is the clone below.

```sh
git clone https://github.com/williamlabdev/order-operations-portal.git ~/dev/source/projects/order-operations-portal
```

**CI status (2026-09-21):** the repository is public (Actions on a private repository would not start until the account's billing was settled; public repositories need none), the `ci` workflow is active and the first run is green — check runs `test` and `build` with conclusion `success` on `main` ([run 35527471989](https://github.com/williamlabdev/order-operations-portal/actions/runs/35527471989)). Those are exactly what the read-back turns into `test=PASS` / `build=PASS`.

## 1. Run ContextRail against the real Project

```sh
cd ~/dev/source/projects/context-rail
git checkout develop
cd frontend && npm ci && npm run build && cd ..
go build -o bin/context-rail ./cmd/context-rail
GITHUB_TOKEN="$(gh auth token)" \
PORT=8080 \
CONTEXT_RAIL_FIXTURE_ROOTS="$HOME/dev/source/projects/order-operations-portal" \
CONTEXT_RAIL_STATIC_DIR=frontend/dist \
CONTEXT_RAIL_STATE_DIR="$HOME/.context-rail/real-run" \
./bin/context-rail
```

`GITHUB_TOKEN` only needs read access (contents, pull requests, checks); a fine-grained PAT scoped to `order-operations-portal` is the cleaner choice than `gh auth token`. The token is read by the server process only; never commit it.

Open <http://127.0.0.1:8080/projects/order-operations-portal>. The Project map must show `github.com/williamlabdev/order-operations-portal` (repository status `active`), not the fixture placeholder.

## 2. Open the Change (REQ-002) and get the Work Order

Changes → **New Change**, from `requests/REQ-002-order-exception-evidence-links.md` in the Project:

| Field | Value |
| --- | --- |
| Title | Order exception evidence links |
| Objective | Let a reviewer attach up to three supporting evidence references (label + http(s) link) to an order exception before deciding, kept in memory next to the order, listed with the order and visible in the UI. |
| Scope excluded | file upload or storage; persistence; authentication; payment or fulfillment; production; `docs/operations/environments.md` |
| Acceptance criteria | AC-001 label and http(s) URL required, else 400 JSON error · AC-002 fourth reference on the same order → 409 · AC-003 unknown order → 404 · AC-004 references in `GET /api/orders` and in the order card · AC-005 existing review tests keep passing; `go test ./...` and `go build ./...` green |
| Allowed paths | `main.go`, `main_test.go`, `web/index.html`, `runs/` |
| Forbidden actions | no storage bucket or persistence; no IAM change; no change to review decision rules; no production deploy |
| Target environment | staging |
| Data classification | internal |
| Expected monthly volume | 200 references per month |

Expect **DECISION_READY** (all inputs supplied). **Accept decision** as `founder-001` (option `minimal_reversible_slice`), then **Compile Work Order** (issuer `founder-001`). Note the Work Order id, `target_branch` (`change/chg-nnn`), `base_branch` (`main`), `work_order_hash` and the required checks (`test`, `build` must have PASS evidence; `smoke` is deferred to promotion). Open **Show pack JSON** and copy the Agent Context Pack.

## 3. Run Claude Code in the Project clone

```sh
cd ~/dev/source/projects/order-operations-portal
git checkout -b change/chg-001 main      # use the exact target_branch from the Work Order
claude
```

Prompt (paste the Agent Context Pack JSON after it):

> Implement this Agent Context Pack exactly. Change only the files listed in `allowed_paths`; do not touch anything else. Run `go test ./...` and `go build ./...` until green. Write an Agent Run Record to `runs/ARR-002-order-exception-evidence-links.json` with: run_id `ARR-002`, work_order_id and work_order_hash from the pack, started_by `<your GitHub login>`, agent `claude-code`, the branch and head commit, the changed paths, and the exact check results. Commit with a message that names the Work Order id, then open a pull request against `main` with `gh pr create --fill`. Do not deploy, do not change environments.md, do not add storage.

When the PR is open, wait for the `test` and `build` check runs to finish (Actions tab). Optionally have a second GitHub account review the PR; with a single operator, the staging policy allows a documented single-operator waiver instead (`single-operator-controls` is in the staging required evidence).

## 4. Submit the candidate with GitHub read-back

In ContextRail → the Change → **Submit candidate**:

- Run id `ARR-002`, started by `<your GitHub login>`, agent `claude-code`, model as reported by Claude Code.
- Branch = the PR branch, base `main`, repository `github.com/williamlabdev/order-operations-portal`.
- Tick **Read back branch, diff, checks and reviews from GitHub**. Leave changed paths / head commit / checks as declared placeholders — the read-back replaces them with what GitHub returns (compare diff, PR reviews, the `test` and `build` check runs of the head commit). Wait for the PR's checks to finish before submitting; a check still running reads back as FAIL.
- If there is no independent reviewer: tick **Single-operator controls documented** with an evidence ref (e.g. the PR URL plus `docs/governance/policies.md#single-operator`) **and** add an AI review (`kind: ai`, verdict `APPROVED`, evidence ref = the review comment on the PR) from a separate agent run — the waiver needs both; without the AI review the candidate is NEEDS_REVIEW (that is exactly what CAND-001 in EB-012 shows).

Expected: `commit_observed`, `branch_matches`, `allowed_paths` (all changed paths within the four allowed entries), `required_checks` PASS from the observed check runs, `independent_review` PASS or WAIVED → verdict **CANDIDATE_ACCEPTABLE**. Then **Accept candidate for promotion** as `founder-001`.

If the diff touched a file outside `allowed_paths`, the verdict is **BLOCKED** with the path named — that is the real UI-10 evidence; fix the branch and resubmit (`Submit another candidate`).

## 5. Record the evidence

Copy into `evidence/EB-012/` of ContextRail: the candidate JSON (`GET /v1/projects/order-operations-portal/changes/<CHG>`), the PR URL, the check-run URLs, a screenshot of the gate table, and the run record from the Project. Update `decisions/DR-007-candidate-gate.json` unknowns: GitHub read-back is now VERIFIED against `<PR URL>`.

Release, build digest, staging and prod-demo stay for 10/8 (`scripts/bootstrap-w1.sh`, then `scripts/record-promotion.sh`).
