# ADR-001：以結構化 Agent Handoff 取代 P0 自主 Agent-to-Agent 協作

- Status: Accepted for P0 architecture direction
- Date: 2026-09-13
- Scope: ContextRail P0 and the Project Context Contract template

## Context

目前的 demo 已經有 Context／Architecture、Engineering、Evidence 與 Release Policy 等角色。若現在直接把它們做成可自主對話的 agents，會同時引入協定、身份驗證、重試、失敗恢復、權限邊界與可觀測性問題，卻還沒有足夠的成熟度或實際 interoperability 需求來證明這些複雜度是必要的。

ContextRail 的核心價值是可追溯的 project context、決策與 delivery evidence，而不是 agent 對話本身。因此需要先驗證角色之間的責任邊界與 evidence lineage。

## Decision

P0 採用 artifact-to-artifact 的結構化 Agent Handoff。各角色以 versioned artifact 交接，不以自由對話互相授權：

- Context／Architecture Agent：產生 context 與影響評估；
- Engineering Agent：只執行已接受的 `AgentWorkOrder`；
- Evidence Agent：整理並核對 implementation、test、review 與 deployment evidence；
- Release Policy Agent：依 deterministic policy 評估是否可進入下一環境。

每個 handoff 至少包含：

`project_id`、`request_id`、`decision_id`、`source_snapshot_hash`、`policy_version`、`input_refs`、`output_artifact`、`evidence_refs`、`status`、`human_gate`。

P0 不實作 autonomous Agent-to-Agent orchestration，也不把 A2A 設為核心路徑的前置條件。A2A 保留為條件式 P1 extension，只有在跨獨立 agent、provider 或 runtime 的任務真的需要 interoperability，且能以 evidence 證明導入收益時才重新評估。即使導入，A2A 也只負責 task／artifact interoperability；ContextRail 仍擁有 `DecisionRecord`、`EvidenceBundle`、authorization、policy 與 human gate。

## Consequences

正面結果：

- 可以先用單一 runtime 驗證責任分工、可重播 handoff 與 evidence lineage；
- 減少 P0 的協定與分散式失敗模式；
- agent 輸出不會直接變成 project fact、權限或 deployment authorization；
- 未來若需要 A2A，可以把現有 handoff schema 作為 adapter 的穩定邊界。

代價與限制：

- P0 不展示 agent 之間的即時協商或自主協作；
- 需要額外保存 handoff artifact 與其 source／policy references；
- 目前的交接效率可能低於自由對話，但可追溯性優先於對話便利性。

## Reconsideration gate

只有在下列條件同時具備時，才建立 A2A P1 spike：

1. 至少兩個獨立 runtime、provider 或 agent service；
2. 一個無法由本地 artifact handoff 合理處理的跨邊界任務；
3. 明確的 task／artifact identity、authentication、authorization、timeout、retry 與 failure semantics；
4. 能比較導入前後的 interoperability、可靠性與治理成本；
5. 不改變 human gate、policy ownership 與 evidence acceptance boundary。
