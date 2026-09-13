# Tasks: Manual Order Review

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md)

**Status**: Pilot task plan; implementation baseline already exists in the demo checkout

## Phase 1: Specification and governance

- [x] T001 Record the feature request in `requests/REQ-001-manual-order-review.md`.
- [x] T002 Record the selected stateless option and forbidden scope in `decisions/DR-001-manual-order-review.json`.
- [x] T003 Establish the project principles in `.specify/memory/constitution.md`.
- [x] T004 Define the Change Slice and acceptance scenarios in `spec.md`.

## Phase 2: Plan and contract

- [x] T005 Confirm the existing Go service and embedded UI are the selected structure in `plan.md`.
- [x] T006 Confirm the HTTP behavior and validation requirements against `main.go` and `main_test.go`.
- [x] T007 Define the evidence and handoff references required for implementation and review.

## Phase 3: Implementation baseline

- [x] T008 Verify the existing review endpoint in `main.go` is within the accepted allowed paths.
- [x] T009 Verify the existing UI in `web/index.html` exposes the result.
- [x] T010 Verify tests cover health, listing, required note, valid review and unknown order.

## Phase 4: Evidence and review

- [x] T011 Record local test evidence in `evidence/EB-001/test-output.txt`.
- [x] T012 Record local build evidence in `evidence/EB-001/build-output.txt`.
- [x] T013 Keep independent review evidence separate from the implementation claim.
- [ ] T014 Reconcile evidence to a committed source snapshot and image identity.
- [ ] T015 Obtain authorized Cloud Run staging configuration and replace the pending receipt.
- [ ] T016 Complete human staging approval; production remains blocked for this demo.

## Exit criteria

The pilot is ready for review when T014–T016 are either completed with evidence or explicitly retained as `NEEDS_INPUT`／`BLOCKED`. No unchecked task may be silently treated as complete.
