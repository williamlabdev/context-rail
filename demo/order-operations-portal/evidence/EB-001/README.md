# EB-001 — Engineering Evidence Bundle

This bundle is the evidence index for `REQ-001` / `DR-001`. The local implementation evidence below is tied to source commit `10876c68ad460d5dbf0edd584c1db7de2e2cdb17` and source snapshot `sha256:840ce51a96fd0ef408c22ef903c93465ba9db857c5744bef45ec03857bf26d11`. The prior review is invalidated by the multi-role contract change; no container image digest or Cloud Run revision is claimed.

| Evidence | Status | Meaning |
| --- | --- | --- |
| `test-output.txt` | PASS | local test result for source commit `10876c6` |
| `vet-output.txt` | PASS | local static analysis result for source commit `10876c6` |
| `build-output.txt` | PASS | local build result for source commit `10876c6`; no image digest claimed |
| `local-smoke.txt` | PASS | local handler-level positive and negative smoke result for source commit `10876c6` |
| `../../work-orders/AWO-001-manual-order-review.json` | BLOCKED | bounded execution contract awaiting human request acceptance |
| `../../runs/ARR-001-manual-order-review.json` | NOT_STARTED | no agent execution or provenance claimed for the brownfield baseline |
| `HUMAN_REVIEW_PACKET.zh-TW.md` | REVIEW_GUIDE | single human-readable entry point for independent review |
| `code-review.md` | review fixture | independent review decision boundary |
| `single-operator-controls.md` | NEEDS_INPUT | compensating controls required for low-risk staging self-approval |
| `cloud-run-staging-receipt.json` | pending cloud access | deploy result or explicit blocker |

The bundle cannot authorize production. A missing or blocked receipt is `NEEDS_INPUT`, not `PASS`.
