# Environments and Promotion Policy

| Environment | Target | Required evidence | Gate |
| --- | --- | --- | --- |
| Development | local Python process | fixture test | change owner |
| Staging | `cloud-run/support-insights-staging` | accepted decision, review, test, smoke | release approver |
| Production | protected/read-only | staging receipt and separate approval | blocked until human decision |

The fixture has no production deployment path.
