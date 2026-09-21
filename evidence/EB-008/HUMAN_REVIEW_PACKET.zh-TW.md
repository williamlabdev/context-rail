# EB-008 人類 Review Packet — VS-006 Staging promotion

這份文件是給人快速閱讀的摘要；原始命令輸出、截圖與 mutation check 仍保留在同一個 Evidence Bundle（EB-008），方便追溯。

## Review result

| 欄位 | 結果 |
| --- | --- |
| Reviewed commit | `17c3fc7` (develop) |
| Decision | `DR-008 / ACCEPTED_FOR_DEVELOPMENT` |
| Human reviewer | `founder-001` |
| Role | `Founder + Solution Architect` |
| Reviewed at | `2026-09-21T04:28:20Z` |
| Human result | `ACCEPTED_FOR_LOCAL_DEVELOPMENT` |
| Review mode | 單一操作者 review（透過 Claude Cowork 逐項提問、由人裁決）；不重跑測試（CI 已跑：三個 job 綠）；不宣稱獨立 reviewer separation |

## 一句話結論

Release bundle 整包擋、build digest 必須涵蓋被接受的 commit、第三次人工 approval 綁 manifest hash（72h）、deployment record 每道 gate 必須 PASS、idempotency key、含 hash 的 Release Receipt；UI-12/13/14 全程有可執行證據，但所有部署觀察都是 declared fixture。

## 審了什麼

1. **方案與範圍**：`DR-008` 的 selected_option 與 scope 是否為 owner 要的 → 是。
2. **證據對驗收標準**：哪些是真驗過、哪些仍是 declared：

| 項目 | 狀態 | 出處 |
| --- | --- | --- |
| 7 個 release domain 測試 + 1 HTTP journey | 已驗 | 見 README.md 的 Result 表 |
| UI-13 GATE_BLOCKED（整包）、UI-12 receipt、UI-14 config drift → STALE 的 Playwright journey | 已驗 | 見 README.md 的 Result 表 |
| consumer fixture 樹 hash 前後相同 | 已驗 | 見 README.md 的 Result 表 |
| 每一筆 deployment record（digest / revision / smoke）都是操作者宣告的 fixture | declared / 未驗 | 留在 DR unknowns |
| scripts/record-promotion.sh 對真實 gcloud 輸出未驗 | declared / 未驗 | 留在 DR unknowns |

3. **unknowns / 邊界**：接受以下邊界，不阻擋本機 development/testing 範圍：

- Cloud Run revision read-back 不在 service 內，P0 靠 operator script
- 真實 staging 部署與第一張真實 receipt 是 10/8 的工作，屬於同一個 DR 的下一步

## 邊界

本次人審接受的是 local development / testing scope，不是 staging 部署或 production release 授權；Cloud Run 相關 unknown 統一在 2026-10-08 處理。
