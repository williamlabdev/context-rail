# REQ-002：Project Registry 唯讀匯入

## Request

Solution Architect 或 Project owner 需要能快速確認一個 consumer Project 的治理邊界與脈絡 readiness。ContextRail 應先提供唯讀匯入，將既有 `project.yaml` 與 readiness inspector 的結果整理成穩定、可回查的 Project Registry snapshot，讓後續 Workspace、Documents / AI Context 與 Change assurance slice 共用同一個讀取契約。

## Scope

- Included:
  - 讀取一個或多個指定的 Project root 與 `project.yaml`；
  - 正規化 Project、Repository、Service、Environment、ProjectDocument、Decision、Context 與 readiness；
  - 支援 singular `service` 與 plural `services` 的已存在 manifest 形狀；
  - 明確保留 `STALE`、`MISSING`、`UNKNOWN` 與 `UNDECLARED`；
  - 輸出 versioned `ProjectRegistrySnapshot`；
  - 保持 read-only，並以現有 Python importer 作為 contract oracle。
- Excluded:
  - Project CRUD、archive、membership、authorization 或任何來源寫入；
  - Firestore、Cloud Storage、GitHub connector、任意 URL crawling、RAG 或 Gemini；
  - Context Pack rebuild；
  - Cloud Run、IAM、deployment、production promotion；
  - 從觀察結果自動建立或確認 Project／Repository／Service 關係。

## Acceptance criteria

- AC-001: 指定 demo Project 時，輸出包含 Project identity、repositories、services、environments、documents、decisions、readiness 與 context status。
- AC-002: demo 與 `examples/support-insights` 兩種 manifest 形狀都能被讀取，且不抹平 singular/plural service 差異造成的語義。
- AC-003: stale Context Pack、未宣告 runtime 與其他未知狀態仍以明確狀態輸出，不被推論成 `PASS` 或 `CURRENT`。
- AC-004: 執行前後來源 Project、Context Pack、DecisionRecord 與其他 governance artifact 的檔案 hash 不變。
- AC-005: 輸出 contract 有 deterministic schema、測試與可回查的 source identity；不能宣稱已完成 API、Firestore 或 Cloud Run。
- AC-006: 未經人類 review 前，不產生 staging deploy work order，也不授予任何 agent 或 runtime 寫入權限。

## Context and evidence

- Product sources: `vision.md` v3.3、`roadmap.md` v1.2、`architecture.md` working baseline。
- Contract source: `docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md` v1.0。
- Related code oracle: `scripts/context_rail_registry.py` 與 `tests/test_context_rail_registry.py`。
- Existing evidence: `docs/validation/CONTEXT_RAIL_PROJECT_REGISTRY_IMPORT_2026-09-14.zh-TW.md`。
- Consumer fixtures: `demo/order-operations-portal`、`examples/support-insights`。
- Environment target: local development only；不含 Cloud Run staging。

## Human decision needed

請決定是否接受「以既有 Python contract oracle 對照的 Go modular-monolith read-only Project Registry」作為 VS-001 的 core implementation boundary。狀態維持 `PENDING_HUMAN_DECISION`；接受本 Request 不等於允許 staging 或 production deployment。
