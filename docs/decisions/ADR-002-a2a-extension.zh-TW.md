# ADR-002：修訂 ADR-001 的 A2A P1 reconsideration gate（提案）

- Status: Proposed（待 William 審查）
- Date: 2026-09-26
- Scope: ContextRail P1 方向；修訂 ADR-001 的 A2A reconsideration gate，不改動 ADR-001 本文
- Amends: [ADR-001](ADR-001-structured-agent-handoff.zh-TW.md) 的 Reconsideration gate 一節

## Context

ADR-001（2026-09-13）決定 P0 以 artifact-to-artifact 的結構化 Agent Handoff 取代自主 A2A，並訂了五個條件同時具備才重新評估 A2A 的 gate。本 ADR 不推翻那個決定，而是檢視：現在浮現的兩個使用情境，是否已經觸及 gate，以及若要動手，動手的邊界在哪裡。

### 為何現在重新檢視

EB-012（`evidence/EB-012/README.md`）記錄了一次真實跑通的 handoff chain（`AWO-001` → `ARR-002` → `CAND-001`/`CAND-002`）。其中「獨立 review」目前是由**同一個 provider 內的不同 actor**（`claude-cowork` 執行、`claude-cowork-reviewer` 審查）完成，透過 GitHub PR review 留痕，`independent_review` 這一 gate 才判 `APPROVED`；沒有這個 review，`CAND-001` 停在 `NEEDS_REVIEW`（見 `internal/change/candidate.go` 的 `VerdictNeedsReview`／`GateNeedsReview`）。這揭露了兩個尚未被 ADR-001 涵蓋的情境：

- **UC1 跨 provider 獨立 review**：Codex 執行 `AgentWorkOrder`，ContextRail 把 review 任務派給另一個 provider／runtime 的 agent（例如 Claude）。這與 EB-012 的差別在於：兩個 runtime **不共享身分、不共享同步呼叫堆疊**——review 是非同步的、可能逾時、可能重試，而且審查結果要能綁定 `work_order_hash` / commit 才能算作 G3 的獨立 review 證據，否則等於是自報家門。本地 artifact handoff（單一 process 內把檔案傳給下一步）處理不了「另一個 runtime 什麼時候完成、完成了沒、要不要重試」這件事——這正是 ADR-001 條件 3 點名的 identity／timeout／retry／failure semantics，而不是 review 本身的邏輯。
- **UC2 範圍外升級**：Engineering agent 撞到 `allowed_paths` 或 `forbidden_actions` 之外的東西時（`demo/order-operations-portal/AGENTS.md` 的 Escalate 一節：「Stop and request human input」），現在的動作是**單純停下**。若這個 agent 是透過 A2A 呼叫的外部 runtime，「停下」在協定層需要一個可辨識的狀態，而不是連線掛掉或無聲逾時；A2A 規格提供 `input-required`（見下文）剛好對應這個語意。ContextRail 收到這個狀態後，要把它轉成一個 `NEEDS_INPUT` 的人工決策請求；只有人接受之後，才發一張**新的** `AgentWorkOrder`，agent 才恢復動作。Agent 自己不能擴權，擴權的決定權留在人。

兩個情境都指向同一個介面問題：**agent 對 ContextRail 之外的邊界要用什麼協定講話**，跟 ADR-001 決定的「ContextRail 內部角色間不自主對話」並不衝突——UC1/UC2 談的是 ContextRail 邊界之外的另一個 runtime，不是把 Context／Engineering／Evidence／Release Policy 四個角色改成互相對話。

### A2A 規格現況（研究依據）

查核時間：2026-09-26。來源：

