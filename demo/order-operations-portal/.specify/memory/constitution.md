# Order Operations Portal Constitution

## Core Principles

### I. Scope before implementation

Every feature starts from an accepted Request and a bounded Change Slice. The implementation must not silently expand into authentication, payment, fulfillment, IAM or production operations.

### II. Synthetic and reversible by default

The prototype uses synthetic orders and stateless process memory. Persistent customer data, payment side effects and irreversible external operations require a new decision and explicit approval.

### III. Evidence before promotion

Implementation claims require reproducible tests, build output, review evidence and an environment-specific receipt. Missing evidence remains `NEEDS_INPUT`, `UNKNOWN`, `STALE` or `BLOCKED`.

### IV. Human authority and least privilege

Agents may propose or execute only the accepted scope. They cannot change IAM, grant approval, bypass review or deploy production. Human acceptance, candidate review and release approval are separate decisions.

### V. Simple contracts and observable behavior

Use the smallest HTTP and Go contracts that prove the user journey. Keep API results, validation failures, logs, tests and evidence understandable without relying on model output.

## Additional constraints

- Runtime: Go 1.22 or newer, Cloud Run as the staging target.
- Data: synthetic fixtures only; no real customer or credential data.
- Review: the same agent that implements the slice cannot be the sole reviewer.
- Deployment: staging may be executed only after its gate passes; production is blocked in this demo.

## Development workflow

Request → Change Slice specification → DecisionRecord → implementation plan/tasks → bounded execution → tests/build → independent review → staging gate.

Spec Kit artifacts support this workflow but do not replace ContextRail governance artifacts. The repository's `AGENTS.md`, development contract and environment policy remain mandatory.

## Governance

This constitution is project guidance for the demo. Changes to scope, authorization, data handling, environment topology or release policy require a new DecisionRecord and human confirmation. A generated specification or plan cannot override this constitution or the canonical Project Context documents.

**Version**: 1.0.0 | **Ratified**: 2026-09-13 | **Last Amended**: 2026-09-13
