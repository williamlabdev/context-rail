# Environments and Promotion Policy

| Environment | Target | Demo behavior | Required evidence |
| --- | --- | --- | --- |
| Development | local process or feature branch | executable | local test |
| Testing | CI candidate | represented | test, build, independent code review |
| Staging | `cloud-run/order-operations-portal-staging` | deploy when GCP access exists | accepted DecisionRecord, matching commit, build, smoke |
| Production | protected/read-only | never deployed by this demo | separate release approval and staging receipt |

Production remains `BLOCKED_IN_DEMO`. A staging PASS is not production readiness. The staging helper requires a staging-approved DecisionRecord, a CURRENT matching Context Pack, independent review identity and reviewed commit, an immutable image digest, and then probes the tagged no-traffic revision rather than the service's default traffic URL.
