---

description: "Task list for the Manual Order Review Change Slice"
---

# Tasks: Manual Order Review

**Input**: Design documents from `specs/001-manual-order-review/`

**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/http.md`, `quickstart.md`

**Status**: Spec Kit-generated execution plan; the checkout contains a brownfield implementation baseline, but tasks are not marked complete without this run's evidence.

## Phase 1: Setup

**Purpose**: Establish the feature execution boundary.

- [x] T001 Confirm the `001-manual-order-review` feature scope in `specs/001-manual-order-review/spec.md`.
- [x] T002 Confirm the selected source structure and constraints in `specs/001-manual-order-review/plan.md`.
- [x] T003 [P] Verify Spec Kit project metadata in `.specify/feature.json`, `.specify/integration.json` and `.specify/init-options.json`.

---

## Phase 2: Foundational

**Purpose**: Confirm the project and governance prerequisites before user-story work.

- [x] T004 Review the non-negotiable principles in `.specify/memory/constitution.md`.
- [x] T005 [P] Confirm `project_id`, `request_id`, `decision_id` and `policy_version` references in `specs/001-manual-order-review/plan.md`.
- [x] T006 [P] Confirm forbidden scope and staging/prod boundary in `decisions/DR-001-manual-order-review.json` and `docs/operations/environments.md`.

**Checkpoint**: Implementation may proceed only after human acceptance, and only within the Change Slice and allowed paths.

---

## Phase 3: User Story 1 - Review an order exception (Priority: P1) 🎯 MVP

**Goal**: Let an operations staff member approve or reject a seeded order exception with a required reason and visible result.

**Independent Test**: Follow `specs/001-manual-order-review/quickstart.md` and verify the list, valid approval, valid rejection, empty-note rejection and unknown-order behavior.

### Contract and behavior tests

- [x] T007 [P] [US1] Add or update health and order-list behavior tests in `main_test.go` for `GET /healthz` and `GET /api/orders`.
- [x] T008 [P] [US1] Add or update review validation tests in `main_test.go` for invalid JSON, unsupported decisions and whitespace-only notes.
- [x] T009 [P] [US1] Add or update state-transition tests in `main_test.go` for valid approval, valid rejection, repeated review and unknown order IDs.

### Implementation

- [x] T010 [US1] Verify the existing synthetic Order and Review Request behavior in `main.go` against `specs/001-manual-order-review/data-model.md`; update only if a documented gap exists.
- [x] T011 [US1] Verify the existing review endpoint in `main.go` against `specs/001-manual-order-review/contracts/http.md`; update only if a documented gap exists.
- [x] T012 [US1] Verify the seeded order list and review outcome in `web/index.html`; update only if a documented gap exists and do not add authentication or persistence.
- [x] T013 [US1] Keep changes within the allowed paths recorded in `decisions/DR-001-manual-order-review.json`.

**Checkpoint**: User Story 1 is independently testable locally; no deployment authorization is implied.

---

## Phase 4: Evidence and review

**Purpose**: Prove the Change Slice against the accepted source and policy.

- [x] T014 [P] Run `go test ./...` and record the exact result in `evidence/EB-001/test-output.txt`.
- [x] T015 [P] Run `go vet ./...` and record the exact result in `evidence/EB-001/vet-output.txt`.
- [x] T016 [P] Run `go build ./...` and record the exact result in `evidence/EB-001/build-output.txt`.
- [x] T017 Run the local handler smoke and negative scenarios from `specs/001-manual-order-review/quickstart.md` and record the result in `evidence/EB-001/local-smoke.txt`; live port smoke remains environment-blocked.
- [x] T018 Reconcile the implementation commit, changed paths, test/build identity and source snapshot hash in `evidence/EB-001/README.md`.
- [x] T019 Complete independent review in `evidence/EB-001/code-review.md`; the implementation agent cannot be the sole reviewer.

---

## Phase 5: Staging gate

**Purpose**: Keep cloud deployment separate from local implementation success.

- [ ] T020 Confirm authorized `GCP_PROJECT_ID`, `CLOUD_RUN_REGION`, `IMAGE_URI` and deploy identity before changing `evidence/EB-001/cloud-run-staging-receipt.json`.
- [ ] T021 Deploy only the reviewed image to the Cloud Run staging target using `scripts/deploy-staging.sh`.
- [ ] T022 Read back the Cloud Run revision and execute the staging smoke check; record the receipt in `evidence/EB-001/cloud-run-staging-receipt.json`.
- [ ] T023 Obtain human staging approval; keep production status `BLOCKED_IN_DEMO` in `evidence/EB-001/cloud-run-staging-receipt.json`.

---

## Dependencies & Execution Order

### Phase dependencies

- Setup (Phase 1) precedes Foundational (Phase 2).
- Foundational (Phase 2) blocks User Story 1 implementation.
- User Story 1 must pass before Evidence and Review (Phase 4).
- Phase 4 must pass before the Staging Gate (Phase 5).
- Phase 5 never authorizes production for this demo.

### Parallel opportunities

- T003, T005 and T006 can be inspected in parallel.
- T007, T008 and T009 can be prepared in parallel because they touch the same test contract but cover independent behaviors; merge them before implementation.
- T014, T015 and T016 can run in parallel after the implementation commit.

## Implementation strategy

1. Complete Setup and Foundational phases.
2. Deliver User Story 1 as the only MVP slice.
3. Validate locally and obtain independent review.
4. Stop at any missing-evidence gate; do not infer staging readiness.
5. Deploy to staging only after the required configuration and human gate are present.

## Traceability

`FR-001` → T007, T010; `FR-002` → T008, T011; `FR-003` → T008, T011; `FR-004` → T009, T011, T012; `FR-005` → T009, T011; `FR-006` → T012; `FR-007` → T010, T013; `FR-008` → T009, T011.

## Phase 6: Convergence

**Purpose**: Close evidence gaps found after the brownfield implementation verification.

- [x] T024 Rebuild the canonical Context Pack and replace `sha256:pending-context-build` in `decisions/DR-001-manual-order-review.json` with the observed source snapshot hash per `plan: source snapshot` (partial).
- [x] T025 Obtain an independent reviewer identity, decision, timestamp and reviewed commit in `evidence/EB-001/code-review.md` per `SC-004`.
- [ ] T026 After T025 and authorized cloud configuration are available, deploy the reviewed image and record the Cloud Run staging revision in `evidence/EB-001/cloud-run-staging-receipt.json` per `REQ-001/AC-006` (missing).
