# Implementation Plan: Project Registry 唯讀匯入

**Branch**: `codex/001-manual-order-review` (current checkout) | **Date**: 2026-09-14 | **Spec**: [spec.md](spec.md)

**Input**: [REQ-002](../../requests/REQ-002-project-registry-import.md), [DR-002](../../decisions/DR-002-project-registry-import.json), [vertical slice map](../../docs/planning/CONTEXT_RAIL_VERTICAL_SLICES_V1.zh-TW.md)

## Summary

Implement the first ContextRail core read boundary as a read-only Project Registry operation. The operation will consume explicitly supplied Project roots, validate the Project manifest, normalize the declared relationships and readiness inspection into a versioned snapshot, and expose the result without writing to the consumer Project. The existing Python importer remains the contract oracle and regression reference while the product runtime follows the Go modular-monolith direction.

Implementation is not authorized until a human accepts `DR-002`. This plan is the technical candidate that follows the Request; it does not itself grant a Work Order, repository write scope or deployment permission.

## Technical Context

**Language/Version**: Go 1.22 or newer, subject to the repository's toolchain check

**Primary Dependencies**: Go standard library where practical; a pinned YAML parser may be introduced only if required by the manifest contract and recorded in `go.mod`/`go.sum`

**Storage**: No persistence in this slice; read from explicitly supplied filesystem Project roots

**Testing**: Go unit/contract tests, `go vet ./...`, `go test ./...`, build verification, fixture comparison with `template/.venv/bin/python` and before/after source hash checks

**Target Platform**: Local POSIX development first; no Cloud Run target in this slice

**Project Type**: Go modular-monolith read-only API/adapter boundary

**Performance Goals**: Not a production performance claim; one local snapshot of the two checked-in fixtures should complete in the documented verification run

**Constraints**: Read-only; no secrets; no provider crawling; no auth; no Firestore; no RAG; no Context Pack rebuild; no deployment; preserve unknown and stale states

**Scale/Scope**: One or more explicitly supplied Project roots; two checked-in fixtures for the first contract test; no enterprise inventory claim

## Constitution Check

- [x] Contract before execution: REQ-002, DR-002 and this Spec Kit chain are linked.
- [x] Provenance over assertion: source root, observed time and source commit are retained; uncertain states are not upgraded.
- [x] Bounded and reversible change: implementation paths are limited by DR-002; the consumer Project is read-only.
- [x] Deterministic gates and human authority: tests can block; no inspection result approves a Project or deployment.
- [x] Evidence is part of the deliverable: fixture comparison, test/build output and mutation check are required.
- [x] Structured handoff: an AgentWorkOrder is not issued until DR-002 is accepted.

**Gate status**: PASS for planning; implementation remains `PENDING_HUMAN_DECISION`.

## Existing contract and adapter boundary

```text
explicit Project root
        ↓
manifest adapter ────────┐
        ↓                 │
readiness inspector       │
        └──────→ normalized ProjectRecord
                         ↓
              ProjectRegistrySnapshot v1
```

The adapter must distinguish these concerns:

1. Manifest validation: required `kind`, metadata and supported Project declarations.
2. Relationship normalization: repository, singular/plural service and environment records.
3. Readiness observation: reuse the semantics already exercised by the Python oracle, including missing/stale/unknown states.
4. Snapshot serialization: stable field names and explicit schema version.

The implementation must not turn the Python script into an internal subprocess dependency. The oracle is used by tests or comparison tooling; the product boundary should own its normalized contract.

## Proposed project structure

```text
cmd/context-rail/
└── main.go                         # local/API entry point, if API exposure is selected

internal/projectregistry/
├── model.go                        # normalized snapshot contract
├── manifest.go                     # read-only manifest adapter
├── readiness.go                    # observation mapping
└── registry.go                     # import orchestration

tests/registry/
├── contract_test.go                # snapshot and status contract
└── mutation_test.go                # source hash remains unchanged

docs/validation/
└── CONTEXT_RAIL_PROJECT_REGISTRY_*.zh-TW.md

evidence/EB-002/
└── README.md                       # checks and source identity
```

**Structure Decision**: Keep a small Go package boundary inside the planned modular monolith. Do not create microservices, a frontend, a database layer or a provider SDK for VS-001. The HTTP route may be included only if it does not expand the read-only contract; a local CLI/adapter is an acceptable first delivery for the contract.

## Contract decisions

- Top-level snapshot kind: `ProjectRegistrySnapshot`.
- Schema version: `project-registry/v1`.
- Each Project record includes an observed root and observation time.
- `repository` and `services` input forms are normalized to arrays; source role and source path are retained.
- Context Pack status is observed from the declared/derived document state; it is never recalculated into a stronger status by the registry.
- Errors for missing or malformed manifests are explicit and must include the input root; a valid sibling Project must not be silently omitted.
- Inspection is not discovery: no ownership confirmation, membership grant, CRUD mutation or deployment inference.

## Implementation sequence

1. Freeze the normalized Go contract from the Python oracle and the two consumer fixtures.
2. Add contract tests for valid, alternate-shape, stale/undeclared and malformed inputs.
3. Implement the read-only manifest and readiness adapters.
4. Add the local/API entry point only after the package contract is stable.
5. Run tests, vet and build; compare outputs and verify zero source mutations.
6. Record EvidenceBundle and request human review before any later Work Order or staging action.

## Complexity Tracking

No constitution violation is proposed. A YAML dependency is conditional and must be justified by the consumer manifest contract; using a subprocess, database or provider SDK would be a larger boundary and is explicitly rejected for this slice.

## ContextRail handoff

```text
REQ-002 → spec.md → DR-002 → plan.md/tasks.md → AWO-002 (future) → ARR-002 → EB-002 → review
```

`DR-002` is the governance SSOT. If the Request, source contract or policy changes, this plan and tasks become stale and require reconciliation before implementation.
