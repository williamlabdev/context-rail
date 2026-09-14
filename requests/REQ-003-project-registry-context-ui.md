# REQ-003：Project Registry／Documents & AI Context 唯讀工作區

## Request

Project owner 或 Solution Architect 需要在瀏覽器理解一個 Project 是否具備足夠的治理脈絡，而不是只能閱讀 CLI JSON。ContextRail 應在 VS-001 normalized read contract 上，提供一個前後端 read-only journey：後端回傳 Project registry/context/readiness，前端呈現文件 provenance、狀態與缺口，讓使用者知道下一步是可讀取、需要補件、資料已過期，或尚未能判斷。

## Scope

- Included:
  - read-only Project list／selection 與 Project detail/workspace；
  - 提供 UI 所需的 Project registry、documents、context、readiness 與 source identity API；
  - 在 UI 明確呈現 `CURRENT`、`STALE`、`MISSING`、`CONFLICT`、`UNKNOWN`、`UNDECLARED`、loading、empty 與 error；
  - 以 `demo/order-operations-portal` 與 `examples/support-insights` 作為 local fixture-backed consumer Projects；
  - frontend integration/browser evidence 與 backend contract evidence；
  - 保持所有操作 read-only，且 UI 與 API 不改變 consumer Project。
- Excluded:
  - Project create/update/archive、文件上傳、Context Pack rebuild、人工決策操作；
  - authentication、authorization、tenant membership、private-repository crawling、Firestore、RAG、Gemini；
  - Cloud Run、IAM、deployment、production promotion；
  - 將 readiness 或 context status 顯示為 authorization 或 approval。

## Acceptance criteria

- AC-001: 使用者可在瀏覽器開啟 Project Registry，看到兩個 fixture-backed Projects 的 identity 與可選入口。
- AC-002: 使用者開啟一個 Project 後，可在同一個 read-only workspace 看到 repositories、services、environments、documents、decisions、context status 與 readiness reason。
- AC-003: `STALE`、`MISSING`、`CONFLICT`、`UNKNOWN` 與 `UNDECLARED` 在 UI 中各自有明確且不等同於成功的呈現；不可被渲染成 `CURRENT` 或 `PASS`。
- AC-004: UI 能處理 loading、empty、invalid/unavailable/error 狀態，並告知使用者可採取的下一步；不提供會改變來源的操作。
- AC-005: backend API contract、frontend integration test 與至少一條 browser journey evidence 可重現於本機。
- AC-006: 執行前後 consumer Project、Context Pack、DecisionRecord 與 evidence source hash 不變。
- AC-007: 未經人類 review 前，不建立 staging deploy work order；本 slice 不產生 Cloud Run 或 production authorization。

## Context and evidence

- Product sources: `vision.md` v3.3、`roadmap.md` v1.2、`architecture.md` working baseline。
- Slice source: `docs/planning/CONTEXT_RAIL_VERTICAL_SLICES_V1.zh-TW.md` v1.1。
- Upstream contract: `REQ-002`、`DR-002` 與 `specs/002-project-registry-import/`。
- Consumer fixtures: `demo/order-operations-portal`、`examples/support-insights`。
- Environment target: local development only；不含 Cloud Run staging。

## Human decision needed

請決定是否接受「在 VS-001 normalized contract 上，建立 fixture-backed Go read API 與 React read-only Project Registry／Documents & AI Context workspace」作為 VS-002 的 implementation boundary。狀態維持 `PENDING_HUMAN_DECISION`；接受本 Request 不等於允許 staging 或 production deployment。
