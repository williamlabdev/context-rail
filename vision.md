# ContextRail Vision

Status: Implementation proposal with validation conditions
Version: 3.3
Last updated: 2026-09-13
Planning baseline: ContextRail Project Plan v4.2
Positioning: Environment-aware AI Change Assurance for Cloud-native Startups, SMBs and Growing Teams (enterprise-ready extension)

## 1 願景

讓團隊在接受需求變更之前，理解架構、成本與風險的代價；在發布之前，確認實際交付符合被接受的決策。

> Verify that an AI-assisted change still matches approved intent before it reaches a governed cloud deployment target.

ContextRail 是面向新創、中小企業與成長型工程團隊的 environment-aware AI Change Assurance layer。它把被接受的需求、公司文件、架構與成本條件，連到 Agent 工作範圍、程式候選、Project 環境拓撲與實際發布身份；來源、程式、環境政策或目標設定改變時，舊核准就失效。每個 Project 可以自行定義 Development、Testing、UAT、Staging、Production 或其他環境名稱；ContextRail 保存環境順序、目標、允許動作、必要證據與核准政策，但不取代雲端平台的 infrastructure source of truth。五週 prototype 先服務 5 至 30 人、尚無專職平台團隊的新創／中小企業，第一個部署切口是 Cloud Run，第一個案例先選內部業務系統，下一個可驗證案例是 B2B SaaS。大型企業不是 P0 的買方驗證對象，而是保留多租戶、私有網路、細粒度權限、更多 provider 與 production operations 的 enterprise-ready extension space。主要使用者先鎖定 Solution Architect、Tech Lead 或 Delivery Architect；FDE、Implementation Engineer 與 DevOps 是待驗證的次要使用者，不把「FDE-first」或已認識 FDE 當成原型啟動前提。

市場定位是 AI Change Assurance；AI Native Software Engineering control layer 是產品的技術架構。AI 參與需求解讀、工程交接、實作或 review 與證據整理，但每個輸出先保持候選狀態，並受版本化契約、可重現檢查與人工權限控制。ContextRail 實作的是一條受限且可驗證的 AINE assurance slice，不宣稱重建完整 Product Foundry、通用 coding agent、CI/CD 平台或無人值守的軟體工廠。

CI/CD 負責建置、測試和部署；ContextRail 負責驗證這次 AI 變更是否仍符合原需求、允許範圍、架構與成本條件，以及被核准的 commit、image、環境拓撲和目標設定。它整合既有 coding agent、Git 與 Cloud Build，不以取代它們為產品價值。環境不是只有狀態標籤：每個 promotion transition 都有自己的政策、必要證據、可執行動作與人工責任。

原型範圍採「窄版可做、完整能力不宣稱」的原則。Vibe coding 可以加速建立固定 adapter、tenant-ready 資料模型、FDE handoff pack 或 Cloud Run revision rollback，但不會自動證明多租戶隔離、內網資料邊界、跨資源回復或企業 production readiness。

## 2 要解決的問題

團隊接受「新增附件」「加入背景通知」等需求時，往往還沒確認資料保存、權限、服務選型、預期用量及營運負擔。開發完成後，即使 CI 通過，最初被接受的架構與成本條件也可能沒有落實。

我們要驗證的問題是：新創與中小企業的團隊是否花太多時間在文件與工具間重建決策脈絡，以及這些落差是否造成返工、漏測或發布延誤。這是產品假設，必須透過真實變更案例與使用者訪談確認；現階段沒有付費客戶或節省成本的實測證據。本原型先以 Solution Architect 的實際或匿名案例做 workflow dogfooding，再檢查相同痛點是否出現在其他小型團隊；這能驗證流程與 artifact 是否合理，但不能單獨證明跨公司需求、FDE 共通性或購買意願。

## 3 第一批使用者與購買者