- [A2A Protocol Specification (latest)](https://a2a-protocol.org/latest/specification/)（Linux Foundation 治理，[a2aproject/A2A](https://github.com/a2aproject/A2A)）
- 目前發佈版本：**v1.0.0**（規格文件頂端標示 "Latest Released Version 1.0.0"；文件未標示明確的發佈日期，只能確認查核當天狀態）
- [A2A GitHub repo](https://github.com/a2aproject/A2A)

確認要點：

- **Agent Card**：由 A2A Server 發佈的 JSON metadata；標準發現路徑 `https://{server_domain}/.well-known/agent-card.json`。欄位含 `name`、`description`、`url`／`supportedInterfaces`、`provider`、`capabilities`（含 `pushNotifications`、`streaming`、`extendedAgentCard`）、`skills`、`securitySchemes`、`security`、`version`、`documentationUrl`。認證過的 client 可呼叫 `GetExtendedAgentCard` 取得更完整（可能含額外 skills／capabilities）的卡片。
- **傳輸綁定**：JSON-RPC 2.0、gRPC、HTTP+JSON/REST 三種，規格要求三者語意上「functional equivalence」。
- **核心方法**（JSON-RPC 命名）：`SendMessage`、`SendStreamingMessage`、`GetTask`、`ListTasks`、`CancelTask`、`SubscribeToTask`、`CreateTaskPushNotificationConfig`、`GetTaskPushNotificationConfig`、`ListTaskPushNotificationConfigs`、`DeleteTaskPushNotificationConfig`。
- **Task 狀態**（`TaskState` enum，共 8 個）：`TASK_STATE_SUBMITTED`、`TASK_STATE_WORKING`、`TASK_STATE_INPUT_REQUIRED`、`TASK_STATE_AUTH_REQUIRED`、`TASK_STATE_COMPLETED`、`TASK_STATE_FAILED`、`TASK_STATE_CANCELED`、`TASK_STATE_REJECTED`。前兩者與 `INPUT_REQUIRED`／`AUTH_REQUIRED` 是可續行的中斷狀態，其餘為終態。
- **Message／Part／Artifact 模型**：`Message` 含 `messageId`、`contextId`、`taskId`、`role`（`ROLE_USER`／`ROLE_AGENT`）、`parts`、`metadata`、`extensions`、`referenceTaskIds`；`Part` 是文字、檔案參照或結構化資料三者之一；`Artifact` 是 task 處理產出的結果，由多個 `Part` 組成，與訊息歷史分開存放，並與某個 `Task` 綁定。
- **認證宣告**：Agent Card 的 `securitySchemes`（API Key、HTTP Bearer、OAuth2、OpenID Connect、mTLS 等）＋ `security` 指出哪些 scheme 適用；伺服端 MUST 拒絕缺少或無效憑證的請求，MUST NOT 洩漏未授權資源的存在。

## 五個 reconsideration 條件逐項檢視

以 UC1＋UC2 對照 ADR-001 的五個條件：

| # | 條件 | 判定 | 說明 / 尚缺證據 |
| --- | --- | --- | --- |
| 1 | 至少兩個獨立 runtime、provider 或 agent service | **符合** | UC1 明確要求「Codex 執行、另一 provider（如 Claude）審查」；EB-012 目前的 review 仍是同 provider 不同 actor，尚未真的跨 provider 跑過一次。 |
| 2 | 一個無法由本地 artifact handoff 合理處理的跨邊界任務 | **符合** | UC1 的非同步、逾時、重試與跨 runtime 身分問題，本地 artifact handoff（單 process 內傳檔案）處理不了；UC2 的「停下並升級」若對方是外部 runtime，也需要協定層可辨識的中斷狀態，不只是掛掉連線。 |
| 3 | 明確的 task／artifact identity、authentication、authorization、timeout、retry 與 failure semantics | **部分符合** | A2A 規格本身把這些定義清楚（securitySchemes、TaskState、方法集）；但 ContextRail 這一側**尚未**訂出自己的預設值（誰簽發身分、逾時多久、重試幾次、失敗要不要自動降級成 `BLOCKED`）。規格提供框架，不等於 ContextRail 已經決定怎麼用它——見下文 Stage B 的 UNKNOWN 清單。 |
| 4 | 能比較導入前後的 interoperability、可靠性與治理成本 | **未符合** | 目前沒有「本地 handoff」與「A2A handoff」的並排量測（延遲、失敗率、人工介入次數、治理稽核成本）。EB-012 只驗證了本地／同 provider 路徑，沒有跨 provider 的對照組。這是五個條件裡缺口最大的一項，必須先有 Stage A 的持久化 handoff 上線一段時間累積基準值，才有東西可比。 |
| 5 | 不改變 human gate、policy ownership 與 evidence acceptance boundary | **符合（設計要求，非既成事實）** | UC1／UC2 的設計本身要求 A2A 訊息不能取代人的決定；本 ADR 把它列為 Stage B 的 Authority rule 強制要求，但這是「打算怎麼做」，不是「已經驗證過」。 |

**總結**：條件 1、2、5 在設計意圖上成立；條件 3 有規格但缺 ContextRail 側的預設值；條件 4 完全空白。**尚不滿足「同時具備」的門檻**——這正是本 ADR 只到 Proposed、且明確分兩階段、Stage B 不承諾時間表的原因。

## Decision（提案）

分兩階段推進，Stage A 現在就可以做、且不依賴 A2A；Stage B 是條件補齊後才啟動的方向，本 ADR 不承諾實作時間。

### Stage A — 先把 ADR-001 的 handoff 做成真正持久化的 artifact + API

在 A2A 存在與否無關的前提下，把 ADR-001 定義的欄位（`project_id`、`request_id`、`decision_id`、`source_snapshot_hash`、`policy_version`、`input_refs`、`output_artifact`、`evidence_refs`、`status`、`human_gate`）做成一個真實的 `AgentHandoff` 物件：

- 落地為持久化 artifact（比照 `internal/change/model.go` 的 `AgentContextPack`／`AgentWorkOrder`、`internal/change/candidate.go` 的 `AgentRun`／`Candidate` 的做法：有 `schema_version`、有 hash、有 lineage）；
- 有對應的 API（比照 `internal/http/changes.go` 現有的 `/v1/projects/{project}/changes/{change}/work-order` 模式，增加對稱的 handoff 端點）；
- lineage 綁回既有的 `DecisionRecord` 與 `AgentWorkOrder`，可用 `work_order_hash` 追溯。

Stage A 的價值獨立於 A2A 是否啟用：它是 UC1／UC2 兩個情境未來要接的**穩定介面**，也是條件 4 需要的量測基準來源。

### Stage B — A2A adapter（僅在條件補齊後）

在 ContextRail 邊界上加一層 A2A adapter，而不是把 A2A 揉進內部角色之間：

**狀態對照表（草案，未實作）**

| A2A `TaskState` | `AgentHandoff.status`（Stage A） | ContextRail 內部狀態 |
| --- | --- | --- |
| `TASK_STATE_SUBMITTED` | `DISPATCHED` | Change 停在目前 gate，等待 |
| `TASK_STATE_WORKING` | `IN_PROGRESS` | 不變 |
| `TASK_STATE_INPUT_REQUIRED` | `NEEDS_INPUT` | 轉成人工決策請求（UC2 核心映射）；比照既有 `StatusNeedsInput`（`internal/change/model.go:29`） |
| `TASK_STATE_AUTH_REQUIRED` | `NEEDS_INPUT`（附認證原因） | 同上，額外標注是認證缺口而非範圍缺口 |
| `TASK_STATE_COMPLETED` | `COMPLETED` | 進入 candidate gate 評估（比照 `GateNeedsReview`／`VerdictNeedsReview`） |
| `TASK_STATE_FAILED` | `FAILED` | 視政策可能轉 `BLOCKED`（比照 `GateBlocked`） |
| `TASK_STATE_CANCELED` | `CANCELED` | Change 保留原狀態，不視為完成 |
| `TASK_STATE_REJECTED` | `REJECTED` | 視為 agent 主動拒絕，需人工檢視原因 |

此表是**設計方向**，欄位與轉換規則未經任何程式碼或測試驗證——標記 UNKNOWN 的部分見下。

**ContextRail 對外暴露的介面（草案）**：以 Agent Card 宣告 `skills`，例如 `review-candidate`（對應 UC1，輸入 `work_order_hash` + 待審 artifact，輸出綁定同一個 hash 的 review 結果）、`request-scope-escalation`（對應 UC2，agent 端呼叫的方向相反：ContextRail 是接收方，把 A2A 的 `input-required` 轉成 `NEEDS_INPUT`）。ContextRail 呼叫出去的方向（作為 A2A client 去呼叫外部 review agent）與被呼叫的方向（作為 A2A server 接受 escalation）都要支援，兩者都還沒設計细節。

**Authority rule（不可退讓）**：

- A2A 訊息、Task 或 Artifact **永遠不能**直接授權任何動作、變成 project fact，或跳過 `human_gate`／policy。
- `DecisionRecord`、`EvidenceBundle`、authorization、policy 的 ownership 全部留在 ContextRail，與 ADR-001 決定的邊界一致。
- 外部 agent 透過 A2A 回報的任何結果（包含 review APPROVED、包含 completed task），在 ContextRail 內都只是**一筆待評估的 observation**，要經過既有 gate 邏輯（如 `independent_review` gate）才能影響 verdict——不因為走了 A2A 協定就自動具備權威性。

**條件 3 的具體預設值（草案，多處 UNKNOWN）**：

- Identity：外部 agent 的身分如何登記與驗證——**UNKNOWN**（現有 `X-ContextRail-Actor` header 模式是「記錄但不驗證」，EB-012 明確標注這是 P0 邊界；Stage B 若要用 A2A 的 `securitySchemes` 做真驗證，需要另外決定信任模型）。
- Timeout：跨 provider review task 的逾時時限——**UNKNOWN**（沒有先例可抄；需先有 Stage A 累積的真實耗時分佈）。
- Retry：逾時或 `FAILED` 之後要不要自動重試、重試幾次——**UNKNOWN**；預設方向傾向不自動重試，逾時直接轉人工決策，避免重試把失敗語意搞模糊。
- Idempotency：同一個 `work_order_hash` 對應的 review 請求重複送達時如何去重——**UNKNOWN**；`AgentHandoff` 的 `request_id` 是 Stage A 就要有的去重鍵，但去重邏輯本身待設計。
- Failure semantics：A2A `FAILED`／`REJECTED` 要不要直接讓對應 gate 判 `BLOCKED`，還是先進 `NEEDS_REVIEW` 讓人判斷——**傾向後者**（比照 EB-012 的 `NEEDS_REVIEW` 而非直接 `BLOCKED`），但未定案。

## Timing

**本 ADR 描述的 Stage B（A2A adapter）在 Google Cloud AI Builder Cup 原型提交截止前不會實作**。截止日期：2026-10-18（依官方網站 https://aibuildercup.com/ ，查核於 2026-09-26；時區 **UNKNOWN**，官網未標示）。

註：`docs/planning/HANDOFF_GUARD_PROJECT_PLAN.zh-TW.md` 第 40 行記載的 9/29 內部完成日與 10/4 官方截止日已過期，將由另一份 PR 更新，本 ADR 不引用該處作為時程依據。

提交材料若要提及本 ADR，只能描述為「提案方向」，不得描述為已實作或已驗證的功能。Stage A（持久化 `AgentHandoff`）是否在截止前完成，是另一個獨立的實作排程決定，不在本 ADR 範圍內。

## Consequences

正面：

- Stage A 讓 ADR-001 的 handoff 從「文件裡的欄位清單」變成可查詢、可追溯的真實 artifact，即使永遠不做 Stage B 也有獨立價值；
- 條件 4 所需的量測基準（延遲、失敗率、人工介入次數）在 Stage A 上線後才第一次有東西可以量；
- Authority rule 把「A2A 只處理 interoperability，權威留在 ContextRail」寫成可稽核的設計約束，延續 ADR-001 的邊界，不是重新開一個口子。

代價：

- Stage A 增加一個新的持久化物件與 API surface，需要額外的 schema、儲存與測試維護成本；
- Stage B 一旦啟動，条件 3 列出的每一個 UNKNOWN 都是真實的實作與治理工作量，目前完全沒有估時；
- 對外揭露 Agent Card／skills 之後，ContextRail 自己也變成一個「被呼叫的 agent」，需要承擔規格要求的伺服端義務（拒絕無效憑證、不洩漏未授權資源存在），這是目前系統沒有的責任面。

風險：

- 若在條件 4 的量測完成前就啟動 Stage B，會重演 ADR-001 想避免的狀況：引入協定與分散式失敗模式，卻沒有證據證明值得；
- UC2 的「agent 自己判斷什麼算範圍外」若沒有和 `forbidden_actions`／`allowed_paths` 的既有機制對齊，可能出現「A2A 层级的升級」與「Work Order 层级的邊界」兩套定義互相打架。

## 給 William 的待決問題

1. Stage A 的 `AgentHandoff` 是否要在本次 Cup 提交前完成，還是列為賽後项目？（本 ADR 不預設答案，只確認它與 Stage B 解耦。）
2. UC1 的第一個真實跨 provider 對象要選哪個？（決定 Agent Card／`securitySchemes` 的第一版信任模型要對誰設計。）
3. 條件 4 的量測要不要現在就開始收集（即使還沒有 A2A，先量 EB-012 這類同 provider handoff 的延遲/失敗率當作 baseline）？

## Acceptance checklist（Proposed → Accepted）

- [ ] Stage A 的 `AgentHandoff` schema 與至少一個真實 lineage 範例（比照 EB-012 的做法）存在於 evidence 目錄
- [ ] 條件 4 有至少一組 before/after 或 baseline 量測數字，不是估計值
- [ ] 條件 3 的五個 UNKNOWN 至少收斂為明確決定或明確「暫不支援」
- [ ] Authority rule 的falsifiable 測試案例（例如：外部 agent 回報 completed，驗證 ContextRail 不會自動 accept，仍停在既有 gate）已設計或已寫成測試
- [ ] William 對三個待決問題給出方向

## 參考

- [A2A Protocol Specification (latest)](https://a2a-protocol.org/latest/specification/)
- [A2A Protocol Specification v1.0.0](https://a2a-protocol.org/v1.0.0/specification/)
- [a2aproject/A2A GitHub repository](https://github.com/a2aproject/A2A)
- [a2aproject/A2A specification.md](https://github.com/a2aproject/A2A/blob/main/docs/specification.md)
- [Google Cloud AI Builder Cup 官方網站](https://aibuildercup.com/)（截止日查核於 2026-09-26）
- [ADR-001：以結構化 Agent Handoff 取代 P0 自主 Agent-to-Agent 協作](ADR-001-structured-agent-handoff.zh-TW.md)
- `evidence/EB-012/README.md`（跨 actor 獨立 review 的真實跑通紀錄）
- `internal/change/model.go`（`AgentContextPack`、`AgentWorkOrder`）
- `internal/change/candidate.go`（`AgentRun`、`Candidate`、gate／verdict 常數）
- `internal/http/changes.go`（`work-order` 端點模式）
- `demo/order-operations-portal/AGENTS.md`（Escalate 一節）
