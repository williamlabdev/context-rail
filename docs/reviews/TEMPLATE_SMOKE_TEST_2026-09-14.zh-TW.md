# Project Context Contract Template Smoke Test

- Date: 2026-09-14 Asia/Taipei
- Commit: `605bad1`
- Scope: `template/` fresh-project instantiation
- Cloud Run: not used

## Result

`PASS`

The automated smoke test copies `template/` to a fresh temporary Project, replaces initialization placeholders, rebuilds the Context Pack, runs strict Project Context validation, and checks both policy modes:

- `multi_operator`: staging self-approval is disabled.
- `single_operator`: low-risk staging self-approval can be enabled with compensating controls.
- Both modes keep `production_policy.requires_distinct_human` enabled.

## Verification

```text
test_fresh_project_rebuilds_and_validates_both_policy_modes ... ok
Ran 13 tests ... OK
Ran 8 tests ... OK
PROJECT CONTEXT PASS
PROJECT CONTEXT STRICT PASS
STAGING GATE PASS
```

The positive staging gate result is from the existing synthetic demo fixture, not from a Cloud Run deployment. The demo remains `staging_verified: NEEDS_INPUT` until a real receipt exists.

## Interpretation

The template is reusable for a new Project at the contract and policy level. This does not yet prove a new application repository can be deployed or operated in Cloud Run; that requires a separate runtime setup and budget decision.