Prototype 第一批客群假設是新創與中小企業中 5 至 30 人的工程團隊，維護至少一個 Cloud-native 業務系統，已有 Git PR、基本測試、至少一種 AI coding assistant，以及不只一個需要被治理的交付環境；第一個 prototype 以 Cloud Run 為部署切口。客戶應用可以是內部系統或 B2B SaaS，但不先追求消費型平台。附件、權限、保存期限或背景工作等變更需要 Solution Architect、Tech Lead 或發布負責人確認，而團隊尚未建立完整內部開發平台。第一批 design partner 必須願意提供一個既有系統的文件、現行人工流程、環境拓撲與去識別化變更案例。這是「團隊特徵優先」而非用公司營收或員工人數硬切市場；大型企業只有在後續驗證同一 workflow 的導入價值後，才進入 enterprise extension。

- PM 或系統負責人提出目標、使用量、可接受成本與驗收條件。
- Solution Architect 或 Tech Lead 比較架構選項，確認限制與需要補足的資訊。
- 工程師與 QA 將已接受的條件編成受限 work order，並把 Agent run、PR、review、測試和 staging 證據連回原決策。
- FDE 或 Implementation Engineer（若有此角色）協助把客戶／內部團隊的交付脈絡轉成可交接的驗收與 runbook；這是次要 persona 假設，不是 P0 的必要依賴。
- 指定的 Release Approver（可由 DevOps 或發布負責人擔任）依版本、設定與測試證據核准發布；該身份必須與候選作者及獨立 review 者分離。
- 初始購買者假設為 Engineering Manager；Delivery／Professional Services leader 是待驗證的相鄰 buyer hypothesis，平台團隊與 FinOps 是共同評估者。

ContextRail 本身可以採 SaaS 交付；客戶管理的是內部系統或 SaaS，屬於另一個維度。P0 先驗證新創／中小企業的導入時間、重複使用與付費意願，不把大型多雲企業當成 prototype 的必要前提；大型企業保留為後續擴展市場，不在 P0 同時解決所有企業治理能力。

## 4 三次決策與選配回饋

第一個檢查點在需求接受之前。Architecture Impact Advisor 讀取 RAG 找到的文件、人工確認的架構基線與有限的雲端現況，提出保留現況及替代方案。系統同時把變更放到 Project 的 environment topology，判斷它預計影響哪些環境與 transition；若環境名稱、順序或 production policy 尚未確認，先列為缺資料。P0 直接使用 Gemini；P1 可匯入 Application Design Center 的架構與 assessment，P2 才評估 Gemini Cloud Assist。Cost Impact Advisor 仍以版本化單價和使用量假設計算成本差異。人員決定接受、修改、延後或拒絕。

第二個檢查點在實作候選完成之後。ContextRail 將核准的 Change Contract 編譯成有範圍、驗收條件與禁止動作的 Agent Work Order；外部 Claude Code、Codex 或其他允許的 coding agent 透過團隊既有流程建立 feature branch 與 PR。系統匯入 Agent Run Record、Git diff、測試、build 與獨立 review，重新核對來源與架構／成本條件。人員決定接受候選、要求修改或拒絕；接受候選只允許合併與進入 staging，不等於發布核准。

第三個檢查點在指定環境驗證之後。Release Assurance 將已接受的架構、成本驅動設定、environment transition 與驗收條件，對照目前的 commit、image digest、目標環境設定、測試與 runtime 證據。來源、環境 policy 或目標版本改變就重新檢查，由與候選作者及獨立 review 分離的 Release Approver 決定是否進入下一個環境；P0 的下一個目標是以同一 digest promotion 至隔離的 prod-demo，smoke 通過後才產生 P0 Release Receipt。

P0 在發布後保存 Release Receipt 和人工修正原因，證明實際 revision 可以回到三次決策。短期 Outcome Observation 與 evidence-linked Evolution Proposal 是 G4 通過且仍保有風險緩衝時的 stretch；即使實作，提案也不能自動修改需求、程式、政策或部署。未來可在帳務資料到齊後比較預估與實際用量；部署當下不能宣稱已驗證月帳單或產品價值。

每次人工接受變更後，ContextRail 以同一份版本化的 Change Contract／DecisionRecord 產出兩個同步視圖，而不是讓模型各寫一份可能互相矛盾的報告。兩個視圖同時引用該 Project 的 environment topology 與本次允許的 promotion transition：

