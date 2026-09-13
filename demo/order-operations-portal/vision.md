# Vision

## Problem

Operations staff need to pause suspicious or incomplete orders before fulfillment. Today the decision is easy to lose in chat and the reason for the decision is not consistently recorded.

## Outcome

An authorized operator can see an order exception, record an approve/reject decision and leave an audit-friendly note. The demo makes the feature small enough to trace from Request to staging evidence.

## Non-goals

- real customer data, payment capture or fulfillment;
- authentication, authorization and persistent storage;
- autonomous approval by AI;
- production deployment;
- pretending that a local fixture is evidence from GitHub or Cloud Run.

## Acceptance criteria

1. The page lists synthetic orders awaiting review.
2. A reviewer can approve or reject an order with a required note.
3. The result is visible in the page and returned by the API.
4. Invalid order IDs and missing notes return a clear client error.
5. The implementation can be tested and built with the commands in `development.md`.
