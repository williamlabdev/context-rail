# EB-009 人類 Review Packet — VS-007 Prod-demo promotion

這份文件是給人快速閱讀的摘要；原始命令輸出、截圖與 mutation check 仍保留在同一個 Evidence Bundle（EB-009），方便追溯。

## Review result

| 欄位 | 結果 |
| --- | --- |
| Reviewed commit | `17c3fc7` (develop) |
| Decision | `DR-009 / ACCEPTED_FOR_DEVELOPMENT` |
| Human reviewer | `founder-001` |
| Role | `Founder + Solution Architect` |
| Reviewed at | `2026-09-21T04:28:20Z` |
| Human result | `ACCEPTED_FOR_LOCAL_DEVELOPMENT` |
| Review mode | 單一操作者 review（透過 Claude Cowork 逐項提問、由人裁決）；不重跑測試（CI 已跑：三個 job 綠）；不宣稱獨立 reviewer separation |

## 一句話結論

promotion release 繼承 staging receipt 的 digest 與凍結 lineage，列出 environment delta 並由新 approval 綁定，staging revision 不能搬到 prod-demo（new_revision），receipt 以 previous_receipt_id 串鏈；只有 staging / prod-demo 可執行，evidence-only 節點跳過；證據完整，部署同樣是 declared。

## 審了什麼

1. **方案與範圍**：`DR-009` 的 selected_option 與 scope 是否為 owner 要的 → 是。
2. **證據對驗收標準**：哪些是真驗過、哪些仍是 declared：

| 項目 | 狀態 | 出處 |
| --- | --- | --- |
| 11 個 release domain 測試（4 個新的） | 已驗 | 見 README.md 的 Result 表 |
| prod-demo promotion Playwright journey，連跑兩輪同一 state dir；翻出 uat 節點擋 promotion_order 的真 bug 並修正 | 已驗 | 見 README.md 的 Result 表 |
| consumer fixture 樹 hash 兩輪後相同 | 已驗 | 見 README.md 的 Result 表 |
| prod-demo deployment record 為 declared fixture；record-promotion.sh 未對 gcloud 驗 | declared / 未驗 | 留在 DR unknowns |

3. **unknowns / 邊界**：接受以下邊界，不阻擋本機 development/testing 範圍：

- 決策之後才加 prod-demo 節點會讓決策 STALE（by design）：10/8 的 runbook 必須先加節點再開 release；real-run state 的 topology v1 目前沒有 prod-demo，這一步要排進 10/8 流程（加節點 → DEC-001 STALE → 重新評估 / 重新決策 → 重發 AWO → 重新 read-back），或把 STALE 當成影片的一個 beat
- Cloud Run read-back 仍靠 operator script

## 邊界

本次人審接受的是 local development / testing scope，不是 staging 部署或 production release 授權；Cloud Run 相關 unknown 統一在 2026-10-08 處理。
