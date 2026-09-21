## AI technical review — claude-cowork-reviewer (separate agent run; not a human approval)

**Verdict: APPROVED** — reviewed against Work Order AWO-001 (ContextRail CHG-001 / DEC-001) and REQ-002.

Verified locally: `gofmt -l .` clean, `go vet ./...` clean, `go build ./...` clean, `go test ./...` 13/13, `go test -race ./...` clean.

**Contract** — changed paths are exactly `main.go`, `main_test.go`, `web/index.html`, `runs/ARR-002-*.json`, all inside `allowed_paths`. Only new import is `net/url`; state is an in-memory map — no bucket, no persistence, no IAM, no `docs/operations/environments.md`, no deploy/CI change, no secrets. The review decision rules are byte-for-byte unchanged; the 8 pre-existing review tests still drive the real router, so the dispatcher refactor is regression-covered.

**Acceptance criteria** — AC-001/002/003/004 (API) each met and tested; AC-005 confirmed. Existence (404) is checked before the cap (409) and check-then-append happen under one write lock, so the 3-reference cap has no race. `evidence` is always an array. URL validation fuzzed with 30 cases (`javascript:`, `data:`, `vbscript:`, `file:`, `//host`, `https://`, embedded spaces, NUL): no bypass.

**UI** — evidence section renders above the review buttons; form only while `PENDING_REVIEW` and under the cap; all text via `textContent`, `href` only takes server-validated http(s) links.

**Findings (none blocking)**
- non-blocking `main.go` — no length bound on `label`/`url` and no `MaxBytesReader` (same exposure as the existing note field; acceptable for synthetic staging).
- non-blocking `main.go` — `writeJSON` runs under the held write lock (pre-existing pattern).
- note `web/index.html` — `MAX_EVIDENCE = 3` duplicates `maxEvidencePerOrder`; the server 409 stays authoritative.
- note — no automated coverage for the UI half of AC-004 (no JS harness in this repo); it rests on the recorded manual smoke.
- note `runs/ARR-002…json` — `declared_commands` omits `./scripts/validate-demo.sh`, which AGENTS.md lists and CI's `build` job runs.
- note — the repository's legacy `work-orders/AWO-001-manual-order-review.json` (REQ-001) shares the id `AWO-001` with the ContextRail Work Order this run cites; the hash in ARR-002 refers to the ContextRail ledger, not to that file.

Everything ARR-002 states about the diff that could be checked is true, including "13 tests: 8 existing untouched, 5 new".
