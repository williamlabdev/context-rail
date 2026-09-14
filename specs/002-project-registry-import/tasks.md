---
description: "Task list for the Project Registry read-only import Change Slice"
---

# Tasks: Project Registry 唯讀匯入

**Input**: Design documents from `specs/002-project-registry-import/`

**Prerequisites**: [spec.md](spec.md), [plan.md](plan.md), [REQ-002](../../requests/REQ-002-project-registry-import.md), [DR-002](../../decisions/DR-002-project-registry-import.json)

**Status**: `DR-002` accepted and `AWO-002` issued for local development; implementation may proceed within the bounded paths.

## Phase 1: Governance and contract setup

**Purpose**: Establish the bounded execution context before implementation work.

- [x] T001 Review `vision.md`, `roadmap.md`, `architecture.md` and `docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md` against `specs/002-project-registry-import/spec.md`.
- [x] T002 Obtain and record the human decision for `decisions/DR-002-project-registry-import.json`.
- [x] T003 [P] Confirm the current source commit and the two fixture roots in `evidence/EB-002/README.md`.
- [x] T004 [P] Create an `AWO-002` only after T002, with paths and checks limited to `DR-002`. Issued as `work-orders/AWO-002-project-registry-import.json`.

**Checkpoint**: No implementation file may be changed before T002 is accepted and the work order is issued.

## Phase 2: Contract tests (P1 MVP)

**Goal**: Prove the normalized read-only snapshot contract independently of a cloud runtime.

**Independent Test**: Run the registry contract tests against `demo/order-operations-portal` and `examples/support-insights`, then compare the relevant fields with the existing Python oracle.

- [x] T005 [P] [US1] Add a valid-fixture contract test in `tests/registry/contract_test.go`.
- [x] T006 [P] [US1] Add singular/plural service normalization coverage in `tests/registry/contract_test.go`.
- [x] T007 [P] [US1] Add stale, missing, unknown and undeclared status coverage in `tests/registry/contract_test.go`.
- [x] T008 [P] [US1] Add malformed/missing manifest error coverage in `tests/registry/contract_test.go`.
- [x] T009 [P] [US1] Add source hash immutability coverage in `tests/registry/mutation_test.go`.

**Checkpoint**: Tests define the accepted contract and fail for an unimplemented registry package.

## Phase 3: Read-only registry implementation

- [x] T010 [US1] Define the versioned snapshot and normalized Project models in `internal/projectregistry/model.go`.
- [x] T011 [US1] Implement read-only manifest validation and relationship normalization in `internal/projectregistry/manifest.go`.
- [x] T012 [US1] Implement readiness/document status mapping in `internal/projectregistry/readiness.go`.
- [x] T013 [US1] Implement multi-root import orchestration and explicit error handling in `internal/projectregistry/registry.go`.
- [x] T014 [US1] Add a local/API entry point in `cmd/context-rail/main.go` only if the contract tests show it does not expand scope.
- [x] T015 [US1] Keep all implementation changes within the paths allowed by `DR-002`.

**Checkpoint**: Both fixtures return independently identified snapshots and no consumer source is modified.

## Phase 4: Verification and evidence

- [x] T016 [P] Run the documented contract tests and record exact output in `evidence/EB-002/test-output.txt`.
- [x] T017 [P] Run `go vet ./...` and record exact output in `evidence/EB-002/vet-output.txt`.
- [x] T018 [P] Run `go build ./...` and record exact output in `evidence/EB-002/build-output.txt`.
- [x] T019 Compare the Go snapshot with the Python oracle and record differences or limitations in `docs/validation/CONTEXT_RAIL_PROJECT_REGISTRY_CORE_2026-09-14.zh-TW.md`.
- [x] T020 Recompute source hashes and record the zero-mutation result in `evidence/EB-002/mutation-check.txt` and `evidence/EB-002/README.md`.
- [ ] T021 Prepare review evidence; do not mark staging or production ready from this local slice.

## Dependencies and execution order

- Phase 1 blocks all implementation phases.
- T005–T009 can be prepared in parallel after the governance boundary is accepted, but must be reconciled before implementation.
- T010–T013 are ordered model → adapter → orchestration; T014 depends on the stable package contract.
- Phase 4 depends on the implementation and must complete before any future staging decision.

## Traceability

- `FR-001` → T013, T016
- `FR-002` → T011, T008
- `FR-003` → T010, T005
- `FR-004` → T011, T006
- `FR-005` → T012, T007
- `FR-006` → T010, T019
- `FR-007` → T009, T020
- `FR-008` → T008, T013
- `FR-009` → T013, T021

## Implementation strategy

1. Obtain the human decision and issue a bounded work order.
2. Write contract tests first and verify they fail before implementation.
3. Implement only the read-only package and the smallest local/API entry point.
4. Validate against both fixtures and the Python oracle.
5. Stop at evidence gaps; a passing local test does not authorize Cloud Run or production.