- **Agent Context Pack**：給 Gemini、Claude Code、Codex 或其他受限 Agent 使用的結構化契約，包含接受範圍、明確不做的事情、允許路徑、禁止動作、成本假設、驗收條件、必要檢查、未知項、證據引用、政策版本、有效期限與 hash；不保存 secret 值，也不授予核准或發布權限。
- **Change Decision Brief**：給 PM、系統負責人、Solution Architect、Tech Lead、DevOps 與非技術利害關係人的人話摘要，先說明一句話結論、使用者影響、成本、風險、需要誰決定與下一步，再展開技術證據。

兩個視圖共享 `decision_id`、版本、`source_snapshot_hash`、`policy_version`、`evidence_refs` 與有效期限；來源、需求、架構或核准改變時，兩個視圖、Agent Work Order 與 Release Approval 一起失效。Engineering Evidence Bundle 與 Release Receipt 也沿用同一組 ID。第一頁不要求讀者理解 RAG、provenance、drift 或 digest；這些只在技術展開層以簡短定義和證據連結呈現。

## 5 產品模型

```text
需求與設計參考
    ↓
具版本與權限的 RAG ＋ 架構基線
    ↓
Gemini 分析 ＋ 可選的 ADC 或 Cloud Assist 建議
    ↓
    架構選項 ＋ 成本情境 ＋ 風險與未知項
    ↓
    人工接受變更並保存 Decision Record
    ↓
    Project Environment Topology ＋ Promotion Policy
    ├→ Change Decision Brief（給人）
    └→ Agent Context Pack（給 Agent）→ 產生 Agent Work Order
    ↓
工程師與受限 coding agent → feature branch 與 PR
    ↓
Agent Run Record ＋ Cloud Build ＋ Independent Review
    ↓
Engineering Evidence Bundle → 人工候選決定
    ↓
Cloud Run staging → 驗證計畫與實作差異
    ↓
Release Approver → 同一 digest promotion 至隔離 prod-demo → smoke
    ↓
Release Receipt
    ↓
選配 Outcome Observation → Evolution Proposal → 下一次人工決策
```

完整工作流的意思是追蹤各階段的責任、輸入、候選、環境 transition 與證據，不是重做 pipeline runner。Environment Topology 是 Project 的治理設定，不是 ContextRail 自己推測出來的環境清單；使用者可以自訂名稱與數量，但不能用自訂名稱繞過 production 的最低保護規則。P0 不在 ContextRail Cloud Run 服務內託管任意 coding agent，也不承諾自主產生或修正全部程式；它產生有 hash 的工作契約，接收一條由開發者啟動的 Agent 執行路徑聲明與證據，再用 Git、CI 與雲端讀值交叉驗證。G4 通過後可選一條固定的 remote adapter，但仍不接受任意 agent orchestration。Agent 回報不能自行成為核准或發布依據。

### Project 是治理邊界，不是普通資料夾

Project 是 ContextRail 的最小可治理單位。它把一個業務系統、受管 repo、環境拓撲、promotion policy、知識來源、角色權限與所有變更證據放在同一個可回查的邊界內；每張 Ticket／Change Request 必須帶有 `project_id`，但 ContextRail 不取代 Jira、GitHub Issues 或其他工作追蹤工具。

P0 提供 Project 的基本增修刪查，但「刪除」採封存語義：

| 操作 | P0 行為 | 治理保護 |
| --- | --- | --- |
| 建立 | 建立 Project identity、repo、owner、業務負責人與初始 EnvironmentTopology／PromotionPolicy | 建立前檢查必要欄位、target reference、角色與 production 最低保護；不自動發現或配置所有雲端資源 |
| 查詢 | Project Registry 與 Project Workspace 顯示變更、環境、gate、風險、證據與最新 receipt | 查詢結果依 Project membership 過濾；Project overview 不是單純 CRUD 清單，而是治理狀態入口 |
| 修改 | 修改 metadata、repo、owner、環境或 policy，產生新版本與 audit event | 受影響的 DecisionRecord、Agent Work Order、Candidate／Release Approval 標記 `STALE`，不得沿用舊核准 |
| 刪除 | 以 `ARCHIVED` 封存，停止新變更與 promotion | 保留 Decision Ledger、Evidence Bundle、Release Receipt 與 append-only audit；不提供 P0 hard delete |

