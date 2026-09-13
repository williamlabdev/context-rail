# Governance Policy Template

- Project owner: `<REPLACE_ME>`
- Technical decision approver: `<REPLACE_ME>`
- Independent code reviewer: must be separate from the change author.
- Release approver: must be separate from the change author and reviewer where practical.
- Role assignments must record actor identity and role for each human action. One actor may hold multiple roles, but independence rules are evaluated by actor identity.
- Declare an `operating_mode`: `multi_operator` requires separated human issuer, reviewer and release approver; `single_operator` may allow low-risk staging self-approval only with explicit compensating-control evidence.
- Data classification and retention: `<REPLACE_ME>`
- Required evidence for staging: accepted DecisionRecord, matching diff/commit, tests, build identity, target configuration and smoke.
- Production policy: a distinct human approval remains required; default to protected/read-only until explicitly enabled, regardless of operating mode.

If a required source, identity, review or runtime fact is unavailable, mark the gate `NEEDS_INPUT`, `UNKNOWN` or `BLOCKED`.
