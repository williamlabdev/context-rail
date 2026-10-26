# EB-019 — G3 headless Claude Code run under a ContextRail Work Order (DR-018)

A real external coding-agent run on the external source repository
`github.com/williamlabdev/order-operations-portal` (public, the single source of
the demo, DR-018). ContextRail issues the Work Order; a headless `claude -p` run
(Fable 5.1) writes the change; the harness `scripts/agent-run.sh` checks scope,
reruns the tests, commits, pushes and opens the pull request; real GitHub CI runs
on it; a separate read-only `claude -p` run (Opus 5.5) reviews it; ContextRail
reads the pull request back from GitHub, accepts the candidate, and the head
commit is built and promoted to local staging (DR-015, DR-017).

Change: CHG-001 "Filter the order list by status" — `GET /api/orders?status=X`
returns only orders in that status, no parameter returns all, an unknown status
is a 400. Work Order allowed paths: `main.go`, `main_test.go`.

There are two runs. **Run 2 is the evidence for G3.** Run 1 completed every step
but ran under weaker tool controls than DR-018 v2 states; it is kept as the
record of that deviation.

## Run 2 (`run-eb019-2`, harness at context-rail `04e0622`)

| Step | Result | File |
| --- | --- | --- |
| Work Order | AWO-001 from DEC-001 v1, repository `github.com/williamlabdev/order-operations-portal`, target `agent/order-status-filter-2` | `agent-run-2.txt` |
| author run | `claude-fable-5-1`, exit 0, `success`, 14 turns, 89 s, **US$0.718** (cap US$20); `permission_mode=dontAsk builtins=Bash,Edit,Read`; 3 permission denials (two compound shell commands, one `go build`) | `agent-run-2.txt`, `author-transcript-2.jsonl`, `author-prompt-2.md` |
| scope check | only `main.go`, `main_test.go` changed | `agent-run-2.txt` |
| harness reruns checks | `gofmt -l` empty, `go vet` ok, `go test -v` all PASS, `validate-demo.sh` PASS | `agent-run-2.txt` |
| pull request | [order-operations-portal#4](https://github.com/williamlabdev/order-operations-portal/pull/4), head `8f39828` | `agent-run-2.txt` |
| external CI | run 40353747316: `test` pass, `build` pass (seen, not inferred) | `agent-run-2.txt` |
| AI review | `claude-opus-5-5`, `permission_mode=dontAsk builtins=Read`, 0 denials, read `main.go`, `main_test.go`, `go.mod`; **APPROVED**, US$0.149 (cap US$5) | `ai-review-2.md`, `review-transcript-2.jsonl`, `review-prompt-2.md` |
| candidate | CAND-001 `CANDIDATE_ACCEPTABLE`, source `github`, head `8f39828`; `independent_review` WAIVED under single-operator controls; accepted | `agent-run-2.txt` |
| local staging `:8095` | head commit built; REL-001 `PROMOTED`, receipt RR-001; every deployment gate PASS | `agent-run-2.txt` |
| behaviour on staging | no param → all orders; `PENDING_REVIEW` → the two pending orders; `APPROVED` → `[]`; `NOT_A_STATUS` → 400 | `agent-run-2.txt` |
| teardown | no listener on 8094/8095, scratch removed | `agent-run-2.txt` |

Every transcript, prompt and diff was checked by the harness for the API key and
the GitHub token before it was written here (`scrub`).

## Run 1 (`run-eb019-1`, harness at context-rail `59cb743`) — deviation

Same flow, also green: [order-operations-portal#3](https://github.com/williamlabdev/order-operations-portal/pull/3),
head `b646caf`, CI run 40351630442 `test`/`build` pass, review APPROVED, candidate
accepted, staging PROMOTED. Author US$0.607, review US$0.072.

Both `claude -p` children started with `permissionMode: "auto"` (init event in
`author-transcript-1.jsonl` and `review-transcript-1.jsonl`): they inherited the
invoking session's mode, and `--allowedTools` only pre-approves tools, it does
not deny the rest. The reviewer, given only `Read`, ran `Bash` twice (`git log`,
`git diff`, `go test -race`, `grep`); the author ran `cat`, `ls`, `git status`.
Every command stayed inside the clone and was read-only or `go test`, but the
limit DR-018 v2 described was not what applied — the auto-mode classifier was.

Fix (`04e0622`), checked by three probes before run 2 (US$0.0249, transcripts in `probes/`, scratch path and home directory redacted):
`--tools` removes the unused built-ins (reviewer: `Read` only) and
`--permission-mode dontAsk` denies any tool call not pre-approved (`probe-author-deny.jsonl`:
`touch` denied, listed in `permission_denials`; `probe-tools-read.jsonl`: init tools `["Read"]`). The harness now records `permission_mode` and the
built-in tool set from each init event and stops if they differ.

## Spend

Run 1 US$0.679 + run 2 US$0.867 + permission probes US$0.025 ≈ **US$1.57**,
against the DR-018 total cap of US$50. Earlier model probes (Fable vs Opus,
`--bare` tool set) were ≈ US$0.03; their transcripts were not kept.

## What this bundle does not show

- **Read-only shell commands are still allowed.** In `dontAsk`, Claude Code auto-
  allows commands it classifies as read-only (`ls`, `cat`, `git status` in the
  probe); only writes outside the allowlist are denied. The author could read any
  file its user can read through such a command.
- **The `--disallowedTools` Read rules were never exercised**, and runs 1 and 2
  passed them with a malformed absolute prefix (`///Users/...`, visible in the
  run headers). The fix to `//Users/...` came after run 2 and is in the same pull
  request as this bundle; no run has used that form, and whether Claude Code reads
  it as an absolute path has not been checked.
- **The pull requests are not merged** and the demo copy in context-rail is not
  re-synced; both need the founder's command (DR-018).
  *Update 2026-10-27, after this bundle:* the founder merged
  order-operations-portal#4 (`74361e9`) and the demo copy was synced from it in
  context-rail #28. Run 1's pull request #3 was closed unmerged.
- **Staging ran the pull request head, not `main`.**
- **`--bare` is not isolation.** The init events list the founder's installed
  plugins (LSP plugins and built-ins); the runs did not depend on hooks, but the
  developer's setup was loaded.
- **Single operator.** The independent review is waived on an AI review by a
  different model from the same vendor plus declared controls; no second human.
- The reviewer could not run tests in run 2 (`Read` only) and says so; the tests
  it relies on were run by the harness and by CI.
- The order list without a filter comes back in map order (`range s.orders`);
  that predates this change (external `main`) and is not something the run touched.
- The `context-rail commit=` lines in `agent-run-1.txt` (`e2209b4`) and
  `agent-run-2.txt` (`d06309f`) are the branch commits before it was rebased onto
  develop; they were never pushed. Their `scripts/agent-run.sh` is byte-identical to
  `59cb743` and `04e0622` here (blobs `ee63b241` and `6bd38757`); after a squash
  merge those two stay reachable from the pull request.
