# Handoff Guard Vision

Status: Draft for implementation  
Version: 1.0  
Last updated: 2026-09-10  
Product: Handoff Guard v2  
Positioning: **Agentic Release Readiness for Internal Web Systems**

## 1. Vision

讓每一次企業內部系統發布，都能被清楚解釋、被證據支持，並在保留人類責任的前提下安全地完成。

> **Every internal web system release should be explainable, evidence-backed, and safely promotable.**

Handoff Guard 的長期目標，是成為人類意圖與 production 現實之間的可信任控制層。它讓團隊不只知道「程式可以部署」，更知道：

- 這個版本是否真的實作了需求。
- 設計、程式碼、測試與實際執行狀態是否描述同一個系統。
- 權限、例外流程與錯誤狀態是否被驗證。
- 即將上線的 Cloud Run revision 是否就是已經被檢查與核准的版本。
- 誰根據哪些證據做了發布決策。

## 2. Mission

Handoff Guard 把分散在文件、設計、Git、CI、測試與 runtime 的交付證據，整理成一個可重跑、可追溯、可解釋的 release review。

它不要求企業替換既有工具，而是讓既有工具的輸出可以被放在同一個決策脈絡中。

## 3. The problem we choose to solve

企業內部 Web 系統的發布風險，通常不是 build 失敗，而是不同交付物之間逐漸產生落差：

- 需求文件描述了 requester、manager 或 admin 的規則，卻沒有轉成可驗證的測試。
- 設計稿只涵蓋 happy path，沒有 loading、empty、error 或 unauthorized 狀態。
- Feature branch 修改了欄位或權限，但 PR 沒有連回原始需求。
- CI 通過了單元測試，卻沒有證明拒絕流程與角色限制在 staging 中正常。
- 部署成功了，但團隊無法確認 production revision 是否就是已驗收的版本。

這不是「缺少另一個聊天機器人」的問題，也不是單純缺少一個 CI/CD 工具的問題。這是 **intent drift across artifacts**：人想交付的系統，逐步偏離了文件、設計、程式、測試與部署證據所共同描述的系統。

## 4. Who we serve

第一批使用者是負責內部 Web 系統交付與發布的團隊：

- Product Manager：需要知道需求是否真的被實作與驗收。
- QA Engineer：需要更快找出角色、例外流程與狀態覆蓋缺口。
- Software Engineer：需要在合併前知道還缺少哪些證據。
- DevOps／Platform Engineer：需要讓 promotion 綁定正確的 image、revision 與核准紀錄。
- Engineering Manager：需要對 production 風險做出可稽核的決策。

最適合的切入場景，是具有角色、狀態與審批流程的內部系統，例如內部申請、審核、權限管理與營運工作台。

## 5. Product promise

對一個即將發布的內部系統版本，Handoff Guard 必須能回答：

1. 這次要交付的需求是什麼？
2. 每項必要需求有哪些來源與驗收證據？
3. 哪些角色、狀態與例外流程已被驗證？
4. 哪些缺口會阻擋發布，為什麼？
5. 被核准的 commit、container image digest 與 Cloud Run revision 是否一致？
6. 發布後能否回讀結果並產生完整 release receipt？

產品承諾是：

> **一鍵完成已通過檢查、保留人工核准且可回溯的受控發布。**

這不是「一鍵繞過檢查的自動部署」，也不是讓 Agent 自行決定 production 是否可以被修改。

## 6. Product model

Handoff Guard 是既有工具之上的 release readiness control plane：

```text
Human intent
    ↓
Requirements · Design · Code · Tests · Runtime
    ↓
Adapters + normalized artifacts and evidence
    ↓
Bounded Gemini analysis
    ↓
Deterministic policy gate
    ↓
Human review and approval
    ↓
Controlled Cloud Run promotion
    ↓
Release receipt and traceable history
```

它的核心不是「支援最多整合」，而是讓同一個 release decision 可以回到：

`requirement → design → PR → CI build → staging evidence → approval → production revision`

## 7. The agentic boundary

Agentic workflow 是產品能力，但自主權必須被設計成有邊界的。

### Agents may

