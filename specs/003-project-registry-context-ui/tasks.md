---
description: "Task list for the Project Registry and Documents AI Context read-only workspace"
---

# Tasks: Project Registry／Documents & AI Context 唯讀工作區

**Input**: Design documents from `specs/003-project-registry-context-ui/`

**Prerequisites**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [API contract](contracts/api.md), [UI contract](contracts/ui.md), [REQ-003](../../requests/REQ-003-project-registry-context-ui.md), [DR-003](../../decisions/DR-003-project-registry-context-ui.json)

**Status**: Planned; implementation is blocked until `DR-003` receives a human decision and `AWO-003` is issued.

## Phase 1: Setup

**Purpose**: Establish the full-stack workspace without changing consumer Projects.

- [ ] T001 Review `vision.md`, `roadmap.md`, `architecture.md`, `docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md` and the VS-001 artifacts before implementation.
- [ ] T002 Obtain and record the human decision for `decisions/DR-003-project-registry-context-ui.json`.
- [ ] T003 Create `work-orders/AWO-003-project-registry-context-ui.json` only after T002, with paths and checks limited to `DR-003`.
- [ ] T004 [P] Record the current source commit, fixture roots and local tool versions in `evidence/EB-003/README.md`.
- [ ] T005 [P] Create the root Go module in `go.mod` and `go.sum` only if VS-001 has not already established them; pin only required dependencies.
- [ ] T006 [P] Initialize the frontend package in `frontend/package.json` and `frontend/package-lock.json` with pinned React/TypeScript/Vite and browser-test dependencies.

**Checkpoint**: No implementation file may be changed before T002 is accepted and T003 is issued.

## Phase 2: Foundational contracts

**Purpose**: Freeze the shared backend/frontend boundary before story work.

- [ ] T007 [P] [US1] Add Go HTTP contract tests for `GET /v1/projects` and `GET /v1/projects/{project_id}` in `tests/http/projects_test.go` using the API contract in `contracts/api.md`.
- [ ] T008 [P] [US1] Add frontend API type and parsing tests in `frontend/tests/api/projectRegistry.test.ts` for `ProjectRegistrySnapshot` and `ProjectRegistryError`.
- [ ] T009 [P] [US2] Add contract assertions in `tests/registry/contract_test.go` that `STALE`, `MISSING`, `CONFLICT`, `UNKNOWN` and `UNDECLARED` survive the VS-001-to-API boundary.
- [ ] T010 [P] [US3] Add service error tests in `tests/http/projects_test.go` for `PROJECT_NOT_FOUND`, `PROJECT_INVALID`, `PROJECT_UNAVAILABLE` and `REGISTRY_UNAVAILABLE`.
- [ ] T011 [P] [US3] Add frontend workspace-state tests in `frontend/tests/state/workspaceState.test.ts` for `LOADING`, `READY`, `EMPTY` and `ERROR`.
- [ ] T012 [US1] Reuse the VS-001 normalized Project model in `internal/projectregistry/` and document any required contract adapter without creating a second semantic model.
- [ ] T013 [US1] Implement safe local fixture configuration for the two explicit roots in `cmd/context-rail/` without arbitrary filesystem traversal or remote URL crawling.

**Checkpoint**: Contract tests define the read-only boundary and fail for missing API/UI behavior; no user story may add mutation routes.

## Phase 3: User Story 1 - Browse and understand a Project context (Priority: P1) 🎯 MVP

**Goal**: Let a Project owner select a Project and see registry, Documents / AI Context and readiness information in a browser.

**Independent Test**: Run the local service with both fixture roots, open the registry, select each Project, and verify identity, relationships, documents, context and readiness are displayed without cross-Project leakage.

### Backend/API implementation

- [ ] T014 [US1] Implement the Project list and detail handlers in `internal/http/projects.go` for the exact read routes in `specs/003-project-registry-context-ui/contracts/api.md`.
- [ ] T015 [US1] Implement safe JSON error responses in `internal/http/errors.go` without secret or credential values.
- [ ] T016 [US1] Wire the read-only service and static frontend host in `cmd/context-rail/main.go`; expose no write route.

### Frontend implementation

- [ ] T017 [P] [US1] Implement the registry fetch and Project selection client in `frontend/src/api/projectRegistry.ts`.
- [ ] T018 [P] [US1] Implement the registry list and selection surface in `frontend/src/components/ProjectRegistry.tsx`.
- [ ] T019 [US1] Implement the selected Project context page in `frontend/src/pages/ProjectContextPage.tsx` with repositories, services, environments, decisions, documents, context and readiness sections.
- [ ] T020 [P] [US1] Implement source-versus-derived document presentation in `frontend/src/components/DocumentStatusList.tsx` with path/version fields.
- [ ] T021 [P] [US1] Implement the non-authoritative status presentation in `frontend/src/components/ReadinessSummary.tsx` and `frontend/src/components/StatusBadge.tsx`.
- [ ] T022 [US1] Compose the registry-to-detail journey in `frontend/src/App.tsx` and keep all interactions read-only.
- [ ] T023 [P] [US1] Add focused layout and status styles in `frontend/src/styles/` without adding a generic design system.

### Verification

- [ ] T024 [P] [US1] Add frontend component/integration tests in `frontend/tests/components/projectContext.test.tsx` for list, selection, detail rendering and Project switching.
- [ ] T025 [US1] Add the positive browser journey in `tests/browser/project-context.spec.ts` covering both fixture-backed Projects and zero cross-Project leakage.

