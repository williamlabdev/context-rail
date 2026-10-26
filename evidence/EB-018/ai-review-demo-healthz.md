# EB-018 AI review: demo revision label, once-per-process digest, restart test

- Reviewer: a Claude `verifier` subagent with no context from the authoring session beyond the file paths and the review goal. Its model is set by the agent definition (`model: sonnet`), not by the authoring session; the exact model id was not captured.
- Kind: `ai` (not a human review; recorded as the AI half of the single-operator waiver, see README).
- Scope: `git diff develop -- demo/order-operations-portal/main.go demo/order-operations-portal/main_test.go`
- Reviewed at: 2026-10-26, against the working tree before the code commit.
- What the change answers: the open items of the EB-017 AI review (one 🟠, three 🟡), which the founder chose to handle on 2026-10-26 (DR-017 version 2).

## Commands the reviewer ran

- `git diff develop -- <2 files>`, a full read of `main.go`: exit 0.
- `go vet ./...`: exit 0.
- `go test -count=1 -v ./...`: PASS; `TestRestartIsANewRevision` passes, `TestHealthHelperProcess` SKIPs outside its helper role.
- `go test -count=30 -race -run 'TestRestartIsANewRevision|TestRevisionLabel'`: 30 runs, no flake.
- `go test -cover`: ok, 87.8%.
- `go version` (go1.26.6) against `go.mod` (`go 1.22`); `grep revision scripts/local-promotion.sh`.

## Findings (reviewer's words, condensed)

- 🔴 none. 🟠 none.
- 🟡 `main_test.go`: the "different revisions" assertion rests on the pid, because both starts usually fall in the same second. Pid reuse between two sequential starts is practically impossible on macOS and Linux. **Fixed before the code commit**: a comment now says the pid is what separates them.
- 🟡 `main.go`: `revisionLabel` does not collapse runs of `-` and has no length cap; `Prod Demo` and `prod-demo` give the same revision prefix. Matches the stated intent. Not fixed.
- 🟡 `main.go`: the label is trimmed in `newServer` and again in `revisionLabel`; redundant, harmless. Not fixed.

Checked and found fine: the sanitiser cases (uppercase lowered, other runes and invalid UTF-8 become `-`, empty becomes `local`); `sync.OnceValue` is safe and compatible with `go 1.22`; the cached digest still matches the independent hash; the race comment is accurate; the subprocess test cannot pass vacuously and does not recurse; the helper bypasses the logging middleware so the first stdout line is the JSON; `/healthz` keys are unchanged and output for `staging` and `prod-demo` is identical to before.

## Verdict

**APPROVED.** "The revision sanitising, once-per-process digest and real two-process restart test are correct, compatible with Go 1.22 and stable over 30 -race runs, with /healthz keys unchanged and only cosmetic nits remaining."
