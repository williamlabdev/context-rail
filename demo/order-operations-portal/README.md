# Order Operations Portal

`Order Operations Portal` is the governed demo Project for ContextRail. It is a small synthetic order-exception application whose first Change is **訂單人工覆核** (manual order review).

The demo is intentionally split into two parts:

- this repository is the Project being governed;
- ContextRail is the control plane that creates the Context, evaluates the Request, records the decision, checks engineering evidence and controls staging promotion.

## Demo path

1. Create the Project from `project.yaml`.
2. Validate the eight source documents and inspect `docs/ai/context-pack.json`.
3. Submit `requests/REQ-001-manual-order-review.md`.
4. Review `decisions/DR-001-manual-order-review.json` and its technical/non-technical summaries.
5. Start a bounded implementation from the Agent Context Pack.
6. Run tests and record the independent review evidence.
7. Deploy the approved image to Cloud Run staging and write a deploy receipt.
8. Show the production gate as `BLOCKED` because this demo never goes live.

## Run locally

```sh
./scripts/validate-demo.sh
go run .
```

Open <http://localhost:8080>. All order data is synthetic. The API has no login, persistence, payment action or production credentials and must not be used as a real order system.

## Repository status

The manifest models the eventual source as one GitHub private repository. The URL is a fixture placeholder until a real private remote is created; `fixture-pending-remote` is deliberate and is not GitHub evidence.

## Evidence

- `docs/ai/context-pack.json` — derived context with source hashes.
- `requests/` — original Request and acceptance criteria.
- `decisions/` — canonical DecisionRecord and two audience views.
- `work-orders/` and `runs/` — bounded handoff and honest Agent Run Record status.
- `evidence/EB-001/` — test, review, build and Cloud Run receipt evidence.
- `scripts/check-staging-gate.py` — read-only pre-deploy validation of DecisionRecord, Context Pack, review identity and commit.
- `scripts/deploy-staging.sh` — real `gcloud run deploy` path with immutable image and tagged-revision smoke; without required evidence or GCP access it writes a `NEEDS_INPUT` receipt rather than claiming deployment.
