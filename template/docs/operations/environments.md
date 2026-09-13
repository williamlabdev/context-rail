# Environments and Promotion Policy

| Environment | Standard type | Target reference | Required evidence | Human gate |
| --- | --- | --- | --- | --- |
| Development | development | `<REPLACE_ME>` | local test | change owner |
| Testing | testing | `<REPLACE_ME>` | test, build, independent review | reviewer |
| Staging | staging | `<REPLACE_ME>` | decision, review, build, config, smoke; single-operator controls when selected | release approver or documented low-risk self-approval |
| Production | production | protected target | staging receipt, release approval, smoke | separate human release approver |

Document the selected operating mode, allowed transitions, target identity, configuration evidence, compensating controls, approver separation, expiry and rollback conditions. A passing lower environment never implies production readiness.
