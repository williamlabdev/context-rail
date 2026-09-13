# ContextRail Project Context Contract

Status: proposed P0 project contract  
Version: 1.0  
Updated: 2026-09-13  
Scope: startup／SMB-first prototype; enterprise-ready extension remains later


## 1. 這是什麼

Project Context Contract 定義一個 Project 在交給人、RAG、coding agent 或 release gate 之前，至少要具備哪些可讀、可驗證、可追溯的脈絡。

它不是要求所有 AI 專案都使用同一棵目錄樹，也不是把所有文件都塞進一個 `AGENTS.md`。它是 ContextRail 用來判斷「這個 Project 是否有足夠資料可以做決策或開始開發」的最小契約。

最重要的邊界如下：

- Project 是治理邊界，不是 repository 的同義詞。
- 文件是來源；RAG index 與 AI Context Pack 是衍生物。
- 缺少資料時顯示 `NEEDS_INPUT`、`UNKNOWN` 或 `STALE`，不能由模型補成事實。
- 一份 Project 可以連結多個 repository；文件中的 repo、service、path 與 environment 必須能被明確對應。
- `README.md`、架構文件與 Agent 指令可以互相引用，但不能各自成為互相矛盾的真相源。

## 2. 它是不是 AI 專案的業界標準

目前沒有一個已被普遍採用、同時涵蓋產品意圖、架構、執行命令、Agent 行為、文件索引、決策、證據與環境發布的單一標準。

可以觀察到的是幾個互補的公開模式：

| 公開模式 | 它解決的問題 | 與 ContextRail 的關係 |
| --- | --- | --- |
| Spec-driven workflow | 將意圖依序變成 Spec、Plan、Tasks、Implementation | 對應 Request／Delivery Item 的切分；不能單獨表示 runtime、權限或 release evidence。 |
| `AGENTS.md`／`CLAUDE.md`／`GEMINI.md` | 告訴 coding agent 專案規則、命令與限制 | 對應 Agent 行為層；不是 Project 的完整 SSOT，也不應保存決策與 secret。 |
| 社群 spec／agent repo template | 將規格、skills、hooks 與 agent 指令放進 repo | 可借用其組織方式；仍需自行定義 Project、證據、權限與 release gate。 |
| AI／Cloud starter pack | 快速建立 agent runtime、CI/CD、evaluation、observability 與 deployment | 對應執行基礎；不負責判斷一次需求變更是否符合被接受的意圖。 |
| Backstage `catalog-info.yaml` | 統一服務、owner、system 與 metadata 的目錄模型 | 可作為 Repository／Service discovery 的來源之一；不取代 ContextRail 的 Change、Decision、Evidence 與 promotion gate。 |

因此本文件的定位是：**一個 AI-assisted software project 的可採用參考模板與治理契約，不是宣稱全業界必須遵循的標準。**

## 3. P0 建議目錄

以下是 ContextRail 第一版建議的最小結構。既有專案不需要一次搬檔；可以先以 Project manifest 登記現有路徑，再逐項補齊。

```text
project.yaml                         # 結構化 Project contract；SSOT
README.md                            # 人類入口與快速開始
AGENTS.md                            # 跨工具的 Agent 行為與安全邊界
CLAUDE.md                            # 可選，Claude-specific bridge，不另立政策
GEMINI.md                            # 可選，Gemini-specific bridge，不另立政策
vision.md                            # 為什麼做、服務誰、不做什麼
architecture.md                      # 系統邊界、資料流、依賴與重要限制
docs/
  engineering/
    development.md                  # 本機開發、測試、build 與 debug 命令
  operations/
    environments.md                 # 環境 topology、target 與 promotion 規則
  governance/
    policies.md                     # 必要 review、核准與證據規則
  decisions/
    ADR-*.md                        # 被接受的架構決策；每筆有狀態與適用範圍
  domain/
    glossary.md                     # 業務名詞與資料語義
  ai/
    context-pack.json               # 由上述來源產生的衍生 context；不是 SSOT
evals/                              # 可選 P1：模型行為的固定案例與評估結果
requests/
  REQ-*.md                          # 原始需求與補件紀錄
evidence/
  EB-*/                             # Git、CI、review、build、runtime 證據
```

文件名稱可以不同。ContextRail 應以 `kind`、用途、來源與驗證狀態辨識文件，而不是以檔名硬編碼所有公司的做法。

## 4. 必要文件與責任

### 4.1 P0 必要集合

