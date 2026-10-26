# REQ-002：訂單例外覆核附加佐證連結

## Request

Before approving or rejecting an order exception, a reviewer wants to attach up to three supporting evidence references (a label and a link, e.g. a ticket, a chat thread or a document) to the order, so the decision can be audited later without leaving the portal.

## Scope

Included: `POST /api/orders/{id}/evidence` accepting `{label, url}`; at most three references per order; references listed with the order in `GET /api/orders`; visible in the UI above the review buttons; validation errors as JSON; the review note stays required.

Excluded: file upload or storage (no Cloud Storage bucket, no persistence), authentication, payment or fulfillment changes, production deployment, any change to `docs/operations/environments.md` or the review decision rules.

## Acceptance criteria

- AC-001: a reference needs a non-empty label and an `http(s)` URL; anything else returns `400` with a JSON error;
- AC-002: the fourth reference on the same order returns `409`;
- AC-003: unknown order returns `404`;
- AC-004: references appear in `GET /api/orders` and in the UI card of that order;
- AC-005: existing review behaviour and tests keep passing; `go test ./...` and `go build ./...` stay green.

## Business constraints (to be confirmed by the business owner)

- data_classification: internal (links only, no customer files);
- expected_monthly_volume: about 200 references per month.

## Human decision needed

Accept the minimal stateless implementation (references kept in memory next to the order) for development and staging; storage-backed attachments are out of scope for this Change.
