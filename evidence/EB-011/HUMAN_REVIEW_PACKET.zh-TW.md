# EB-011 人類 Review Packet — UI-15～19 Workspace persistence + locale

這份文件是給人快速閱讀的摘要；原始命令輸出、截圖與 mutation check 仍保留在同一個 Evidence Bundle（EB-011），方便追溯。

## Review result

| 欄位 | 結果 |
| --- | --- |
| Reviewed commit | `17c3fc7` (develop) |
| Decision | `DR-011 / ACCEPTED_FOR_DEVELOPMENT` |
| Human reviewer | `founder-001` |
| Role | `Founder + Solution Architect` |
| Reviewed at | `2026-09-21T04:28:20Z` |
| Human result | `ACCEPTED_FOR_LOCAL_DEVELOPMENT` |
| Review mode | 單一操作者 review（透過 Claude Cowork 逐項提問、由人裁決）；不重跑測試（CI 已跑：三個 job 綠）；不宣稱獨立 reviewer separation |

## 一句話結論

URL 綁 Project（/projects/{id}，refresh / back / deep link 不混 Project），英文 key + zh-TW 字典、localStorage 記住 locale、StatusBadge 人類標籤 + 機器 code 並排、資料值不翻譯；UI-15～19 有可執行證據。

## 審了什麼

1. **方案與範圍**：`DR-011` 的 selected_option 與 scope 是否為 owner 要的 → 是。
2. **證據對驗收標準**：哪些是真驗過、哪些仍是 declared：

| 項目 | 狀態 | 出處 |
| --- | --- | --- |
| UI-15 與 UI-16～19 Playwright journey，全套 13 條連跑兩輪 | 已驗 | 見 README.md 的 Result 表 |
| 字典覆蓋 unit test 掃描所有 t() key（現以 import.meta.glob 讀原始碼，CI typecheck 通過） | 已驗 | 見 README.md 的 Result 表 |
| consumer fixture 樹 hash 前後相同；/projects/* deep link 200 | 已驗 | 見 README.md 的 Result 表 |
| （無） | — | — |

3. **unknowns / 邊界**：接受以下邊界，不阻擋本機 development/testing 範圍：

- 伺服器產生的 gate detail / reason / audit 文字保持英文（影片切繁中時會看到中英混合，接受為技術細節）
- 只有 en 與 zh-TW

## 邊界

本次人審接受的是 local development / testing scope，不是 staging 部署或 production release 授權；Cloud Run 相關 unknown 統一在 2026-10-08 處理。
