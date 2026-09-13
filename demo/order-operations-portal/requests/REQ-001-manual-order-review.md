# REQ-001：訂單人工覆核

## Request

Operations staff need a controlled action to review an order exception before fulfillment. A reviewer must explicitly approve or reject the order and provide a reason.

## Scope

Included: synthetic order list, review action, required note, API validation, visible result, `/healthz` smoke endpoint.

Excluded: authentication, persistence, payment, fulfillment, real customer data, production deployment.

## Acceptance criteria

- AC-001: list at least two synthetic pending orders;
- AC-002: approve and reject require a non-empty note;
- AC-003: unknown order returns `404`;
- AC-004: result is returned as JSON and visible in the UI;
- AC-005: tests and build are reproducible from `development.md`;
- AC-006: staging deployment, when authorized, is tied to the approved image digest.

## Human decision needed

Accept the minimal stateless demo implementation and allow development/testing. Cloud Run staging remains a separate release decision; production is not requested.
