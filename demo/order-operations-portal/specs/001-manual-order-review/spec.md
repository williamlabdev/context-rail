# Feature Specification: Manual Order Review

**Feature Branch**: `001-manual-order-review` (planned; current checkout remains `develop`)

**Created**: 2026-09-13

**Status**: Pilot baseline; brownfield implementation already exists

**Input**: `requests/REQ-001-manual-order-review.md` and `decisions/DR-001-manual-order-review.json`

## Change Slice

Provide a small, independently testable flow for an operations staff member to review a synthetic order exception before fulfillment. The slice includes an order list, an explicit approve/reject action, a required reason, validation and a visible result. It does not introduce persistence, authentication or real fulfillment side effects.

## User Scenarios & Testing

### User Story 1 - Review an order exception (Priority: P1)

As an operations staff member, I want to approve or reject a pending order with a reason so that the demo shows a controlled human review step.

**Why this priority**: This is the smallest journey that proves the product intent and release-evidence path.

**Independent Test**: Start the service, open the order list, submit an approval or rejection with a note, and confirm the result is visible to the operator.

**Acceptance Scenarios**:

1. **Given** at least two synthetic pending orders, **when** the operator opens the portal, **then** the orders and their review reasons are visible.
2. **Given** a known order, **when** the operator submits `APPROVED` or `REJECTED` with a non-empty note, **then** the status and trimmed note are returned and displayed.
3. **Given** a known order, **when** the note is empty or whitespace, **then** the request is rejected with a validation error and the order is unchanged.
4. **Given** an unknown order ID, **when** a review is submitted, **then** the system clearly reports that the order does not exist and makes no review change.

### Edge Cases

- Invalid JSON returns `400`.
- A decision other than `APPROVED` or `REJECTED` returns `400`.
- State is process-local and resets after restart; persistence is explicitly out of scope.
- Authentication, authorization, payment, fulfillment and production deployment remain out of scope.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST list at least two synthetic orders in `PENDING_REVIEW` state.
- **FR-002**: The system MUST require `APPROVED` or `REJECTED` for a review decision.
- **FR-003**: The system MUST require a non-empty trimmed note.
- **FR-004**: The system MUST present the reviewed order, its new status and the review time to the operator.
- **FR-005**: The system MUST clearly reject an unknown order without changing any order state.
- **FR-006**: The browser UI MUST make the review result visible.
- **FR-007**: The implementation MUST use synthetic, stateless data for this slice.

### Key Entities

- **Order**: Synthetic order exception with ID, customer label, amount, reason, status and optional review note.
- **Review Request**: A decision and note submitted for one order.

## Success Criteria

- **SC-001**: A reviewer can complete one approval and one rejection through the UI using only the seeded orders.
- **SC-002**: The five documented happy-path and negative behaviors pass during verification, including validation and unknown-order behavior.
- **SC-003**: A reviewer can start the prototype from the documented setup and observe the primary journey without undocumented steps.
- **SC-004**: The change can be independently reviewed against the allowed paths in `DR-001`.

## Assumptions and Open Questions

- Synthetic data is sufficient for the prototype demonstration.
- The state reset on restart is acceptable for this slice.
- Real authentication, persistence and authorization require a future Request and DecisionRecord.
- Cloud Run staging remains unavailable until project, region, image and deploy identity are supplied; this is recorded in `EB-001` rather than assumed.
