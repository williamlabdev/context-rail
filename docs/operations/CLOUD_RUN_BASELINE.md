# Cloud Run Baseline (staging, read-only workspace)

Status: baseline for roadmap G1
Version: 1.0
Updated: 2026-09-20

This runbook deploys the current ContextRail read-only workspace (VS-001 importer + VS-002 registry API and React UI) to one Cloud Run **staging** service. It exists so that roadmap gate G1 ("Go build, model, Firestore, Storage, identity and Cloud Run path work for real") has a real service URL instead of a local-only claim.

What the deployed service is:

- a GET-only registry API at `/v1/projects` and `/v1/projects/{id}`;
- the built React workspace served from `/` (single-page fallback for client routes);
- a liveness endpoint at `/healthz` that reports process liveness only;
- two governed Project fixtures baked into the image: `demo/order-operations-portal` and `examples/support-insights`.

What it is **not**: it has no persistence, no authentication, no write path, no Gemini/Firestore/Storage integration, no production service and no IAM beyond an unauthenticated demo endpoint. Deploying it does not change any Project's readiness or authorization state.

## Container contract

The binary honours the [Cloud Run container contract](https://cloud.google.com/run/docs/container-contract):

| Variable | Purpose | Image default |
| --- | --- | --- |
| `PORT` | Listen on `0.0.0.0:$PORT`; without it the service stays on `127.0.0.1:8080` for local runs | `8080` |
| `CONTEXT_RAIL_FIXTURE_ROOTS` | Comma-separated explicit Project roots (no filesystem scan) | both fixtures under `/app/fixtures` |
| `CONTEXT_RAIL_STATIC_DIR` | Built workspace directory | `/app/static` |
| `CONTEXT_RAIL_FIXTURE_SCENARIO` | `normal`, `empty`, `invalid`, `unavailable` — verification scenarios only | `normal` |

`SIGTERM` triggers a graceful shutdown with a 10 s drain, which is what Cloud Run sends before stopping an instance.

## Deploy

Prerequisites on the operator machine: `gcloud` authenticated as an identity that can enable APIs, create an Artifact Registry repository, run Cloud Build and deploy Cloud Run in the target GCP project; billing enabled on that project.

```sh
scripts/deploy-cloud-run.sh <gcp-project-id> [region=asia-east1] [service=context-rail-staging]
```

The script enables `run`, `cloudbuild` and `artifactregistry`, creates the `context-rail` Artifact Registry repository if missing, submits `cloudbuild.yaml`, deploys the resolved **image digest** (never a mutable tag) to the staging service, and prints `url`, `revision`, `image@digest` and `commit`. It then runs the smoke check below.

To re-run only the smoke check against an existing URL:

```sh
scripts/deploy-cloud-run.sh --smoke https://context-rail-staging-<hash>-<region>.a.run.app
```

## Smoke check (what counts as PASS)

1. `GET /healthz` returns `{"status":"ok","read_only":true,"mode":"container"}`.
2. `GET /v1/projects` lists both fixture Projects (`order-operations-portal`, `support-insights`).
3. `GET /` returns the workspace HTML (HTTP 200).
4. `POST /v1/projects` returns HTTP 405 — the read-only boundary holds in the cloud exactly as it does locally.

## Evidence to record

For each deployment, record in the slice's EvidenceBundle: service URL, Cloud Run revision name, image digest, git commit, deploying identity, timestamp, and the smoke output. A deployment without these is `NEEDS_INPUT`, not evidence.

## Local equivalent

```sh
cd frontend && npm ci && npm run build && cd ..
go build -o bin/context-rail ./cmd/context-rail
PORT=8080 CONTEXT_RAIL_FIXTURE_ROOTS=demo/order-operations-portal,examples/support-insights ./bin/context-rail
```

Without `PORT` the server binds to loopback only; the flags `--fixture-root`, `--static-dir`, `--addr` and `--fixture-scenario` still work and take precedence over the environment.
