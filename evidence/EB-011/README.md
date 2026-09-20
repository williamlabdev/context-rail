# EB-011 — UI-15 … UI-19 Workspace persistence and locale (deep links, zh-TW / en, human status labels, data values untouched)

- Checked at: 2026-09-21 Asia/Taipei
- Base source commit: `00cf52a` (feat/ui-20-context-pack)
- Decision: `decisions/DR-011-workspace-locale.json` — `ACCEPTED_FOR_DEVELOPMENT`
- Scope: frontend only; no Go change (suites re-run for the record)
- Review: AI-assisted implementation with automated evidence; human review of DR-011 pending

## What the slice shows

1. **UI-15** — the selected Project lives in the URL (`/projects/{id}`). Refresh keeps the Project and its topology version; switching Projects changes the URL and shows only that Project's ledger (support-insights shows *No Change opened yet*); back restores the previous Project with the same version; a deep link to `/projects/support-insights` opens that Project with its card pressed. The static handler serves every `/projects/…` path (200) so a deep link never 404s.
2. **UI-16** — the header switch (`EN` / `繁中`) flips every static copy: headings (變更 / 發布 / 晉升路徑 / 文件 / AI 情境 / 就緒度), buttons (新增變更 / 新增環境 / 開啟變更並評估), labels, notes and empty states; `<html lang>` follows. A unit test scans every `t()` key in the sources (340+) and fails if zh-TW lacks one.
3. **UI-17** — after a refresh the locale is still zh-TW (localStorage `context-rail.locale`) and the Project context is unchanged; switching Projects keeps the locale.
4. **UI-18** — badges show the human label with the machine code beside it (`情境 已衍生 DERIVED`, `啟用 ACTIVE`, `需要輸入 NEEDS_INPUT`; in English `Context Stale STALE`, `Gate blocked GATE_BLOCKED`). Identifiers such as `RR-001` render verbatim.
5. **UI-19** — Project name, source root, repository path, environment ids and target refs, document paths and a user-entered Change title read back unchanged in zh-TW and after switching back to en; a Change opened from the zh-TW form carries the same data.

## Result

| Check | Result | File |
| --- | --- | --- |
| `go vet ./...`, `go test ./...` (unchanged backend) | PASS | `go-test-output.txt` |
| Frontend typecheck / build / unit tests (25: +3 locale tests incl. dictionary coverage) | PASS | `frontend-quality-output.txt` |
| Browser: UI-15 and UI-16…19 journeys plus every earlier journey, **two consecutive runs on one state dir** | 13 + 13 PASS | `browser-output.txt` |
| Consumer fixture trees byte-identical; `/projects/*` deep links served | PASS | `mutation-check.txt` |
| Visual: workspace and topology in zh-TW, topology in en | — | `workspace-zh-TW.png`, `topology-zh-TW.png`, `topology-en.png` |

## Findings while verifying

- The first full run surfaced two spec assumptions, not product defects: `project-context.spec.ts` matched the old badge text `Context STALE` (now label + code), and the new locale spec hard-coded the staging target ref that an earlier journey moves. Both assertions were made semantic (badge label + code; target ref read from the API).

## Boundaries and known gaps

- Server-generated text — gate details, reasons, impacts, advisor unknowns, audit reasons — stays English in both locales as technical detail (DR-011 defers a server-side catalogue).
- Only `en` and `zh-TW` exist.
- Same persistence and actor caveats as DR-005…DR-010.
