# Architecture

## Scope

This is a stateless Go HTTP service for a synthetic demo. It serves a small browser UI and a JSON API from one Cloud Run container.

```text
Browser
  │ GET /, GET /api/orders, POST /api/orders/{id}/review
  ▼
Go HTTP service
  ├── in-memory synthetic order fixture
  └── review validation and response
```

## Manual-review change boundary

The Change is limited to the order-review interaction, its HTTP handler and the browser view. It must not modify deployment policy, IAM, payment, fulfillment or external data stores. A real implementation would persist the review and enforce identity outside this demo.

## Failure and safety behavior

- Unknown order IDs return `404`.
- Empty reviewer notes return `400`.
- Production target is represented as read-only in the Project policy.
- The service exposes `/healthz` for staging smoke testing.
- Data is synthetic and resets when the process restarts.

## Deployment identity

The container is designed for Cloud Run. `scripts/deploy-staging.sh` requires an explicit GCP project, region and image reference; it never deploys to the production target. Cloud Run, Artifact Registry and GitHub remain external sources of deployment and source evidence.
