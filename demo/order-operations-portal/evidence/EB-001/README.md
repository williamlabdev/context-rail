# EB-001 — Engineering Evidence Bundle

This bundle is the evidence index for `REQ-001` / `DR-001`. It is valid only for the commit and image identity recorded by the implementation run. Placeholder values are intentionally visible until the corresponding source is connected.

| Evidence | Status | Meaning |
| --- | --- | --- |
| `test-output.txt` | PASS | local test result |
| `build-output.txt` | PASS | local build result; no commit/digest claimed |
| `code-review.md` | review fixture | independent review decision boundary |
| `cloud-run-staging-receipt.json` | pending cloud access | deploy result or explicit blocker |

The bundle cannot authorize production. A missing or blocked receipt is `NEEDS_INPUT`, not `PASS`.