Project 設定變更本身也是一種受治理的變更。Project Registry 負責找到與管理邊界，Project Workspace 負責處理 Ticket；兩者都回到同一份 DecisionRecord，避免為了支援多個 Project 而建立互相矛盾的報告或政策。

### Project 文件契約與 AI Context

Project 也必須有一個可檢查的文件基線，讓人與 AI 使用同一個脈絡。P0 的最小基線包含 `project.yaml`、`README.md`、`vision.md`、`architecture.md`、`docs/engineering/development.md`、`docs/operations/environments.md` 與 `AGENTS.md`；每份文件都要記錄用途、受眾、來源版本、有效時間、存取範圍與目前狀態。這是一個 ContextRail 的最小 Project Context Contract，不宣稱是所有 AI 專案都必須照抄的業界標準。

`ProjectDocument` 是文件來源與狀態的可查詢紀錄；`docs/ai/context-pack.json` 則是由已確認文件、DecisionRecord 與證據索引產生的衍生物。Context Pack 必須能回到來源 path、version／commit、hash、effective time 與 access scope，不能取代文件、修改 SSOT 或把缺件補成事實。文件缺少、過期、衝突或撤回時，Project readiness 應顯示 `MISSING`、`STALE`、`CONFLICT` 或 `REVOKED`，而不是讓 Agent 繼續用看似完整的摘要。

完整的文件欄位、目錄建議、import／validate／rebuild 流程與 Ready for decision／development／staging 門檻，集中記錄在 [Project Context Contract](docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md)。Project Workspace 的「文件與 AI Context」頁是這份契約的第一個 UI 投影；它展示文件清單、SSOT／derived、索引狀態、缺件原因與 readiness，不只是上傳附件。

## 6 一個產品工作與六項支援能力

第一版只證明一個產品工作：讓團隊在 AI 產生的變更進入指定環境前，確認它仍符合被接受的意圖與該環境的放行條件。產品從 Project Registry／Workspace 開始，但不是再做一個專案管理平台；Project CRUD 的價值在於建立可治理邊界，讓每張 Ticket 都能繼承正確的環境、政策、來源與權限。產品產出不是自由文字的 AI 報告，而是由同一份決策資料渲染出的 Change Decision Pack：給人的 Change Decision Brief、給 Agent 的 Agent Context Pack，以及後續的 Engineering Evidence Bundle 和 Release Receipt。以下能力是這條 assurance path 的證據來源與檢查階段，不是六套獨立產品。

### Architecture Impact Advisor

每個建議連回目前元件、資料流、ADR 或已確認的限制。輸出受影響的服務、IAM、資料模型、可靠性、測試、migration 與 rollback 條件。顯示保留現況與可行替代方案；資訊不足時指出缺口，避免為了展示 GCP 服務而推薦新增元件。

P0 的建議由 Gemini workflow 和人工確認的架構基線產生。條件式 P1 的優先候選是 Application Design Center REST Adapter，匯入 application、revision 與 assessment report；Gemini Cloud Assist MCP 目前需要 private preview 權限，因此只列為 P2。不同 provider 的輸出都轉成相同的 ArchitectureAssessment，保存 provider、外部 resource ID、來源版本、原始輸出 hash、假設與限制。外部建議是候選資料，不能直接核准需求或取得發布權限。

### Cost Impact Advisor

成本是一級決策資訊。報告分開呈現既有成本、變更後成本、每月增量及一次性開發／移轉工時，並展示低、基準、高三組使用量情境。

Gemini 協助整理成本驅動因子；單價、單位、分級計費及加總由確定性計算器處理。每筆金額可回溯 region、currency、SKU 或官方價格頁、價格時間與假設。未知費項顯示為未知；估算與實際帳務保持分離。

### Decision Support

將架構、成本、風險與使用者提供的業務價值放在同一個比較畫面。系統可以指出違反規範、超過門檻或需要補證據；產品 ROI 與需求優先順序由人決定。接受需求、接受候選並允許進入 staging、核准正式發布是不同決策。

### Environment Topology & Promotion Policy

