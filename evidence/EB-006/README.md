# EB-006 — VS-004 Change Decision Pack (Request → NEEDS_INPUT → Decision → Brief + Agent Context Pack → Work Order)

- Checked at: 2026-09-20 Asia/Taipei
- Base source commit: `8a058e2` (feat/vs-003-topology)
- Decision: `decisions/DR-006-change-decision-pack.json` — `ACCEPTED_FOR_DEVELOPMENT`
- Scope: local development/testing; JSON ledger; rule advisor; no agent run, Cloud Run, IAM or production action
- Review: AI-assisted implementation with automated evidence; **human review `ACCEPTED_FOR_LOCAL_DEVELOPMENT`** by `founder-001` on 2026-09-21 (single-operator; see [HUMAN_REVIEW_PACKET.zh-TW.md](HUMAN_REVIEW_PACKET.zh-TW.md))

## What the slice shows

1. Opening a Change with only a title, objective and target → **NEEDS_INPUT** listing each missing input with its owner role and reason (acceptance criteria → requester; allowed paths → tech lead; data classification and expected volume → business owner). Candidates and unknowns are still shown so the human sees the shape of the decision; accepting is refused (`422 NEEDS_INPUT`).
2. Supplying inputs → **change v2**, re-evaluated to **DECISION_READY** with the topology-derived transition (`testing-to-staging`); v1 stays readable in history.
3. A named human selects one candidate and accepts → **DEC-001 v1 ACCEPTED_FOR_DEVELOPMENT**. The Change Decision Brief (people) and the Agent Context Pack (agent) are rendered from that one record and show the same lineage strip: decision id + version, source snapshot hash, topology version, policy version.
4. Compile Work Order → **AWO-001 ISSUED** with `work_order_hash`, repository/branches, allowed paths, forbidden actions (policy defaults + request), acceptance ids and required checks (target environment evidence + diff check + independent review).
5. A material topology edit (staging `target_ref`) → the decision, brief headline, pack status and work order all show **STALE** with the reason; re-issuing the Work Order is refused (`409 DECISION_STALE`).

## Result

| Check | Result | File |
| --- | --- | --- |
| `go vet ./...`, `go test ./...` — registry, topology, HTTP, change (9 domain + 2 HTTP journey/error tests) | PASS | `go-test-output.txt` |
| Frontend typecheck / build / unit tests (22, incl. 5 ChangesPanel) | PASS | `frontend-quality-output.txt` |
| Browser: UI-06→09 journey incl. STALE on topology change, plus VS-003 and VS-002 journeys | 7 PASS | `browser-output.txt` |
| Consumer fixture trees byte-identical before/after; ledger written only under the state dir (`changes.json`: v1 NEEDS_INPUT → v2 DECISION_READY, DEC-001, AWO-001, 4 audit events) | PASS | `mutation-check.txt` |
| Visual: Changes panel with stale decision, dual view and work order | — | `changes-panel.png` |

## Boundaries and known gaps

- **GeminiAdvisor is UNVERIFIED.** The adapter exists (`GEMINI_API_KEY` enables it; falls back to the rule advisor on any error) but no call was made: no key and no egress from the verification workspace. Rule-advisor candidates are what every test and screenshot shows.
- Invalidation on material topology change is Project-wide (all observed decisions) at the topology layer; the change ledger additionally checks the decision's own bound topology version, target environment and source snapshot.
- Persistence is instance-local JSON (see DR-005); no authentication; `X-ContextRail-Actor` is recorded, not verified.
- Cost handling is limited to naming UNKNOWN drivers; no deterministic price calculator yet.
- No agent is executed and no candidate/PR is read back — that is VS-005.
