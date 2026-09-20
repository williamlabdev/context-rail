# ContextRail

**Environment-aware AI Change Assurance for cloud-native business systems.**

Before an AI-assisted change advances to the next governed environment, ContextRail proves that the change still matches the accepted intent, scope, policy and release identity — and blocks it when it does not.

> Google Cloud AI Builder Cup 2026 entry. Deployed workspace: `<deployed Cloud Run URL — filled in at submission>` · Demo video: `<link — filled in at submission>`

## The problem

Startup and SMB engineering teams already use Git, CI and an AI coding assistant, and ship to two or more environments. What they lack is a control plane that keeps three questions answerable while the AI works fast:

1. What exactly was accepted, by whom, against which sources?
2. Did the coding agent stay inside that scope, with evidence rather than claims?
3. Is the image that reached staging — and then prod-demo — the reviewed change, deployed to the declared target, with the environment differences approved?

## What ContextRail does

One governed **hero path**, three separate human decisions, deterministic gates that block but never approve:

```
Project Context Contract ─▶ Environment Topology (versioned) ─▶ Change request
     ▼
Decision Pack: NEEDS_INPUT or DECISION_READY ─▶ 1st human decision ─▶ Brief + Agent Context Pack (same lineage)
     ▼
Agent Work Order (hashed) ─▶ coding agent run ─▶ Candidate gate (allowed paths, checks, independent review …)
     ▼
2nd human decision ─▶ Release bundle ─▶ build digest ─▶ 3rd human approval (bound to manifest hash)
     ▼
Deployment record (digest / target / config drift, smoke, idempotency) ─▶ Release Receipt
     ▼
Same digest ─▶ prod-demo, environment delta bound by a new approval ─▶ chained Release Receipt
```

Everything that can drift is hashed and versioned: topology config, source snapshots, work orders, manifests, receipts. When a source, an environment or a candidate moves after a decision, the dependent decision or approval is marked **STALE** — never silently re-used.

### Product surfaces (all runnable today)

| Surface | What it governs | Slice |
| --- | --- | --- |
| Project Registry | Read-only import of the Project Context Contract (`project.yaml`, documents, decisions, readiness) | VS-001/002 |
| Environment Topology | Add / edit / reorder / retire / restore environments as immutable versions; production stays read-only; material changes invalidate decisions | VS-003 |
| Change Decision Pack | Deterministic readiness (missing inputs name their owner), rule or Gemini advisor, first human decision, Brief + Agent Context Pack from one DecisionRecord, hashed Work Order | VS-004 |
| Candidate gate | Agent run + observed evidence (optionally read back from GitHub) → BLOCKED / NEEDS_EVIDENCE / NEEDS_REVIEW / CANDIDATE_ACCEPTABLE, second human decision | VS-005 |
| Staging promotion | Release bundle gate (no partial promotion), build digest, third human approval bound to the manifest hash, deployment verification, Release Receipt | VS-006 |
| Prod-demo promotion | Same digest to the next environment, environment delta listed and bound by a new approval, new revision required, receipts chained | VS-007 |
| Documents / AI Context | Versioned document baseline, missing documents mapped to readiness stages, rebuildable Context Pack (`PARTIAL` when anything is missing — nothing is synthesized) | UI-20 |
| Workspace | URL-bound Project, zh-TW / en, human status labels with machine codes | UI-15…19 |

## Google Cloud

- **Cloud Run** hosts the ContextRail workspace (Go binary serving the API and the built React UI) and is the promotion target of the governed demo application (staging and prod-demo services).
- **Cloud Build** + **Artifact Registry** build the image; the deploy script promotes by **digest only**, never by tag (`scripts/deploy-cloud-run.sh`, `cloudbuild.yaml`).
- **Gemini** is the optional decision advisor (`GEMINI_API_KEY`): it proposes candidate options and unknowns; it never makes the decision and its output is marked UNVERIFIED.
- `scripts/record-promotion.sh` reads the live Cloud Run revision and digest with `gcloud`, runs the smoke check and records the observation with an idempotency key, so the receipt reflects what actually landed.

## Quick start (local)

```sh
cd frontend && npm ci && npm run build && cd ..
go build -o bin/context-rail ./cmd/context-rail
PORT=8080 CONTEXT_RAIL_FIXTURE_ROOTS=demo/order-operations-portal,examples/support-insights \
  CONTEXT_RAIL_STATIC_DIR=frontend/dist CONTEXT_RAIL_STATE_DIR=.context-rail-state ./bin/context-rail
# open http://127.0.0.1:8080/projects/order-operations-portal
```

Tests: `go test ./...`, `cd frontend && npm test`, and the browser journeys `cd frontend && npm run test:e2e` against a running server (`BASE_URL`, `PLAYWRIGHT_EXECUTABLE_PATH`). Every journey re-hashes the consumer fixtures afterwards: the governed Projects are read, never written.

Cloud Run: `scripts/deploy-cloud-run.sh <gcp-project-id>` — see [docs/operations/CLOUD_RUN_BASELINE.md](docs/operations/CLOUD_RUN_BASELINE.md) for the container contract, environment variables and the smoke check.

## Repository layout

```
cmd/context-rail/        HTTP entry point (Cloud Run container contract, graceful shutdown)
internal/projectregistry read-only Project Context Contract import
internal/topology        versioned environment topology
internal/document        document baseline + Context Pack rebuild
internal/change          change decision pack, advisor, work order, candidate gate, GitHub read-back
internal/release         release bundle, approval, deployment verification, receipts, prod-demo promotion
internal/http            HTTP handlers, static SPA, health
frontend/                React + Vite workspace (zh-TW / en)
tests/                   Go domain + HTTP tests, Playwright browser journeys
demo/, examples/         governed consumer Projects (fixtures, never mutated)
decisions/, evidence/    DecisionRecords (DR-nnn) and Evidence Bundles (EB-nnn) for every slice of ContextRail itself
docs/                    architecture, design, planning, operations
template/                the reusable Project Context Contract template
```

ContextRail is built the way it asks its users to build: every slice has a DecisionRecord and an Evidence Bundle with test output, browser output, fixture-mutation checks and screenshots.

## Boundaries (P0)

- Production is represented as a protected, read-only environment; releases execute only for staging and an isolated prod-demo.
- Governance state is JSON under `CONTEXT_RAIL_STATE_DIR` (instance-local on Cloud Run); Firestore is a later step.
- The `X-ContextRail-Actor` header is recorded, not authenticated.
- ContextRail verifies what was deployed; it does not run deployments itself in P0.
- Server-generated gate details stay English; static workspace copy is localised.

## Documents

- [Vision](vision.md) · [Roadmap](roadmap.md) · [Architecture](architecture.md)
- [Project Context Contract](docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md) · [UI flow and state contract](docs/design/CONTEXT_RAIL_UI_FLOW_STATE_CONTRACT.zh-TW.md)
- [Cloud Run baseline runbook](docs/operations/CLOUD_RUN_BASELINE.md) · [Real run playbook (Claude Code + GitHub read-back)](docs/operations/REAL_RUN_PLAYBOOK.md)
- [Demo video script](docs/submission/VIDEO_SCRIPT.md) · [Deck outline](docs/submission/DECK_OUTLINE.md)
- Template: [Project Context Contract template](template/README.md) · Example Project: [Order Operations Portal](demo/order-operations-portal/README.md) (fixture) — real governed repository: [williamlabdev/order-operations-portal](https://github.com/williamlabdev/order-operations-portal)
