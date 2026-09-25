# EB-013 — Change Decision Brief v2: reader-ordered, human-written owner summary, localised labels

- Checked at: 2026-09-25 Asia/Taipei
- Source commit: `9f5b3ff` (feat/brief-readability; builds on `59b3197`)
- Decision: `decisions/DR-012-brief-readability.json` — `ACCEPTED_FOR_DEVELOPMENT`
- Origin: [readability review of the CHG-001 Brief and Receipt](../../docs/reviews/CONTEXT_RAIL_BRIEF_READABILITY_REVIEW_2026-09-25.zh-TW.md), items 1–3
- Review: AI-assisted implementation with automated evidence; **human review pending** (see [HUMAN_REVIEW_PACKET.zh-TW.md](HUMAN_REVIEW_PACKET.zh-TW.md))
- CI: not run for this commit; all evidence below is local

## What the slice shows

1. **Owner summary is human-written.** `owner_summary` is a request field; without it the Change is `NEEDS_INPUT` with `owner_summary` owned by the requester. The decision copies it verbatim; nothing synthesises it.
2. **Reader-question order.** The Brief opens with the owner summary, then the approach (option title, not id), next stop and "Production is not authorized by this decision", then accepted risk (a warning box when unknowns exist), included / not included, what must happen before the next environment (evidence codes with plain labels, code kept beside), who decided (local time, ISO in the tooltip, rationale quoted), and alternatives not chosen.
3. **Machine identity moves down.** Decision/Change/option ids, technical objective, transition → target ref, full snapshot and topology hashes and policy sit in a collapsed *Traceability (for audit)* block; the lineage strip stays at the foot.
4. **Older decisions are not rewritten.** A decision without `owner_summary` shows *No plain-language summary was supplied* and prints the technical objective unedited.
5. **Route diagram.** The decision records the promotion path of its bound topology version; the Brief draws it top to bottom — *comes from here*, *approved up to here* (green), a ✕ after the target, then later environments dashed as *needs another decision* and production as *not authorized*. A decision without a recorded path shows only source and target, with a note.
6. **System wording in the reader's language.** RuleAdvisor's options and unknowns come from a coded table; zh-TW shows them in Chinese (also in the decision form and decision card). Owner summary, rationale and any uncoded text (a model's unknowns) are shown as recorded.
7. **Locale.** The renderer emits `change-decision-brief/v2` structured values; headings and labels follow zh-TW / en, data values read back unchanged.

## Result

| Check | Result | File |
| --- | --- | --- |
| `go vet ./...`, `go test ./...` (Brief v2 assertions, owner_summary NEEDS_INPUT, missing-summary flag, promotion path order, rule-advisor codes, Brief/Pack lineage parity, STALE reason) | PASS | `go-test-output.txt` |
| Frontend typecheck / build / unit tests (30: +5 Brief tests — DOM order with trace closed, missing-summary notice, path places, path-less fallback, zh-TW coded wording vs recorded text; dictionary coverage) | PASS | `frontend-quality-output.txt` |
| Browser: all 13 journeys on a fresh state dir, system Chrome | 13 PASS | `browser-output.txt` |
| Consumer fixture trees byte-identical before/after | PASS | `mutation-check.txt` |
| Visual: accepted Brief with unknowns, en and zh-TW | — | `brief-en.png`, `brief-zh-TW.png` |

## Boundaries and known gaps

- Readability is shown by structure and screenshots, not by readers. The three-question role test (review item 6, G5) is still open and is the real acceptance for "human-readable".
- Receipt conclusion-first layout and a reviewer Technical Report (review items 4–5) are not in this slice.
- Other server-generated text — missing-input reasons, observations, gate details — is still English in zh-TW; only rule-advisor options and unknowns are coded.
- The screenshots' owner summary and rationale are English because the seeded demo data was written in English; they are shown as recorded. `uat-…` in the path is an environment an earlier browser journey adds.
- Evidence-code labels cover the nine codes current topologies use; others render as the code.
- The EB-012 CHG-001 decision predates `owner_summary` and shows the missing-summary notice.
- Brief JSON changed from v1 (`headline`, `sections`) to v2; no other consumer in the repo reads v1.