| 文件／來源 | 主要回答的問題 | 最低必要內容 | 真相角色 |
| --- | --- | --- | --- |
| `project.yaml` 或等價 manifest | 這個 Project 是誰、連了哪些來源、允許哪些環境？ | identity、owner、repository links、service links、environment links、required documents、commands、classification、version | 結構化 SSOT |
| `README.md` | 新成員如何理解並啟動？ | 一句話用途、使用者、目錄入口、前置條件、快速開始、限制 | 人類入口 |
| `vision.md` 或 `product.md` | 為什麼做、成功是什麼、不做什麼？ | problem、users、outcomes、non-goals、已知假設 | 產品意圖 |
| `architecture.md` | 變更會影響什麼？ | component/data flow、依賴、boundary、security、failure、重要 trade-off | 架構基線 |
| `development.md` 或 README 的等價區段 | 怎麼在本機開發與驗證？ | runtime version、install、run、test、lint、build、fixture、常見失敗 | 工程操作來源 |
| `environments.md` 或等價設定 | 可以送往哪裡？誰可以送？需要什麼證據？ | environment type/order、target、allowed transition、required evidence、approver separation | 發布治理來源 |
| `AGENTS.md` 或相容的 Agent 指令來源 | Agent 可以做什麼、不能做什麼？ | scope、commands、verification、protected paths、forbidden actions、escalation | Agent 行為契約 |

「必要」是依目標 action 決定的。只想閱讀 Project 可以少一點文件；要建立 Decision、Work Order 或進入 staging，則必須通過相應 readiness gate。這避免把所有專案一律要求成大型企業的完整文件包。

### 4.2 P1 建議集合

- `roadmap.md`：目前順序、未來方向與明確裁切條件。
- `docs/governance/policies.md`：角色、核准分離、資料分類、保存與例外流程。
- `docs/decisions/ADR-*.md`：被接受的架構選擇、替代方案、後果與有效範圍。
- `docs/domain/glossary.md`：業務名詞、事件、狀態與資料擁有者。
- API／event／schema contract：讓 Agent 與測試不必從實作反推介面。
- `cost.md` 與 `security.md`：需要成本或資安判斷時才列為 gate input。

若 Project 的核心行為包含模型輸出，建議再加入 `evals/`（案例、預期、評估結果）與 `docs/ai/model-policy.md`（模型／prompt 版本、資料邊界、成本、延遲與安全限制）。它們是 AI 行為的驗證與政策來源，不是把模型回答直接升格為事實；ContextRail 的 P0 仍以固定 fixture 驗證，避免把完整 evaluation platform 偷渡進第一版。

### 4.3 衍生文件

`docs/ai/context-pack.json` 可以把目前有效文件、Project manifest、已接受決策與引用整理成 Agent 易讀格式。它必須包含來源路徑、版本或 commit、content hash、effective time、access scope 與生成時間；它可以重建，但不能被當成唯一真相，也不能覆寫已接受的 DecisionRecord。

## 5. `project.yaml` 最小契約

以下是示意，不是要求所有客戶採用這個檔名或所有欄位。

```yaml
apiVersion: context-rail/v1alpha1
kind: Project
metadata:
  id: order-operations
  name: Order Operations Portal
  version: 3
  classification: internal
spec:
  purpose: Give support staff a controlled way to find and resolve order exceptions.
  owners:
    technical: team/platform
    business: team/operations
  repositories:
    - id: order-portal
      provider: github
      url: github.com/example/order-portal
      role: primary
      paths: [web/, api/]
  services:
    - id: order-api
      repository: order-portal
      path: api/
      role: primary
  environments:
    - id: development
      type: development
      sequence: 1
      targetRef: local-or-branch
      requiredEvidence: [pr]
    - id: staging
      type: staging
      sequence: 2
      targetRef: cloud-run/order-portal-staging
      requiredEvidence: [candidate-review, build, smoke]
  commands:
    test: go test ./...
    build: go build ./...
    deployStaging: ./scripts/deploy-staging.sh
  documents:
    - path: README.md
      kind: human-entry
      requiredFor: [decision, development]
    - path: architecture.md
      kind: architecture-baseline
      requiredFor: [decision, work-order]
    - path: docs/ai/context-pack.json
      kind: derived-context
      sourceOfTruth: false
```

驗證器至少要檢查：ID 是否穩定、repository link 是否有角色與 path、environment sequence 是否無歧義、command 是否可執行或明確標為未知、文件路徑是否存在、`sourceOfTruth: false` 是否只出現在衍生物，以及每個 required document 是否有版本與驗證時間。

## 6. SSOT 與衍生物分層

```text
Project manifest + approved documents + observed source/runtime evidence
                              │
                              ▼
                    ContextRail canonical records
                              │
              ┌───────────────┴────────────────┐
              ▼                                ▼
       Human Decision Brief              Agent Context Pack
       readable projection               structured projection
              │                                │
              └───────────────┬────────────────┘
                              ▼
                 Work Order / Evidence / Receipt
```

每個投影都要保留 `project_id`、`decision_id`（若適用）、document versions、source snapshot hash、policy version 與 evidence references。來源更新、撤回、權限改變或 Project topology 改變時，ContextRail 只能使相依產物失效並說明原因，不能靜默重寫歷史核准。

## 7. Readiness gate

Project 文件完整度不應被壓成一個漂亮的百分比。ContextRail 應依 action 顯示 readiness：