- 從文件與設計描述抽取需求、角色、狀態與驗收條件候選。
- 比對需求、PR、測試與 staging evidence，提出缺口。
- 找出可能的角色、權限、error state 或部署風險。
- 提供 remediation suggestion 與需要人工澄清的問題。

### Agents may not

- 自行把 release 改成 `APPROVED`。
- 自行跳過 policy gate 或必要測試。
- 執行任意 shell command、刪除資料或寄送外部訊息。
- 直接寫入 production 或自行決定 promotion target。
- 以沒有來源的信心分數取代證據。

### Authority model

```text
Gemini proposes → deterministic rules verify → human decides → controller executes
```

模型負責理解與提出候選；程式與平台規則負責驗證必要條件；人負責正式核准；受控 controller 才能執行發布。

## 8. Product principles

### 8.1 Evidence before confidence

任何 finding 都必須能回到來源、版本、段落、PR、build、revision 或測試結果。沒有 evidence reference 的模型推論只能是候選，不得直接成為發布依據。

### 8.2 Integrate before replace

Handoff Guard 不取代 Figma、Git、CI、Cloud Run 或專案管理工具。它先把這些工具的輸出連成決策，再透過 Adapter 擴展資料來源。

### 8.3 Bounded autonomy

Agent 可以加速理解、比較與找缺口，但不能取得未被授權的 production 權限。所有高風險動作都必須有固定輸入、清楚 target 與人工核准。

### 8.4 Make failure useful

`Blocked` 不是錯誤訊息的終點。每個阻塞項目都應該說明：缺少什麼、證據在哪裡、如何修正、修正後如何重新執行檢查。

### 8.5 Version identity matters

文件版本、commit SHA、image digest、Cloud Run revision、model ID、prompt version 與 approval record 必須能互相對應。沒有版本身份的「看起來成功」不算完成。

### 8.6 Start with one complete vertical slice

先完成一個可重現的 approval workflow，再擴展整合數量。三週 MVP 的價值在於證明從需求到 production 的完整鏈路，而不是列出尚未完成的 connector 清單。

## 9. North-star experience

使用者應該能在同一個 Release Review 中看到：

1. 本次 release 的需求與來源。
2. 相關設計、PR、測試與 Cloud Run staging 證據。
3. Gemini 找到的缺口與每個 finding 的引用。
4. 確定性 policy gate 的結果。
5. `Blocked`、`Needs Review` 或 `Ready` 的原因。
6. 人工核准與 promotion 的責任紀錄。
7. production smoke test、revision 與 release receipt。

理想的體驗不是「AI 幫我按下發布」，而是：

> **我可以快速理解風險，知道哪些證據支持這個版本，並放心地對發布決策負責。**

## 10. Strategic scope

### Near-term: the three-week MVP

三週內只承諾一條完整垂直切片：

- 一個內部 approval workflow。
- 本機文件上傳或文字輸入。
- 一個可選的設計 snapshot。
- Feature branch、PR metadata 與 Cloud Build。
- Cloud Run staging 與 production。
- Requirement、QA、DevOps 三個 bounded agents。
- Readiness gate、人工核准、受控 promotion 與 release receipt。

### Next horizon

- Google Drive／Google Docs 唯讀 Adapter。
- Figma frame 或 export Adapter。
- Cloud Deploy pipeline、approval 與 Cloud Run target。
- Candidate tag、final tag、PR comment 與 remediation task。
- 更多 holdout cases 與跨團隊 release history。

### Long-term direction

- SharePoint、Confluence、內網檔案伺服器與企業私有連接器。
- 多 repo、多服務與跨環境 release orchestration。
- 企業 SSO、SCIM、政策管理與稽核報告。
- 可由企業治理的 Adapter SDK、policy library 與 evidence graph。

## 11. Explicit non-goals

Handoff Guard 不會成為：

- 通用專案管理平台。
- 完整 CI/CD 取代品。
- 只回答文件問題的 Gemini chatbot。
- 讓 Agent 自主修改 production 的自動化平台。
- 三週內完成所有企業資料源與私有網路的整合工程。

