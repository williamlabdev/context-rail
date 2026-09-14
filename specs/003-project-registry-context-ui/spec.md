# Feature Specification: Project Registry／Documents & AI Context 唯讀工作區

**Feature Branch**: `codex/001-manual-order-review` (current checkout)

**Spec Kit Feature**: `003-project-registry-context-ui`

**Created**: 2026-09-14

**Status**: Accepted for local development; ready for `AWO-003`

**Input**: [REQ-003](../../requests/REQ-003-project-registry-context-ui.md) and [DR-003](../../decisions/DR-003-project-registry-context-ui.json)

## Change Slice

Provide the first user-visible ContextRail workspace. A Project owner or Solution Architect can open a Project, understand its declared sources and readiness, and identify what is current, stale, missing or unknown. The workspace is read-only and consumes the normalized Project Registry contract from VS-001.

The slice includes the service boundary needed by the workspace and the browser journey needed to validate it. It does not add Project management writes, authentication, persistence or deployment.

## User Scenarios & Testing

### User Story 1 - Browse and understand a Project context (Priority: P1)

As a Project owner or Solution Architect, I want to open a Project and see its repositories, services, environments, documents and readiness so that I can judge whether the Project has enough context for a governed action.

**Why this priority**: This is the first complete product journey beyond a raw contract. It turns the Project Context Contract into something a person can inspect and validate.

**Independent Test**: Start the local workspace with the checked-in consumer fixtures, open the Project Registry, select each Project, and verify that the detail view displays the declared relationships, documents, context status and readiness reasons.

**Acceptance Scenarios**:

1. **Given** two valid consumer Projects, **when** the owner opens the registry, **then** both Projects are shown with a stable identity and enough summary information to choose one.
2. **Given** a selected Project, **when** its detail view loads, **then** repositories, services, environments, documents, decisions, context status and readiness reasons are visible.
3. **Given** a Project with current documents and a derived Context Pack, **when** the owner opens its context view, **then** source documents and derived context are distinguishable and the source path/version information is visible.

### User Story 2 - Understand incomplete or uncertain context (Priority: P1)

As a Project owner, I want incomplete, stale or uncertain context to be obvious so that I do not mistake an incomplete Project for one that is ready or approved.

**Why this priority**: ContextRail's value depends on refusing to hide uncertainty. A visually polished page that turns stale or missing evidence into success would be unsafe.

**Independent Test**: Open the support-insights fixture and any fixture with an undeclared value, then confirm the corresponding statuses and readiness reasons remain visible and are not rendered as a positive state.

**Acceptance Scenarios**:

1. **Given** a stale Context Pack, **when** the owner views the Project, **then** the page labels it `STALE` and explains the observed source drift or missing update.
2. **Given** a missing required document, **when** the owner views the Documents / AI Context section, **then** it labels the document `MISSING` and does not display the Project as fully ready.
3. **Given** an undeclared runtime or unavailable observation, **when** the owner views the service or readiness section, **then** it displays `UNDECLARED` or `UNKNOWN` and does not replace it with an inferred value.

### User Story 3 - Handle read-only loading and failure states (Priority: P2)

As a Project owner, I want the workspace to explain loading, empty and failure states so that I know whether I should wait, select another Project or provide missing input.

**Why this priority**: A governance UI must make unavailable evidence visible instead of presenting a blank or misleading success state.

**Independent Test**: Exercise the workspace with a delayed response, an empty registry and an invalid/unavailable Project response, and confirm each state has an explicit message and no write action.

**Acceptance Scenarios**:

1. **Given** a registry request in progress, **when** the user waits for the response, **then** the workspace shows a loading state without stale content presented as current.
2. **Given** no Projects are available, **when** the registry loads, **then** the workspace shows an empty state with a clear next step that does not create a Project.
3. **Given** an invalid or unavailable Project response, **when** the user opens it, **then** the workspace shows an error state with the source problem and no mutation control.

### Edge Cases

- A Project with no current documents remains inspectable, but its readiness reason must explain the missing or stale inputs.
- A Project with no services or environments must not crash or render a different Project's records.
- A source status such as `CONFLICT` is not equivalent to `CURRENT`, `PASS` or approval.
- Switching between Projects must replace all detail data; no records may leak from the previously selected Project.
- Refreshing or revisiting the workspace is read-only and must not rebuild Context Pack, confirm ownership or create an audit event.
- The workspace must not display secrets, credential values or unobserved provider data.

## Requirements

### Functional Requirements

- **FR-001**: The workspace MUST present an explicitly supplied Project Registry as a list or selection surface.
- **FR-002**: The workspace MUST present the selected Project's identity, repositories, services, environments, documents, decisions, context status and readiness reasons.
- **FR-003**: The workspace MUST distinguish source documents from derived Context Pack content and show source path/version information when available.
- **FR-004**: The workspace MUST visibly preserve `CURRENT`, `STALE`, `MISSING`, `CONFLICT`, `UNKNOWN` and `UNDECLARED` states without mapping uncertain states to success or approval.
- **FR-005**: The workspace MUST present explicit loading, empty, invalid/unavailable and error states.
- **FR-006**: The workspace MUST keep registry and context operations read-only; no view action may modify a consumer Project or governance artifact.
- **FR-007**: The service boundary MUST return independently identified Project records and must not mix detail data when the user switches Projects.
- **FR-008**: The service boundary MUST preserve the observed source root, observation time and readiness reason needed for provenance display.
- **FR-009**: The workspace MUST NOT expose secrets or infer ownership, authorization, deployment readiness or production approval from the displayed data.
- **FR-010**: The browser journey MUST be reproducible against both checked-in consumer fixtures without Cloud Run or a paid external connector.

### Key Entities

- **Project Registry Entry**: A selectable Project identity and summary in the registry.
- **Project Context View**: The selected Project's relationships, documents, derived context and readiness information.
- **Document Status**: The observed state and provenance of a source document.
- **Readiness Status**: A derived explanation of whether required inputs for a defined action are available; it is not authorization.
- **Workspace State**: Loading, populated, empty or error state for registry and Project detail views.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A user can open the registry, select a Project and reach its context view in one documented browser journey with no undocumented setup step.
- **SC-002**: Both checked-in consumer Projects are independently selectable, and switching between them shows zero cross-Project record leakage in the browser verification.
- **SC-003**: All six governed source states (`CURRENT`, `STALE`, `MISSING`, `CONFLICT`, `UNKNOWN`, `UNDECLARED`) are represented by distinct, non-success UI states when supplied by fixtures.
- **SC-004**: Loading, empty and error scenarios each produce an explicit user-visible state and no mutation request.
- **SC-005**: A reviewer can trace every displayed document/context status to a source path, version or explicit unavailable value in the evidence bundle.
- **SC-006**: A before/after hash comparison shows zero mutations to consumer Projects and governance artifacts after the complete browser journey.

## Assumptions and Boundaries

- The workspace consumes the VS-001 normalized read contract; it does not redefine Project or document semantics.
- Local fixture-backed execution is sufficient for this slice; Cloud Run is a later environment slice.
- The initial UI can be a focused workspace rather than a complete navigation shell, provided the registry-to-detail journey is complete.
- Project creation, editing, archiving, authentication, membership, persistence, uploads, RAG, Gemini, agent execution and deployment remain out of scope.
- `DR-003` is accepted for local development/testing only. VS-001 implementation/evidence has been human-reviewed and accepted for local development. VS-002 implementation still requires the bounded `AWO-003`; this specification does not grant environment authorization.
