# Governance Policy Template

- Project owner: `<REPLACE_ME>`
- Technical decision approver: `<REPLACE_ME>`
- Independent code reviewer: must be separate from the change author.
- Release approver: must be separate from the change author and reviewer where practical.
- Data classification and retention: `<REPLACE_ME>`
- Required evidence for staging: accepted DecisionRecord, matching diff/commit, tests, build identity, target configuration and smoke.
- Production policy: human approval required; default to protected/read-only until explicitly enabled.

If a required source, identity, review or runtime fact is unavailable, mark the gate `NEEDS_INPUT`, `UNKNOWN` or `BLOCKED`.