Project owner 或 admin 定義本專案的環境拓撲。環境可以叫做 `dev`、`development`、`qa`、`uat`、`staging`、`production` 或企業自己的名稱，但每個環境都要映射到一個標準類型，並保存順序、target reference、允許動作、必要證據、approver policy、owner 與 config hash。`Code Review`、`Candidate Review` 與 `Release Approval` 是 gate／決策，不是環境本身。

這個能力的差異不在於顯示四個環境，而在於把「哪個變更可以進哪個環境」變成可回查的政策：Development 可以接受本機或 PR 證據，Testing 要求測試與 build evidence，Staging 要求候選核准與環境設定核對，Production 即使在 prototype 中也只能維持 read-only 或 blocked。P0 只執行 staging 與隔離 prod-demo；其他環境以定義、證據匯入與 gate 狀態驗證，不把 ContextRail 宣稱成完整環境或 IaC 管理器。

### Agent Handoff Assurance

將經接受的需求轉成 Agent Work Order，限制 repository、base branch、允許修改路徑、驗收 ID、必要檢查、禁止動作、有效期限與來源版本。P0 將 Context／Architecture Agent、Engineering Agent、Evidence Agent 與 Release Policy Agent 視為受限角色；它們透過 Request、DecisionRecord、AgentWorkOrder、EvidenceBundle 等結構化 artifact 交接，不以自由對話互相授權。P0 支援一條由開發者啟動的 Claude Code 或 Codex 工作路徑；ContextRail 不保存個人 coding-agent 憑證，也不讓外部 Agent 取得 prod-demo、IAM 或核准權限。

Agent Run Record 保存 adapter、agent、provider／model／version、work-order hash、時間、commit、變更路徑、執行檢查、sandbox／權限聲明與原始證據 hash。這份紀錄是待驗證聲明，不是因果證明；ContextRail 仍從受管 Git 與 CI 讀回 commit、diff、測試和 build identity。獨立 review 與人員候選決定分開保存，同一個 Agent 自評不能取代人工接受。P0 不實作 autonomous Agent-to-Agent orchestration；A2A 只保留為跨 provider、remote agent 或不同 runtime 有明確需求與證據後的 P1 extension。即使採用 A2A，也只處理 interoperability／task transport，不取代 DecisionRecord、EvidenceBundle、權限與 human gate。

### Release Assurance

CI 提供測試結果，ContextRail 檢查證據是否對應這次變更與目標版本。受控執行器只部署已核准的 digest 與允許的設定。部署後讀回 revision、執行 smoke test 並產生 receipt；failure 與未知結果都有可見狀態。自動 Git tag、一次 Cloud Run revision rollback 與 outcome proposal 可在 G4 後選作窄版延伸；跨資料庫、物件與 IAM 的完整 rollback 仍不在原型範圍。

## 7 從第一天建立企業記憶

RAG 是文件檢索與引用層；Decision Ledger 保存誰在何時基於哪些條件作出什麼決定；Evidence Graph 以結構化關聯連接需求、Agent Work Order、執行紀錄、候選決定、PR、測試、部署及 receipt。第一版以 Firestore 的紀錄與關聯表實作，不另建圖資料庫。

資料累積要包含來源、版本、有效期間、存取範圍與人工接受狀態。P0 驗證一次來源更新使受影響判斷失效；完整撤權、刪除與企業文件生命週期在後續版本擴充。模型產出的建議保留候選身分；被接受的決策也保留適用範圍，不自動成為全公司的規範。

文件上傳會將內容送至本產品的雲端處理。P0 只使用合成、明確授權的上傳或 fixture；local folder 只用來示範來源契約，不列為可宣稱完成的產品 adapter。G4 後的單一 Drive／Docs 文件唯讀匯入才是候選 P1，並以明確授權的資料驗證重新匯入與版本失效。但這不等於解決公司禁止文件出內網的限制；真正的私有檢索、VPN／VPC、內網身份與不出網部署仍屬後續方案。

## 8 自主權與證據原則

Gemini 可以檢索已授權來源、提出影響候選、整理 work order 與找出 evidence gap；只有執行 stretch 時才依發布後觀測提出下一步候選。第一版使用有固定步驟的 Gemini workflow，保存模型、提示詞及檢索版本；沒有實作工具自主選擇時，對外使用 multi-step AI workflow 的描述。Claude Code、Codex 等外部 coding agent 是受治理的工程執行者，不等於產品內的 Gemini provider，也不能因團隊用 AI 開發就宣稱產品閉環已完成。

