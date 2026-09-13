# ContextRail 參賽與開發計畫書

Google Cloud AI Builder Cup 2026  
主題 Future of Work and Enterprise Productivity  
版本 4.2　更新日期 2026-09-12  
工程窗口暫定 2026-09-15 至 2026-10-18　兩人五週；10/16 凍結開發（官方交件日期待 G0 確認）

本版將 ContextRail 的市場定位收斂為 environment-aware AI Change Assurance for AI-assisted changes to cloud-native business systems，技術上仍採 AI Native Software Engineering control layer。Prototype 明確採新創／中小企業優先：先服務 5 至 30 人、已使用 Git／CI 與 AI coding assistant、沒有專職平台團隊且有兩個以上交付環境的工程團隊；大型企業保留為 enterprise-ready extension space，不是 P0 的買方驗證前提。Cloud Run-first 只描述五週 prototype 的第一個部署切口，不限制長期市場只能是 Cloud Run 或內部系統。P0 只證明一條真實英雄流程：Project Registry 建立並管理 Project，Project 先定義可自訂的環境拓撲與 promotion policy，被接受的需求形成受限 Agent Work Order，外部 coding agent 建立候選，系統以 PR／獨立 review、Git／CI 與環境證據完成三次人工決策、prod-demo smoke 及 Release Receipt。P0 也保留 tenant-ready 的資料契約，但不宣稱完成多租戶營運或完整環境管理。每次人工接受後，系統從同一份決策資料產出給 Agent 的 Agent Context Pack 與給人的 Change Decision Brief，兩者共享 ID、版本、來源 hash、環境拓撲與證據引用。Release Bundle 可包含多個 Change，但每個 Change 仍必須保留自己的 scope、evidence 與 gate。只有 G4 PASS、總容量至少 300 人時、至少保留 40 人時風險緩衝，且契約／API／IAM／location 已驗證後，才可從固定 remote agent adapter、FDE Handoff Pack、Application Design Center REST Adapter、單一 Drive／Docs 文件唯讀來源或 Cloud Run revision rollback 中選擇至多一項窄版 P1；local folder 只作來源契約 fixture。這些不能被包裝成任意 agent、autonomous FDE、真正內網整合或完整 enterprise rollback。主要使用者先以 Solution Architect、Tech Lead 或 Delivery Architect 為起點；FDE、Implementation Engineer 與 DevOps 是待驗證的相鄰 persona。v4.2 保留 React 前端與 Go 後端，本文為目前實作範圍的真相源；v1、v2、v3 保留為歷史文件。Word 版由本 Markdown 產生。

## 01 決策摘要

ContextRail 協助新創與中小企業的 Solution Architect、Tech Lead 與工程團隊，在使用 AI coding agent 的變更進入指定環境前證明它仍符合被接受的意圖與環境政策。第一個案例是既有 Cloud Run 內部簽核系統新增附件：Project 先定義 Development、Testing、Staging、Production 四個環境及各自 gate，需求接受前比較架構、成本與未知項，Agent 實作後核對工作範圍與 PR／候選證據，發布前再核對 commit、image、目標環境設定、prod-demo 結果與 Release Receipt。大型企業的組織級權限、私有網路與多 provider 治理保留到後續 extension，不讓 P0 被企業級整合拖慢。

產品只有一個主要工作：environment-aware AI Change Assurance。Project Registry／Workspace 是這條工作流的入口與治理邊界，不是要取代 Jira 或建立完整 PM 平台。Architecture Impact Advisor、Cost Impact Advisor、Decision Support、Environment Topology、Agent Handoff Assurance 與 Release Assurance 是同一條證據路徑的支援能力，不是六套產品。RAG 提供具版本的文件脈絡，Gemini 提出解讀、work-order 草案與 evidence gap，計算器和規則處理金額及必要條件；外部 Claude Code、Codex 或其他允許的 coding agent 依受限契約產生程式候選。人員分別負責需求接受、候選接受與環境 promotion／發布核准。

技術決策為 React 前端加 Go 後端。ContextRail API 與 demo approval app 的 server-side 程式均使用 Go，但保留不同 build context 與部署身份。Go 是原型實作限制，不是客戶應用的語言限制。

產品定位鎖定為 Cloud-native 業務系統的 environment-aware AI Change Assurance；第一個案例是內部 Cloud Run 系統。ContextRail 擁有 Project Environment Topology、Promotion Policy、Change Contract、Agent Work Order、候選 evidence、人工核准、失效判定與 release identity；架構、成本與 RAG 是決策證據。Gemini、coding agent、ADC 或 Cloud Assist 的產出都先保持可引用候選。P0 不依賴 ADC 或 Cloud Assist 權限，也不在產品 Cloud Run 內託管任意 coding agent，避免外部 preview、個人憑證與通用 orchestration 阻塞核心交付。FDE 是交付角色與 operating model，不是本原型必須複製的產品類別。

CI/CD 負責建置、測試與部署；ContextRail 驗證 pipeline 正在處理的候選，是否仍對應被接受的需求、允許範圍、必要檢查、環境 promotion policy 及核准身份。它使用 Cloud Build 和 Cloud Run 作為第一條可驗證整合路徑，不重建 runner、source control、test framework 或通用部署平台。

核心 go／no-go 指標為：所有指定的意圖、工作範圍與 release identity 漂移都被阻擋，正常案例沒有誤阻擋；五組同條件比較必須完整計入首次設定與人工修正時間；至少兩位外部目標使用者能把產品描述為 AI 變更核對或放行，而不是 CI/CD、自動部署或 coding agent。淨 review 時間改善 30% 保持探索目標，不為達標而降低 mandatory checks。團隊自身的 Solution Architect 案例可作為 workflow 起點，但不計入外部市場驗證門檻。

五週原型的判斷為有條件可行。暫定容量為兩人各每週 25 小時，共 250 人時，分為 190 人時核心工作與 60 人時風險緩衝。使用者已確認兩人五週，尚未確認每週時數。這個估算假設兩人均能參與開發、第一週取得雲端權限，而且只實作一個受管應用。

目前 repository 仍無產品實作、無 commit、無 remote；以上為 2026-09-11 的本機查核結果。文件完成不代表功能完成。Agent provenance、競品重疊、付費意願、工時、檢索品質與成本估算準確度仍待驗證，詳見 [Critical Thinking Review](../reviews/CONTEXT_RAIL_CRITICAL_REVIEW.zh-TW.md)。

## 02 產品定位與目標市場

英文定位為 AI Change Assurance for AI-assisted changes to cloud-native business systems。

ContextRail verifies that an AI-assisted change still matches approved intent before it reaches a governed cloud deployment target.

AI Native Software Engineering control layer 是技術架構描述，不是第一層市場類別。Cloud Run 是初始部署切口，不表示未來只能服務 Google Cloud 或內部系統。

第一批客群假設是新創與中小企業中的 5 至 30 人工程團隊，維護至少一個有角色、審批、權限或資料保存規則的 Cloud-native 業務系統，使用 Git PR、CI 與至少一種 AI coding assistant，卻缺少專職平台團隊與 Agent 治理證據；更精確的切入條件是：同一 Project 有兩個以上由團隊自行管理的交付環境，且環境之間的 promotion 依賴人工判斷或不一致的清單。第一個 prototype 以 Google Cloud Run 為部署切口。客戶應用可以是內部系統或 B2B SaaS，先不追求消費型平台、多雲大型企業或重度法遵組織。附件、權限、保存期限及背景工作等跨層變更仍依賴 Solution Architect 或 Tech Lead 人工重建脈絡。主要使用者為 Solution Architect、Tech Lead 或 Delivery Architect；FDE、Implementation Engineer 與 DevOps 是次要驗證對象，PM 提供業務目標，工程師啟動受限 Agent 工作，QA 與 DevOps 補足驗證證據。初始購買者假設為 Engineering Manager；Delivery／Professional Services leader 是待驗證的相鄰 buyer hypothesis。大型企業保留為後續 extension 市場，只有在小型團隊能重複使用且願意持續投入後才驗證。

「企業內部系統」與「B2B SaaS」都是待驗證的問題類別，但 P0 的採用對象先是新創與中小企業，而不是大型企業。實際切入以附件、權限或保存期限等會跨越需求、架構與發布的變更為主，先找願意分享去識別化案例的團隊。本團隊具 Solution Architect 工作流經驗，因此先用自身歷史或匿名案例做 dogfooding；這可驗證流程和 artifact，不可直接推論跨公司需求或 FDE 市場成立。內部系統是第一個案例，B2B SaaS 是下一批驗證 cohort；Airbnb 類型的消費平台、多雲大型企業與重度法遵組織暫不列入初始市場，但保留未來擴展空間。

ContextRail 可以採 SaaS 交付；它服務的客戶應用可以是內部系統或對外 SaaS。這兩個維度分開，不以「AI 導致 SaaS 萎縮」作為選市場的前提，也不以未經訪談的痛點敘述估算 TAM。

## 03 問題與價值主張

要驗證的主要問題是：需求被接受時，架構代價與用量假設沒有被記錄；coding agent 開始工作時，原始意圖又被壓縮成缺乏權限與驗收邊界的 prompt；發布時，團隊必須重新從文件、Agent 對話、PR、測試與雲端設定中還原原決策。ContextRail 的單一工作不是讓 pipeline 跑得更快，而是讓團隊知道 pipeline 正在發布的 AI 變更是否仍是被核准的那一個。

| 角色 | 需要作出的決定 | 第一版提供的資訊 |
| --- | --- | --- |
| PM 或系統負責人 | 是否接受、修改或延後需求 | 業務目標、替代方案、成本增量與未知項 |
| Solution Architect／Tech Lead | 採用哪個架構方案 | 現況、ADR、依賴、IAM、資料流及風險 |
| 工程師與 QA | 如何限制 Agent 工作並驗證候選 | 接受條件、work order、run provenance、PR、review 與測試對照 |
| FDE／Implementation Engineer | 如何讓交付可交接與可重複 | 客戶或內部流程脈絡、驗收 owner、handoff gap 與 runbook 候選；P0 先以既有 evidence 產生，不另建完整 FDE 平台 |
| DevOps | 是否可以發布指定版本 | digest、設定、staging 證據與有效核准 |
| Engineering Manager | 是否值得導入工具 | 淨節省時間、關鍵遺漏、導入與維護負擔 |

產品價值必須以使用者實測支持。不能把「多整合幾個服務」「更多 Agent」或「更多文件」直接等同生產力提升。

## 04 核心展示案例

既有簽核系統已使用 Cloud Run 與 Firestore。新需求是讓申請人上傳附件，主管可依案件權限下載，附件有保存期限、容量限制與稽核需求。這是 P0 英雄案例，因為同一個需求同時需要業務決策、架構選擇、成本假設、Agent 範圍與發布身份。

系統讀取需求、現有架構、權限規則、保存政策與既有 ADR，顯示三個選項：

| 選項 | 架構變化 | 成本驅動因子 | 需要確認 |
| --- | --- | --- | --- |
| 維持現況或延後 | 不增加附件能力 | 既有成本與人工流程工時 | 延後的業務影響 |
| Cloud Run 代理傳輸 | Cloud Storage 保存檔案，API 代理上下載 | Run 計算時間、Storage、流量、metadata 讀寫 | 大檔案負載、逾時與權限測試 |
| 限時簽名 URL | Cloud Run 核發有限期 URL，檔案直接傳輸 | Storage、流量、核發 API、metadata 讀寫 | URL 有效期、重放風險、撤銷限制與 IAM |

兩個實作方案均有優缺點，不能預設簽名 URL 一定比較便宜或安全。產品只實作其中一個經人工接受的方案；另一個保留為可檢查的分析候選。IAM 與 bucket 由開發者按固定設定建立，不由模型臨時授權。

主展示包含三個不同層級的阻擋。第一次因下載量或保存期限未確認，成本狀態為 UNKNOWN，使用者補資料後才接受方案。第二次 coding agent 的候選超出允許路徑或缺少 acceptance test，候選不能被接受；修正後由獨立 review 與 CI 重新評估。第三次 staging 已部署，但部署設定不符原方案，發布被阻擋；修正後重新建置、驗證與核准，再部署到 prod-demo。

