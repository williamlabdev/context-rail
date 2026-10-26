# EB-018 — G4 local promotion against the topology project.yaml declares (DR-017 v2)

- Checked at: 2026-10-26 Asia/Taipei, on the developer's Mac (Darwin 25.6.0 arm64, Go 1.26.6). The promotion run was captured at 14:19:33Z (22:19 local); the build/test capture is in `go-build-test-output.txt` with its own timestamp.
- **Gate result: G4 MET for the local scope, with the limits listed at the end. The founder ruled on DR-017's design choices on 2026-10-26 (version 2); this bundle reruns the promotion under that ruling.** The difference from EB-017 is where the topology comes from: staging's local target and the prod-demo environment are declared in the demo `project.yaml` and read back, not created through the topology API at run time.
- Source commit: `46482f5`, the code under test and the harness as run. It is two commits on top of `65a7d44` (DR-017 v2's `source_commit`, develop after PR #24). `35477b2` declares the topology in `project.yaml` and `docs/operations/environments.md`, rebuilds the Context Pack with `template/scripts/rebuild_context_pack.py`, updates the four test files that pinned the old four-environment topology, makes the demo revision label `[a-z0-9-]` only, hashes the executable once per process, adds a real two-process restart test, and switches `scripts/local-promotion.sh` from topology edits to a read-back. `46482f5` corrects the harness's candidate record after the fresh-eyes review: `CHG-001` now covers the declared topology instead of listing `project.yaml` as out of scope, the candidate's `base_commit` is `65a7d44` and its `changed_paths` are the five demo files `35477b2` changes (demo-relative; the root test files and the harness are outside the demo Project and are governed by DR-017, not by `CHG-001`), and the run record names `claude-opus-5-5`. The tracked changes not in `46482f5` when the run started were `README.md`, the DR-017 file and `roadmap.md` (`local-promotion-run.txt` line 4).
- Decision: [`decisions/DR-017-local-promotion-read-back.json`](../../decisions/DR-017-local-promotion-read-back.json) version 2 — `ACCEPTED_FOR_DEVELOPMENT`; this bundle is its acceptance criteria 4 and 5. Its `human_decision` records the founder's choices.
- Supersedes as the current G4 run: [`EB-017`](../EB-017/README.md), which stays unchanged as the record of version 1 (runtime topology).
- Gate: roadmap §6 **G4** — "Local staging and isolated local prod-demo promotion, drift checks, idempotency and receipt work."
- Review: not yet human-reviewed; composed by the agent session that ran the commands, from the captured outputs in this directory. The AI review (`ai-review-demo-healthz.md`, APPROVED) covers the demo code only, `main.go` and `main_test.go`; the `project.yaml`, `environments.md` and Context Pack declaration, the four test files and the harness had no AI review, only the fresh-eyes review of this bundle and DR-017. That review checks the fixes for the EB-017 AI review's open items, which the founder chose to handle ("處理 AI 審查的 🟡"). Of its own three new 🟡 findings one was fixed (the pid comment) and two were declined by the agent as matching the stated intent (no collapse or length cap in `revisionLabel`; the redundant trim).
- CI: **not relied on as evidence**; every line below is a local run.
- No API key was used: the server ran with `CONTEXT_RAIL_ADVISOR=rule`, and the invoking shell had no `ANTHROPIC_API_KEY` (`local-promotion-run.txt` line 7).

## What this bundle shows

1. **Build, vet and tests pass, against five declared environments.** `go build ./... && go vet ./...` exit 0; `go test -count=1 ./...` all six test packages `ok`, including `tests/http`, `tests/registry` and `tests/topology` against development, testing, staging, prod-demo and production; the demo module's `go vet` and `go test -v` pass (with `TestRevisionLabelIsSanitised` and `TestRestartIsANewRevision`), and so does its own `scripts/validate-demo.sh` (a file-presence check). File: `go-build-test-output.txt`. Separately, recomputing the sha256 of all 9 sources the rebuilt Context Pack declares gives no mismatch (`context-pack-hash-check.txt`), so the pack is CURRENT, which is what `tests/registry` reads as `DERIVED` rather than `STALE`.
2. **Each instance says who it is.** The same binary (`sha256:bee951949284…`) runs twice, as staging on `:8091` and prod-demo on `:8092`; a `-trimpath` build of the same source (`sha256:fa5365ca873c…`) runs on `:8093` labelled prod-demo. Each `/healthz` reports its environment label, a revision new to that start, and the sha256 of its own executable. File: `local-promotion-run.txt` lines 9–27.
3. **The topology is the declared one.** `GET …/environments` returns version 1, bootstrapped from `project.yaml`: `staging=local/order-operations-portal-staging`, `prod-demo=local/order-operations-portal-prod-demo`, `production=protected/read-only`. The script checks both local targets and would stop on a mismatch; it makes no topology edit. File: lines 28–152 (summary line 151).
4. **A governed change reaches the release.** `CHG-001` → decision accepted → work order `AWO-001` → candidate `CAND-001` at `head_commit` `46482f5`, with local `test` and `build` checks and the AI review → `CANDIDATE_ACCEPTED`. The `independent_review` gate is `WAIVED` under single-operator controls, not passed (line 2800); the `allowed_paths` gate passes on the five changed paths (line 2781). File: lines 153–4355.
5. **Staging promoted from what the instance said.** The deployment request carried only `idempotency_key`, `source: "local"` and `service_url`; ContextRail read `:8091/healthz` and filled revision, digest and target from it: all gates PASS, `PROMOTED`, receipt `RR-001`. File: lines 4356–5081.
6. **Idempotency.** The same request with the same key returned attempt `REL-001-D01` and `RR-001` with the same `receipt_hash` (`sha256:266c512d…`); the replay added no `GET /healthz` to the staging instance (4 in total: readiness probe, the direct read in step 2, the deployment, attempt B). File: lines 5082–5445 and 8818–8820.
7. **Drift is caught from the read-back.** `REL-002` promotes `REL-001` to prod-demo.
   - Attempt A against `:8093`: `digest_drift` BLOCKED (`deployed sha256:fa5365ca873c ≠ approved sha256:bee951949284`), receipt `RR-002` `PROMOTION_FAILED`.
   - Attempt B against `:8091`, the staging instance: `target_drift` BLOCKED and `new_revision` BLOCKED, receipt `RR-003` `PROMOTION_FAILED`.
   - Attempt C against `:8092`: all gates PASS, `PROMOTED`, receipt `RR-004` with `previous_receipt_id` `RR-001`.
   All three receipts point back to `RR-001`, the source release's receipt, not to each other. File: lines 5446–7763.
8. **Teardown recorded.** No listener on 8090–8093; the state directory was scratch and is not committed. File: lines 8822–8827.

| Step | Result | File |
| --- | --- | --- |
| build, vet, test (root + demo + `validate-demo.sh`) | exit 0, all `ok` | `go-build-test-output.txt` |
| topology | declared v1 read back, both local targets match, no edit | `local-promotion-run.txt` |
| candidate | `CANDIDATE_ACCEPTED`, independent review `WAIVED` | `local-promotion-run.txt` |
| staging deploy `:8091` | `PROMOTED`, `RR-001` | `local-promotion-run.txt` |
| same-key replay | same attempt, same `RR-001` hash, no new read | `local-promotion-run.txt` |
| prod-demo `:8093` (other binary) | `digest_drift` BLOCKED, `RR-002` | `local-promotion-run.txt` |
| prod-demo `:8091` (staging instance) | `target_drift` + `new_revision` BLOCKED, `RR-003` | `local-promotion-run.txt` |
| prod-demo `:8092` | `PROMOTED`, `RR-004`, previous receipt `RR-001` | `local-promotion-run.txt` |
| teardown | no listener on 8090–8093 | `local-promotion-run.txt` |

Reproduce from the repo root with `scripts/local-promotion.sh` (it overwrites `local-promotion-run.txt` by default). Revisions, pids, timestamps and receipt hashes differ per run; binary digests change with every commit because Go stamps VCS information into the binary, a limit the founder chose to keep (DR-017 v2). The harness's log label for attempt C now reads "previous receipt RR-001" (`46482f5`); EB-017 kept the older "receipt chained to RR-001" wording.

## G4 verdict

MET for the local scope DR-015 defines, now with the topology declared in the governed Project rather than held only in ContextRail's runtime state. The fallback "report as staging-only" is not needed. This says nothing about production readiness: `production` stays protected and read-only, and no attempt was made against it.

## What this bundle does not show

- **An OCI image digest.** `build_digest` is the sha256 of a local executable; no registry or container runtime was involved.
- **`config_drift`.** `/healthz` reports no config hash, so the gate is skipped for local records (DR-017 unknown 1).
- **Isolation beyond process and port.** Staging and prod-demo share the machine, the user and the ContextRail state directory. Attempt B shows the gate catching a misdirected deployment; nothing prevents one.
- **A second human.** One operator accepted the candidate and approved both releases under the single-operator waiver.
- **A followed redirect.** Refusing 3xx is shown by a unit test only.
- **prod-demo owner and approver policy.** The manifest has no fields for them, so the declared prod-demo carries neither (EB-017's runtime version had both); neither feeds a gate.
- **Real CI or an external coding-agent run**, and **the `docker run` path.**
