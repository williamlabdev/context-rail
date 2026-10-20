# EB-015 — G1 local run path: Go build, `go run`/`docker run`, and the first labelled local staging / prod-demo pair

- Checked at: 2026-10-20 Asia/Taipei, on the developer's Mac (Darwin 25.6.0 arm64, Go 1.26.6, Docker Desktop server 29.6.1). Build/test outputs 15:31–15:35; the three run-path read-backs were re-captured 15:38 after a review asked for a status code on every call and a recorded stop step (timestamps are the first line of each `*-readback.txt`).
- **Gate result: G1 PARTIAL.** Build and both run paths pass; the "model access" clause is not met, see the verdict below.
- Source commit: `d5803c4` on `develop` (ContextRail, Dockerfile, frontend all unchanged); plus the one code change this bundle introduces, `demo/order-operations-portal/main.go` (PORT / ORDER_PORTAL_ENV), committed in the same PR as this bundle
- Decision: [`decisions/DR-015-local-only-deployment.json`](../../decisions/DR-015-local-only-deployment.json) — `ACCEPTED_FOR_DEVELOPMENT`; this is the bundle its acceptance criterion 4 asks for ("the first local staging / prod-demo run … lands as a new evidence bundle (EB-015 or later) with the actual local invocation and read-back output")
- Gate: roadmap §6 **G1** — "Go build, model access and the local run path (`go run` / `docker run`) work for real; no cloud resource is required"
- Review: not yet human-reviewed; composed by the agent session that ran the commands, from the captured outputs in this directory
- CI: **not consulted**; every line below is a local run. The repo is public so CI does run on the PR, but this bundle does not depend on it.
- EB-008 / EB-009 are untouched (DR-015 acceptance criterion 4). They remain declared fixtures.

## What this bundle shows

1. **Go build and tests, on a clean tree.** `go build ./...`, `go vet ./...`, `go test -count=1 ./...` on ContextRail (6 test packages) and on the demo app (10 tests, two of them new). File: `go-build-vet-test-output.txt`.
2. **Frontend build by the README's own command.** `cd frontend && npm ci && npm run build` produced a fresh `frontend/dist`. `npm run typecheck` and `npm test` (9 files, 41 tests) pass. Files: `frontend-build-output.txt`, `frontend-quality-output.txt`.
3. **`go run` path (local binary).** README.md "Quick start (local)" command, verbatim except for `CONTEXT_RAIL_STATE_DIR` pointed at a scratch directory so the run does not touch the working tree. The startup line reports `advisor=rule-advisor read_back=none` (and `mode=container`, which in `cmd/context-rail/main.go:121-124` only means `PORT` was set, so the server binds `0.0.0.0` instead of loopback; no container is involved in this run). Five `curl` calls each record `http=<code> bytes=<n>` plus the first 240 bytes of the body: `GET /` and `GET /projects/order-operations-portal` serve the built workspace; `GET /v1/projects/order-operations-portal/environments` returns the 4-environment topology bootstrapped from `project.yaml` (version 1, `origin=manifest`); `GET …/changes` returns an empty ChangeList; `GET /v1/projects/support-insights/documents` returns the 9-document baseline. The file ends with the `kill` and an `lsof` showing the port released. File: `go-run-readback.txt`.
4. **`docker run` path.** `docker build` of the unchanged Dockerfile (28 steps, distroless nonroot runtime) and `docker run -d -p 8090:8080 context-rail:g1-d5803c4`. The container's own log line shows the baked-in fixture roots and `/app/static`; four `curl` calls through the mapped port each record their status code and body prefix; the file ends with `docker stop`, `docker rm` and a `docker ps -a` count of 0 for that name. Files: `docker-build-output.txt`, `docker-run-readback.txt`.
5. **Two local instances of the demo application, told apart.** The demo app (`demo/order-operations-portal`) previously hard-coded `:8080`, so two instances could not coexist on one machine. It now reads `PORT` (default `:8080`, so the documented `go run .` is unchanged) and `ORDER_PORTAL_ENV`, and echoes the label in its startup line and in `GET /healthz`. Two instances were started from the same binary, `PORT=8091 ORDER_PORTAL_ENV=staging` and `PORT=8092 ORDER_PORTAL_ENV=prod-demo`. The file ties the chain together end to end: `ps eww` prints each PID's own environment (`PORT=…` and `ORDER_PORTAL_ENV=…`), `lsof` maps each PID to its port, and `curl` on each port gets `/healthz` carrying that label and `/api/orders`, all with status codes; it ends with the `kill` and an `lsof` count of 0. File: `demo-two-instances-readback.txt`.

## Result