| Readiness | 最低條件 | 不能推論的事情 |
| --- | --- | --- |
| Ready for decision | manifest、vision／problem、architecture、目前 environment policy 與可引用來源有效 | 需求一定值得做、成本一定正確 |
| Ready for development | decision 已接受、development commands 可重現、allowed paths、acceptance criteria 與必要 Agent rules 有效 | Agent 可以任意修改 repo |
| Ready for staging | candidate、PR／review、test／build、target config 與 staging policy evidence 對得上 | 已經通過 production 或已具備 production readiness |

狀態應至少包括：`CURRENT`、`DRAFT`、`MISSING`、`STALE`、`CONFLICT`、`REVOKED` 與 `DERIVED`。`DERIVED` 只描述來源，不代表內容一定完整；context pack 若缺必要文件，應顯示 `PARTIAL` 或 `NEEDS_INPUT`。

## 8. 既有 Project 的匯入方式

匯入不是把 repo 複製到 ContextRail，也不是把每個 repo 強迫建立成一個新 Project：

```text
選擇 Project 或建立 DRAFT Project
        ↓
匯入 manifest／repo metadata／文件路徑／environment target
        ↓
產生 observed 與 inferred inventory
        ↓
人確認 primary、dependency、path scope 與刻意不納管的來源
        ↓
建立 Project baseline 與 context pack
        ↓
依 action 顯示缺件與下一步
```

GitHub、GitLab、self-hosted GitLab、Google Drive、local fixture 都是來源 adapter；P0 只要求一個受控來源或 fixture，不能把未連線的 private repository 顯示成不存在。文件若提到一個 repo，最多先形成 `INFERRED` link，仍要以 inventory 或人確認。

## 9. ContextRail P0 的實作邊界

- UI 先提供 Project Workspace 的「文件與 AI Context」頁：文件清單、audience、SSOT／derived、狀態、版本、索引狀態、readiness 與缺件原因。
- P0 使用固定 fixture／明確授權文件驗證 status、引用與失效傳播；不做任意 URL 抓取、PDF／OCR、內網 connector 或完整文件生命週期。
- `Validate document set` 只產生可回讀的 validation result；`Rebuild AI Context` 只能更新衍生 pack，不能將缺件變成通過。
- P0 不建立新的 microservice 或獨立 graph database；Go modular monolith、Firestore records／edges 與既有 RAG adapter 足夠。
- Project 文件契約不取代 Git、Backstage、Jira、Trello、CI、Cloud Build、Cloud Run 或既有 code review；它把這些來源的證據放回同一個 Project decision boundary。

## 10. 驗收與停止條件

最小驗收證據如下：

1. 一個完整 Project 能列出來源文件、版本、責任人與目前 readiness。
2. 缺 `development.md`、過期 `architecture.md` 或衝突 policy 時，UI 顯示明確的 `NEEDS_INPUT`／`STALE`／`CONFLICT`，而不是產生完整假象。
3. `context-pack.json` 能回到每一個來源路徑、版本、hash 與生成時間，且明確標示 derived。
4. 同一個 Project 的 Decision Brief 與 Agent Context Pack 能共享 canonical IDs，不各自重新解讀需求。
5. 文件更新後，受影響的 Decision、Work Order 或 gate 會標記失效並留下 audit event。
6. 第二個使用者可以依 README 與 development contract 啟動並驗證一個受限 Change；這是可用性的驗收，不是文件數量競賽。

若文件維護時間高於它節省的重查時間，或小型團隊只需要在 PR 旁看一個 policy check，應縮成 Git／CI connector，而不是繼續增加文件種類。

## 11. 相關公開參考

- [GitHub Spec Kit](https://github.github.com/spec-kit/)：Spec → Plan → Tasks → Implement 的 intent-driven workflow。
- [AGENTS.md](https://github.com/agentsmd/agents.md)：跨工具的 coding-agent 指令格式方向。
- [GoogleCloudPlatform/agent-starter-pack](https://github.com/GoogleCloudPlatform/agent-starter-pack)：Google Cloud agent runtime、CI/CD、evaluation 與 observability starter。
- [Backstage Software Catalog descriptor format](https://backstage.io/docs/features/software-catalog/descriptor-format/)：YAML catalog entity 與 relation metadata。
- [IgniteUI/ai-repo-structure](https://github.com/IgniteUI/ai-repo-structure)：結合 `AGENTS.md`、`CLAUDE.md`、`.github/`、skills 與 hooks 的社群參考 repo。
- [claude-spec-driven-template](https://github.com/dashdanilo/claude-spec-driven-template)：以 spec-driven workflow 組織 Claude Code 的社群 template，可參考其文件與 agent 工具分層。

這些專案各自解決不同層次的問題；它們支持本契約的分層設計，但不證明 ContextRail 的整體 workflow 已是市場標準或已被客戶驗證。
