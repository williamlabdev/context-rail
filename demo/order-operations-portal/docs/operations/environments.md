# Environments and Promotion Policy

| Environment | Target | Demo behavior | Required evidence |
| --- | --- | --- | --- |
| Development | local process or feature branch | executable | local test |
| Testing | CI candidate | represented | test, build, independent code review |
| Staging | `cloud-run/order-operations-portal-staging` | deploy when GCP access exists | accepted low-risk DecisionRecord, matching commit, AI review, compensating controls, build, smoke |
| Production | protected/read-only | never deployed by this demo | separate human release approval and staging receipt |

Production remains `BLOCKED_IN_DEMO`. A staging PASS is not production readiness. In `single_operator` mode, the staging helper permits low-risk self-approval only when the declared AI review and compensating-control evidence pass. It still requires a staging-approved DecisionRecord, a CURRENT matching Context Pack, an immutable image digest, and then probes the tagged no-traffic revision rather than the service's default traffic URL. Production always requires a distinct human release approval.
