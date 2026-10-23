# EB-016 — G1 model access: one real Anthropic advisor call over the local `go run` path

- Checked at: 2026-10-20 Asia/Taipei, on the developer's Mac (Darwin 25.6.0 arm64, Go 1.26.6). The real call was captured at 08:08:42Z (16:08 local); the build/test capture is in `go-build-test-output.txt` with its own timestamp.
- **Gate result: G1 MET.** Build and both run paths were shown in [EB-015](../EB-015/README.md); this bundle closes the one clause EB-015 left open, "model access", with one real model call that produced the candidates of a Change decision.
- Source commit: `c3d4fe1` (PR #22 head, the EB-015 bundle) plus the DR-016 change set committed together with this bundle: `internal/change/advisor.go` (shared prompt), `internal/change/advisor_anthropic.go` (new), `cmd/context-rail/main.go` (advisor selection), `tests/change/advisor_anthropic_test.go` (new), `README.md`, `roadmap.md`.
- Decision: [`decisions/DR-016-advisor-provider-selection.json`](../../decisions/DR-016-advisor-provider-selection.json) — `ACCEPTED_FOR_DEVELOPMENT`; this bundle is its acceptance criterion 4.
- Gate: roadmap §6 **G1** — "Go build, model access and the local run path (`go run` / `docker run`) work for real; no cloud resource is required"
- Review: not yet human-reviewed; composed by the agent session that ran the commands, from the captured outputs in this directory.
- CI: **not relied on as evidence**; every line below is a local run. One CI fact is observed and recorded in the last section (PR #22's `go` job failing, run 39388003970), as the reason for a change outside the evidence proper.
- The API key was read from a file outside the repository into the environment of the single server process and nowhere else. It was not exported into the shell, it is not in any file in this directory (a search for the Anthropic key prefix over `evidence/` and `decisions/` finds no key), and the adapter sends it only in the `x-api-key` header, never in a URL or an error message.

## What this bundle shows

1. **Build, vet and the full test suite pass** with the DR-016 change set in the tree: `go build ./... && go vet ./...` exit 0; `go test -count=1 ./...` all six test packages `ok`. The new `tests/change/advisor_anthropic_test.go` covers the wire contract (path, headers, model, `output_config` schema with `additionalProperties:false` and the `pricing_ref` enum), the HTTP-error, refusal, truncation and zero-option paths, the default model, and the `SelectAdvisor` table (no key, one key, both keys, explicit provider with and without its key, unknown provider). File: `go-build-test-output.txt`.
2. **Startup names the model.** `./bin/context-rail` started with `CONTEXT_RAIL_ADVISOR=anthropic`, `PORT=8080`, the two fixture roots, the built workspace and a scratch state directory. The startup line reads `advisor=anthropic:claude-opus-5-5 (falls back to rule-advisor on error)`; `GET /healthz` returned `http=200`. File: `anthropic-advisor-run.txt` lines 1–16.
3. **One real advisor call inside a Change decision.** `POST /v1/projects/order-operations-portal/changes` with the complete "Manual order review" request (the test suite's complete fixture plus a `requested_by` field) returned `http=200` in `time_total=13.15s`. The response is `CHG-001`, status `DECISION_READY`, with **two options both carrying `"advisor_source": "anthropic:claude-opus-5-5"`**, `"observations": []` (so the rule-advisor fallback did **not** run), eight model-written unknowns followed by the one rule-advisor unknown (`repository URL is a fixture placeholder…`) merged in by `mergeUnique`, and `pricing_ref: "none"` on both options, which the deterministic calculator correctly reports as `cost_estimate.status: UNKNOWN` with the reason naming the price table version. File: `anthropic-advisor-run.txt` lines 18–230.
4. **The structured-output contract held.** The answer parsed as the `{"options":[…],"unknowns":[…]}` object the schema asks for, every option has all seven required fields, and `pricing_ref` is a value from the enum. This is the first evidence in this repository that a real model honours the advisor schema; DR-014 had recorded it as UNVERIFIED for Gemini, and for Gemini it still is.
5. **Teardown recorded.** `kill <pid>`, the server's own `shutting down` line, and `lsof -nP -iTCP:8080 -sTCP:LISTEN` showing no listener. The state directory was scratch and is not committed. File: `anthropic-advisor-run.txt` lines 232–236.

| Step | Result | File |
| --- | --- | --- |
| `go build ./... && go vet ./...` | exit 0 | `go-build-test-output.txt` |
| `go test -count=1 ./...` | 6 packages `ok` | `go-build-test-output.txt` |
| startup with `CONTEXT_RAIL_ADVISOR=anthropic` | `advisor=anthropic:claude-opus-5-5 (falls back to rule-advisor on error)` | `anthropic-advisor-run.txt` |
| `GET /healthz` | `http=200` | `anthropic-advisor-run.txt` |
| `POST …/changes` (real model call) | `http=200`, 13.15 s, 2 options from `anthropic:claude-opus-5-5`, no fallback observation | `anthropic-advisor-run.txt` |
| stop | no listener on 8080 | `anthropic-advisor-run.txt` |

## G1 verdict

**MET.** EB-015 showed the Go build and both local run paths; this bundle shows model access through the same `go run` path with no cloud resource: a real model produced the candidate options of a real Change decision, attributed in the ledger by `advisor_source`. The gate is met by one real provider through the `go run` path only; the `docker run` path was not re-run with an advisor. EB-015 had phrased the closing step as a Gemini run; DR-016 (the founder's decision not to be bound to one provider) is why it was closed with Anthropic instead, and roadmap §6 G1 names no provider. The Gemini adapter is unchanged and still UNVERIFIED against the real API; the roadmap row says so rather than implying both paths were exercised.

## What this bundle does not show

- **Cost was not measured.** The server does not log the API's `usage` field, so the token count of this call is not captured. DR-016 carries a list-price estimate (cents), not a measurement.
- **One call, one request.** This is not a quality or consistency evaluation of the advisor; it is proof of access and of the schema contract on one real response. Judging whether the candidates are good is the human decision step, by design.
- **The 60 s server write timeout was not approached** (13 s measured). The adapter's own timeout is 45 s so that a slow model still leaves the rule fallback time to answer; this limit was not exercised.
- **`docker run` with the advisor** was not repeated; the container path was shown in EB-015 with the rule advisor, and the advisor selection is environment-only, so the same variables apply inside the container. Not re-run, so not claimed.

## Also in this change set (not evidence, but recorded here)

PR #22 (EB-015) edited `demo/order-operations-portal/README.md`, which is a declared source of that demo Project's Context Pack. The edit made the pack `STALE` and failed two tests (`tests/http`, `tests/registry`) on that branch and in its CI `go` job (observed via `gh pr checks 22`, run 39388003970). This change set restores that README to its `develop` content; the `PORT` / `ORDER_PORTAL_ENV` behaviour stays documented in `demo/order-operations-portal/main.go` and in EB-015. The demo Project's Context Pack and DR-001 are not rewritten.