第二個驗證案例使用同一應用，將附件保存期限由 30 天改為 365 天，證明新分析能找出歷史決策並重新計算成本與政策影響；舊 Change Decision、Agent Work Order 與 Release Approval 會因來源版本變動失效。案例內容可合成，但模型、檢索、至少一次外部 coding-agent run、Git／CI 讀回、Cloud Build、Cloud Run 與 receipt 必須真實運作。

每個 demo case 都要同時展示兩份輸出：Change Decision Brief 先用一般語言說明「結論、影響、風險、需要誰決定與下一步」；Agent Context Pack 則以固定 schema 保存 accepted scope、out-of-scope、allowed paths、forbidden actions、acceptance tests、required checks、unknowns 與 evidence refs。兩份輸出不可各自重新解讀需求，必須共享 `decision_id`、版本與 `source_snapshot_hash`。主展示依序演示：正常接受附件方案、Agent 超出範圍而被阻擋、staging 設定或 digest 漂移而被阻擋，以及修正後同 digest promotion 至 prod-demo 並產生 Release Receipt。

## 05 使用流程與決策邊界

Project 在進入需求分析前，先建立 [Project Context Contract](../architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md)。P0 最少登錄 `project.yaml`、`README.md`、`vision.md`、`architecture.md`、`docs/engineering/development.md`、`docs/operations/environments.md` 與 `AGENTS.md` 的用途、受眾、SSOT／derived 角色、來源版本、有效時間、存取範圍、索引狀態與缺件原因。`ProjectDocument` 是來源登錄；`docs/ai/context-pack.json` 是可重建的衍生視圖，必須回指來源 path、version／commit、hash 與生成時間。文件缺少或衝突時進入 `NEEDS_INPUT`／`CONFLICT`，不能讓 Agent 以摘要補造事實。Project Workspace 的 Documents / AI Context 頁是這份契約的 P0 UI 投影。

```text
需求及設計參考
  → 文件與架構基線
  → RAG 檢索與來源核對
  → Gemini 分析與可選的 ADC 或 Cloud Assist 建議
  → 架構選項
  → 成本計算與風險說明
  → 人工接受變更
  → Project Environment Topology 與 Promotion Policy
  → 版本化 Agent Work Order
  → 外部 coding agent 建立 feature branch 和 PR
  → Agent Run Record 與 Git／CI 交叉驗證
  → Independent Review 與 Engineering Evidence Bundle
  → 人工接受、退回或拒絕候選
  → Cloud Build 與 Artifact Registry
  → 目標環境 gate 與 Cloud Run staging 驗收
  → 版本及政策 gate
  → 人工發布核准
  → prod-demo 部署與 smoke test
  → Release Receipt
  → 選配 Tag／Rollback／Outcome Proposal stretch
```

接受需求只允許建立 work order；接受候選只允許合併與進入指定的下一個環境；發布核准才允許 promotion。開發由團隊啟動其既有 coding agent 完成，ContextRail 不在 P0 託管任意 agent 或保存個人 agent 憑證。系統產生工作契約、接收執行聲明並用 Git／CI／Cloud Run 讀值交叉驗證，不把 Agent 自述當作 PASS。

### Project Environment Topology

環境拓撲是 Project 層級的治理設定，不是全域固定的四個下拉選項。使用者可以建立 `Development`、`Testing`、`Staging`、`Production`，也可以加入 `QA`、`UAT` 或企業自己的名稱；每個環境同時要指定標準類型、順序、target reference、允許動作、必要證據、owner、approver policy 與 config hash。環境名稱可以自訂，但標準類型為 `production` 的最低保護不能被改名或繞過。

| Project 設定 | P0 行為 | 非 P0 承諾 |
| --- | --- | --- |
| Environment definition | 保存環境 ID、顯示名稱、標準類型、順序、target ref 與 owner | 不宣稱自動發現所有雲端環境或成為 IaC source of truth |
| Promotion policy | 定義 predecessor／successor、允許 action、必要 evidence 與 approver separation | 不取代 Cloud Deploy、Git branch protection 或企業既有 RBAC |
| Environment evidence | 匯入 branch／PR／test／build／staging／runtime evidence，並綁定目標環境 | 不把一個環境的 PASS 複製成另一個環境的 PASS |
| Prototype execution | 真實執行 staging 與隔離 prod-demo；Development／Testing 以 evidence 與 gate 驗證 | Production 維持 read-only／blocked，不宣稱 production readiness |

`Code Review`、`Candidate Review` 與 `Release Approval` 是跨環境的決策 gate，不是環境。這個區分是本版的 niche 假設：產品治理的是變更穿越自訂環境拓撲時的意圖、證據與責任，不是再做一個 pipeline dashboard。

Decision Support 將業務價值、架構、成本與風險並列。若沒有使用者提供的收益、時效與優先順序，就只呈現技術代價，不輸出「ROI 已成立」或自動否決需求。政策門檻由專案負責人設定，版本化保存。

### Project Registry 與 Project CRUD

Project Registry 是所有受管 Project 的入口；Project Workspace 則處理某一個 Project 內的 Ticket／Change Request。P0 的 CRUD 是治理前置條件，不是獨立的專案管理產品：

| 操作 | P0 行為 | 驗收重點 |
| --- | --- | --- |
| 建立 | 建立 Project key、名稱、repo、owner、業務負責人、重要性、資料分類與初始 topology／policy | 缺少 repo、owner、target 或 production 最低保護時不能建立為 ACTIVE；建立操作可重試且不重複產生 Project |
| 查詢 | 顯示 active／paused／archived Project、環境數、開啟變更、阻擋 gate、`STALE` 判斷與最近 receipt | 讀取依 Project membership 過濾；列表是治理入口，不只顯示名稱 |
| 修改 | 修改 metadata、repo、owner、環境拓撲或 promotion policy，建立新版本與 audit event | topology／policy／repo 等必要輸入變更時，受影響的 DecisionRecord、Agent Work Order、Candidate／Release Approval 自動標記 `STALE` |
| 刪除 | API 可提供 archive action，但不做 P0 hard delete | 封存後不能建立新變更或 promotion；Decision Ledger、Evidence Bundle、Release Receipt 與 append-only audit 仍可查詢 |

最小 Project 契約為：

```text
Project
  project_id, tenant_id, key, name, description
  repo_ref, owner, business_owner, criticality, data_classification
  status, environment_topology_id, environment_topology_version
  promotion_policy_version, config_hash, version
  created_at, updated_at, archived_at, created_by, updated_by
```

P0 API 採明確的 archive 語義：`POST /v1/projects`、`GET /v1/projects`、`GET /v1/projects/{project_id}`、`PATCH /v1/projects/{project_id}` 與 `POST /v1/projects/{project_id}:archive`。`DELETE` 若日後提供，也只能映射到同一個封存流程，不得繞過證據保留。Project update 與 Ticket workflow 使用同一個 `project_id`；Project 設定變更不是靜默更新，而是會讓下游核准重新回到 `NEEDS_REVIEW` 或 `STALE`。

UI 至少包含 Project List、Create Project、Project Workspace 與 Project Settings。Settings 需明示「儲存後哪些判斷會失效」；Archive dialog 需明示「停止新變更與 promotion，但保留歷史證據」。本版另提供 [Figma-ready SVG](../../output/figma/context-rail-project-crud.svg) 作為三個畫面的靜態向量基線；它保留文字與形狀，不宣稱已完成 Figma component、Auto Layout 或 Prototype interaction。P0 不加入 billing、完整 SSO／SCIM、跨組織租戶管理、批量 migration 或完整 PM 功能。

### 雙視圖產出契約

Change Contract 與 DecisionRecord 是唯一真相來源；產品不讓 Gemini 直接各寫一份「給人」和「給 AI」的自由文字報告。系統先保存結構化決策，再以同一份資料渲染兩個視圖：

| 視圖 | 讀者 | 必要內容 | 不能做的事 |
| --- | --- | --- | --- |
| Agent Context Pack | Gemini、Claude Code、Codex、受限 coding agent | 目標、accepted scope、out-of-scope、架構與成本限制、allowed paths、forbidden actions、acceptance tests、required checks、unknowns、evidence refs、policy version、expiry、hash | 不含 secret 值；不授予核准、IAM 或發布權；不以模型自由敘述取代規則 |
| Change Decision Brief | PM、系統負責人、SA、Tech Lead、DevOps、非技術利害關係人 | 一句話結論、使用者影響、架構影響、成本假設、風險、需要誰決定、下一步；技術細節可展開 | 不把 UNKNOWN 寫成確定結論；不隱藏阻擋原因；不要求讀者先理解 RAG 或 provenance |

兩個視圖以及 Engineering Evidence Bundle、Release Receipt 共享 `decision_id`、版本、`source_snapshot_hash`、`policy_version`、`evidence_refs`、核准人與有效期限。任何必要輸入變動都使兩個視圖、Work Order 與 Release Approval 失效。人類視圖以固定句型翻譯技術狀態，例如「可以進下一步」「需要人工確認」「目前不能發布」「資料不足，無法判定」；原始 JSON 與證據只在技術展開層提供。

## 06 原型範圍與完成定義

### P0 五週核心成果

| ID | 功能 | 可觀察的完成證據 |
| --- | --- | --- |
| F01 | Project Registry／Workspace CRUD、環境拓撲與 tenant-ready 身份 | 兩個測試身份；可建立、查詢、修改與封存 Project；Project 與文件／決策資料具有穩定 tenant_id；可建立 Development、Testing、Staging、Production 四個自訂環境、排序並設定 target／policy／owner；Project 設定變更會產生新版本、audit event 並使受影響判斷標記 `STALE`；封存後不能新增變更或 promotion 但歷史證據可讀；未授權使用者無法跨 workspace 讀寫；不宣稱完成計費、SSO、商業化多租戶營運、hard delete 或自動環境發現 |
| F02 | Project Context Contract、版本化參考資料與 AI Context | ProjectDocument 登錄必要文件的用途、受眾、SSOT／derived、來源版本、有效時間、access scope、狀態與索引結果；可驗證 `project.yaml` 等最小文件集合，產生可重建且帶 source hash 的 Context Pack；8 至 12 份 TXT／Markdown；真實 embedding、retrieval、引用及一次來源更新失效 |
| F03 | 變更影響證據 | Gemini 對一個確認基線提出保留現況及兩個方案；Cloud Run、Storage、Firestore 的成本驅動因子、情境與未知項可回查 |
| F04 | Change Contract、需求決策與雙視圖 | 接受、修改、延後或拒絕；保存方案、成本假設、驗收條件、環境邊界、下一個允許 transition、理由與輸入版本；由同一 DecisionRecord 產出 Change Decision Brief 與 Agent Context Pack |
| F05 | Agent Context Pack、工程交接與 adapter 契約 | Agent Context Pack 編譯成一個 repo、branch 與目標環境的版本化 work order；一條真實 Claude Code 或 Codex run；provider／model／version、範圍與 commit 可回讀；P0 由開發者啟動，不在產品內託管任意 remote agent |
| F06 | 候選證據與雙視圖決策 | Git diff、Agent Run Record、PR／測試、build 與獨立 review 組成 Evidence Bundle；Evidence 與決策 brief 共享 ID；超範圍候選或缺少環境必要證據被阻擋，人員接受、退回或拒絕 |
| F07 | 建置與 staging | 已接受候選的真實 Cloud Build、digest、目標環境設定 hash、Cloud Run staging revision、整合與角色測試；Development／Testing 的證據不可被冒充為 staging PASS |
| F08 | 受控發布與雙層 receipt | 舊核准、錯 digest、缺測試、環境 policy 不符或 target drift 不能發布；人工核准後以同一 digest 部署隔離 prod-demo，回讀含人話摘要與機器欄位的 Release Receipt；一次 release 可包含多個 Change，但每個 Change 的 gate 都必須通過 |
| F09 | 差異化評估與交件 | 8 個情境案例、其中 2 個 holdout；5 組同條件 baseline、兩位操作、影片、英文材料與重現說明 |

P0 的 ContextRail API 與 demo approval app 後端均使用 Go，不保留 Python backend runtime。React 靜態資產與 Go API 可封裝在同一個產品映像中，維持三個 Cloud Run 服務的預算與操作模型；示範應用仍使用獨立映像和服務身份。Release Bundle 是 release-level grouping：P0 demo 以 2 至 3 個 feature、需求變更或 bug fix 組成一個 bundle，每個 Change 保留自己的 DecisionRecord、Work Order、Evidence Bundle 與阻擋原因，bundle 不能以其他 Change 的 PASS 代替缺失證據。