**Checkpoint**: The registry-to-detail browser journey is independently usable; it does not imply Project readiness, approval or deployment authorization.

## Phase 4: User Story 2 - Understand incomplete or uncertain context (Priority: P1)

**Goal**: Keep stale, missing, conflict, unknown and undeclared context visibly distinct from success.

**Independent Test**: Open fixtures that contain stale Context Pack/source drift and undeclared values, then verify the exact states and readiness reasons in the browser.

- [ ] T026 [US2] Preserve document/context/readiness status and reason fields through `internal/projectregistry/` and `internal/http/projects.go` without upgrading uncertain values.
- [ ] T027 [P] [US2] Add status fixtures and API assertions in `tests/registry/contract_test.go` for `STALE`, `MISSING`, `CONFLICT`, `UNKNOWN` and `UNDECLARED`.
- [ ] T028 [P] [US2] Add frontend status rendering tests in `frontend/tests/components/statusRendering.test.tsx` proving uncertain states are not rendered as `CURRENT` or `PASS`.
- [ ] T029 [US2] Add readiness reason and source provenance rendering to `frontend/src/components/ReadinessSummary.tsx` and `frontend/src/components/DocumentStatusList.tsx`.
- [ ] T030 [US2] Add the uncertain-context browser journey to `tests/browser/project-context.spec.ts` and record the observed labels/reasons.

**Checkpoint**: Negative and uncertain states remain visible and non-authoritative.

## Phase 5: User Story 3 - Handle read-only loading and failure states (Priority: P2)

**Goal**: Explain loading, empty and failure conditions without displaying stale data as current or offering mutation controls.

**Independent Test**: Run delayed, empty and invalid/unavailable fixture configurations and verify explicit UI states with no write requests.

- [ ] T031 [P] [US3] Add controllable delay, empty and invalid/unavailable local fixtures in `cmd/context-rail/` and `tests/http/projects_test.go` without changing consumer sources.
- [ ] T032 [US3] Implement loading, empty and error transitions in `frontend/src/App.tsx` and `frontend/src/components/WorkspaceState.tsx`.
- [ ] T033 [P] [US3] Add component tests in `frontend/tests/components/workspaceState.test.tsx` for loading, empty and error messages.
- [ ] T034 [US3] Add delayed, empty and error browser journeys in `tests/browser/project-context.spec.ts` and assert that no mutation request is issued.
- [ ] T035 [US3] Verify the UI contains no create, edit, archive, upload, approve or deploy control in `frontend/src/`.

**Checkpoint**: All required UI state families are explicit and read-only.

## Phase 6: Polish and evidence

- [ ] T036 [P] Run Go contract/unit tests and record exact output in `evidence/EB-003/go-test-output.txt`.
- [ ] T037 [P] Run `go vet ./...` and `go build ./...`; record exact output in `evidence/EB-003/go-quality-output.txt`.
- [ ] T038 [P] Run frontend typecheck, unit/integration tests and production build; record exact output in `evidence/EB-003/frontend-quality-output.txt`.
- [ ] T039 [P] Run the Playwright browser journeys and record exact output plus browser evidence in `evidence/EB-003/browser-output.txt`.
- [ ] T040 Compare Go/API output with the VS-001 Python oracle and record limitations in `docs/validation/CONTEXT_RAIL_PROJECT_REGISTRY_CONTEXT_UI_2026-09-14.zh-TW.md`.
- [ ] T041 Recompute before/after hashes for both consumer Projects and governance artifacts; record zero-mutation result in `evidence/EB-003/README.md`.
- [ ] T042 Prepare review evidence in `evidence/EB-003/README.md` and keep staging/production status out of this local slice.

## Dependencies and execution order

- Phase 1 blocks all implementation phases; T002 and T003 are mandatory governance gates.
- Phase 2 freezes the shared API/UI contract before story implementation.
- User Story 1 depends on VS-001 and the Phase 2 contracts; it is the MVP.
- User Story 2 depends on the read path from User Story 1 but can add status coverage independently once the boundary exists.
- User Story 3 depends on the application shell from User Story 1 and can then be tested independently with controlled local states.
- Phase 6 depends on all selected stories and must finish before review preparation.

## Parallel opportunities

- T004–T006 can run in parallel after the human gate and Work Order exist.
- T007–T011 can be prepared in parallel because they touch separate contract/test concerns.
- T017, T018, T020, T021 and T023 can be prepared in parallel after the API shape is fixed.
- T024 and T025 can proceed after the US1 components and local service are available.
- T027 and T028 can proceed in parallel for the backend and frontend status assertions.
- T036–T039 can run in parallel after implementation is complete.

## Traceability

- `FR-001` → T013, T014, T017, T025
- `FR-002` → T014, T019, T024
- `FR-003` → T020, T024, T025
- `FR-004` → T009, T021, T027, T028, T029, T030
- `FR-005` → T010, T011, T032, T033, T034
- `FR-006` → T016, T022, T035
- `FR-007` → T012, T014, T024, T025
- `FR-008` → T026, T029, T040
- `FR-009` → T015, T021, T035
- `FR-010` → T025, T030, T034, T039

## Implementation strategy

1. Obtain human acceptance and issue AWO-003.
2. Complete shared contracts and write failing tests.
3. Deliver User Story 1 as the MVP full-stack journey.
4. Add uncertain-state handling and verify it independently.
5. Add loading/empty/error handling and verify no mutation.
6. Run all local evidence checks, then stop for review; no Cloud Run action is implied.
