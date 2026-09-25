# EB-013 人類 Review Packet — Change Decision Brief v2（可讀性第 1～3 點）

這份文件是給人快速閱讀的摘要；原始命令輸出與截圖在同一個 Evidence Bundle（EB-013）。

## Review result

| 欄位 | 結果 |
| --- | --- |
| Reviewed commit | `9f5b3ff`（feat/brief-readability） |
| Decision | `DR-012 / ACCEPTED_FOR_DEVELOPMENT` |
| Human reviewer | （待填） |
| Reviewed at | （待填） |
| Human result | **待審** |
| Review mode | （待填）；CI 未跑，證據為本地 go test／vitest／Playwright 13 條 |

## 一句話結論

Brief 改成照讀者的問題排序、開頭是人寫的白話摘要（缺少時 NEEDS_INPUT，不代寫）、多了一張路徑圖標出核准到哪一站、系統產生的方案與未知問題在中文版會顯示中文、hash 與 ID 收進折疊的追溯區；是否「讀得懂」尚未經讀者驗證。

## 請審的事

1. **方案與範圍**：DR-012 的 selected_option 與 scope 是否為 owner 要的？
2. **看截圖**：`brief-zh-TW.png`、`brief-en.png`。只看畫面，能不能回答：核准了什麼？沒核准什麼？最大的風險是什麼？
3. **證據對驗收標準**：

| 項目 | 狀態 | 出處 |
| --- | --- | --- |
| 缺 owner_summary → NEEDS_INPUT（requester） | 已驗（Go 測試） | `go-test-output.txt` |
| 白話摘要置頂、machine identity 只在追溯區 | 已驗（unit test 檢查 DOM 順序） | `frontend-quality-output.txt` |
| 舊決策顯示「未提供摘要」並原樣列出技術目標 | 已驗（Go + unit test） | 同上 |
| zh-TW 標題翻譯、資料值不變 | 已驗（字典覆蓋測試 + 截圖） | 同上、截圖 |
| 路徑圖依決策當時的 topology 畫出、標出核准終點；舊決策只畫兩站並註明 | 已驗（Go + unit test） | 同上 |
| zh-TW 下 advisor 方案／未知問題顯示中文，人寫的內容不變 | 已驗（unit test + 截圖） | 同上、截圖 |
| 全部 13 條 browser journey、fixture 未被改動 | 已驗 | `browser-output.txt`、`mutation-check.txt` |
| 讀者讀得懂（三問測試，G5） | **未驗** | — |

4. **unknowns／邊界**：是否接受以下邊界，不阻擋本機 development/testing：

- 可讀性只有結構與截圖證據，讀者測試（review 第 6 點）未做
- Receipt 結論優先與 Technical Report（review 第 4、5 點）不在本 slice
- EB-012 的 CHG-001 決策會顯示「未提供白話摘要」，不回頭補寫
- Brief JSON 從 v1 改為 v2
- 其他伺服器產生的文字（缺少輸入的原因、觀察、gate 細節）在中文版仍是英文
- 截圖資料：`brief-zh-TW.png` 用中文示範資料、在乾淨狀態下拍攝；`brief-en.png` 用英文示範資料，路徑中的 `uat-…` 是前面測試新增的環境。白話摘要與決策理由在兩張圖中都是原文照登

## 邊界

人審接受的是 local development / testing scope，不是 staging 部署或 production release 授權。
