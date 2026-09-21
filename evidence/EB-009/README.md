# EB-009 — VS-007 Prod-demo promotion (same digest → next environment, delta bound by approval, chained receipt)

- Checked at: 2026-09-21 Asia/Taipei
- Base source commit: `a2cf649` (feat/vs-006-promotion)
- Decision: `decisions/DR-009-prod-demo-promotion.json` — `ACCEPTED_FOR_DEVELOPMENT`
- Scope: local development/testing with declared deployment fixtures; no Cloud Run deployment executed, no production action
- Review: AI-assisted implementation with automated evidence; **human review `ACCEPTED_FOR_LOCAL_DEVELOPMENT`** by `founder-001` on 2026-09-21 (single-operator; see [HUMAN_REVIEW_PACKET.zh-TW.md](HUMAN_REVIEW_PACKET.zh-TW.md))

## What the slice shows

1. **Staging receipt first** — a change is decided, implemented, accepted and promoted to staging exactly as in EB-008 (`RR-001`).
2. **Promote** — the PROMOTED staging release offers "Promote"; opening it creates a second release that inherits the digest (`build` gate: *inherited from REL-001 / RR-001; a promotion never rebuilds*), freezes the change lineage of the staging receipt, and lists the **environment delta** between staging and prod-demo (`type`, `target_ref`, `required_evidence`, `approver_policy`) with its own hash inside the manifest. Gates `source_release`, `promotion_order`, `environment_delta`, `change`, `environment` all PASS; the build form is absent.
3. **Third human decision again** — `approver-1` approves the promotion manifest (delta hash included, 72h).
4. **Wrong then right** — recording the *staging* revision name on prod-demo is `PROMOTION_FAILED` with `new_revision BLOCKED` (revisions belong to one service); the approval survives. Recording a new revision on the prod-demo target with the same digest and a passing smoke is **PROMOTED** with `RR-003` whose `previous_receipt_id` is `RR-001` and whose evidence refs include `receipt:RR-001`.
5. **Once per target** — promoting the same staging receipt again is `GATE_BLOCKED` (`source_release`: already promoted by REL-002 / RR-003). No further promotion is offered after prod-demo: production stays blocked.
6. **Domain tests additionally cover:** promotion to production (environment + promotion_order both BLOCKED); promotion to an evidence-only node (`uat`) blocked, while a `uat` node *between* staging and prod-demo is skipped and named by `promotion_order`; a non-PROMOTED or unknown source refused (`SOURCE_RELEASE_INVALID`); build on a promotion refused (`BUILD_INHERITED_FROM_SOURCE`); a newer accepted candidate after the staging receipt makes the promotion **STALE** (change gate names `RR-001`) and its deployment fails; a plain release to `testing` is blocked with the P0 boundary.

## Result

| Check | Result | File |
| --- | --- | --- |
| `go vet ./...`, `go test ./...` — 11 release domain tests (4 new) + every earlier suite | PASS | `go-test-output.txt` |
| Frontend typecheck / build / unit tests (22) | PASS | `frontend-quality-output.txt` |
| Browser: prod-demo promotion journey plus VS-006, VS-005a, VS-004, VS-003 and VS-002 journeys, **two consecutive runs on one state dir** | 10 + 10 PASS | `browser-output.txt` |
| Consumer fixture trees byte-identical after both runs; ledger shows staging receipts and prod-demo receipts chained (`prev=RR-…`) | PASS | `mutation-check.txt` |
| Visual: promoted prod-demo release with delta, failed staging-revision attempt and chained receipt; blocked second promotion | — | `prod-demo-receipt.png`, `prod-demo-double-promotion-blocked.png` |

## Findings while verifying

- The first full run of the browser suite after this slice surfaced a real cross-journey interaction: the VS-003 journey leaves a `uat` node right after staging, and the promotion order then pointed at `uat` instead of prod-demo. Fixed in the domain rule (evidence-only nodes are skipped and named), covered by `TestPromotionOrderSkipsEvidenceOnlyEnvironments`, and the second consecutive run now passes on top of the first run's topology.
- `tests/browser/environment-topology.spec.ts` read the topology version as a snapshot right after a row status change and raced the re-render once; changed to a retrying assertion on the heading text. No product change.

## Boundaries and known gaps

- **No real deployment.** Both receipts in evidence are declared. `scripts/record-promotion.sh` (now documented for the prod-demo service too) is UNVERIFIED until Cloud Run services exist (GCP deferred to 2026-10-08 by the owner).
- Adding prod-demo after decisions were made marks them STALE (material topology change) — by design; the demo adds prod-demo first.
- The service does not read Cloud Run itself; the operator script is the read-back path.
- Same persistence and actor caveats as DR-005…DR-008.
