# Feature Specification: Project Registry 唯讀匯入

**Feature Branch**: `codex/001-manual-order-review` (current checkout)

**Spec Kit Feature**: `002-project-registry-import`

**Created**: 2026-09-14

**Status**: Accepted for local development; implementation awaits `AWO-002`

**Input**: [REQ-002](../../requests/REQ-002-project-registry-import.md) and [DR-002](../../decisions/DR-002-project-registry-import.json)

## Change Slice

Provide a read-only Project Registry snapshot for ContextRail consumer Projects. A project owner should be able to see the declared Project identity and linked governance context without modifying the source repository or inferring facts that the source does not declare.

The slice is a contract and inspection capability. It is not Project CRUD, a cloud inventory product, a database-backed workspace or a deployment controller.

## User Scenarios & Testing

### User Story 1 - Inspect a Project governance boundary (Priority: P1)

As a Solution Architect or Project owner, I want to inspect a Project's declared repositories, services, environments, documents and readiness so that I can decide whether the Project has enough context for the next governed action.

**Why this priority**: Project Registry is the shared read boundary for every later ContextRail slice. Without it, each UI or assurance feature would reconstruct Project context independently.

**Independent Test**: Point the inspection operation at the two checked-in consumer fixtures, read the normalized snapshots, and verify that identity, relationships, documents, decisions, context and readiness are visible with their source status.

**Acceptance Scenarios**:

1. **Given** a valid Project manifest, **when** the owner requests a snapshot, **then** the result includes Project identity, repositories, services, environments, documents, decisions, readiness and context status.
2. **Given** a Project using a singular `service` declaration or a plural `services` declaration, **when** it is inspected, **then** the snapshot normalizes the declared service records without dropping their source relationships.
3. **Given** a stale Context Pack or an undeclared runtime, **when** it is inspected, **then** the snapshot preserves `STALE`, `MISSING`, `UNKNOWN` or `UNDECLARED` rather than upgrading the state to a positive readiness result.
4. **Given** a valid Project, **when** the inspection operation completes, **then** the source files and governance artifacts have not been modified.

### User Story 2 - Inspect more than one Project (Priority: P2)

As a Solution Architect, I want to inspect multiple explicitly supplied Projects in one operation so that I can compare their governance readiness without allowing one Project's data to be silently assigned to another.

**Why this priority**: Multi-Project visibility is needed by a registry, but it must remain explicit and read-only before provider discovery or membership authorization exists.

**Independent Test**: Supply both fixtures in one request, verify two separately identified Project entries, and confirm that each entry retains its own root and source-derived statuses.

**Acceptance Scenarios**:

1. **Given** two valid Project roots, **when** both are supplied explicitly, **then** the result contains two independently identifiable Project records.
2. **Given** one invalid Project root, **when** it is included with a valid root, **then** the operation reports the invalid input clearly and does not silently omit or mutate the valid Project record.

### Edge Cases

- A missing or malformed `project.yaml` is an input error, not an empty Project.
- A required document that is absent remains `MISSING`; a source drift remains `STALE`.
- Unknown fields are preserved where the normalized contract can represent them; unsupported runtime or manifest values remain `UNDECLARED` or `UNKNOWN`.
- An inspection result does not create a Project, confirm ownership, grant access or trigger an agent, deployment or external connector.
- A source path outside the explicitly supplied Project root is not crawled implicitly.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST accept one or more explicitly supplied Project roots for inspection.
- **FR-002**: The system MUST validate that each Project manifest identifies a `Project` before producing its normalized record.
- **FR-003**: The system MUST expose Project identity, repositories, services, environments, documents, decisions, readiness and context status in a versioned snapshot.
- **FR-004**: The system MUST normalize both supported singular and plural service declarations without silently discarding declared source fields.
- **FR-005**: The system MUST preserve negative and uncertain states, including `MISSING`, `STALE`, `CONFLICT`, `UNKNOWN` and `UNDECLARED` when present.
- **FR-006**: The system MUST identify the observed source root and observation time for each Project record.
- **FR-007**: The system MUST be read-only and MUST NOT modify source manifests, documents, Context Packs, decisions or evidence.
- **FR-008**: The system MUST fail clearly for missing, unreadable or malformed Project manifests.
- **FR-009**: The system MUST NOT infer repository ownership, authorization, deployment readiness or production approval from an inspection result.

### Key Entities

- **Project Registry Snapshot**: Versioned read result containing one or more independently identified Project records.
- **Project Record**: Normalized identity, purpose, ownership and lifecycle metadata for one Project.
- **Source Relationship**: A declared Repository, Service or Environment relationship retained with its role and source values.
- **Project Document Status**: A source document's declared or observed state, including provenance and readiness impact.
- **Readiness Result**: A derived inspection result that reports whether a defined action has enough observed inputs; it is not an authorization.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Both checked-in consumer fixtures produce a valid versioned snapshot in one local verification run.
- **SC-002**: The snapshot contains all eight top-level inspection areas: Project, repositories, services, environments, documents, decisions, readiness and context.
- **SC-003**: Every stale, missing, unknown or undeclared fixture state remains explicitly represented in the result; zero such states are converted to an unsupported positive state.
- **SC-004**: A before/after file hash comparison shows zero source or governance file mutations for a successful inspection.
- **SC-005**: The contract tests cover at least one valid fixture, one alternate service shape, one stale/undeclared state and one malformed input.

## Assumptions and Boundaries

- The first consumer inputs are checked-in local fixtures or explicitly authorized Project roots; provider crawling is a later slice.
- The current Python importer is a comparison oracle for the contract spike, not the product's long-term runtime boundary.
- The product implementation is expected to fit the Go modular-monolith direction in `architecture.md`; implementation details belong in `plan.md`, not this user-facing specification.
- Authentication, membership, persistence, RAG, Gemini, Cloud Run, IAM and production release remain out of scope.
- `DR-002` is accepted for local development/testing only. Implementation still requires the bounded `AWO-002`; local planning and execution artifacts do not imply staging authorization.