prod-demo 是合成資料的展示發布目標；通過原型驗收不等於可以操作客戶正式環境。產品本身的 hosting、受管 staging 與 prod-demo 共三個 Cloud Run 服務，預算均須涵蓋。P0 會展示一個可自訂四環境 topology，但只對 staging／prod-demo 做真實部署；Production 節點維持受保護的 read-only／blocked 狀態。

### P0 stretch 不屬於核心驗收

- 自動建立對應 commit 的 annotated Git release tag，包含重試與衝突狀態。
- 對上一個已知良好 revision 執行一次非破壞性 rollback 演練。
- 由短期 Outcome Observation 產生一份 evidence-linked Evolution Proposal。
- 擴充文件撤權、刪除與歷史決策回查測試。

只有 G4 PASS、總容量至少 300 人時、至少保留 40 人時風險緩衝，且契約、API／IAM／location 已驗證，才選擇至多一項 stretch 或 P1。這是所有窄版 P1／stretch 的唯一啟動 gate；250 人時基線只完成 P0。窄版 revision rollback 只涵蓋 Cloud Run revision，不代表資料庫、物件或 IAM 已回復。未完成不影響 F01 至 F09 驗收，也不能用示意資料宣稱已實作企業級能力。

### P1 額外容量且 G4 通過後才開始

- Application Design Center REST Adapter，匯入一個 sandbox application、revision 與 assessment report 並正規化為 ArchitectureAssessment；估計另需 12 至 20 人時，含 API、IAM、location、版本與失敗測試。
- Google Drive／Docs 選定單一文件的唯讀匯入及重新匯入；估計另需 12 至 20 人時，含 OAuth、權限與撤回測試。
- FDE Handoff Pack，將需求、背景、決策、驗收條件、未知項與 runbook 組成可交接資料；不自動代表客戶執行或發布。
- 一個固定的 remote Agent adapter，例如受限的 CI／GitHub Actions workflow；不支援任意 agent、任意 shell 或產品代管個人 coding-agent 憑證。
- 單一 Drive／Docs 文件唯讀來源 adapter；只驗證來源契約、版本與權限，不宣稱已解決 VPN、VPC、資料駐留或內網身份。local folder 只作本機 fixture／來源契約示範，不列為可宣稱完成的產品 adapter。
- Cloud Run revision rollback，回到上一個 approved digest 並保存 operation／receipt；不涵蓋資料庫、物件或 IAM 的跨資源復原。
- 一種簡單的 design snapshot 附件，保存引用，不宣稱已能分析所有 Figma 狀態。
- Cloud Tasks 或 Pub/Sub 的第二種成本模型及案例。
- 已授權帳務資料的離線匯入與觀測比較。

P1 沒有納入 190 人時核心基線，也不能消耗至少 40 人時風險緩衝。AI Change Assurance P0、G4 與交件優先；只有 G4 PASS、總容量提高至至少 300 人時、契約／API／IAM／location 已驗證且至少保留 40 人時風險緩衝後，才從上述候選選擇至多一項。Vibe coding 可以降低 boilerplate 工時，但不降低 adapter、權限、失敗、資料隔離與可回讀證據的驗收門檻。

### P2 後續產品化

Gemini Cloud Assist MCP、ADC remote MCP、完整 Figma API、Cloud Deploy、多 repo、多雲、任意 Terraform 解析與自動修改、企業 SSO／SCIM、細粒度文件 ACL、多租戶營運、真正 on-prem／private-network connector、任意 remote Agent orchestration、autonomous FDE agent、跨資料庫／物件／IAM 的完整 rollback、完整 FinOps、自動程式修復與 enterprise production readiness。Cloud Assist MCP 目前需要 private preview 權限；未取得不構成 P0 或 P1 失敗。

### 第一版支援層級

| 服務 | 有限情境架構說明 | 成本模型 | 自動資源配置 |
| --- | --- | --- | --- |
| Cloud Run | P0 | P0 限定計費模式 | 只更新已建立的展示服務 |
| Cloud Storage | P0 | P0 含容量、操作、指定路徑流量 | 不自動建立或變更 IAM |
| Firestore | P0 | P0 含已建模讀寫與容量 | 不自動移轉資料模型 |
| Application Design Center | P1 匯入 application、revision 與 assessment | 不作為價格來源 | P1 不執行 production deploy |
| Cloud SQL | 選項說明 | P2 或外部估算證據 | P2 |
| Cloud Tasks | 選項說明 | P1 | P2 |
| Pub/Sub | 選項說明 | P1 | P2 |
| Secret Manager | 選項說明 | 列為費項，未建模時顯示缺項 | P2 |

七種服務的 catalog 不代表七種服務都能估算或部署。Cloud Build、Artifact Registry、Gemini 等另屬產品執行成本，須獨立計入。

## 07 Architecture Impact Advisor

輸入為需求差異、設計參考、人工確認的架構基線、現況快照、公司規範、ADR 及使用量。來源優先序為當前受管資源讀值、版本化設定、經確認的文件；互相衝突時揭露差異，不自行挑選對模型有利的內容。