規則引擎對必要條件產生 PASS、FAIL 或 UNKNOWN。來源引用或 Agent Run Record 存在不代表推論一定成立，因此關鍵判斷仍需人工驗證。模型不能自行核准、更改預算政策、授予 IAM、寫入 production，或從文件、PR 與 log 內容接收未列入 work order 的工具指令。超出允許路徑、來源已變動、執行者資訊缺失或 review 不獨立時，候選必須阻擋或進入 NEEDS_REVIEW。

人工核准綁定需求版本、架構方案、成本假設、policy、commit、digest 與目標設定。核准後改了任一必要輸入，舊核准就失效。多次點擊發布不得造成重複部署或錯誤 receipt；若執行 tag stretch，也不得覆寫既有 tag。

## 9 五週原型的承諾

主案例為內部簽核系統新增附件；次案例為文件更新造成原決策失效。Demo 的一個 Release Bundle 可包含 2 至 3 個 feature、需求變更或 bug fix，但每個 Change 仍保留自己的契約、證據與 gate。原型交付：

- 一個 Project Registry、一個 Project Workspace、一個 tenant-ready 的資料契約、一個可自訂四環境拓撲的受管 Project、一個 repo 與 8 至 12 份小型 TXT／Markdown 文件；P0 同時展示 Project Context Contract 的必要文件、索引與 readiness，並產生可回溯來源的 Agent Context Pack；P0 可建立、查詢、修改與封存 Project，修改會使受影響的決策與核准標記 `STALE`；一個 Release Bundle 可聚合多個 Change，但不能繞過任何單一 Change 的必要 gate；不宣稱完成商業化多租戶營運或完整環境管理。
- 真實 embedding、檢索、來源引用與決策紀錄回查。
- 限定架構選項與 Cloud Run、Cloud Storage、Firestore 三類成本模型。
- 人工接受方案、同一決策資料產生的 Change Decision Brief 與 Agent Context Pack、版本化 Agent Work Order、一條真實外部 coding-agent feature PR 與 Agent Run Record。
- Git／CI 交叉驗證、PR／獨立 review、Engineering Evidence Bundle 與人工候選決定；Code Review 只作證據與 gate，不在本產品內重建 inline review editor。
- Cloud Build 與 Cloud Run staging。
- staging 與隔離的 prod-demo 兩個受管服務，使用同一個已建置 digest 驗證受控 promotion；G4 後才評估一項窄版 P1。
- 權限、work-order／來源版本失效測試、8 個情境案例、其中 2 個 holdout，以及可回查的 release receipt。

自動 Git tag、Cloud Run revision rollback、FDE Handoff Pack、固定 remote agent adapter、Application Design Center REST Adapter、單一 Drive／Docs 文件唯讀來源 adapter、擴充文件生命週期與短期 outcome-to-proposal 是條件式 P1／stretch；local folder 仍只作契約 fixture。只有 G4 PASS、總容量至少 300 人時、至少保留 40 人時風險緩衝，且契約、API／IAM／location 已驗證時，才選擇至多一項；未完成不影響核心原型驗收，也不能宣稱已完成企業級能力或 AINE evolution。

prod-demo 是使用合成資料的展示發布目標，與客戶 production readiness 分開評估。GCP catalog 可描述額外服務，但有建議說明、能估算、能部署是三種不同支援程度。

五週工程窗口暫定為 2026-09-15 至 2026-10-18；10/16 凍結開發與 10/17 至 10/18 交件只是內部排程假設，不是已確認的官方截止日。官方首頁與 FAQ 的組隊／prototype 日期仍有衝突，須在 G0 取得主辦方或實際提交入口確認；在確認前以最早可能日期作提交風險準備，若官方日期早於 10/18，立即重排 P0 範圍。已確認人數為兩人；工時先沿用每人每週 25 小時的規劃假設，合計約 250 人時，其中 190 人時核心工作、60 人時風險緩衝。每週實際可投入時數仍需在開發啟動時核實；這是有條件的原型交付窗口，並非企業正式產品的完成保證。開發與驗證並行：先用團隊的 Solution Architect 案例驗證工作流，再用第二個匿名案例與 3 至 5 位外部相鄰角色檢查可複製性；這些是驗證活動，不新增 P0 功能。

