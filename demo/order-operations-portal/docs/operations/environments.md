# Environments and Promotion Policy

| Environment | Target | Demo behavior | Required evidence |
| --- | --- | --- | --- |
| Development | local process or feature branch | executable | local test |
| Testing | CI candidate | represented | test, build, independent code review |
| Staging | `local/order-operations-portal-staging` | local process, read back through its loopback `/healthz` (DR-015, ContextRail DR-017) | accepted low-risk DecisionRecord, matching commit, AI review, compensating controls, build, smoke |
| Prod-demo | `local/order-operations-portal-prod-demo` | isolated local process, promoted only from a staging receipt | release approval, staging receipt, smoke, single-operator controls |
| Production | protected/read-only | never deployed by this demo | separate human release approval and staging receipt |

Production remains `BLOCKED_IN_DEMO`. A staging PASS is not production readiness, and neither is a prod-demo PASS: prod-demo is a second local process, isolated only by process and port. The Cloud Run staging path below (`scripts/deploy-staging.sh`) is the pre-DR-015 history behind EB-001; it is kept, not the current target. In `single_operator` mode, the staging helper permits low-risk self-approval only when the declared AI review and compensating-control evidence pass. It still requires a staging-approved DecisionRecord, a CURRENT matching Context Pack, an immutable image digest, and then probes the tagged no-traffic revision rather than the service's default traffic URL. Production always requires a distinct human release approval.
