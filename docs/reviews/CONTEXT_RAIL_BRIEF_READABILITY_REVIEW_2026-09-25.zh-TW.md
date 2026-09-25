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
| 4 | Receipt 結論優先：開頭一句話說明結果；下方只列非 PASS 的項目，PASS 合併成一行 | `feat/receipt-conclusion-first` 已實作（見第 7 節） |
| 5 | 完整 gate 表與 hash 移到給審查者看的 Technical Report，讓「role-specific」真正成立 | `feat/release-technical-report` 已實作（見第 8 節） |
| 6 | 驗收：找 2–3 位不同角色的人讀完後回答三個問題（核准了什麼？沒核准什麼？最大的風險是什麼？），結果記入 G5 | 未做；是第 1～3 點能否算完成的依據 |

## 6. 第 1～3 點實作後的已知限制

- EB-012 的 CHG-001 決策早於 `owner_summary`，因此 Brief 會顯示「未提供白話摘要」。這是刻意的設計：不回頭補寫歷史決策，要補應該由 requester 提出新版本。
- 「下一關」的證據標籤只涵蓋目前 topology 用到的 9 個代碼；未知代碼會直接顯示代碼本身。
- Brief v2 的 schema 已改變（`Headline`／`Sections` 已移除）。如果有外部消費者讀 v1 JSON，需要跟著調整；repo 內沒有其他消費者。

## 7. 第 4 點實作（Receipt 結論優先）

- 發布詳情最上方新增一句結論（`release-outcome`），依發布狀態顯示：已發布、已拒絕、部署失敗、被擋下、等待建置／核准／部署。句子只用已記錄的欄位組成，不新增事實。
- 如果先前有失敗的部署，結論下方列出**第一個未通過的 gate 和它的細節**，例如 D01 `new_revision` BLOCKED。這原本是 Receipt 頁上最重要、卻埋在中間的訊息。
- Release Receipt 卡片移到結論下方，不再放在頁面最底。
- 每張 gate 表只展開未通過的項目，PASS 收合成「N 項閘門中有 M 項通過」，點開後可以看到完整的列。原本的 row testid 保留在 DOM 中，稽核時仍能取得。
- 「發布 Receipt」改為「發布收據」。
- 高度參考：改版前 EB-009 的整頁截圖約 3200px；改版後，同一個 prod-demo 流程的發布詳情區塊約 1780px。兩者量的範圍不同，這個數字只能當大略參考。

仍有的限制：gate 的 detail、轉換名稱等由 server 產生的文字在 zh-TW 仍是英文；manifest、delta 等技術區塊仍和決策者的資訊放在同一頁，這部分屬於第 5 點。

## 8. 第 5 點實作（決策者摘要 vs. 技術報告）

- 發布詳情新增兩顆切換鈕（`release-view-summary` / `release-view-technical`），預設落在「摘要」。這是純前端的顯示切換，renderer 不因切換而新增或改變任何事實，只決定同一批資料要不要顯示、如何排列。
- **摘要視圖**（決策者預設看到的）留下：結論句（`release-outcome`，第 4 點的產出）、一段人讀的收據摘要（`release-receipt-summary`：上線到哪裡、部署到第幾個 revision、誰核准、何時核准——不含完整 sha256，最多顯示短 hash）、核准卡片（`release-approval`）、目前未通過的 gate（存活的和每次失敗部署各自的）、以及全部操作表單（建置／核准／部署／晉升）。表單刻意設計成兩個視圖都能用，不因切換而被鎖住。
- **技術報告視圖**新增：完整標頭（release id、transition、target、topology、config hash、manifest hash，`release-heading-technical`）、完整 Receipt 卡片（含完整 sha256）、晉升卡片與 Delta 表、MANIFEST 表與 build 資訊、完整 gate 表（含所有 PASS 列）、以及每一次部署嘗試（含已成功晉升的那次，摘要視圖只留非 PROMOTED 的嘗試）。
- 因為 `ReleaseDetail` 在每次建置／核准／部署動作後都會依 key 重新掛載（既有設計），視圖選擇不做 localStorage 記憶——每次動作完成都乾淨地回到摘要，行為可預期，也不需要额外處理跨動作的狀態一致性。這點屬於任務書允許的「optional」，選擇不做。
- 判斷取捨：`ReleaseOutcome` 結論句與核准卡片（`release-approval`）在兩個視圖都顯示，未嚴格照最初條列收進摘要限定——結論句是導向性的一句話，核准卡片對審查者同樣有用，沒有理由只留在技術報告。
- 已知限制：MANIFEST 表（這次發布包含哪些變更）完全收進技術報告，決策者在摘要視圖看不到「這次上線了哪些變更」的清單，需要切到技術報告才看得到；如果日後驗收（第 6 點）發現決策者確實需要這份清單，屬於下一輪要處理的項目。
- 驗證：`frontend/tests/components/releasesPanel.test.tsx`（新增，3 個案例，覆蓋摘要隱藏技術細節／技術報告顯示完整內容／摘要 gate 表只列未通過項）；既有的 `tests/browser/release-promotion.spec.ts`、`tests/browser/prod-demo-promotion.spec.ts` 已更新為在需要技術細節的斷言前先切換視圖，並新增摘要視圖不含完整 hash／manifest 的斷言。全數本地跑過，CI 未跑。