如果一項功能不能改善「內部 Web 系統 release decision 的證據、理解或安全性」，它不應該進入 MVP。

## 12. Success definition

### Product success

- 使用者可以從一份需求建立完整 Release Review。
- 系統可以在故意植入缺口時產生可解釋的 `Blocked` finding。
- 使用者修正缺口後可以重新分析並得到 `Ready`。
- 只有通過確定性檢查與人工核准，才可以執行 production promotion。
- 每次發布都有完整 release receipt，可回到需求、commit、digest、revision、approval 與結果。

### Prototype targets

以下是 MVP 調校目標，不是事先宣稱的成果：

- 至少 90% 的必要需求具備 evidence reference。
- 至少 80% 的預先植入缺口被阻擋或標記。
- False blocking 不高於 20%。
- 相對人工檢查，review time 減少至少 30%。
- Unauthorized promotion 為 0 件。
- Release receipt completeness 為 100%。

### Demonstration success

評審看完三分鐘展示後，應該能清楚理解三件事：

1. Handoff Guard 解決的是 artifact drift，而不是再做一個聊天介面。
2. Gemini 真的參與了產品流程，但沒有取代人類權限與確定性檢查。
3. 這是一條真的走到 Cloud Run 的 release workflow，而不是只停留在 mockup。

## 13. Business thesis

Handoff Guard 的商業價值，不在於把所有既有工具重新包裝，而在於替高風險的 release decision 建立證據與責任邊界。

初期商業假設：

- 客戶是有正式發布流程的內部 Web 系統團隊，而不是只需要一般文件問答的使用者。
- Pilot 以一個產品團隊、一條 release pipeline 與有限案例建立 design partner。
- Team 方案依 active product team、release volume 或使用量收費。
- Enterprise 方案提供私有 connector、SSO、稽核、政策與導入支援。

這些是需要透過 design partner 驗證的假設，不是已驗證的市場事實。

## 14. Technology decisions that serve the vision

技術選擇必須服務產品願景，而不是反過來讓願景被工具綁住：

- **Gemini／Vertex AI**：產品執行時的文件理解、需求抽取、QA 與 DevOps 分析。
- **Cloud Run**：展示與部署的實際 runtime，支援 staging、production 與受控 promotion。
- **Cloud Build／Artifact Registry**：建置、測試、container 與不可變 image identity。
- **Firestore／Cloud Storage**：保存 project、artifact、evidence、check、approval 與 receipt。
- **Firebase Authentication／Secret Manager**：身份、角色與秘密管理。
- **Adapter contracts**：讓 Google Workspace、本機文件、Git、CI、runtime 與未來私有系統可替換。
- **Claude Code／Codex 等 coding assistant**：開發輔助工具，不等同於產品執行時的 AI 能力。

## 15. Decision filter

後續任何功能、整合或設計提案，都要先回答：

1. 它改善了哪一個 release decision？
2. 它增加了哪一種可驗證 evidence？
3. 它是否讓使用者更容易理解 blocker 與修正路徑？
4. 它是否保留 deterministic gate 與 human authority？
5. 它能否在不破壞 P0 垂直切片的情況下交付？
6. 它是否需要新的高風險權限、資料暴露或不可逆操作？

若答案不清楚，功能應先停留在探索，不應直接進入 MVP。

## 16. Relationship to the project plan

本文件定義「為什麼做、服務誰、哪些原則不可違反」。

詳細的三週排程、P0／P1／P2 範圍、Business Model Canvas、競爭分析、GCP 元件與交件策略，請參考：

- [Handoff Guard v2 Project Plan](docs/planning/HANDOFF_GUARD_PROJECT_PLAN_V2.zh-TW.md)

目前 v1 計畫保留為歷史基線，不作為 v2 的實作範圍：

- [Handoff Guard v1 Project Plan](docs/planning/HANDOFF_GUARD_PROJECT_PLAN.zh-TW.md)

## 17. One-sentence test

如果 Handoff Guard 最終只能讓團隊「更快地部署」，但不能讓團隊更清楚地知道「為什麼這個版本可以安全地部署」，那它就沒有完成自己的願景。
