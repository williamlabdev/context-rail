# EB-005 人類 Review Packet — VS-003 Environment Topology

這份文件是給人快速閱讀的摘要；原始命令輸出、截圖與 mutation check 仍保留在同一個 Evidence Bundle（EB-005），方便追溯。

## Review result

| 欄位 | 結果 |
| --- | --- |
| Reviewed commit | `17c3fc7` (develop) |
| Decision | `DR-005 / ACCEPTED_FOR_DEVELOPMENT` |
| Human reviewer | `founder-001` |
| Role | `Founder + Solution Architect` |
| Reviewed at | `2026-09-21T04:28:20Z` |
| Human result | `ACCEPTED_FOR_LOCAL_DEVELOPMENT` |
| Review mode | 單一操作者 review（透過 Claude Cowork 逐項提問、由人裁決）；不重跑測試（CI 已跑：三個 job 綠）；不宣稱獨立 reviewer separation |

## 一句話結論

環境拓撲以不可變版本 + JSON state 管理，material 變更讓該 Project 的觀察到決策 STALE，production 不能 retire / 改 type，expected_version 衝突回 409；UI-05 全程有可執行證據。

## 審了什麼

1. **方案與範圍**：`DR-005` 的 selected_option 與 scope 是否為 owner 要的 → 是。
2. **證據對驗收標準**：哪些是真驗過、哪些仍是 declared：

| 項目 | 狀態 | 出處 |
| --- | --- | --- |
| 18 個 topology Go 測試 | 已驗 | 見 README.md 的 Result 表 |
| UI-05 Playwright journey：add → edit → reorder → retire → restore（v2…v6）+ 版本衝突處理 | 已驗 | 見 README.md 的 Result 表 |
| consumer fixture 樹 hash 前後相同 | 已驗 | 見 README.md 的 Result 表 |
| （無） | — | — |

3. **unknowns / 邊界**：接受以下邊界，不阻擋本機 development/testing 範圍：

- P0 一次 material 變更會讓該 Project 所有觀察到的決策 STALE，不是只影響用到該環境的決策（保守設計，接受）
- X-ContextRail-Actor 只記錄不驗證
- JSON 檔 state，無 Firestore

## 邊界

本次人審接受的是 local development / testing scope，不是 staging 部署或 production release 授權；Cloud Run 相關 unknown 統一在 2026-10-08 處理。