Application Design Center REST Adapter、固定 remote agent adapter、FDE Handoff Pack、單一 Drive／Docs 文件唯讀匯入與 Cloud Run revision rollback 都是候選窄版 P1；local folder 只作來源契約 fixture，不列為 P1 產品能力。它們不納入 190 人時核心工作。AI Change Assurance 核心路徑、G4 發布與交件緩衝優先；只有 G4 PASS、總容量至少 300 人時、至少保留 40 人時風險緩衝，且契約／API／IAM／location 已驗證後，才選擇至多一項 P1。Gemini Cloud Assist MCP、Figma API、Cloud Deploy、完整成本觀測及企業 RBAC 排入 P2；若 private preview 已取得，也只做隔離 spike，不讓它成為核心依賴。完整多租戶營運、任意 agent orchestration、無人值守 FDE、真正內網 connector、跨資料資源 rollback 與 production readiness 仍保留為後續產品化能力。

## 10 市場與商業假設

Gemini Cloud Assist、Application Design Center、ServiceNow、Harness 和 Infracost 已涵蓋我們的重要能力。我們不能用「沒人做」「只有我們在部署前分析成本」或「既有系統是獨家市場」作為競爭論點。公開產品文件也不足以證明其他產品缺少某種客製流程。

待驗證的差異化是 environment-aware AI Change Assurance：對採用 coding agent、擁有自訂多環境交付路徑的新創／中小企業 Cloud-native 團隊，以較少設定建立 Change Contract，把被核准的需求與架構、成本假設、Project Environment Topology、Promotion Policy、Agent Work Order、run provenance、commit、image digest、target config、各環境證據及實際 revision 綁在一起。第一個 cohort 是 Cloud Run 內部系統，下一個可驗證 cohort 是 B2B SaaS；Cloud Run 是第一個部署切口，不是產品只能服務的市場類別。大型企業的複雜權限、私有網路、組織級 inventory 與多雲治理只作後續 extension 假設。ContextRail 不取代 coding agent、Cloud Assist、ADC 或 CI/CD；它把外部產出轉成有來源與版本的候選，再由人工決策與確定性 gate 控制候選接受和環境 promotion。如果既有工具加一份清單已能低成本解決問題，優先考慮 GitHub Check、connector 或 plugin，而非擴大獨立平台。

初始商業模式先測試固定範圍的付費 design-partner pilot，優先找新創／中小企業團隊，確認 environment topology 建模、導入時間、使用頻率與可量化價值後，再評估以受管 Project 或團隊為單位的訂閱；大型企業 connector、固定 Agent adapter、FDE handoff 與導入服務是後續 extension，可獨立報價。付費意願、導入成本與毛利都需要實測，詳見 v4.2 Business Model Canvas。

## 11 成功與停止條件

第一階段成功表示產品能讀取未預先寫死的需求，引用正確且有權限的文件，展示可重算的成本差異，產生受限 work order，並在 Agent provenance、修改範圍、必要 review、資料或部署身份不符時拒絕放行。完成的部署必須能回到人工接受的需求、候選與發布核准紀錄。

第二階段以相同資料、權限和時間限制，比較 PR template 加 Gemini 與 Cloud Build、可取得的 Google Cloud 原生工具及完整 ContextRail。主要結果是所有植入的意圖、範圍與 release identity 漂移是否被阻擋，並同時報首次設定、每次淨 review 時間、誤阻擋與一個 Release Bundle 內多個 Change 的處理負擔。若取得 ADC 或 Cloud Assist 權限，另記錄其設定時間、建議覆蓋與可回溯性；未取得就標記未測。新創／中小企業案例的結果只報樣本數、原始分數及限制，不推論企業事故率；大型企業需求只作後續 extension 假設。

若使用者不願提供文件、每案整理時間高於收益、只有雲端問答需求，或現有工具已達成相同效果，應縮小到特定整合功能或停止擴大。累積文件量與使用 Agent 數量不作為成功指標。

## 12 技術選擇與文件關係

