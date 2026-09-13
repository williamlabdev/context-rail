# Implementation Plan: Manual Order Review

**Branch**: `001-manual-order-review` (planned) | **Date**: 2026-09-13 | **Spec**: [spec.md](spec.md)

**Input**: `requests/REQ-001-manual-order-review.md`, `decisions/DR-001-manual-order-review.json`

## Summary

The Change Slice is a stateless Go HTTP flow backed by seeded synthetic orders and a browser page. The brownfield implementation already exists; this plan records the intended implementation boundary and the verification path without claiming that a new implementation run occurred.

## Technical Context

**Language/Version**: Go 1.22 or newer

**Primary Dependencies**: Go standard library only

**Storage**: In-process memory; no persistence

**Testing**: `go test ./...`, `go vet ./...`, local HTTP smoke checks

**Target Platform**: Local POSIX development and Cloud Run staging

**Project Type**: Go web service with embedded static HTML

**Performance Goals**: Not measured for this synthetic prototype

**Constraints**: No real customer data, no IAM changes, no production deployment

**Scale/Scope**: Two seeded orders and one review interaction

## Constitution Check

- [x] Scope is limited to the accepted Request and `DR-001` allowed paths.
- [x] Data is synthetic and process-local.
- [x] Tests and build evidence are required.
- [x] IAM, production and external side effects are forbidden.
- [x] Human review remains separate from agent execution.

## Project Structure

```text
demo/order-operations-portal/
├── main.go                         # HTTP routes, validation and in-memory state
├── main_test.go                    # behavior tests
├── web/index.html                  # review UI
├── requests/REQ-001-*.md           # canonical request
├── decisions/DR-001-*.json          # accepted decision and boundaries
├── evidence/EB-001/                 # test, build, review and receipt evidence
└── specs/001-manual-order-review/   # Spec Kit pilot artifacts
```

**Structure Decision**: Keep the existing single Go service. Do not introduce a database, authentication layer, separate frontend or deployment controller for this slice.

## ContextRail handoff

The spec and plan are inputs to `DR-001`, not replacements for it. The bounded execution handoff is:

```text
REQ-001 → spec.md → DR-001 → plan.md / tasks.md → AgentRunRecord → EB-001 → review
```

Required references: `project_id=order-operations-portal`, `request_id=REQ-001`, `decision_id=DR-001`, `policy_version=environments-v1`, and the source snapshot hash recorded when the canonical context is rebuilt.

## Design artifacts

- [Data model](data-model.md) defines the synthetic order and review state boundary.
- [HTTP contract](contracts/http.md) defines the externally observable service behavior.
- [Quickstart](quickstart.md) defines the reproducible local validation path.

## Constitution Check: post-design

- [x] The design remains a single reversible Change Slice.
- [x] No design artifact introduces real data, persistence, IAM or production access.
- [x] Every externally observable behavior has a validation scenario or evidence path.
- [x] Cloud Run staging remains a separate gate and is not implied by local validation.
