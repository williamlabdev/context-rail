# EB-004 — Cloud Run baseline (container contract + static workspace)

- Checked at: 2026-09-20 Asia/Taipei
- Base source commit: `c1c2e2c` (develop, fast-forwarded from `codex/001-manual-order-review`)
- Decision: `decisions/DR-004-cloud-run-baseline.json` — `ACCEPTED_FOR_DEVELOPMENT`
- Scope: make the existing read-only workspace deployable to one Cloud Run staging service
- Cloud Run, IAM, production: **not yet executed** — this bundle proves the container contract locally; the first real deployment must add its own receipt (see below)

## Result

| Check | Result | File |
| --- | --- | --- |
| `go vet ./...` and `go test ./...` (registry, HTTP, new health/static tests) | PASS | `go-test-output.txt` |
| Frontend typecheck / build / unit tests (11) | PASS | `frontend-quality-output.txt` |
| Binary started with `PORT=8091` in an image-shaped layout (`/app/context-rail`, `/app/static`, `/app/fixtures/*`) | listens on `0.0.0.0:8091`, mode=container | `smoke-output.txt` |
| `GET /healthz` liveness only | 200 `{"status":"ok","read_only":true,"mode":"container"}` | `smoke-output.txt` |
| `GET /v1/projects` lists both fixtures; `support-insights` context stays `STALE` | PASS | `smoke-output.txt` |
| `GET /`, `/assets/*.js`, client route fallback, missing asset 404 | PASS | `smoke-output.txt` |
| `POST /v1/projects`, `DELETE /` | 405, read-only boundary holds | `smoke-output.txt` |
| `SIGTERM` graceful shutdown; no `PORT` → binds `127.0.0.1:8080` | PASS | `smoke-output.txt` |
| Existing browser journey (4 Playwright tests) against the container-mode server | PASS | `browser-output.txt` |

## Not verified here

- The Docker image build itself: no Docker daemon was available in the verification workspace. `Dockerfile`, `.dockerignore` and `cloudbuild.yaml` are reviewed but unexecuted until the first `scripts/deploy-cloud-run.sh` run. Treat the image build as `NOT VERIFIED` until then.
- Cloud Run deployment, service URL, revision and image digest: `NEEDS_INPUT` — requires the G0 items (GCP project, billing, deploying identity).

## Deployment receipt (fill in after the first real deployment)

```text
gcp_project:        NEEDS_INPUT
region:             NEEDS_INPUT
service:            context-rail-staging
url:                NEEDS_INPUT
revision:           NEEDS_INPUT
image_digest:       NEEDS_INPUT
git_commit:         NEEDS_INPUT
deploying_identity: NEEDS_INPUT
smoke_result:       NEEDS_INPUT   (paste scripts/deploy-cloud-run.sh --smoke output)
```

Until this block is filled with observed values, roadmap gate G1 remains open.
