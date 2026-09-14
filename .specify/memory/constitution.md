# ContextRail Constitution

Status: proposed working baseline; human review pending

## Core Principles

### I. Contract before execution

Every material feature starts with a bounded Request and a traceable specification. A DecisionRecord, policy version and human gate determine whether implementation, candidate review or environment promotion may proceed. Spec Kit artifacts help form the work; they do not grant authority.

### II. Provenance over assertion

Every readiness, review and release claim must point to the observed source version, commit, digest or evidence artifact that supports it. Missing, stale, conflicting or unavailable information remains `NEEDS_INPUT`, `STALE`, `CONFLICT`, `UNKNOWN` or `INCONCLUSIVE`; an agent must not fill it with an invented fact.

### III. Bounded and reversible change

An implementation is limited to the accepted repository, branch, paths, acceptance criteria and forbidden actions. Prefer local, stateless and reversible slices before adding cloud resources, persistence, external connectors or production effects. No hard deletion of evidence-bearing governance records.

### IV. Deterministic gates and human authority

Tests, build checks and policy gates may block a transition but cannot approve it by themselves. AI review is technical evidence, not human approval. Production and any high-impact exception require an explicitly identified human authority; single-operator low-risk staging may use documented compensating controls without weakening the production gate.

### V. Evidence is part of the deliverable

Each completed slice records the exact checks, changed paths, source identity, review identity and known evidence gaps. Every logical repository change is committed with an auditable message; unrelated dirty work is preserved and deployment is never inferred from local success.

### VI. Structured handoff before agent collaboration

P0 agents exchange versioned Requests, DecisionRecords, AgentWorkOrders, AgentRunRecords and EvidenceBundles. Free-form agent conversation is coordination only. Autonomous Agent-to-Agent orchestration is not a P0 prerequisite and may be considered only after a measured interoperability need.

## Additional Constraints

- The Project Context Contract, `vision.md`, `roadmap.md` and `architecture.md` define product direction and boundaries; this constitution does not silently override them.
- `docs/ai/context-pack.json` is derived context, never a source of truth.
- Local development does not imply Cloud Run, IAM or production authorization.
- The reusable consumer template remains runtime-agnostic; Python tooling is used for contract validation and rebuild, while the ContextRail product API is planned as a Go modular monolith.
- Secrets, real credentials, unapproved personal data and provider write access are out of scope unless a separate accepted decision authorizes them.

## Development and Review Workflow

1. Define the vertical slice against Vision, Roadmap, Architecture and the Project Context Contract.
2. Create or update the Request, then use Spec Kit `specify → plan → tasks` to form the executable work description.
3. Record the DecisionRecord and human gate before implementation files or deployment work are authorized.
4. Execute only the bounded Work Order, run documented tests/build checks, and reconcile Git/CI/runtime evidence.
5. Review the evidence independently where policy requires it; keep staging and production decisions separate.
6. Commit each logical change and leave unavailable evidence explicitly unresolved.

## Governance

This is a proposed working baseline and must be reviewed by the project owner before it is treated as ratified. Amendments require a committed diff, a short reason, and review of affected Spec Kit templates or governance documents. A feature plan must include a Constitution Check and record any justified exception rather than silently bypassing a principle.

**Version**: 0.1.0 | **Ratified**: pending human review | **Last Amended**: 2026-09-14
