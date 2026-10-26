# EB-017 AI review: demo /healthz identity change

- Reviewer: a Claude `verifier` subagent with no context from the authoring session beyond the file paths and the review goal. Its model is set by the agent definition (`model: sonnet`), not by the authoring session; the exact model id was not captured.
- Kind: `ai` (not a human review; recorded as the AI half of the single-operator waiver, see README).
- Scope: `git diff develop -- demo/order-operations-portal/main.go demo/order-operations-portal/main_test.go`
- Reviewed at: 2026-10-26, against the working tree before the code commit.

## Commands the reviewer ran

- `git diff develop -- <2 files>`: exit 0, only the two files changed.
- `go vet ./...`: exit 0, no output.
- `go test -count=1 -v ./...`: PASS, 11 tests, including the new `TestHealthReportsRevisionAndBuildDigest`.
- Read the full `main.go`.
- Not run: `./scripts/validate-demo.sh` and `go build ./...` (outside the requested scope).

## Findings (reviewer's words, condensed)

- 🟠 `main_test.go`: the digest test only checked the format, never the value against an independent hash of `os.Executable()`. **Fixed before the code commit**: the test now recomputes the hash itself (`independentExecutableDigest`) and compares.
- 🟠 `main_test.go`: "a restart is a new revision" is tested only through the pure `revisionName` with a fixed time and pid. Uniqueness rests on the pid, which differs between real starts. **Not fixed**; the EB-017 run shows it on real processes (staging and prod-demo instances report different revisions).
- 🟡 `main.go`: the environment label goes into the revision unsanitised. Harmless for a local demo. Not fixed.
- 🟡 `main.go`: the executable is hashed once at startup; a binary replaced on disk between exec and `newServer` would be misreported. Narrow race. Not fixed.
- 🟡 `newServer()` hashes on every call; in tests that is about 11 hashes, no visible cost. Not fixed.

Checked and found fine: the right file is hashed and closed; every error path returns "" without panic; the digest is stable within one executable; the revision format matches the test; `/healthz` reads three immutable strings, adds no lock, I/O or failure path; existing keys `status`, `service`, `environment` are unchanged.

## Verdict

**APPROVED.** "The implementation is correct, runs once per process, and adds no failure or blocking path to /healthz."
