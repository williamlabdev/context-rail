# Implementation Plan: Project Registry／Documents & AI Context 唯讀工作區

**Branch**: `codex/001-manual-order-review` (current checkout) | **Date**: 2026-09-14 | **Spec**: [spec.md](spec.md)

**Input**: [REQ-003](../../requests/REQ-003-project-registry-context-ui.md), [DR-003](../../decisions/DR-003-project-registry-context-ui.json), [VS-001 plan](../002-project-registry-import/plan.md)

## Summary

Implement the first user-visible ContextRail journey on top of the VS-001 normalized Project Registry contract. A small Go read API will expose explicitly supplied fixture-backed Project records, and a React/TypeScript workspace will let a Project owner select a Project and inspect registry, Documents / AI Context and readiness states. The browser surface is read-only and must preserve stale, missing, conflict, unknown and undeclared information.

The feature depends on VS-001's contract and implementation boundary. It does not add Project writes, authentication, persistence, provider crawling, RAG or deployment. `DR-003`, the VS-001 implementation/evidence and `AWO-003` are accepted/issued for local development only.

## Technical Context

**Language/Version**: Go 1.22+ for the service; Node.js 25.8.1 and TypeScript/React for the local UI, with versions pinned in package metadata during implementation

**Primary Dependencies**: Go standard library and the VS-001 registry package; React, TypeScript and Vite for the UI; Playwright for the browser journey; exact package versions are local development dependencies, not runtime services

**Storage**: No database; explicitly supplied fixture roots are read by the local service; browser state is ephemeral

**Testing**: Go unit/contract tests, frontend unit/component tests, Playwright browser journey, `go vet ./...`, `go test ./...`, frontend typecheck/build and source hash comparison

**Target Platform**: Local POSIX development with a browser; no Cloud Run target in this slice

**Project Type**: Go read-only web service plus React/TypeScript read-only web workspace

**Performance Goals**: The two checked-in Projects should become selectable in one local browser verification run; no production latency or scale claim is made

**Constraints**: Read-only; no secrets; no auth; no membership; no writes; no provider crawling; no persistence; no Context Pack rebuild; no RAG/Gemini; no Cloud Run/IAM/production

**Scale/Scope**: Two checked-in consumer fixtures, one registry list, one Project detail/context view and three explicit UI state families: populated, incomplete/uncertain and unavailable

## Constitution Check

- [x] Contract before execution: REQ-003, DR-003, spec, plan and tasks are linked; implementation is gated by human acceptance and the VS-001 prerequisite.
- [x] Provenance over assertion: source path, source version/observed time and readiness reason remain visible; no uncertain state is upgraded.
- [x] Bounded and reversible change: paths are limited to DR-003; fixture roots are read-only; browser actions do not mutate governance records.
- [x] Deterministic gates and human authority: UI status is informational; no screen state approves a Project or deployment.
- [x] Evidence is part of the deliverable: backend tests, frontend tests, browser evidence, build identity and zero-mutation proof are required.
- [x] Structured handoff: implementation proceeds only under the bounded AWO-003; no agent or UI action receives write permission.

**Gate status**: PASS for local development implementation; `DR-003` and VS-001 implementation/evidence are accepted, and `AWO-003` is `ISSUED`. Staging/production remain out of scope.

## Dependencies and design decisions

1. VS-001 owns the normalized Project Registry contract. VS-002 consumes it and must not create a second normalization model.
2. The local Go service exposes read-only JSON endpoints and serves the built UI for browser verification. During UI development, the frontend dev server may proxy the same endpoints.
3. The browser receives explicit status values and provenance fields. It must not infer readiness from an empty error-free response.
4. Fixture configuration is explicit and local. No endpoint accepts arbitrary filesystem traversal or arbitrary remote URLs.
5. The UI has no mutation controls in this slice. Future write actions require a new Request/Decision and cannot be added as a convenience to this plan.
6. VS-001 is a hard prerequisite: its Go package/contract implementation and evidence must be accepted before VS-002 implementation begins. If VS-001 is not complete, VS-002 stops at contract design and does not create a duplicate registry model.
7. Research decisions are implementation candidates within the accepted local-development scope; they do not override `unknowns`, authorize dependencies or permit staging/production activity.

## Project Structure

### Documentation (this feature)

```text
specs/003-project-registry-context-ui/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── api.md
│   └── ui.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/context-rail/
└── main.go                         # read-only local service and static UI host

internal/projectregistry/
├── model.go                        # VS-001 normalized models, reused by VS-002
├── manifest.go
├── readiness.go
└── registry.go

internal/http/
├── projects.go                     # GET registry/detail handlers
└── errors.go                       # explicit unavailable/invalid responses

frontend/
├── package.json
├── package-lock.json
├── tsconfig.json
├── vite.config.ts
├── playwright.config.ts
├── index.html
├── src/
│   ├── App.tsx
│   ├── api/projectRegistry.ts
│   ├── components/
│   ├── pages/
│   └── styles/
└── tests/

tests/registry/
└── contract_test.go

tests/http/
└── projects_test.go

tests/browser/
└── project-context.spec.ts

evidence/EB-003/
└── README.md
```

**Structure Decision**: Use a small Go modular-monolith boundary and a focused frontend application. The Go service owns source reading and normalized status; the frontend owns presentation and browser state. No microservices, database, auth layer, cloud adapter or generic design system is introduced for this slice.

## Implementation sequence

1. Confirm VS-001 is complete and available as a reusable package with evidence; otherwise stop before implementation and do not duplicate semantics.
2. Freeze API and UI contracts from the spec and fixtures.
3. Write Go contract/handler tests and frontend state tests before implementation.
4. Implement the read API, then the registry/detail/context UI.
5. Build the frontend, serve it through the local service at the fixed local address in `quickstart.md`, and execute the browser journey.
6. Compare fixture outputs, verify zero mutation, record EvidenceBundle and prepare review.

## Post-design Constitution Check

- [x] The design adds a user-visible path without changing the governance authority boundary.
- [x] The API and UI contracts preserve explicit negative/uncertain states.
- [x] Local fixture execution is separated from cloud deployment and external authorization.
- [x] All UI actions are read-only and do not create audit events or decisions.
- [x] The plan has no unresolved implementation placeholder; version pinning is an implementation task, not an authorization assumption.

## Complexity Tracking

No constitution violation is proposed. Playwright adds browser verification weight, but it is required to prove this is a front/back end slice rather than a backend-only contract. Serving the built UI from the Go service avoids introducing a second runtime or deployment target.

## ContextRail handoff

```text
REQ-003 → spec.md → DR-003 → plan.md/research.md/data-model.md/contracts/ → tasks.md → AWO-003 → ARR-003 → EB-003 → review
```

`DR-003` remains the governance SSOT. A successful local browser journey does not authorize staging or production.