架構基線以結構化 JSON／YAML 保存元件、連線、region、runtime identity、資料類型、保存規則及負責人。P0 只讀取允許清單內的 Cloud Run 設定與已知 bucket／Firestore metadata，並使用人工確認的關聯。Cloud Asset Inventory 可在後續擴大資源查核；其資源資訊不能自行證明完整的應用語意依賴。[Cloud Asset Inventory](https://docs.cloud.google.com/asset-inventory/docs/asset-inventory-overview)

分析維度包括元件與依賴、API 與 schema、資料流與保存、IAM、安全、可靠性、效能、營運與 rollback。可參考 Google Cloud Well-Architected Framework，但框架檢查結果不是架構正確性的認證。[Well-Architected Framework](https://docs.cloud.google.com/architecture/framework)

架構 provider 使用同一個契約。P0 的 Gemini provider 讀取本產品已授權的 RAG 與架構基線；條件式 P1 的優先候選 ADC REST Adapter 只匯入指定 sandbox application、revision 及 assessment report，不讓 ADC 直接操作 prod-demo；P2 的 Cloud Assist MCP Adapter 只有在 private preview 權限已取得時才啟用。每個 provider 的建議都保持候選狀態，必須由使用者在 ContextRail 中選擇，不能用外部 agent 的執行成功取代人工需求核准。

ArchitectureImpactReport 的最小契約：

```text
 assessment_id, change_id, provider, provider_version
 provider_resource_refs[], raw_output_hash
 baseline_version, source_refs
alternatives[]
  option_id, description, affected_components[]
  required_changes[], dependencies[], tradeoffs[]
  security_notes[], reliability_notes[], migration_plan
  rollback_conditions[], acceptance_tests[], unknowns[]
findings[]
  finding_id, type, evidence_refs[], inference, status
selected_option_id, reviewed_by, reviewed_at
```

每個關鍵 finding 要能區分直接觀察、推論與未知。只找到文件中的服務名稱，不能推論該服務已存在；只有 snapshot 也不能推論所有下游都已盤點。必要資料缺少時列入 NEEDS_REVIEW。

## 08 Cost Impact Advisor

### 成本報告的責任

成本報告分開處理客戶應用的變更成本與 ContextRail 自己的營運成本。本節處理前者，第 15 節列出後者的預算。每個方案顯示同一時間範圍、幣別與使用量下的基線、變更後估算、差額和一次性開發／移轉工時。

Gemini、ADC 或 Cloud Assist 可以提出可能的成本驅動因子與建議情境；所有金額由 Decimal 計算器依版本化價格算出。計費單位、分級價格、region 與模式必須匹配，不能讓模型憑記憶填價。Cloud Assist 的 Cloud Billing 對話介面不回傳產品價格或特定 GCP 成本，因此不能作為 PriceRecord 或核准金額的來源。[Gemini Cloud Assist in Cloud Billing](https://docs.cloud.google.com/billing/docs/how-to/gemini/overview)

### 所需輸入與範圍

| 費項 | 必要輸入 | 常見遺漏 |
| --- | --- | --- |
| Cloud Run | 計費模式、billable CPU／記憶體時間、請求數、最小實例 | 冷啟動、閒置、並行與背景工作 |
| Cloud Storage | 平均保存 GiB、操作類別／次數、來源與目的地、保存期 | egress、版本、soft delete、操作費 |
| Firestore | region、讀寫刪除、儲存、適用索引讀取 | 查詢放大、向量索引讀取、備份 |
| 工程及營運 | 人工提供的工時區間與內部時薪 | 移轉、回復演練、值班與文件維護 |
| 其他服務 | 服務、region、使用量及價格來源 | 未被建模的項目不得計為零 |

P0 鎖定單一 region、USD 與一種 Cloud Run 計費模式，實際值於 Week 1 的雲端驗證後確定。費率來源為官方價格頁或可用的價格資料匯入；每筆保留 SKU 或來源表格定位、擷取時間、有效時間及來源 hash。價格快照有效期起始設為 30 天，期間若收到費率或方案變更則立即失效；這是產品政策，不是 Google 保證費率 30 天不變。[Cloud Run pricing](https://cloud.google.com/run/pricing)、[Cloud Storage pricing](https://cloud.google.com/storage/pricing)、[Firestore pricing](https://cloud.google.com/firestore/pricing)

MVP 不實作所有合約折扣、承諾用量折扣、稅金和帳戶層級免費額度的分攤。主要比較採未扣抵公開牌價；另列已知折扣及未估項目，避免把整個帳戶免費額度重複分配給每個方案。

### 計算方式與示例

每個 low、base、high 情境都使用同一套定價規則，先算各費項再相加。分級價格以各級距實際用量計算；每月增量為同情境的變更後金額減去基線金額。

```text
subtotal = sum(tiered_cost(quantity_i, price_record_i))
delta = proposed_subtotal - baseline_subtotal
one_time_cost = supplied_hours * supplied_hourly_rate
```

以下是計算器測試用假單價，並非 Google Cloud 報價，也不是產品成本預測。令測試儲存單價為每 GiB-month USD 0.02，測試流量單價為每 GiB USD 0.10：

| 情境 | 平均容量 GiB | 對外流量 GiB | 已知儲存及流量小計 USD |
| --- | --- | --- | --- |
| 低 | 20 | 10 | 1.40 |
| 基準 | 100 | 50 | 7.00 |
| 高 | 500 | 250 | 35.00 |

若 Run 計算、Firestore 或操作費未提供，小計依然可顯示，但 total 為未知、coverage 為 PARTIAL；不能把 USD 35.00 當成總成本或預算上限。展示 UI 必須區分 TEST_FIXTURE 與官方價格快照。正式 demo 改用已查核單價、保留相同計算測試。

low／base／high 是使用者確認的三種情境，不是統計信賴區間。Cloud Run 的 billable time 不能簡化為請求數乘回應時間，除非所選模式與並行假設已明確成立；用少量 staging 觀測校正假設後仍須顯示局限。

### 成本契約與 gate

```text
CostImpactReport
  estimate_id, change_id, option_id, estimator_version
  region, currency, period, price_snapshot_id
  assumption_set_id, baseline_ref, scenarios[]
  line_items[], known_subtotal, total_if_complete
  coverage, excluded_items[], unknowns[]
  one_time_effort_range, budget_limit, budget_status

PriceRecord
  service, sku_or_locator, unit, region, currency
  tiers[], billing_mode, source_url
  retrieved_at, effective_at, source_hash, price_status
```

P0 的預算規則比較同範圍的「變更後 high 情境總額」與使用者指定門檻。報告明列 cost_scope 與 required_cost_items；已排除稅金或合約折扣的結果標為限定範圍估算。缺關鍵用量、必要費項或過期價格時為 UNKNOWN；超門檻為 FAIL；範圍內資料完整且符合設定門檻才是 PASS。這只代表在已列明的假設下通過，不是未來帳單保證。即使差額很小，只要使用者關心的總額缺項，也不能顯示整體預算通過。

### 實際成本的回饋

staging 可查核 max instances、CPU、記憶體、保存設定等成本驅動配置。短期觀測與完整月帳務分開保存。未來透過 Cloud Billing export 匯入實際成本，記錄資料涵蓋期間與到齊程度；它是持續匯出流程，不能當成部署完成瞬間的月帳單驗證。[Cloud Billing export](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery)

## 09 RAG 與累積機制

### 從第一週建立

原始文件放入 Cloud Storage；Firestore 保存 DocumentVersion、chunk、embedding、來源 metadata 及決策關聯。Embedding 由指定 Google 模型產生，Firestore 執行向量檢索；它不會自行替文件產生 embedding。使用向量索引與前置條件過濾，維度需符合所選服務限制。[Firestore vector search](https://docs.cloud.google.com/firestore/native/docs/vector-search)

P0 限制為 8 至 12 份 TXT／Markdown、每份不超過 100 KB、每個 corpus 最多 300 chunks。初始 chunk 大小約 400 至 700 tokens、保留標題路徑，檢索 top-k 起始值為 6；這些是待評估參數。Embedding 初選 768 維，Week 1 驗證模型支援該輸出維度，避免超過 Firestore 2048 維限制；不同模型或維度的索引不得混查。PDF／OCR、任意 URL 抓取與整個 repo 的自動語意解析延後。

### 文件生命週期

```text
SELECTED → AUTHORIZED → PARSED → VERSIONED
  → CHUNKED → EMBEDDED → ACTIVE
  → SUPERSEDED 或 REVOKED 或 DELETED
```

使用者需明確選擇要上傳的檔案。先驗證工作區身份與文件狀態，再檢索；必要政策與架構基線可用 ID 定位直接取回，語意檢索處理補充資料。不能因 top-k 沒找出某條規範就宣稱不存在規範。

DocumentVersion 至少包含 workspace_id、source_type、source_id、version、content_hash、locator、collected_at、effective_at、owner、access_scope、status 與 supersedes。Chunk 另含 document_version_id、heading_path、offset、embedding_model、embedding_dimension、index_version。

P0 只支援專案共用文件與私有上傳文件兩種範圍。採可驗證的實體集合或權限 cohort 與前置 filter；後端再核對結果和目前 membership。後端 server SDK 會繞過 client Security Rules，須同時使用 IAM 與應用層授權，不能只測前端隱藏效果。[Firestore Security Rules](https://docs.cloud.google.com/firestore/native/docs/security/get-started)

重複匯入相同 hash 不重建 embedding。P0 驗證更新使舊版退出預設檢索，並使依賴該版的判斷失效。產品化仍要求撤權或刪除後立即禁止檢索、清除 chunk、embedding 與相關 cache，且稽核紀錄只保留可依法保留的 metadata；完整撤權、刪除及重試測試列為 stretch，核心 demo 僅使用合成或明確授權資料。

### RAG 與決策記憶的分工

RAG 找文件；Decision Ledger 保存接受或拒絕的理由；Evidence Graph 保存 requirement、work order、agent run、candidate decision、PR、test、deployment 與 receipt 的關聯。P0 使用 Firestore 集合與 edges，不建獨立 graph database；outcome 與 evolution proposal 只有在對應 stretch 被選中時加入。

AI 建議是候選資料。人工接受的 ADR 具專案和有效期間，不自動升格為公司政策。歷史結果可用於後續檢索與離線評估，不進行線上自我訓練或自動改政策。引用可定位只證明來源存在，還要抽查來源是否真的支持該主張。

## 10 AINE Assurance 與資料契約

ContextRail 實作一條受限的 AI Native Software Engineering vertical slice，不宣稱重建完整 Product Foundry。AINE 在本產品的三項責任如下：

| AINE 責任 | P0 對應能力 | 明確邊界 |
| --- | --- | --- |
| Constitution | 權限、版本、證據、PASS／FAIL／UNKNOWN 與三次人工決策 | Agent、模型、CI 或部署成功都不能自行核准 |
| Compiler | Change Contract 編譯成 Agent Work Order，再組成可重現的候選與 release evidence | P0 只有一個 repo 和一條由開發者啟動的 Agent path；P1 最多增加一個固定 adapter，不是通用 executor |
| Evolution Engine | P0 stretch 以 Release Receipt 和短期 outcome 產生 evidence-linked proposal | 不屬於 AINE assurance slice 的核心必要條件；只有宣稱 Evolution Engine 時才需實作，且提案不自動改需求、程式、政策、部署或 rollback |

需求決策、候選決策與發布決策分開，避免「需求已接受」或「Agent 已完成」在 UI 或程式裡變成「可直接上線」。

| 物件 | 主要狀態 | 誰能改變關鍵狀態 |
| --- | --- | --- |
| Environment Topology | DRAFT、ACTIVE、STALE、ARCHIVED | Project owner／admin 定義；環境政策、順序或 target 變動由系統標記 STALE |
| Promotion Policy | DRAFT、ACTIVE、STALE、BLOCKED | Project owner／admin 定義；不得移除 production 最低保護 |
| Change Decision | DRAFT、ASSESSING、NEEDS_REVIEW、ACCEPTED、REVISE、DEFERRED、REJECTED、STALE | 人員接受或拒絕；輸入變動由系統標記 STALE |
| Agent Work Order | DRAFT、READY、ISSUED、STALE、CLOSED | 人員確認後簽發；來源或政策變動由系統標記 STALE |
| Agent Run Record | RECEIVED、INVALID、EVIDENCE_MISMATCH、MATCHED、NEEDS_REVIEW | 系統驗證 schema 與外部讀值；人員處理未知 |
| Candidate Review | COLLECTING、BLOCKED、NEEDS_REVIEW、READY、ACCEPTED、REVISE、REJECTED、STALE | 規則產生 readiness；人員接受、退回或拒絕 |
| Release Review | COLLECTING、BLOCKED、NEEDS_REVIEW、READY、APPROVED、DEPLOYING、RELEASED、FAILED、UNKNOWN | 規則產生 readiness；人員核准；執行器更新結果 |
| Policy Check | PASS、FAIL、UNKNOWN | 確定性檢查；保存政策版本 |

Candidate READY 必須滿足 work order 有效、commit 與 diff 可讀回、修改路徑在範圍內、required checks 完整、架構／成本 drift 已處理、目標環境存在於 ACTIVE topology，而且有分開的 review record。Release READY 另需 promotion policy 允許該 transition、所有 mandatory checks 為 PASS、沒有未處理的 required finding、目標 revision 與 target identity 相符。任一 FAIL 為 BLOCKED；資訊缺少、Agent 自述與 Git／CI 不符、環境 policy 不明或模型輸出不合法為 NEEDS_REVIEW。UNKNOWN 不能悄悄當作 PASS。

EnvironmentTopology 保存 topology_id、project_id、version、environment_id、display_name、normalized_type、sequence、target_ref、allowed_actions、required_evidence、owner、approver_policy、config_hash、effective_at 與 status。PromotionPolicy 保存 predecessor／successor、transition_id、required_evidence、approval_separation、allowed_delta 與 policy_version。它們是 ContextRail 的治理 metadata，不取代 Terraform、Cloud Run 或 Cloud Deploy 的 infrastructure／deployment source of truth。

DecisionRecord 保存 change_version、architecture_assessment_id、selected_option_id、cost_estimate_id、source_snapshot_refs、environment_topology_id／version、target_environment_id、allowed_transition_id、policy_version、reviewer、decision、reason 與 timestamp。

AgentWorkOrder 保存 work_order_id、change_id／version、selected_option_id、repo、base_branch、target_branch、target_environment_id、allowed_transition_id、allowed_paths、forbidden_actions、acceptance_ids、required_checks、source_snapshot_refs、policy_version、expires_at 與 work_order_hash。

AgentRunRecord 保存 run_id、work_order_id／hash、adapter、agent、provider、model／version、started_at、finished_at、commit_sha、changed_paths、declared_commands、sandbox／permission declarations、raw_evidence_hash 與 validation_status。P0 由開發者匯入或 CI 提交結構化紀錄；它是來源聲明，必須與受管 Git 和 CI 讀值比對，不能單獨證明誰產生了每一行程式。

EngineeringEvidenceBundle 綁定 candidate_id、work order、run record、commit_sha、diff_hash、pull_request_ref、target_environment_id、environment_config_hash、test／build／review evidence、architecture／cost drift、source snapshots 與 candidate verdict。Independent Review 必須使用不同 run_id、reviewer role 與人工 reviewer identity；reviewer 不能同時是候選作者或 work-order issuer。兩人原型若只能做到角色分離，仍要記錄兩個 actor identity；無法分離時狀態為 INCONCLUSIVE／NEEDS_REVIEW，不能宣稱 independent review。相同模型可以作第二次執行，但不能把同一次生成回覆標成獨立審查。人員 CandidateDecision 保存 ACCEPTED、REVISE 或 REJECTED 及理由。

ReleaseManifest 綁定已接受 candidate、change decision、commit_sha、image_digest、test_report_hash、source_environment_id、target_environment_id、staging_revision、staging_config_hash、prod_demo_revision、prod_demo_config_hash、allowed_environment_delta_hash、target_service、policy_version。每個實際執行的環境設定分別 hash；allowed environment delta 必須列出可接受差異並納入核准。設定 hash 涵蓋 region、runtime service account、CPU／記憶體、伸縮、環境變數和 secret version references，不包含 secret 值。

Approval 保存 manifest_hash、approver、approved_at、expires_at。MVP 預設 24 小時有效期為產品政策假設；執行前重新核對身份、來源撤權、work order、candidate、版本與目標狀態。任何必要輸入變動需重算與重新核准。設定 hash 使用正規化的語意欄位，排除單純重新讀取造成的 collected_at 變化，避免每次查詢都使核准失效。

若執行 stretch，OutcomeObservation 保存 release_receipt_id、observation_window、runtime metrics、product_signal、cost coverage、limitations 與 result；EvolutionProposal 保存 observation refs、proposed_change、baseline、candidate evaluation criteria、rollback requirement、reviewer decision 與 lineage。這些物件不屬於核心完成定義，也不能用短期 staging 或 prod-demo 資料宣稱已證明長期業務價值。

## 11 Google Cloud 系統架構

### 產品與受管應用分離

```text
使用者
  → ContextRail Cloud Run
      React SPA + Go HTTP API
      Go 模組化單體
      身份驗證、Project 與環境拓撲授權
      RAG / 架構分析 / 成本計算 / 規則
      ├→ Gemini 與 embedding
      ├→ ADC REST Adapter P1
      ├→ Cloud Assist MCP Adapter P2
      ├→ Firestore 文件索引及決策
      ├→ Cloud Storage 原文及證據
      ├→ Agent Work Order / Evidence ingestion
      └→ 固定 promotion trigger
             ↓
開發者 → 外部 Claude Code／Codex adapter
             ↓ Agent Run Record
Git feature PR → Independent Review → Candidate Decision
             ↓
        Cloud Build → Artifact Registry
             ↓             同一 digest
    demo-app-staging → demo-app-prod-demo
             ↓                  ↓
           測試              smoke 與回讀
             └────→ Release Receipt → 選配 Outcome Proposal
```

ContextRail 的應用部署與受管示範應用使用不同 build 設定，避免工具更新意外改變評審正在看的示範版本。MVP 以單一 sandbox GCP project、三個 Cloud Run 服務、分開的 service accounts 與資料命名空間實作；這不是企業正式環境的隔離架構。

| 元件 | P0 責任 | 取捨 |
| --- | --- | --- |
| React 與 Go API | UI、API、工作流狀態與計算 | React build 封裝入 Go 服務映像；維持模組化單體 |
| Cloud Run | 產品 hosting 與兩個展示 target | min instances 起始為 0；記錄冷啟動 |
| Gemini 與 embedding | 有限步驟分析及檢索向量 | Week 1 鎖定可用 model、region、SDK |
| Coding Agent Adapter | P0 輸出 work order、匯入一條外部 run 與 review evidence；P1 可接一個固定 remote adapter | 不在產品服務內保存個人憑證或執行任意 shell |
| Application Design Center | P1 匯入指定 application、revision 與 assessment | REST API 可用、額外容量確認且 G4 通過後才實作 |
| Gemini Cloud Assist | P2 雲端分析候選；若資格已取得只做隔離 spike | MCP 為 private preview，不是交付依賴 |
| Firestore | Project、專案 membership、環境拓撲、promotion policy、版本、向量、決策、edges、核准與 audit event | 保存治理 metadata；不另建 SQL 或圖資料庫，也不取代 IaC／雲端資源狀態 |
| Cloud Storage | 原文、測試與分析快照 | 有期限與存取範圍；public demo 不公開原文 |
| Cloud Build | PR checks、建置、staging 及 promotion | 以 build ID 和已知 repo 核對來源 |
| Artifact Registry | 不可變 image identity | digest 為發布身份，tag 為人可讀標記 |
| Firebase Authentication | 登入及已驗證身份 | 專案 membership 由後端檢查 |
| Secret Manager | Git 等必要秘密的引用 | GCP 服務盡量使用 service identity |
| Cloud Logging | 精簡執行與錯誤紀錄 | 限制原文與高量 debug log |

Go API 以標準庫 `net/http` 為預設，不在沒有明確需求時增加大型 Web framework。依賴由 `go.mod` 與 `go.sum` 鎖定；Week 1 選定同時符合所用 SDK 支援條件的 Go 版本，Cloud Build 不使用未固定的 `latest` 工具鏈。主要 client 為 `google.golang.org/genai`、`cloud.google.com/go/firestore`、`cloud.google.com/go/storage` 與 `firebase.google.com/go/v4`。Firebase ID token 先由 Admin SDK 驗證，再由應用層檢查 workspace membership。[Google Gen AI SDK for Go](https://cloud.google.com/vertex-ai/generative-ai/docs/sdks/overview)、[Firestore Go client](https://docs.cloud.google.com/go/docs/reference/cloud.google.com/go/firestore/latest)、[Cloud Storage Go client](https://docs.cloud.google.com/go/docs/reference/cloud.google.com/go/storage/latest)、[Firebase ID token verification](https://firebase.google.com/docs/auth/admin/verify-id-tokens)

Go domain 定義 `ProjectStore`、`ProjectMembershipStore`、`EnvironmentTopologyStore`、`PromotionPolicyEvaluator`、`ArchitectureAdvisor`、`AgentEvidenceAdapter` 與 `ReviewAdapter` 介面，以及版本化 `Project`、`EnvironmentTopology`、`PromotionPolicy`、`ArchitectureAssessment` 和工程證據契約。Project store 的 update／archive 必須以 transaction 或等價的版本檢查保存 audit event，並觸發受影響判斷的 stale propagation；不能只更新 UI 狀態。Provider adapter 只負責驗證身份、呼叫外部服務及正規化結果。P1 以 Design Center REST API v1 為優先，只有必要功能尚未提供時才明示使用 v1alpha；保存 application、revision、assessment 與 long-running operation ID。P2 的 MCP agent 輸出同樣必須通過 schema、來源與權限檢查，不得直接寫入 DecisionRecord、CandidateDecision 或 ReleaseManifest。[Design Center REST API](https://docs.cloud.google.com/application-design-center/docs/reference/rest)、[Cloud Assist MCP integration](https://docs.cloud.google.com/cloud-assist/configure-mcp)

服務遵守 Cloud Run container contract，從 `PORT` 啟動 HTTP server，為外部呼叫設定 deadline，並傳遞 `context.Context` 以支援取消。goroutine 只處理當前請求內可取消的並行工作，不能作為持久背景佇列；跨請求分析、build 與 deploy 狀態寫入 Firestore，並以外部 operation ID 恢復。[Cloud Run Go quickstart](https://docs.cloud.google.com/run/docs/quickstarts/build-and-deploy/deploy-go-service)

P0 不把 Cloud Deploy 加為第二套發布控制系統。先用受控 Cloud Build 及固定 target 做到底；後續 adapter 可以對接 Cloud Deploy。[Cloud Build to Cloud Run](https://docs.cloud.google.com/build/docs/deploying-builds/deploy-cloud-run)

分析採有限步驟與可重試 operation record；每次 operation 有 timeout、輸入 hash 和狀態。Cloud Run request 結束後不依賴無人持有的記憶體背景工作繼續跑。較長的 build／deployment 以外部 operation ID 回讀，重啟可恢復狀態。

### Adapter 的範圍

DocumentAdapter、ArchitectureAdvisor、AgentEvidenceAdapter、ReviewAdapter、GitAdapter、CIAdapter、RuntimeAdapter、PriceAdapter 與 DeploymentAdapter 只做必要契約，沒有泛用 plugin marketplace。相同 domain model 可更換來源，但每個新 adapter 仍需處理權限、網路、版本、刪除與同步。P0 的 coding-agent adapter 可以是本機／CI 輔助命令與結構化匯入，不要求 ContextRail 遠端啟動 Agent。

Drive／Docs 是便利的來源；P1 至多做一個選定文件的唯讀匯入，本機 local folder 只作來源契約 fixture／示範，不列為產品 adapter。上傳本機文件會將內容送上雲；若企業規定文件不得離開內網，需另做企業端檢索與雲端模型資料邊界設計。僅有 adapter 介面不能承諾解決資料駐留、斷網使用或內網身份驗證。

## 12 AI 工作流與安全執行

第一版由程式安排固定的 Gemini 步驟：需求抽取、架構候選、成本因子整理、環境影響與 promotion path、work-order 草案、evidence gap 與發布說明；outcome proposal 只在 stretch 執行。P1 可把既有 evidence 組成 FDE Handoff Pack，但仍需人員確認。成本計算器、來源查核、環境 policy 評估、Git／CI 比對與部署 controller 是一般程式。外部 coding agent 依團隊明確啟動的一份 work order 工作；對外可稱 AINE-compatible multi-step workflow，但未完成自主工具選擇及其評估前，不宣稱多個自主 Agent 已能無人協作完成開發。

模型輸出用固定 schema 驗證；引用必須來自本次允許的 source set。重試最多兩次，輸出失敗保留錯誤供人處理；同一模型與 prompt 並不保證生成文字完全一致。

ADC、Cloud Assist、coding agent 與 review agent 的回傳和一般模型輸出相同，都是不受信任的候選資料。Adapter 使用最小 IAM，保存 provider、model／version、resource、revision、operation 或 run ID 與回傳 hash；權限不足、資源不可見或結果不明時輸出 UNKNOWN 或 NEEDS_REVIEW，不能推論資源不存在。外部 advisor 與 review agent 沒有 Git 寫入、IAM 變更、prod-demo 部署或人工核准權限；coding agent 的 Git 權限只限指定 repo／branch，不取得 deployment secrets。

文件、PR 說明、issue 與測試 log 都是資料，不能擴張 work order 或成為額外工具指令。Gemini 無 Git 寫入、IAM 變更或發布權限。Coding agent 只依簽發 work order 和本身明確啟動時的沙箱規則執行；任何超出 allowed_paths、base commit、禁止動作、目標環境或時效的結果都不能成為 READY。需執行的雲端動作由後端解析既定 action，檢查 target allowlist、environment transition 與有效核准。

Agent Run Record 不等於不可否認的 provenance。P0 以 work-order hash、Git commit／diff、CI build identity、時間與 adapter 版本做交叉一致性檢查，仍明示無法證明 Agent 是每一行變更的唯一作者。獨立 review 需有不同 run_id 與 reviewer role；即使 review PASS，人工候選決定與發布核准仍然分離。

CI 中執行的是 repo 程式碼，與純讀取文件的信任不同。PR 檢查使用無 deployment secrets、無 prod-demo 寫權限的身份。合併後 build 和 promotion 分別用有限角色；在 API 身份下不能執行任意傳入的 build YAML 或 shell。Promotion 使用服務端固定版本的定義並重新讀取有效 Approval；使用者只能提供 manifest ID，不能覆寫 target、service account 或 build definition。

## 13 Feature 開發與測試

沿用使用者目前的 develop 分支作為唯一長期整合分支，不採完整 Git Flow。遠端建立後由兩人設定保護規則；新 feature branch 以變更 ID 命名，工具建立時採 codex/REQ-123-attachments 等前綴，其他成員可使用團隊慣例。

每個 PR 包含 change_id、work_order_id／hash、agent_run_id、被接受的方案、source／target environment、allowed transition、相關 acceptance IDs、測試計畫、風險與 feature flag。ContextRail 從受管 Git 讀取 base／head commit、diff 與 changed paths，和 Agent Run Record 比對。短命分支只有在 Candidate Review READY 且人員選擇 ACCEPTED 後才合併回 develop；未完成功能保持關閉，不因 Agent 完成、review PASS 或 merge 就宣稱需求完成。Feature branch 是工程協作的實作單位，Environment Topology 是 promotion 的治理單位；兩者不可混為同一個狀態。

Go PR gate 依序執行格式、靜態檢查、測試與編譯：`gofmt` 不得留下差異，`go vet ./...`、`go test ./...` 與 `go build ./...` 必須通過。涉及交易鎖、核准失效或並行狀態的 package 另執行 race test。Go struct 是程式內型別，跨前後端與模型輸出的契約仍以版本化 JSON Schema 和 golden fixtures 驗證，避免只靠編譯通過判定相容。

| 層級 | 執行位置 | 目的 |
| --- | --- | --- |
| Unit | 開發與 PR build | Go package、成本單位、狀態轉移、政策與版本失效 |
| Contract | PR build | 模型 schema、來源引用、adapter 與 manifest |
| Agent Evidence | PR build 與 ContextRail | work-order hash、允許路徑、run metadata、diff、review 與 commit 對照 |
| Integration | 受控 CI | Firestore、交易鎖、權限、相同 digest 的識別 |
| E2E | Cloud Run staging | 匯入、分析、接受、檢查、阻擋與核准 |
| Scenario QA | staging | requester、manager、admin 的允許及拒絕流程 |
| Release smoke | prod-demo | 目標 revision、基本操作與 receipt 回讀 |

因此產品管理的證據不限於整合測試。Coding agent、review agent、CI 和測試工具都只產生候選或結果，ContextRail 判斷它們是否對應本次需求、工作範圍與 commit；它不自行取代測試框架，也不宣稱 Agent 自述或閱讀 test 名稱即可證明行為已驗證。

## 14 Cloud Run 發布與 Release Receipt

Feature PR 的 Engineering Evidence Bundle 通過、人工接受候選並合併後，develop build 產生一次 image，保存 commit SHA 與 digest，先部署 staging。與候選作者及獨立 review identity 分離的 Release Approver 查看來源、work order、Agent／review evidence、架構／成本方案、staging 測試結果及兩環境允許差異後作出發布核准，後端固定 action 才能以同一 digest 啟動 promotion。

staging 與 prod-demo 是不同 Cloud Run services，revision 名稱不能跨服務搬移。promotion 是把相同 digest 部署至 prod-demo，取得新的 revision，再核對兩個環境的設定 hash 與已核准的 allowed environment delta；digest 相同不代表環境相同。[Cloud Run deployment](https://docs.cloud.google.com/run/docs/deploying)

執行器在交易中取得 manifest 對應的鎖與 idempotency key；重複請求返回同一 deployment operation。build/deploy timeout 時先讀回真實狀態，結果不明標記 UNKNOWN，不再盲目重跑。

P0 的成功終點是 prod-demo smoke 通過、deployment receipt 已持久化且能從 receipt 回到 Change Contract、CandidateDecision、Approval、commit、digest、staging／prod-demo config hash、allowed environment delta 與實際 revision。Tag、revision rollback 與 outcome 不作為核心發布成功條件；revision rollback 只可在唯一 P1 gate 通過後作為一項窄版 P1／stretch。

### P0 stretch 的 Tag Rollback 與 Outcome

若選擇 Git tag stretch，系統在 receipt 完成後於受管 repo 的該 commit 建立 annotated tag。名稱採 release-YYYYMMDD-shortSHA，同日相同版本重試沿用同一 tag；不實作 semantic-release。Tag annotation 保存 receipt ID、digest 和 revision。

| 事件 | 正確行為 |
| --- | --- |
| tag 已存在且指向同 commit | 核對 receipt 後回報已完成，不覆寫 |
| 同名 tag 指向不同 commit | 回報衝突，停止 tag 步驟 |
| 部署成功但 tag 建立失敗 | RELEASED 且 stretch tag_status FAILED；可重試 tag |
| 部署失敗或 outcome UNKNOWN | 不建立成功 tag |
| tag 被推送 | 不因此自動觸發 prod-demo 部署 |

Git tag、Artifact Registry tag、Cloud Run traffic tag 與 GCP labels 是不同物件。Stretch 自動化的是 Git release tag；環境標籤另由部署設定管理。CI trigger 必須排除 release tag，避免循環部署。

若選擇 rollback stretch，以同一 prod-demo service 的上一個已知良好 revision 為目標，保存操作與回讀結果。應用 rollback 不代表資料庫、物件或 IAM 已回復；核心案例只使用可向後相容的變更，破壞性 migration 在發布前阻擋。若選擇 outcome stretch，短期觀測只能形成待人工評估的 proposal。

## 15 開發與展示預算

以下是五週開發加交件後保留期間的暫定預算分配，不是已發生費用或可保證的報價。雲端 billing 與實際 region 尚未在本次文件工作中查驗，也沒有執行付費部署。

| 預算項目 | 規劃額度 USD | 範圍 |
| --- | --- | --- |
| Gemini inference 與 embeddings | 80 | 開發、8 案例多次評估、保留案例重跑及展示 |
| Cloud Run | 45 | 產品、staging、prod-demo 三服務 |
| Cloud Build 與 Artifact Registry | 35 | 建置、失敗重試與映像保存 |
| Firestore 與 Cloud Storage | 25 | 文件、向量、決策與證據 |
| Log、網路與其他用量 | 25 | 日誌、egress、secret 操作等 |
| 交件後保留 | 40 | 至少規劃至官網評估期結束 11/6 |
| 費用緩衝 | 50 | 未預期用量與必要修復 |
| 合計 | 300 | 不含人力與既有 coding assistant 訂閱 |

USD 300 是本版建議的預算容量，須在啟動時由帳單負責人確認。免費額度、比賽 credits 或合約折扣皆未預先扣抵。若只能使用 USD 200，先減少模型評估重跑與閒置資源、重新確認評審期間保留成本，不把必要測試刪成零。

每次 analysis 保存 input/output tokens、embedding 使用量、retry 次數與估算執行成本；設定每日分析次數、檔案與 chunk 上限，受理前拒絕超出配額的新增工作。小型 Cloud Run 採 min instances 0 並限制伸縮；這些措施降低開銷，不構成總帳單硬上限。

設定 50%、80%、100% 的 alerts-only 預算提醒。一般提醒不會自動停止計費；若選用其他 spend-cap 功能，需另驗證其服務支援與限制，不能假定所有 GCP 費用都被涵蓋。[Cloud Billing budgets](https://docs.cloud.google.com/billing/docs/how-to/budgets)

## 16 Business Model Canvas

以下九項是待驗證的商業假設；Solution Architect-first 是目前的進入假設，FDE-compatible 是待外部驗證的延伸假設。Environment-aware 是本版用來縮小市場問題的 wedge，不是已證明的護城河。

| 區塊 | v4.2 假設 |
| --- | --- |
| Customer Segments | **Prototype 先找新創與中小企業**：已用 Google Cloud（首個切口為 Cloud Run）與 AI coding assistant、維護審批型內部系統或 B2B SaaS、沒有專職平台團隊且有兩個以上自訂交付環境的 5 至 30 人工程團隊；主要接觸 Solution Architect／Tech Lead／Delivery Architect，並驗證 FDE、Implementation 與 DevOps 的相鄰需求。大型企業保留為 enterprise-ready extension，先找願意分享實際變更的 design partner，不把大型企業導入列為 P0 成功條件 |
| Value Propositions | 提供 environment-aware AI Change Assurance，以同一份版本化決策同時產出人看得懂的 Change Decision Brief 與 Agent 可執行的 Context Pack，再連接 Agent 工作、候選證據、環境政策與部署；來源、範圍、環境拓撲或交付身份漂移時使舊核准失效，減少重查、錯放行與返工 |
| Channels | Git／PR workflow、Solution Architecture 與交付社群、團隊既有企業接觸、Google Cloud 社群與雲端顧問；無既成合作承諾 |
| Customer Relationships | 協助匯入一個系統與一條 Agent 工作路徑、以團隊自身案例 dogfood、共同整理案例、短期 pilot 與定期回顧 |
| Revenue Streams | 固定範圍的付費 design-partner pilot，優先驗證新創／中小企業是否願意為可重複的 Project assurance 付費；驗證後採每個受管應用或團隊訂閱；大型企業導入、connector 與 Agent adapter 作為後續 extension 單獨報價 |
| Key Resources | Change Contract、EnvironmentTopology、PromotionPolicy、Agent Work Order、Engineering Evidence Bundle、provider-neutral assessment、release／outcome 契約、版本化價格與評估案例 |
| Key Activities | 驗證 Agent provenance、語意／範圍／環境漂移、candidate／release identity、引用品質與成本；維護 connector 並降低導入工時 |
| Key Partners | 潛在 Google Cloud 生態、coding-agent／Git／CI 平台、系統整合商；不假設已有合作或 preview 權限 |
| Cost Structure | AI 用量、資料與運算、Agent／connector 維護、客戶導入、支援及安全工程 |

定價先驗證客戶是否願意為一個固定範圍 pilot 付費，以及誰擁有預算和採購權。只有當單一應用的首次設定可在可接受時間內完成、每月確有重複變更，而且淨節省時間可重現時，才測試 recurring subscription；現階段不使用 USD 199 或每次變更計價作為既定方案，避免低估導入和支援成本，或鼓勵使用者略過必要複查。

單位經濟需把 pilot 或訂閱收入，扣除 AI、運算、儲存、持續支援及攤提導入成本；不能只扣 API 費就宣稱高毛利。定價、支援工時和導入攤提尚無實測，因此本版只定義計算欄位，不提供看似精確的毛利案例。

五週目標是 3 至 5 位潛在使用者訪談、至少兩位外部角色操作原型、先完成一個團隊自身的 Solution Architect 案例，再嘗試取得第二個去識別化案例或有明確後續條件的 pilot 意向。外部驗證對象可先包含 Solution Architect、Tech Lead、DevOps、Implementation 或 FDE 類似角色；沒有 FDE 聯絡人不阻擋原型，但不能把自身案例當成 FDE 市場證據。是否允許對外聯繫由團隊決定，本計畫不代表已聯繫或有意向客戶。

## 17 競爭與差異化驗證

市場已有直接重疊能力。下表根據官方功能文件，不是實際採購試用；不能以某頁沒有寫出某功能，就認定競品不支援。產品最強的待驗證假設不是「產生更好的架構」「有更多 Agent」或「更快部署」，而是以較低設定成本，把自訂環境拓撲、AI 變更意圖、證據與 promotion 責任綁成可回查的 AI Change Assurance。

| 產品 | 已公開的重疊能力 | 對本案的意義 |
| --- | --- | --- |
| Gemini Cloud Assist 與 ADC | 自然語言架構、IaC、部署、既有資源更新、assessment 與成本優化 | 作為 ArchitectureAssessment 的候選來源；競爭點是能否把建議綁到 Project environment policy、版本與漂移，不是再做一個 advisor |
| ServiceNow | Enterprise Architecture、架構決策、變更影響及 DevOps change 流程 | 決策記憶與變更前審查不是空白市場 |
| Harness | 治理、AI verification、部署、審批、知識關聯及 audit | 一鍵發布、證據與 human approval 都有強競爭 |
| Infracost | 開發與 PR 前成本估算、cost diff、policy 與 guardrails | 不能主張只有本案把成本分析提前 |
| Humanitec 與 Backstage | 資源編排、服務目錄、依賴與開發工作流 | 應利用既有工具輸出，避免重新建平台 |
| Coding assistant 與 review agent | 產生、修改、解釋或審查程式碼 | 執行速度不是差異；要驗證的是 work-order、PR／review、候選、環境 gate 與部署證據能否保持一致 |
| 團隊自建清單與 Gemini | 低轉換成本、符合既有習慣 | 是必須實測的替代方案 |

來源：[Gemini Cloud Assist](https://cloud.google.com/products/gemini/cloud-assist)、[ServiceNow EA](https://www.servicenow.com/products/enterprise-architecture.html)、[ServiceNow DevOps](https://www.servicenow.com/standard/resource-center/data-sheet/ds-servicenow-devops.html)、[Harness](https://www.harness.io/products/continuous-delivery)、[Infracost](https://www.infracost.io/docs/)、[Humanitec](https://developer.humanitec.com/platform-orchestrator/docs/platform-orchestrator/overview/)、[Backstage](https://backstage.io/docs/features/software-catalog/)。

可測試的切入點是：對有自訂多環境路徑的新創／中小企業 Cloud Run 團隊，用較少設定建立一份 Change Contract，把需求與來源 hash、所選架構、成本假設與未知項、EnvironmentTopology、PromotionPolicy、Agent Work Order、run／review evidence、acceptance IDs、commit、image digest、每個目標環境的 config、staging evidence、三次人工決策、實際 revision 與 receipt 綁在一起。一次 release 可用 Release Bundle 聚合多個 Change，但任何必要輸入改變都使相應 work order、candidate 或核准失效。Cloud Run 是技術切口；客戶購買的是對 AI 變更穿越自訂環境時的可核對放行，不是另一條 pipeline。主要使用者先以 Solution Architect／Tech Lead 為起點，FDE／Implementation 是待驗證的交付延伸；必須比較首次 setup、每次淨 review 時間、bundle 內多個 Change 的管理負擔、關鍵漏放行及誤阻擋；優勢尚未驗證，不能使用「唯一」「沒人做」或「市場已確定很大」。

架構生成、RAG、Agent、多步 workflow、成本建議與一鍵部署都不是單獨的差異化主張。ContextRail 應匯入 Gemini、coding agent、review agent、ADC、Cloud Assist、Infracost、Git 或 CI 的既有輸出，再對來源、版本、工作範圍、人工決定與交付身份做驗證；不重建完整 coding agent、架構設計、FinOps 或 CD 平台。

RAG、Evidence Graph 和歷史資料是產品資產，只有在帶來更好檢索與決策結果、且客戶持續使用時才可能形成競爭優勢。PR template 加 coding assistant、Gemini 與 Cloud Build 是新創／中小企業最重要的 baseline。若它或既有平台能以低成本配置同等流程，產品應轉向 GitHub Check、connector 或 plugin，而不是維持另一個獨立工作台。大型企業的 extension 只有在小型團隊 workflow 重複成立後才值得投入。

## 18 評估設計

先建立 8 個人工標記案例，其中 6 個開發案例、2 個保留測試案例。保留案例於第四週前不提供給 prompt 調整者；人員 B 整理後由 A 依凍結規則執行。兩人知道同一業務案例的限制要寫進報告，不能稱為大規模獨立驗證。

### 使用者與市場驗證

開發與驗證採雙軌進行，不先等待 FDE 聯絡人，也不因訪談尚未完成而擴大產品範圍。第一個案例使用團隊的 Solution Architect 工作流或去識別化歷史案例；第二個案例必須檢查同一流程是否能處理不同需求或不同交付脈絡。Week 5 至少完成 3 至 5 位外部相鄰角色訪談、至少兩位外部使用者操作原型，以及一次一句話定位測試。

市場驗證的最低判斷門檻如下：

- 5 人中至少 3 人描述最近反覆發生的需求／架構／交付或交接問題。
- 至少 2 人認為系統應能核准、阻擋或交接，而不只是產生報告。
- 至少 2 人願意提供去識別化案例、討論 pilot，或介紹實際買方。
- 多數受訪者將產品描述為 AI change／delivery assurance，而不是 chatbot、CI/CD 或自動部署。

若只在 Solution Architect 自身案例成立，保留 Solution Architect-first；若 FDE／Implementation 角色也反覆指出相同問題，才把 FDE-compatible wedge 寫入下一版定位。若外部角色只喜歡摘要或報告，先強化狀態機、決策 gate 與 evidence，再增加 AI 功能。

案例涵蓋正常變更、缺用量或價格過期、文件衝突或無權文件、來源更新使 work order 失效、work-order hash 或 Agent allowed paths 不符、缺少必要測試或獨立 review、commit／digest／target config 漂移，以及部署部分失敗。部分案例同時涵蓋兩種條件；region／單位、prompt injection、provider metadata 和冪等性另以單元或整合測試覆蓋，不用 8 個場景取代 API 邊界驗證。

競爭比較固定五組任務：正常變更、來源在核准後改變、缺價格或用量、Agent 超出範圍或 provenance 缺失，以及 commit／digest／target config 漂移。三個主要比較組為 PR template 加 coding assistant、Gemini 與 Cloud Build，可取得的 Google Cloud 原生工具，以及完整 ContextRail；Infracost 或 Harness 僅在有合法試用與相同案例時加入。

| 指標 | 原型門檻或目標 | 報告方式 |
| --- | --- | --- |
| 引用可定位 | 必要引用 100% 指向本次允許版本 | 分子／分母與失敗例 |
| 引用支持性 | 人工標註，目標至少 90% | 區分來源存在與主張成立 |
| 檢索與影響遺漏 | 報 recall@k、impact precision／recall | 小樣本原始計數，不報泛化保證 |
| 成本計算 | 指定輸入與人工基準差距不超過 USD 0.01 | 僅驗證算術；不等於帳單誤差 |
| UNKNOWN 處理 | 所有指定缺資料案例不得產生 PASS | 費項、版本、權限分開列 |
| 未授權與過期核准 | 所有對應負向測試阻擋 | 只能宣稱測試範圍內通過 |
| Agent 候選完整性 | 指定 work-order、allowed-path、run metadata 與 review 負向案例全部阻擋 | 分開報 Agent 聲明、Git／CI 讀值與人工修正 |
| Release identity | 每次成功發布完整綁定 manifest | commit、digest、設定、revision、核准 |
| 誤阻擋 | 記錄正確案例被誤阻擋的比例 | 與漏放行分開，不混成 accuracy |
| 檢查時間 | 探索目標節省 30% | 包含導入、修正和重跑的淨時間 |
| 首次設定 | 記錄每組接上來源與案例所需時間 | 不把預先整理 ContextRail 資料排除 |
| 證據完整度 | 必要 Change Contract、Agent 與 release 欄位 100% 可回讀 | 分組列出缺欄與人工補正次數 |
| 雙視圖一致性 | Agent Context Pack 與 Change Decision Brief 100% 共享 decision_id、版本、來源 hash 與 evidence refs | 抽查差異、失效案例與重新產生次數 |
| 人類可理解性 | 至少 2 位外部使用者能在未讀 glossary 下說出結論、主要風險、決策人與下一步 | 任務後問答與誤解案例；不以喜歡摘要取代理解測試 |
| 第二人重現 | 至少一個正常與一個負向案例可由另一人完成 | 記錄提示、介入和失敗步驟 |

公平比較使用相同來源、變更題目、角色權限與相近 context budget，記錄實際工具、版本、設定時間、人工介入、修正次數、token 與雲端成本。ADC 或 Cloud Assist 只有在取得權限時加入 Google Cloud 原生組；未取得就標記未測，不推估輸贏，也不視為 ContextRail 通過。

技術原型門檻為：所有指定 mandatory blocker 案例都阻擋、正常案例沒有誤阻擋、Agent candidate 與成功發布的必要 identity 欄位完整率 100%，且探索性淨檢查時間目標改善 30%。至少一位非主要實作者完成第二個案例，或取得具明確下一步的 pilot 訊號。這不取代產品 go／no-go 的外部市場門檻；後者仍要求至少兩位外部目標使用者能描述 AI 變更核對／放行。樣本很小，因此需同時報原始計數與失敗內容，不能只報百分比。

產品 eval 有真實模型呼叫；常規 CI 使用固定 fixtures 以降低成本和不穩定。保存失敗案例及未達目標的結果。若已有引用卻持續遺漏關鍵政策，先改善資料與檢索，不透過降低 mandatory checks 換取 demo PASS。

## 19 五週時程與人員分工

工程日曆從 9/15 起算，10/18 只是暫定工程窗口終點；10/16 凍結開發，10/17 至 10/18 只處理提交。官方交件／組隊日期在 G0 前仍是 UNKNOWN，若實際入口確認更早日期，立即重排 P0 與提交材料。每週容量 50 人時，兩人各 25 小時；總容量約 250。A 偏 Go 後端／GCP／部署，B 偏 React 前端／資料／QA；兩人都需能執行 Go 測試並維護資料契約。

| 週次 | 日期 | 工作重點 | 出口條件 |
| --- | --- | --- | --- |
| Week 1 | 9/15 至 9/21 | Go module、HTTP 骨架、雲端 spike、身份、tenant-ready schema、Project Registry／CRUD API、Project environment topology／policy schema、文件版本、最小 RAG、advisor／adapter access inventory，以及團隊自身的 Solution Architect 案例定義 | G1 Go API 真實 URL；Gemini、embedding、Firestore、Storage 與身份成功；Project create／list／detail／update／archive、Change Contract、EnvironmentTopology、PromotionPolicy、Agent Work Order 與 evidence schema 有 fixtures；完成第一個 workflow baseline |
| Week 2 | 9/22 至 9/28 | Project List／Workspace／Settings UI、Project 設定版本與 stale propagation、架構基線、方案比較、三類成本、環境影響與 promotion path、未知項、需求決策 UI、work-order 產生與自身案例 dogfood | G2 一個新需求有引用、可重算成本、綁定 target environment／allowed transition、人工決定與不可變 work-order hash；Project 設定修改可回查 audit 並使受影響核准失效；記錄第一次 setup 與人工修正時間 |
| Week 3 | 9/29 至 10/5 | fixture app、外部 coding-agent run、feature PR、獨立 review、candidate decision、Cloud Build、Development／Testing evidence 與 staging，以及第二個匿名案例與外部回饋 | G3 從接受方案到目標 staging；超範圍修改、缺 provenance／review／測試／環境證據會阻擋；第二案例不需新增核心 schema |
| Week 4 | 10/6 至 10/12 | prod-demo promotion、Release Receipt、來源更新失效、環境 policy／target drift、冪等與負向測試；以 concierge demo 試演示交付／交接；選定至多一項 P1 | G4 同 digest 依 promotion policy 受控發布；部分失敗可回讀；舊決策、work order、環境拓撲或核准失效時拒絕放行；P1 只有在核心證據已通過後才啟動 |
| Week 5 | 10/13 至 10/16 | 五組 baseline、2 個 holdout、3 至 5 位外部角色訪談、至少兩位外部使用者操作、定位測試、費用核對、英文材料與影片；不新增功能 | G5 第二人能重現；外部使用者能把產品描述為 AI 變更核對或放行；10/16 freeze；10/17 至 10/18 只完成交件（若 G0 確認更早截止，依該日期提前） |

前四週每日記錄主要未知與實際花費工時；第五週不加新功能。10/17 至 10/18 為交件外部緩衝，不預先安排新開發。若開工日期延後，不把測試和影片時間默默壓縮。

### 工時配置

| 工作包 | A 人時 | B 人時 | 合計 |
| --- | --- | --- | --- |
| 範圍、Go 契約與 GCP spike | 7 | 3 | 10 |
| UI、身份、Project CRUD 與環境拓撲 | 4 | 8 | 12 |
| RAG、來源版本與引用 | 7 | 5 | 12 |
| 架構基線與方案分析 | 7 | 7 | 14 |
| 成本模型與價格快照 | 8 | 6 | 14 |
| Change Contract 與三次決策 | 8 | 6 | 14 |
| 示例應用與 feature 變更 | 4 | 6 | 10 |
| Agent work order、run、review 與 candidate evidence | 12 | 12 | 24 |
| Git、Go build、Cloud Build 與 staging 證據 | 10 | 4 | 14 |
| Environment promotion、receipt 與 drift checks | 12 | 6 | 18 |
| 自動化測試與 holdout | 7 | 9 | 16 |
| Baseline comparison、訪談與可用性測試 | 4 | 14 | 18 |
| 英文文件、影片與交件 | 5 | 9 | 14 |
| 核心工作合計 | 95 | 95 | 190 |
| 風險緩衝 | 30 | 30 | 60 |
| 總容量 | 125 | 125 | 250 |

工時為本版拆解估算，並非統計預測。若 B 不具備預期開發能力，A 的路徑會超載；不能僅因總工時相同就判定時程可行。

### 容量下降與裁切順序

| 每人每週 | 五週合計 | 計畫處理 |
| --- | --- | --- |
| 15 小時 | 150 人時 | 保留 Change Contract、單一 Agent candidate 與 staging；縮小文件、成本和 prod-demo，明示 staging-only |
| 20 小時 | 200 人時 | 理論上可完成 190 人時核心但只剩 10 人時緩衝；G2 即重估，必要時 staging-only |
| 25 小時 | 250 人時 | 190 人時核心加 60 人時風險緩衝；本版有條件基準 |
| 30 小時 | 300 人時 | 優先補真實案例與可靠性；G4 後且保留 40 人時緩衝才評估一項窄版 P1 或 stretch |

G1 若 Go build、所選 SDK、模型或 deployment 權限未通過，48 小時內先解決環境並重排，不能用 mock 宣稱雲端整合成功。Go 後端是已選定的約束；若某個 client library 阻塞，先評估同一官方 API 的 Go／REST 路徑。ADC API／IAM／location 無法驗證就延後該 P1，不阻擋 Gemini 核心；Cloud Assist private preview 未取得一律維持 P2。G2 若成本仍只回傳模型猜測金額，或 work order 無法綁定輸入版本，就停止新增連接器。G3 若無法完成一條真實 Agent run、Git／CI 交叉驗證與人工候選決定，對外只能稱 decision-to-deployment workflow，不能稱已實作 AINE assurance slice，並取消全部 stretch 與 P1。G4 未完成受控發布則交件明確標記 staging-only。Outcome-to-proposal 未執行時不列為缺陷，但不能宣稱已完成 AINE evolution。

## 20 Critical Thinking Review 與修正

本次為同一撰寫者進行的反證檢查，不等於外部或獨立工程審查。完整 findings、對應章節、尚待實測的條件另存於 review 文件。雙視圖只增加可讀性與 Agent 交接契約，不增加新的自主權；兩者仍必須回到同一份決策資料。

| 被挑戰的假設 | v4.2 處理 |
| --- | --- |
| 這種產品很少人做 | 納入 Infracost、ServiceNow、Google 與 Harness 的直接重疊，差異化改為假設 |
| 多兩週就能做成企業產品 | 以 250 人時、單應用和明確 P0 計算；Vibe coding 可加速實作，但不消除整合、權限、資料隔離、雙視圖一致性與證據風險 |
| 文件有 RAG 就會越用越準 | 加入來源更新、撤權、候選分類與引用支持性評估 |
| AI 可以判斷需求值不值得做 | 價值輸入由人提供；AI 比較代價，不創造 ROI |
| 成本可以由 RAG 或模型直接給 | 改成版本化價格與確定性計算，缺項輸出 UNKNOWN |
| staging 通過即可驗證月成本 | 區分設定驗證、用量觀測與延後取得的帳務資料 |
| 同 digest 表示同環境 | 分開保存 staging／prod-demo config hash、allowed environment delta、secret version 與新 revision 回讀 |
| tag 代表部署成功 | 獨立 tag 狀態，先記部署結果再 tag，避免觸發循環 |
| 有 adapter 就支援內網企業 | 分開資料來源與資料出網限制，私有連接器後移 |
| 改用 Go 就自然縮短時程或降低成本 | 不預先計入效能收益；Week 1 驗證 SDK、身份、build、冷啟動與可恢復 operation |
| 導入 Cloud Assist 就一定更完整 | 核心閉環不依賴外部 advisor；Cloud Assist MCP 只有在資格已取得時做隔離 spike，額外價值需實測 |
| Cloud Assist 可以提供核准用成本 | 只採候選成本驅動因子；金額由版本化 PriceRecord 與確定性計算器產生，缺資料維持 UNKNOWN |
| 使用 coding assistant 就算 AI Native Software Engineering | P0 assurance slice 要求 work order、run provenance、候選評估與三次人工決策；只有宣稱 Evolution Engine 時才要求 outcome proposal。缺少 P0 條件就只能稱 decision-to-deployment workflow；沒有 outcome proposal 就不能宣稱完整 AINE evolution |
| Agent Run Record 可以證明程式由誰產生 | 將紀錄視為來源聲明，以 Git／CI 交叉比對並保留 provenance limitation |
| AI 報告可以同時服務所有讀者 | 同一 DecisionRecord 渲染 Agent Context Pack 與 Change Decision Brief；兩者共享 ID、版本、來源 hash 與 evidence refs，並以雙視圖一致性與非技術讀者理解測試驗收 |
| 同一 Agent 自我 review 足夠 | 要求不同 run_id 與 reviewer role，再由人員決定候選；review PASS 不授權 merge 或 deploy |
| 加入 Agent workflow 不影響五週時程 | 保留 24 人時給 Agent evidence 核心；總核心降至 190 人時並增加 60 人時風險緩衝 |
| Control Plane 定位不會被當成 CI/CD | 市場定位改為 environment-aware cloud-native AI Change Assurance；Project environment topology 與 promotion policy 成為差異化假設，CI/CD 是被整合的執行層，Cloud Run 是第一個技術切口 |
| Prototype 應先服務大型企業 | 市場策略改為新創／中小企業優先：先驗證 5 至 30 人、無專職平台團隊的重複使用與付費意願；大型企業保留為 enterprise-ready extension，不把組織級權限、私有網路與多 provider 導入塞進 P0 |
| Solution Architect 自身經驗等於市場驗證 | 先用自身或匿名案例 dogfood workflow，再訪談 3 至 5 位外部相鄰角色；自身案例只證明 founder-problem fit。將跨公司需求、FDE 共通性、買方身份與付費意願維持為待驗證，不提前改成 FDE-first。 |
| FDE 必須是原型啟動前提 | FDE 作為次要 persona／驗證 cohort；P0 不依賴 FDE 聯絡人或專用 connector。先完成 SA-first vertical slice；若外部 FDE／Implementation 證據不足，保留 FDE-compatible 而不擴大範圍。 |
| 所有窄版能力都應放進 P0 | F01 至 F09 保持英雄流程；Release Bundle 可在 P0 聚合多個 Change，但不共用 gate 結果；tenant-ready schema 留在 P0，固定 adapter、FDE Handoff Pack、來源 adapter 與 revision rollback 只能在 G4 後選一項；完整企業能力仍後移 |
| 使用者自訂環境只是另一個下拉選單 | F01 與 EnvironmentTopology／PromotionPolicy 契約要求順序、target、allowed action、required evidence、approver policy 與 config hash；P0 只執行 staging／prod-demo，production 維持 read-only／blocked |
| Project CRUD 只是一般資料管理，無法形成治理價值 | Project Registry／Workspace 分層；每張 Ticket 綁定 `project_id`，Project update 會產生版本與 audit event，並使受影響決策、work order 與 approval 標記 `STALE`；archive 保留完整 evidence。P0 不延伸為 Jira、billing 或完整租戶管理 |

本版修正的是計畫缺陷；只有完成實作與測試才能關閉工程風險。Environment-aware wedge 與新創／中小企業優先的市場順序目前仍是待驗證假設：如果使用者只把它理解成環境狀態儀表板，或既有 Cloud Deploy／CI policy 已能低成本覆蓋，範圍應縮成 metadata／policy connector；如果小型團隊導入成本高於收益，則不應直接跳到大型企業銷售。仍開放的高優先事項為實際工時、GCP／Git 權限、同條件競品比較、可用資料及付費意願。

## 21 比賽交件與展示

2026-09-10 查核官網，首頁列 prototype 截止日 10/18、評估 10/19 至 11/6；首頁的組隊截止日為 10/11，但 FAQ 某回答仍寫 10/4，時區也未在頁面明示，因此日期狀態維持 UNKNOWN。G0 必須在開發啟動時以實際 submission entry 或主辦方確認唯一有效日期；確認前以最早可能的 10/4 作保守提交準備，不能把 10/18 當成已確認的官方交件日。若更早日期適用，先縮小 P0 或改為 staging-only，不能默默壓縮測試與證據。[官網時程](https://aibuildercup.com/)、[FAQ](https://aibuildercup.com/Faqs.html)

官方要求可運作且部署的 prototype、Google AI／指定平台、公開 GitHub、影片及簡報；submission materials 需英文。Themes 頁的部分分類文字與六大主題不一致，送件時選定 Future of Work and Enterprise Productivity 並核對表單。本中文文件為團隊開發規劃，不能取代英文交件文件。[參賽要求](https://aibuildercup.com/themes.html)

| 評分面向 | 比重 | 本案提供的證據 |
| --- | --- | --- |
| 技術與生成式 AI | 40% | 真實 RAG、Gemini、Agent Work Order／evidence、成本計算及 Cloud Run assurance path |
| 問題與影響 | 25% | 實際變更案例、人工 baseline 與淨時間 |
| 創新與差異化 | 25% | 從核准意圖、Agent 候選到部署結果的完整 lineage，以及輸入漂移造成的分層失效 |
| 使用體驗 | 10% | 來源查看、未知項、方案選擇與可重試流程 |

影片目標 2 分 50 秒，避免超過 FAQ 的 under 3 minutes。先用 15 秒交代案例，30 秒展示 RAG 與架構／成本比較，20 秒展示接受方案與 Agent Work Order，40 秒展示 coding-agent PR、evidence gap 與候選修正，30 秒展示 staging drift 阻擋，25 秒展示人工核准、部署與 receipt，最後 10 秒說明相較 PR template baseline 的結果和限制。建置等待可明示加速或剪輯，保留對應 run／commit／build ID，不假裝所有工作即時完成。

交件包包括英文 README、架構與成本方法、模型使用、重現步驟、測試與限制、公開 repo、部署 URL、PDF deck、影片及 frozen manifest。公開入口使用獨立 demo 身份與合成資料；初次開啟不應因企業 OAuth 申請阻塞。

新專案條款與第三方 coding assistant 的具體解釋如仍有疑問，由團隊向主辦方取得書面答覆；本計畫不聲稱已得到允許或已寄出詢問。模型執行採 Gemini，開發工具選擇與產品模型使用分開記錄。

## 22 工程起手順序與文件管理

第一批工程動作依序為：核實兩人容量、Go 經驗與一種可用 coding agent；確認 GCP sandbox／billing／remote；先閱讀並凍結 [UI Flow & State Contract](../design/CONTEXT_RAIL_UI_FLOW_STATE_CONTRACT.zh-TW.md)，用 UI-01 至 UI-20 走一次 Project → Documents / AI Context → Change → Decision → Work Order → Evidence → staging → Receipt 的成功與阻擋路徑，並驗證 `zh-TW`／`en` 切換不改變 canonical state；在 UI flow 的前置條件、狀態、角色、保存、locale 與 evidence 契約尚未通過前，不開始 Go API 實作。UI gate 通過後才建立 Go module 與鎖版依賴，完成 Go API 的 deploy、身份、模型、embedding、Firestore 與 Storage spike；再凍結 Project、ProjectDocument、ContextPack、EnvironmentTopology、PromotionPolicy、Change Contract、AgentWorkOrder、AgentRunRecord 與 EngineeringEvidenceBundle schema 以及四個環境的 promotion matrix；先做 Project Registry 的 create／list／detail／update／archive 與 stale propagation，再實作文件 validate／rebuild、ArchitectureAdvisor、成本與三次人工決策。ADC 與 Cloud Assist 只做權限盤點，P0 完成前不開發外部 advisor adapter。

建議目錄為 `apps/web`、`cmd/handoffguard`、`cmd/demo-approval`、`internal/api`、`internal/domain`、`internal/rag`、`internal/architecture`、`internal/cost`、`internal/decision`、`internal/agent`、`internal/evidence`、`internal/release`、`internal/outcome`、`internal/adapters`、`contracts`、`pricing`、`infra`、`tests` 與 `docs`。Project 內的最小文件集合與 `project.yaml` 欄位依 [Project Context Contract](../architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md) 管理；`docs/ai/context-pack.json` 是衍生產物，不是另一份真相源。Go unit tests 跟著 package 放置，`tests` 保留 E2E、fixtures 與跨服務驗證。先用少量 internal packages 區分責任，不為每個分析步驟或 Agent 建立獨立微服務。

五週中每個完成宣稱都要有 commit、test、artifact 或可讀回的雲端結果。計畫中的路徑和 schema 是待實作設計，沒有把它們當成已存在程式。

[vision.md](../../vision.md) 定義方向；本 v4.2 定義交付範圍；[roadmap.md](../../roadmap.md)、[architecture.md](../../architecture.md) 與 [Project Context Contract](../architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md) 定義工程與文件切片及邊界；[UI Flow & State Contract](../design/CONTEXT_RAIL_UI_FLOW_STATE_CONTRACT.zh-TW.md) 定義開發前的畫面、狀態、action 與 evidence gate；[review](../reviews/CONTEXT_RAIL_CRITICAL_REVIEW.zh-TW.md) 定義尚未解除的風險；[產品一頁式說明](../product/CONTEXT_RAIL_ONE_PAGER_V2.3.zh-TW.md) 是同步的對外摘要。修改範圍需同步更新 Vision、Roadmap、Architecture、Project Context Contract、Project Plan、UI Flow & State Contract、One Pager 與 Critical Review；Word 從 Markdown 再生，不做獨立內容編輯，歷史 v1／v2／v3 保留不覆寫。

## 23 來源與查核限制

主要來源查核日為 2026-09-10。官方產品說明支持功能存在，不代表所有功能為 GA、同一方案可用、價格不變或已在本原型驗證。

- [AI Builder Cup](https://aibuildercup.com/)
- [AI Builder Cup themes and requirements](https://aibuildercup.com/themes.html)
- [AI Builder Cup FAQ](https://aibuildercup.com/Faqs.html)
- [Gemini Cloud Assist](https://cloud.google.com/products/gemini/cloud-assist)
- [Gemini Cloud Assist MCP integration](https://docs.cloud.google.com/cloud-assist/configure-mcp)
- [Gemini Cloud Assist pricing and preview status](https://cloud.google.com/products/gemini/pricing)
- [Gemini Cloud Assist in Cloud Billing](https://docs.cloud.google.com/billing/docs/how-to/gemini/overview)
- [Application Design Center REST API](https://docs.cloud.google.com/application-design-center/docs/reference/rest)
- [Application Design Center remote MCP](https://docs.cloud.google.com/application-design-center/docs/use-app-design-center-mcp)
- [Application Design Center deployment](https://docs.cloud.google.com/application-design-center/docs/deploy-applications)
- [ServiceNow Enterprise Architecture](https://www.servicenow.com/products/enterprise-architecture.html)
- [ServiceNow DevOps Change Velocity](https://www.servicenow.com/standard/resource-center/data-sheet/ds-servicenow-devops.html)
- [Harness Continuous Delivery](https://www.harness.io/products/continuous-delivery)
- [Infracost documentation](https://www.infracost.io/docs/)
- [Infracost usage estimates](https://www.infracost.io/docs/features/usage_based_resources/)
- [Humanitec overview](https://developer.humanitec.com/platform-orchestrator/docs/platform-orchestrator/overview/)
- [Backstage catalog](https://backstage.io/docs/features/software-catalog/)
- [Firestore vector search](https://docs.cloud.google.com/firestore/native/docs/vector-search)
- [Cloud Asset Inventory](https://docs.cloud.google.com/asset-inventory/docs/asset-inventory-overview)
- [Well-Architected Framework](https://docs.cloud.google.com/architecture/framework)
- [Cloud Run pricing](https://cloud.google.com/run/pricing)
- [Cloud Storage pricing](https://cloud.google.com/storage/pricing)
- [Firestore pricing](https://cloud.google.com/firestore/pricing)
- [Cloud Billing export](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery)
- [Cloud Billing budgets](https://docs.cloud.google.com/billing/docs/how-to/budgets)
- [Cloud Build deployment to Cloud Run](https://docs.cloud.google.com/build/docs/deploying-builds/deploy-cloud-run)
- [Cloud Run deployment semantics](https://docs.cloud.google.com/run/docs/deploying)
- [Cloud Run Go quickstart](https://docs.cloud.google.com/run/docs/quickstarts/build-and-deploy/deploy-go-service)
- [Google Gen AI SDK for Go](https://cloud.google.com/vertex-ai/generative-ai/docs/sdks/overview)
- [Firestore Go client](https://docs.cloud.google.com/go/docs/reference/cloud.google.com/go/firestore/latest)
- [Cloud Storage Go client](https://docs.cloud.google.com/go/docs/reference/cloud.google.com/go/storage/latest)
- [Firebase Admin SDK setup](https://firebase.google.com/docs/admin/setup)
- [Firebase ID token verification](https://firebase.google.com/docs/auth/admin/verify-id-tokens)