前端使用 React，ContextRail API 與示範簽核系統的後端使用 Go。Go 服務以模組化單體部署至 Cloud Run，透過官方 Go client 連接 Gemini、Firestore、Cloud Storage 與 Firebase Authentication；Cloud Build 和 Artifact Registry 負責產生可識別映像。Go 是本原型的實作決策，不表示未來只能分析或部署 Go 客戶系統。

架構 provider 採分級策略：P0 使用 Google Gen AI SDK；P1 可透過 Design Center REST API 匯入 ADC 的 application、revision 與 assessment，也可選擇一個固定 Agent、單一 Drive／Docs 文件或 rollback adapter；P2 才使用 Cloud Assist MCP。核心 domain 只依賴版本化的 ArchitectureAssessment、EnvironmentTopology、PromotionPolicy、DecisionRecord、AgentWorkOrder、AgentRunRecord、EngineeringEvidenceBundle、ReleaseManifest 與 Release Receipt 契約；rollback operation 只有在選定 P1 時才加入，不使用未定義的替代契約名稱。任何 provider 無法使用時都不能破壞既有 DecisionRecord 或 release gate。Cloud Assist 不是價格來源，成本仍由 ContextRail 的版本化資料與確定性計算器產生。

Go、SDK、模型與 region 的版本在開發第 1 至 2 天完成端到端 spike 後鎖定。Week 1 同時查核 ADC API、IAM、location 與 Cloud Assist MCP preview 資格，但只把前者列為可選 P1。五週估算不預先計入 Go 或外部 advisor 可能帶來的效能、時間或成本改善；這些項目必須由實際 build、操作與評估資料驗證。

本文件描述產品方向；[v4.2 開發計畫書](docs/planning/CONTEXT_RAIL_PROJECT_PLAN_V4.2.zh-TW.md) 是當前交付範圍與驗收依據；[Roadmap](roadmap.md)、[Architecture](architecture.md) 與 [Project Context Contract](docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md) 保存工程與文件邊界；[Critical Thinking Review](docs/reviews/CONTEXT_RAIL_CRITICAL_REVIEW.zh-TW.md) 保存反證、修正與未解假設；[產品一頁式說明](docs/product/CONTEXT_RAIL_ONE_PAGER_V2.3.zh-TW.md) 是同步的對外摘要。Markdown 為內容真相源，Word 由其產生。

舊版 [Vision v1](docs/planning/history/VISION_V1.md)、[計畫書 v2](docs/planning/HANDOFF_GUARD_PROJECT_PLAN_V2.zh-TW.md) 保留作歷史參考。目前 repository 仍在文件規劃階段；本版本不代表已完成上述產品功能。

## 13 主要查核來源

查核日期為 2026-09-10。下列資料支持競品能力與技術選項，不構成客戶需求或產品差異化已獲驗證的證據。

- [Gemini Cloud Assist](https://cloud.google.com/products/gemini/cloud-assist)
- [Gemini Cloud Assist MCP integration](https://docs.cloud.google.com/cloud-assist/configure-mcp)
- [Application Design Center REST API](https://docs.cloud.google.com/application-design-center/docs/reference/rest)
- [Application Design Center remote MCP](https://docs.cloud.google.com/application-design-center/docs/use-app-design-center-mcp)
- [Gemini Cloud Assist in Cloud Billing](https://docs.cloud.google.com/billing/docs/how-to/gemini/overview)
- [ServiceNow Enterprise Architecture](https://www.servicenow.com/products/enterprise-architecture.html)
- [Harness Continuous Delivery](https://www.harness.io/products/continuous-delivery)
- [Infracost](https://www.infracost.io/docs/)
- [Firestore vector search](https://docs.cloud.google.com/firestore/native/docs/vector-search)
- [Cloud Run Go quickstart](https://docs.cloud.google.com/run/docs/quickstarts/build-and-deploy/deploy-go-service)
- [Google Gen AI SDK for Go](https://cloud.google.com/vertex-ai/generative-ai/docs/sdks/overview)
- [Firebase Admin SDK setup](https://firebase.google.com/docs/admin/setup)
- [AI Builder Cup requirements](https://aibuildercup.com/themes.html)
