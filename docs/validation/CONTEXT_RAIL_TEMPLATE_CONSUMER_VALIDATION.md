# Project Context Template Consumer Validation

Date: 2026-09-13
Status: `PARTIAL_PASS`

## Purpose

Verify that the Project Context Contract template can be consumed by Projects with different runtimes and environment shapes.

## Consumers

| Consumer | Runtime | Environment shape | Context validation | Runtime checks |
| --- | --- | --- | --- | --- |
| `demo/order-operations-portal` | Go HTTP service | Development → Testing → Staging → Production | `PROJECT CONTEXT STRICT PASS` | Go tests, vet, build, health/review smoke PASS |
| `examples/support-insights` | Python report generator | Development → Staging → Production | `PROJECT CONTEXT STRICT PASS` | Python test, compileall, report output PASS |

## Commands observed

```sh
template/.venv/bin/python template/scripts/rebuild_context_pack.py --root demo/order-operations-portal
template/.venv/bin/python template/scripts/validate_project_context.py --root demo/order-operations-portal --strict
template/.venv/bin/python template/scripts/rebuild_context_pack.py --root examples/support-insights
template/.venv/bin/python template/scripts/validate_project_context.py --root examples/support-insights --strict
```

## Findings

- The required document registry and derived Context Pack shape work across Go and Python consumers.
- The template can represent four environments or a smaller three-environment topology.
- Environment sequence uniqueness, declared document existence and source hash lineage are validated.
- Template usage still requires each Project to define its own runtime commands, owners, target references and policy.

## Not yet proven

- A second human has not yet initialized a Project from the template.
- No real GitHub private remote has been connected.
- No real Cloud Run staging deployment receipt exists; current receipts are explicit `NEEDS_INPUT` or fixture evidence.
- ContextRail UI/API import and readiness rendering have not yet consumed both Projects.

The next gate is a human walkthrough of both consumer READMEs, followed by a read-only ContextRail import/readiness slice.

## Read-only import/readiness slice

`scripts/context_rail_readiness.py` now reads one or more Project roots and emits a JSON or summary report. It does not mutate Project files. It conservatively separates context readiness from human acceptance, engineering evidence and production permission.

Observed results:

- `order-operations-portal`: decision context `READY`; local development `NEEDS_INPUT` because the DecisionRecord still has a pending human decision; cloud testing `NEEDS_INPUT` because independent review is still required; staging `NEEDS_INPUT` because the local, testing and Cloud Run receipt gates are incomplete; production `BLOCKED`.
- `support-insights`: decision context `READY`; local development, cloud testing and staging are `NEEDS_INPUT` because no accepted DecisionRecord or Evidence Bundle exists; production `BLOCKED`.

The readiness names intentionally separate local development from deployment:

- `ready_for_local_development` does not require Cloud Run credentials or a staging receipt.
- `ready_for_cloud_testing` evaluates test, build and independent review evidence for the testing/CI boundary; it does not mean that a Cloud Run service has been deployed.
- `ready_for_staging` is the Cloud Run pre-deployment gate and does not require a receipt that cannot exist before the first deploy.
- `staging_verified` is the post-deployment state and requires a matching PASS staging receipt, review, test/build evidence and the upstream deployment gate.

The inspector also detects a changed source document as `STALE` by comparing the derived Context Pack hash with the current file.
