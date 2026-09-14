# Research: Project Registry／Documents & AI Context 唯讀工作區

**Date**: 2026-09-14

**Status**: Candidate decisions pending human acceptance of `DR-003`

## Decision 1: Reuse the VS-001 normalized contract

- **Decision**: VS-002 consumes the VS-001 `ProjectRegistrySnapshot` semantics and does not create a second Project/Document/Readiness model.
- **Rationale**: Project Registry is a shared read boundary. Duplicating normalization in the UI would allow singular/plural manifests and stale states to diverge.
- **Alternatives considered**: Let the frontend read `project.yaml` directly (rejected because it bypasses the service boundary and provenance rules); create a separate UI-specific registry model (rejected because it duplicates the governance contract).

## Decision 2: Go service plus React/TypeScript UI

- **Decision**: Use the planned Go modular-monolith boundary for read APIs and a focused React/TypeScript workspace for the browser surface.
- **Rationale**: This follows `architecture.md` and `vision.md`, keeps source reading server-side, and makes the first user journey independently testable without adding persistence or cloud services.
- **Alternatives considered**: Static HTML only (rejected because it would not prove a real frontend/API boundary); separate backend service and frontend deployment (rejected because it adds runtime/deployment complexity before the local contract is proven).

## Decision 3: REST-style JSON read contract

- **Decision**: Expose `GET /v1/projects` and `GET /v1/projects/{project_id}` with explicit error responses and no write routes.
- **Rationale**: The contract is simple, inspectable, browser-friendly and maps directly to the two user journeys. Versioning makes later changes explicit.
- **Alternatives considered**: GraphQL (rejected as unnecessary for two read views); frontend filesystem access (rejected as unsafe and not representative of the product boundary).

## Decision 4: Serve the built UI from the Go service for verification

- **Decision**: Local development may run a frontend dev server with API proxying, but the verification path serves the built frontend from the Go service.
- **Rationale**: One local service makes the browser journey reproducible and avoids claiming a second deployment target. Development feedback remains convenient without changing the release boundary.
- **Alternatives considered**: Browser test against two independent processes (deferred; useful only if a real deployment split is later selected); Cloud Run preview (out of scope and costs/authorizes more than this slice requires).

## Decision 5: Browser evidence with Playwright

- **Decision**: Use a pinned local Playwright test for the registry-to-detail journey and explicit uncertain/error states.
- **Rationale**: A browser test provides reproducible evidence that the frontend consumes the backend and renders the governance statuses; it does not require a remote service.
- **Alternatives considered**: Manual screenshot only (rejected as difficult to reproduce); backend-only tests (rejected because they cannot prove the UI slice).

## Decision 6: No authentication in this slice

- **Decision**: Treat local fixture access as an explicitly authorized development input and keep authentication/membership out of VS-002.
- **Rationale**: Authentication changes the actor, tenancy and authorization contract and requires a separate Request/Decision. The absence must not be interpreted as production readiness.
- **Alternatives considered**: Add a demo user/password (rejected because it creates security semantics without a policy); use a cloud identity provider (rejected as outside the no-cost local scope).
