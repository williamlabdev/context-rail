# EB-005 — VS-003 Environment Topology (add / edit / reorder / retire / restore)

- Checked at: 2026-09-20 Asia/Taipei
- Base source commit: `a82df93` (feat/cloud-run-baseline)
- Decision: `decisions/DR-005-environment-topology.json` — `ACCEPTED_FOR_DEVELOPMENT`
- Scope: local development/testing; JSON-file state; no Cloud Run, IAM or production action
- Review: AI-assisted implementation with automated evidence; **human review `ACCEPTED_FOR_LOCAL_DEVELOPMENT`** by `founder-001` on 2026-09-21 (single-operator; see [HUMAN_REVIEW_PACKET.zh-TW.md](HUMAN_REVIEW_PACKET.zh-TW.md))

## What the slice shows

1. Opening a Project bootstraps **topology v1** from `project.yaml` (origin `manifest`) without touching the manifest.
2. Add, edit, reorder, retire and restore each create a new immutable version with actor, reason, computed impact and a field-level diff; the panel shows `Topology vN` and the config hash.
3. Material changes (target, type, order, evidence, approver policy, status) publish `STALE` invalidations for the Project's observed decisions; the Decisions list overlays `→ STALE (topology vN)`. Display-only changes are `INFORMATIONAL`.
4. Retired environments stay in the table as `RETIRED` (history readable) and can only be restored through a new version; production shows `READ_ONLY · BLOCKED IN P0` and its Retire control is disabled.
5. A stale `expected_version` is refused with `409 TOPOLOGY_VERSION_CONFLICT` and the operator is offered a reload.

## Result

| Check | Result | File |
| --- | --- | --- |
| `go vet ./...`, `go test ./...` — registry, HTTP, topology (18 new named tests) | PASS | `go-test-output.txt` |
| Frontend typecheck / build / unit tests (17, incl. 6 topology) | PASS | `frontend-quality-output.txt` |
| Browser: UI-05 journey add → edit → reorder → retire → restore (v2…v6), version-conflict handling, plus the 4 VS-002 journeys | 6 PASS | `browser-output.txt` |
| Consumer fixture trees byte-identical before/after the browser journey; state written only under the state dir | PASS | `mutation-check.txt` |
| Visual: topology panel after the journey | — | `topology-panel.png` |

## Boundaries and known gaps

- Persistence is JSON files under `CONTEXT_RAIL_STATE_DIR`; on Cloud Run that is instance-local `/tmp` and is lost on redeploy or scale-to-zero. Durable storage is `NEEDS_INPUT` (DR-005 unknowns).
- `X-ContextRail-Actor` is recorded in the audit trail as declared; there is no authentication or membership check.
- Invalidation granularity: every observed decision of the Project is marked STALE on a material change. Binding decisions to specific environments needs the VS-004 DecisionRecord contract.
- The VS-002 browser assertion "no mutation controls" was narrowed to Project-level controls (`create project|archive|upload|approve|deploy`); topology controls are now in scope by DR-005.
- Docker image build still `NOT VERIFIED` locally (no daemon); unchanged from EB-004.