| Check | Result | File |
| --- | --- | --- |
| `go build ./...`, `go vet ./...`, `go test -count=1 ./...` (ContextRail) | PASS, 6 packages | `go-build-vet-test-output.txt` |
| demo app `go vet`, `go test -count=1 -v` | PASS, 10/10 (8 existing + `TestListenAddrHonoursPort`, `TestHealthReportsEnvironmentLabel`) | same file |
| `npm ci && npm run build` (README verbatim) | PASS, `vite build` ok | `frontend-build-output.txt` |
| `npm run typecheck`, `npm test` | PASS, 41/41 | `frontend-quality-output.txt` |
| Local binary on `:8080`, 5 read-backs, then stopped | 5 × `http=200` recorded per call; `advisor=rule-advisor`; port released | `go-run-readback.txt` |
| `docker build` + `docker run -p 8090:8080`, 4 read-backs, then `docker stop`/`rm` | image `sha256:4bc90617…`; 4 × `http=200` recorded per call; container gone afterwards | `docker-build-output.txt`, `docker-run-readback.txt` |
| demo app, two instances `:8091` staging / `:8092` prod-demo, then stopped | PID→env (`ps eww`) → port (`lsof`) → `/healthz` label, 4 × `http=200`; ports released | `demo-two-instances-readback.txt` |
| **Gemini advisor against the real API** | **NOT RUN — no `GEMINI_API_KEY` on this machine** (searched shell rc files and `.env` files by name; none). The server used `rule-advisor`, which is the documented fallback, so the Gemini HTTP path (`internal/change/advisor.go`, `gemini-3.6-flash`) is still UNVERIFIED exactly as DR-014 / DR-015 say. | `go-run-readback.txt` (startup line) |
| Playwright e2e suite | NOT RUN this session (needs a running server plus Chromium; not part of G1) | — |

**G1 verdict: PARTIAL.** The Go build and both local run paths work for real with no cloud resource. The gate's "model access" clause is **not met**: the server ran on the rule-advisor, which is a deterministic fallback, not model access, and roadmap §6 says not to claim an integration that only a mock has exercised. The real Gemini call stays UNVERIFIED; DR-015 unknown 2 makes it optional for the product, but it is still the one G1 clause without evidence. To close it, run the README command once with the key scoped to that command only, open a Change decision in the workspace, and capture the startup line (it will read `advisor=gemini-3.6-flash (falls back to rule-advisor on error)`) plus the decision's advisor field:

```sh
GEMINI_API_KEY=… PORT=8080 CONTEXT_RAIL_FIXTURE_ROOTS=demo/order-operations-portal,examples/support-insights \
CONTEXT_RAIL_STATIC_DIR=frontend/dist CONTEXT_RAIL_STATE_DIR=.context-rail-state ./bin/context-rail
```

## How two local environments are told apart (settles DR-015 unknown 1)

DR-015 left open whether local staging and local prod-demo would be distinguished by ports, containers, state dirs or a registry digest. This first recorded run settles it with the smallest mechanism that leaves a trace in the evidence itself:

- **Separate `PORT` per instance** (`8091` staging, `8092` prod-demo here; any two free ports work). This is what makes two instances possible at all.
- **`ORDER_PORTAL_ENV` label, echoed by the instance.** The label appears in the process's first stdout line and in `GET /healthz` (`"environment":"staging"` / `"prod-demo"`). A deployment evidence bundle must therefore record *both* the invocation (port + label) *and* the `/healthz` read-back; a port number alone proves nothing about which role the instance was playing, and a label alone could be typed into a document without any process behind it.
- **Separate processes, not separate state.** Two PIDs on two ports prove nothing by themselves about roles; the role is proven by reading each process's own environment (`ps eww`) and its `/healthz` answer, which is why the read-back file records both. The demo app keeps its orders in memory, so there is nothing to isolate on disk; if it ever gains persistence, each instance gets its own state directory, following the same pattern ContextRail already uses with `CONTEXT_RAIL_STATE_DIR`.
- **Containers are optional.** The same two commands work as two `docker run -p` mappings once the demo app has a Dockerfile; nothing in this bundle requires one.

Not changed by this bundle: `demo/order-operations-portal/project.yaml` still declares the staging `target_ref` as `cloud-run/order-operations-portal-staging` and has no `prod-demo` environment. Changing a target or adding an environment is a topology change that goes through the Environment Topology UI/API as a new version (DR-005, roadmap §2), not a text edit alongside an evidence bundle. That remains open for the first real local promotion (G3/G4).

## Boundaries and known gaps

- This bundle proves the run path, not a promotion. No Change, Release or receipt was created; the ContextRail state directory used here was a scratch directory and was not kept; the container and all four local processes were stopped inside the recorded runs.
- The Gemini advisor line above is the only G1 clause not exercised. Nothing in this bundle should be read as evidence that `gemini-3.6-flash` is a valid model name or that the endpoint accepts it.
- `docker build` ran against the Docker Desktop daemon on this Mac; the image was not pushed anywhere and the tag `context-rail:g1-d5803c4` is local only.
- All output files were captured by shell redirection from the live processes, then trimmed only where noted inside the file (`docker-build-output.txt` omits the repeated BuildKit step lines; `frontend-build-output.txt` keeps the summary lines; the `*-readback.txt` files keep the first 240 bytes of each response body). Nothing was retyped.
- Form: unlike EB-014 this bundle has no `HUMAN_REVIEW_PACKET` and no mutation-check file. It is a gate measurement, not a PR compilation, and has not been human-reviewed yet.
