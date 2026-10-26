# EB-017 — G4 local promotion: staging to an isolated local prod-demo, gates reading each instance back

- Checked at: 2026-10-26 Asia/Taipei, on the developer's Mac (Darwin 25.6.0 arm64, Go 1.26.6). The promotion run was captured at 13:42:08Z (21:42 local); the build/test capture is in `go-build-test-output.txt` with its own timestamp.
- **Gate result: G4 MET for the local scope, with the limits listed at the end; the design behind it (DR-017) still awaits the founder's confirmation.** One locally built demo binary was promoted through local staging and local prod-demo; digest drift, target drift and a moved revision were each BLOCKED by a gate reading the live instance; a same-key retry replayed without a new read; every attempt left a receipt.
- Source commit: `624804a`, the code under test and the harness as run. It is two commits on top of `7d371b8` (DR-017's `source_commit`, the base the decision was made against): `4a4a142` adds the demo `/healthz` identity, `internal/release/observer.go`, the `source: "local"` path in `internal/release/service.go`, `cmd/context-rail/main.go`, `tests/release/local_readback_test.go` and `scripts/local-promotion.sh`; `624804a` applies two fresh-eyes findings (the read-back never follows redirects; `deployed_config_hash` is refused on a local request like the other identity fields) and harness fixes. The only tracked changes not in that commit when the run started were `README.md` and `roadmap.md` (`local-promotion-run.txt` line 4).
- Decision: [`decisions/DR-017-local-promotion-read-back.json`](../../decisions/DR-017-local-promotion-read-back.json) — `ACCEPTED_FOR_DEVELOPMENT`; this bundle is its acceptance criterion 4. The design choices in it were made by the agent and await the founder's confirmation (see its `human_decision`).
- Gate: roadmap §6 **G4** — "Local staging and isolated local prod-demo promotion, drift checks, idempotency and receipt work."
- Review: not yet human-reviewed; composed by the agent session that ran the commands, from the captured outputs in this directory. The demo code change has one AI review (`ai-review-demo-healthz.md`, APPROVED).
- CI: **not relied on as evidence**; every line below is a local run.
- No API key was used: the server ran with `CONTEXT_RAIL_ADVISOR=rule`, and the invoking shell had no `ANTHROPIC_API_KEY` (`local-promotion-run.txt` line 7).

## What this bundle shows

1. **Build, vet and tests pass.** `go build ./... && go vet ./...` exit 0; `go test -count=1 ./...` all six test packages `ok`; the demo module's `go vet` and `go test -v` pass, and so does its own `scripts/validate-demo.sh`. File: `go-build-test-output.txt`.
2. **Each instance says who it is.** The same binary (`sha256:16e850d836cf…`) runs twice, as staging on `:8091` and prod-demo on `:8092`; a second build of the same source with `-trimpath` (`sha256:9c984ef7c418…`, different bytes) runs on `:8093` labelled prod-demo. Each `/healthz` reports its environment label, a revision new to that start, and the sha256 of its own executable. The startup line of ContextRail reads `local_read_back=loopback`. File: `local-promotion-run.txt` lines 9–27.
3. **Topology changed at runtime, the governed Project did not.** `PATCH …/environments/staging` set `target_ref` to `local/order-operations-portal-staging` (version 1 → 2) and `POST …/environments` added `prod-demo` at sequence 4 with target `local/order-operations-portal-prod-demo` (version 2 → 3). `project.yaml` and the Context Pack are untouched (`git status` clean under `demo/`). File: lines 28–705.
4. **A governed change reaches the release.** `CHG-001` (the `/healthz` identity change itself) → decision accepted → work order `AWO-001` → candidate `CAND-001` at `head_commit` `624804a`, with local `test` and `build` checks and the AI review → `CANDIDATE_ACCEPTED`. The `independent_review` gate is `WAIVED` under single-operator controls, not passed (line 3224). File: lines 706–4706.
5. **Staging promoted from what the instance said.** `REL-001` build evidence names the binary digest; approval waived for the single operator; the deployment request carried only `idempotency_key`, `source: "local"` and `service_url`. ContextRail read `:8091/healthz` and filled revision, digest and target from it: all gates PASS, outcome `PROMOTED`, receipt `RR-001`. File: lines 4707–5432.
6. **Idempotency.** The same request with the same key returned the same attempt `REL-001-D01` and receipt `RR-001` with the same `receipt_hash`; the release still has one deployment. The staging instance logged 4 `GET /healthz` in total: the script's readiness probe, the direct read in step 2, the deployment, and prod-demo attempt B below; the replay added none. File: lines 5433–5796 and 9218–9221.
7. **Drift is caught from the read-back.** `REL-002` promotes `REL-001` to prod-demo (build inherited, never rebuilt; the four-field environment delta is bound by the approval).
   - Attempt A against `:8093`: `digest_drift` BLOCKED (`deployed sha256:9c984ef7c418 ≠ approved sha256:16e850d836cf`), every other gate PASS, receipt `RR-002` `PROMOTION_FAILED`.
   - Attempt B against `:8091`, the staging instance: `target_drift` BLOCKED and `new_revision` BLOCKED (the revision is the one behind `RR-001`), receipt `RR-003` `PROMOTION_FAILED`.
   - Attempt C against `:8092`: all gates PASS, `PROMOTED`, receipt `RR-004` with `previous_receipt_id` `RR-001`.
   All three receipts point back to `RR-001`, the source release's receipt, not to each other. File: lines 5797–8154.
8. **Teardown recorded.** All four listeners gone (`lsof … -sTCP:LISTEN` empty on 8090–8093); the state directory was scratch and is not committed. File: lines 9223–9228.

| Step | Result | File |
| --- | --- | --- |
| build, vet, test (root + demo + `validate-demo.sh`) | exit 0, all `ok` | `go-build-test-output.txt` |
| startup | `local_read_back=loopback`, `advisor=rule-advisor` | `local-promotion-run.txt` |
| topology | staging retargeted (v2), prod-demo added (v3) | `local-promotion-run.txt` |
| candidate | `CANDIDATE_ACCEPTED`, independent review `WAIVED` | `local-promotion-run.txt` |
| staging deploy `:8091` | `PROMOTED`, `RR-001` | `local-promotion-run.txt` |
| same-key replay | same attempt, same `RR-001` hash, no new read | `local-promotion-run.txt` |
| prod-demo `:8093` (other binary) | `digest_drift` BLOCKED, `RR-002` | `local-promotion-run.txt` |
| prod-demo `:8091` (staging instance) | `target_drift` + `new_revision` BLOCKED, `RR-003` | `local-promotion-run.txt` |
| prod-demo `:8092` | `PROMOTED`, `RR-004`, previous receipt `RR-001` | `local-promotion-run.txt` |
| teardown | no listener on 8090–8093 | `local-promotion-run.txt` |

Reproduce from the repo root with `scripts/local-promotion.sh` (ports and output path are environment-configurable; it overwrites `local-promotion-run.txt` by default). Revisions, pids, timestamps and receipt hashes differ per run. The binary digests are stable for one commit but change with every commit even when the demo source does not, because Go stamps `vcs.revision` and `vcs.modified` into the binary (`go version -m bin/order-portal`).

## G4 verdict

MET for the local scope DR-015 defines. Promotion, drift checks, idempotency and receipts all ran against live processes, and the failing cases failed for the reason the gate names. The fallback "report as staging-only" is not needed. This says nothing about production readiness: `production` stays protected and read-only, and no attempt was made against it.

## What this bundle does not show

- **An OCI image digest.** `build_digest` is the sha256 of a local executable. It is compared with the build evidence's `image_digest` field because both name the bytes that run; no registry or container runtime was involved.
- **`config_drift`.** `/healthz` reports no config hash, so the gate is skipped for these records, as for any record without `deployed_config_hash` (DR-017 unknown 1).
- **Isolation beyond process and port.** Staging and prod-demo share the machine, the user and the ContextRail state directory. Attempt B shows the gate catching a misdirected deployment; nothing prevents one.
- **A second human.** One operator accepted the candidate and approved both releases under the single-operator waiver; a distinct human approver was not exercised.
- **A followed redirect.** The read-back refuses to follow 3xx; that is shown by a unit test only, the real run had no redirecting instance.
- **Real CI or an external coding-agent run.** The candidate's checks are local captures and its run record describes this session; that is G3's ground, not this bundle's.
- **The `docker run` path.** Only the `go build` binaries were promoted.
