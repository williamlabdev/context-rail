# Change Decision Brief 與 Release Receipt 可讀性 Review

- Review target: `d89053ff6fe743fd94cd274f50a58bdcbe93f8a9`（develop；EB-012 的 CHG-001 實際渲染結果）
- Date: 2026-09-25
- Scope: CHG-001 的 Change Decision Brief（`change-decision-brief/v1`）、DEC-001 決策卡、prod-demo Release Receipt 畫面（`evidence/EB-009/prod-demo-receipt.png`，這張是 fixture 資料，不是實際部署結果）
- 對應目標: roadmap Week 2「role-specific report rendering」、DoD #5「human-readable Change Decision Brief」、DoD #11（語系）、G5 usability（3–5 次外部角色測試）
- Review identity: AI review；不取代 G5 的人類可讀性測試，也不取代 human review
- Follow-up branch: `feat/brief-readability`（處理下文第 1～3 點）

## 1. 結論

可讀性屬於 P0，不是 P1。現在的 Brief 在技術上是正確的，每一項事實都能追溯到 DecisionRecord，但它寫給的是機器和稽核者，不是做決策的人。Receipt 的狀況一樣，而且更嚴重。

## 2. 根本原因

同一份產出同時要給三種人看：做決策的人、審查或稽核的人、agent。

「renderer 不加事實」這條原則本身沒錯。問題在實作時被放大成「也不整理、不排序、不換說法」，結果畫面上呈現的就是資料結構本身。

**原則保留，但把邊界講清楚：renderer 可以挑選、排序、改名稱、翻譯；不能新增事實。**

## 3. Brief 逐段診斷（v1）

| 段落 | 問題 |
|---|---|
| 排序 | 第一段是兩串完整 sha256；真正的結論排在第 3 段，而且和 headline 重複 |
| 一句話結論 | `Proceed with "minimal_reversible_slice" in staging; verify in staging…`：直接露出 option ID，staging 出現兩次，也沒有說明核准的是什麼功能 |
| Why | 這段寫的是誰、何時、用什麼工具，不是理由。真正的理由在 request 裡，Brief 沒有帶出來 |
| Objective／核准範圍 | 用 API 的寫法（`POST /api/orders/{id}/evidence`），業主讀不懂；業務能力和 API 契約混在同一張清單 |
| 下一關條件 | 用 `single-operator-controls` 這類內部 ID；「任何變動會讓決策 STALE」描述的是系統機制，不是讀者要採取的行動 |
| **已接受的未知** | **最嚴重。** `project context is STALE…` 是真正的風險警示，卻排在倒數第三段，旁邊還寫著 `risk low`，讀者無法把兩者對上 |
| 誰決定 | 顯示 `founder-001` 和 UTC ISO 時間，沒有轉成人讀的格式 |
| 替代方案 | 內容不錯，但沒有說明為什麼沒選 |
| 語系 | 標題和內文都是 Go 裡寫死的英文，切到 zh-TW 後整份 Brief 仍是英文 |

## 4. Receipt／發布畫面診斷

- 整頁高約 3200px，同一組 11 個 gate 的表格重複出現 3 次，約 30 列都是 PASS。
- 整頁唯一重要的訊息被埋在中間：D01 因 `new_revision` BLOCKED 失敗，D02 才成功。
- 結論（Release Receipt）排在最底部。
- Gate 名稱用 snake_case，欄位窄時會從字中間斷開（`environ/ment`）。
- 翻譯只做了一半（「發布 Receipt」）。

## 5. 改版方向與狀態

| # | 方向 | 狀態 |
|---|---|---|
| 1 | Brief 依讀者的問題排序：白話摘要 → 路線 → 風險與接受的未知（有未知時以警示框呈現）→ 範圍內／外 → 下一關需要什麼 → 誰決定 → 未採用的方案；hash 與 lineage 收進「追溯資訊」折疊區 | `feat/brief-readability` 已實作 |
| 2 | 白話摘要由人撰寫：request 新增 `owner_summary`，缺少時回傳 `NEEDS_INPUT`（owner：requester），renderer 不自動產生。舊決策沒有這個欄位時，明確顯示「未提供」，並原封不動列出技術目標，不拿其他內容代替 | `feat/brief-readability` 已實作 |
| 3 | Renderer 只輸出結構化資料（`change-decision-brief/v2`）；前端負責翻譯標題和標籤，資料值不翻譯 | `feat/brief-readability` 已實作 |
| 4 | Receipt 結論優先：開頭一句話說明結果；下方只列非 PASS 的項目，PASS 合併成一行 | 未做 |
| 5 | 完整 gate 表與 hash 移到給審查者看的 Technical Report，讓「role-specific」真正成立 | 未做 |
| 6 | 驗收：找 2–3 位不同角色的人讀完後回答三個問題（核准了什麼？沒核准什麼？最大的風險是什麼？），結果記入 G5 | 未做；是第 1～3 點能否算完成的依據 |

## 6. 第 1～3 點實作後的已知限制

- EB-012 的 CHG-001 決策早於 `owner_summary`，因此 Brief 會顯示「未提供白話摘要」。這是刻意的設計：不回頭補寫歷史決策，要補應該由 requester 提出新版本。
- 「下一關」的證據標籤只涵蓋目前 topology 用到的 9 個代碼；未知代碼會直接顯示代碼本身。
- Brief v2 的 schema 已改變（`Headline`／`Sections` 已移除）。如果有外部消費者讀 v1 JSON，需要跟著調整；repo 內沒有其他消費者。
