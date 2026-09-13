# Handoff Guard v2 參賽與開發計畫書

Google Cloud AI Builder Cup 2026

Future of Work and Enterprise Productivity

版本 2.0　編製日期 2026 年 9 月 9 日　開發期間 2026 年 9 月 9 日至 9 月 29 日

規劃對象為兩位成員，共三週開發。本文取代目前以顧問交付變更為核心的 v1 產品假設，改以內部 Web 系統的需求到部署發布為主題。v1 文件保留作為歷史基線，不作為 v2 實作範圍。

## 01 決策摘要

Handoff Guard 是一個針對內部 Web 系統的發布準備檢查與受控部署工具。它把企業文件、設計、程式碼變更、測試結果、權限情境及 Cloud Run staging 證據放在同一個 release review 中，協助團隊判斷版本是否可以進入 production。

核心問題不是團隊缺少文件或 CI，而是這些來源彼此沒有被驗證為同一個版本。文件可能要求拒絕流程，設計稿可能只有核准畫面，程式碼可能只測試正常角色，CI 可能仍然通過。Handoff Guard 將這些差異轉為可追溯的阻塞項目，並在人工核准後透過受控流程完成 Cloud Run promotion。

本計畫的主要成果是一條可以在展示中重現的流程：

需求文件或本機文件 → Gemini 需求與驗收條件抽取 → Feature branch 與 Pull Request → Cloud Build 建置與測試 → Cloud Run staging → Handoff Guard readiness review → 人工核准 → 受控 promotion 與 release tag → Cloud Run production 與發布回執。

產品承諾為「一鍵完成受控發布」，不是讓 Agent 任意執行 production 指令。按下發布按鈕前，系統必須完成必要檢查，並保留人工核准、版本、映像摘要、Cloud Run revision 及檢查證據。

官方比賽頁面要求 prototype 可運作、使用 Google AI 模型或指定 agentic platform，並部署在 Cloud Run 或 Firebase；正式提交材料為英文。[AI Builder Cup 官方要求](https://aibuildercup.com/themes.html) 本計畫因此把 Gemini 放在產品執行流程，把 Cloud Run 放在實際展示流程，並把其他 AI 工具限定為開發輔助。

## 02 產品定位

### 產品名稱

Handoff Guard

### 英文定位

Agentic Release Readiness for Internal Web Systems

### 一句話說明

Handoff Guard verifies that an internal web system release is consistent across requirements, design, code, tests, permissions, and Cloud Run evidence before promotion to production.

### 中文定位

在內部系統上線前，確認需求、設計、程式碼、測試、權限與 Cloud Run 部署證據一致。

### 產品邊界

Handoff Guard 是既有工具之上的 release readiness control plane。它不取代 Figma、Git、CI、Cloud Run 或專案管理工具，也不把所有企業資料源一次做完。它先處理一個具體的內部系統發布情境，再以 Adapter 擴充不同資料來源與部署環境。

## 03 問題與目標使用者

### 問題定義

內部系統的發布風險通常不是程式無法建置，而是不同交付物描述了不同的系統：

- 需求文件描述了角色、規則或例外流程，但沒有被轉成可驗證的測試。
- Figma 顯示了主要畫面，但沒有涵蓋 loading、empty、error 或 unauthorized 狀態。
- Feature branch 的程式碼加入了新欄位或權限，PR 沒有連回原始需求。
- CI 通過了單元測試，但沒有證明拒絕流程、角色限制或實際 staging 畫面正確。
- 部署可以成功，但團隊不知道這個 Cloud Run revision 是否就是已驗收的版本。

### 目標角色

| 角色 | 目前工作 | Handoff Guard 提供的價值 |
| --- | --- | --- |
| Product Manager | 維護需求、驗收條件與優先順序 | 看見需求是否被實作、測試與部署證據覆蓋 |
| QA Engineer | 設計測試與驗證例外流程 | 從文件與差異自動產生檢查缺口 |
| Software Engineer | 開發 feature、修正 PR 問題 | 在合併前知道缺少哪些證據 |
| DevOps Engineer | 維護 CI、環境與發布 | 讓 promotion 綁定明確的映像與 readiness 結果 |
| Engineering Manager | 對 production 風險負責 | 以一份 release receipt 做核准與稽核 |

### 第一個展示場景

展示一個內部申請與審批系統。產品團隊在 Google Docs 或本機文件中定義 requester、manager 與 admin 的權限，Figma 描述申請、核准與拒絕畫面，工程師在 feature branch 加入拒絕原因欄位，Cloud Build 將版本部署到 Cloud Run staging。

Handoff Guard 發現拒絕流程缺少角色測試，且 Figma 沒有 error state，因此把 production promotion 標記為 Blocked。團隊補上測試與畫面後，重新執行檢查，人工核准，再將同一個 image digest promotion 到 production。

## 04 使用者流程

### Release review 流程

1. 使用者建立一個 Release Review，選擇專案、版本與要檢查的資料來源。
2. Document Adapter、Design Adapter、Git Adapter 與 CI Adapter 取得來源的版本與 metadata。
3. Gemini 以固定 schema 抽取需求、角色、狀態、驗收條件及風險候選。
4. 程式將候選需求與 PR、測試、staging URL、Cloud Run revision 及部署記錄對應。
5. Agentic workflow 執行需求、QA 與 DevOps 檢查，提出 evidence-backed findings。
6. Deterministic policy gate 檢查必要條件，例如 build status、image digest、必需測試、權限與人工核准。
7. 使用者閱讀來源與缺口，選擇修正、要求澄清或核准 release。
8. 受控 release action 建立 candidate tag，將指定版本 promotion 到 Cloud Run production。
9. Smoke test 通過後產生 final release tag 與 Release Receipt。

### 狀態模型

DRAFT → COLLECTING → ANALYZING → BLOCKED 或 NEEDS_REVIEW → READY → APPROVED → PROMOTING → RELEASED 或 ROLLED_BACK

AI 可以提出 Blocked 或 Needs Review，但不能單獨把狀態改成 Approved。正式發布需要 deterministic checks 與人員核准同時成立。

## 05 MVP 範圍

### P0 必須完成

| 項目 | MVP 定義 | 驗收條件 |
| --- | --- | --- |
| 內部系統場景 | 一個 approval workflow | 可展示 requester、manager、admin 與拒絕流程 |
| 文件來源 | 本機文件上傳或貼上文字 | 保留檔名、版本、段落與 hash |
| Google 文件 Adapter | 唯讀介面與可選實作 | 若 OAuth 在 Day 5 前通過才納入展示 |
| Gemini workflow | 需求、QA、DevOps 三個 bounded steps | 固定 schema、來源引用、可重現結果 |
| Feature branch | 短命 feature branch 與 PR metadata | PR 可連回 requirement ID 與 test plan |
| CI | Cloud Build 建置與測試 | PR 或 branch trigger 可取得 build status |
| Cloud Run | staging 與 production 服務 | 同一個 image digest 可被驗證與 promotion |
| Readiness gate | Ready、Blocked、Needs Review | 阻塞條件可見且不能被 Agent 靜默跳過 |
| 受控發布 | 人工核准後的一鍵 promotion | 保存 release receipt 與 Cloud Run revision |
| 測試 | 一條完整 E2E 加 focused checks | 匯入、分析、阻塞、核准、部署、回讀可完成 |

### P1 有餘裕才完成

- Google Drive folder picker 與 Google Docs 唯讀匯入。
- Figma export 或指定 frame snapshot 的 Adapter。
- Cloud Deploy 單一 pipeline、staging target 與 production approval。
- candidate tag 與 final tag 的自動建立。
- PR comment、release label 與自動建立 remediation task。
- 12 至 15 個新場景的 holdout evaluation。

### P2 賽後處理

- SharePoint、Confluence、內網檔案伺服器與其他文件 Adapter。
- 完整 Figma API、GitHub App 與多 repo orchestration。
- 企業私網連線、Cloud VPN、Private Service Connect 與 on-premises connector。
- 多租戶管理、SSO、SCIM、政策管理與合規報告。
- Agent 產生 PR patch 或直接修正程式碼。

### 明確不做

- 不做通用專案管理平台。
- 不做完整 CI/CD 取代品。
- 不讓模型執行任意 shell command、寄信、刪除資料或直接寫入 production。
- 不在三週內掃描整個企業 Drive 或內網檔案系統。
- 不把尚未實作的 connector、Figma integration 或 Cloud Deploy 宣稱為完成。

## 06 Adapter 與資料契約

### Adapter 分層

核心 domain 不直接依賴 Google Drive SDK、GitHub SDK 或 Cloud Run API。每個外部系統透過 Adapter 轉成標準 Artifact 與 Evidence。

| Adapter | 第一版用途 | 後續擴充 |
| --- | --- | --- |
| DocumentAdapter | 本機文件與 Google Docs | Drive、SharePoint、Confluence、內網檔案 |
| DesignAdapter | Figma snapshot 或人工提供的設計參考 | Figma API、Storybook、視覺測試 |
| GitAdapter | branch、PR、commit、diff | GitHub、GitLab、Bitbucket |
| CIAdapter | Cloud Build status、test result、artifact | GitHub Actions、GitLab CI |
| RuntimeAdapter | Cloud Run staging URL、revision、health | GKE、其他雲端 runtime |
| DeploymentAdapter | controlled promotion 與 release receipt | Cloud Deploy、其他 deployment controller |
| AIAdapter | Gemini structured analysis | 企業核准的其他模型 provider |

### 標準資料

Artifact 包含 kind、source、external_id、version、locator、content_ref 與 collected_at。

Evidence 包含 artifact_id、claim、evidence_type、locator、status、severity 與 confidence_note。

ReleaseCheck 包含 requirement_id、check_type、status、evidence_refs、severity 與 human_decision。

Adapter 負責取得資料、驗證存取權限、保存版本與來源定位；核心流程負責追蹤需求、整理證據、套用政策與產生 release decision。這使得公司文件放在 Drive、本機、私有檔案伺服器或其他 SaaS 時，不需要重寫核心判斷邏輯。

## 07 Agentic Workflow 設計

### 三個 bounded agents

| Agent | 輸入 | 輸出 | 是否可以寫入外部系統 |
| --- | --- | --- | --- |
| Requirement Agent | 文件、設計描述、版本 metadata | requirement、role、state、acceptance candidate | 不可以 |
| QA Agent | requirement、diff、test result、staging evidence | test gap、role gap、state gap | 不可以 |
| DevOps Agent | build、image、revision、logs、deployment status | deployment risk、rollback evidence、release readiness candidate | 不可以 |

三個 Agent 不互相任意呼叫工具，而是由 Orchestrator 以固定順序執行。模型輸出必須符合 schema，程式檢查來源是否存在、版本是否相符、severity 是否在允許值內。Policy Gate 再用確定性規則檢查必要條件。

### AI 與確定性規則的分工

| 工作 | Gemini | 程式與平台規則 |
| --- | --- | --- |
| 從文件理解需求 | 負責 | 驗證來源與版本 |
| 找出設計與需求落差 | 提出候選 | 要求 evidence reference |
| 判斷測試缺口 | 提出候選 | 檢查測試是否實際執行 |
| 判斷是否可上線 | 提出 readiness candidate | 決定必需條件與人工核准 |
| 產生 release tag | 提議版本與原因 | 只有 release workflow 可建立 |
| production promotion | 不可執行 | 受控 Cloud Build 或 Cloud Deploy |

這種設計仍然是 agentic workflow，因為模型在多個階段理解與整理資訊，但沒有把安全邊界交給模型自由決定。

## 08 Team Development Workflow

### 分支策略

三週內使用單一受保護的 develop trunk，避免引入完整 Git Flow。每個成員從 develop 建立短命 feature branch，完成 PR 後 squash merge 回 develop。main 與長期 release branch 延後到比賽後再決定。

Feature branch 範例為 feature/REQ-123-rejection-flow。每個 PR 至少包含 requirement ID、scope、test plan、feature flag 狀態與風險等級。尚未完成的功能使用 feature flag，不能因為程式已 merge 就被標記為完成。

### 測試層級

| 階段 | 檢查 | 是否阻擋 merge 或 release |
| --- | --- | --- |
| Feature PR | lint、type check、unit、contract、dependency scan | 阻擋 PR merge |
| develop build | build、container scan、資料契約、focused integration | 阻擋 staging |
| Cloud Run staging | smoke、E2E、角色、權限、loading、empty、error、unauthorized | 阻擋 promotion |
| Handoff Guard review | 文件、設計、PR、CI、staging、部署證據 | 產生 readiness gate |
| production 後 | health check、smoke、release receipt | 觸發 rollback 或完成 release |

Handoff Guard 不是測試執行器，也不是 CI 取代品。它消化 CI 與 runtime 的結果，確認這些結果是否回答了本次需求的風險。

### Tag 與發布流程

develop build 通過 → candidate tag release-candidate-short-sha → Handoff Guard READY → Human Approval → 同一 image digest promotion 到 production → smoke test 通過 → final tag submission-v0.1.0。

MVP 不做 semantic-release。版本號由 PR label 或人工確認產生，tag 只在受控 release workflow 中建立。任何 tag 都必須連回 commit SHA、image digest、Cloud Run revision、模型識別與提示詞版本。

## 09 Google Cloud 技術架構

### 元件

| 元件 | MVP 責任 | 交件證據 |
| --- | --- | --- |
| Cloud Run | Web UI、API、release review、staging 與 production | 公開 URL、revision、health check |
| Gemini on Vertex AI | 文件理解、需求抽取、QA 與 DevOps analysis | model ID、prompt version、structured output |
| Cloud Build | PR checks、container build、測試與部署 workflow | build ID、logs、status |
| Artifact Registry | 保存 immutable container image | image digest |
| Firestore | project、artifact、evidence、check、approval、receipt | version、audit record |
| Cloud Storage | prototype 文件與 analysis snapshot | object version、hash |
| Firebase Authentication | user identity 與 team role | verified token、role decision |
| Secret Manager | Gemini、Git provider 與 deployment credentials | secret reference，不記錄 secret 值 |
| Cloud Deploy | preferred 的 promotion controller | pipeline、target、approval、release |

Cloud Build 可以依 GitHub repository 的 push 或 pull request 觸發 build，並把 build status 回傳到 GitHub 與 Google Cloud console。[Cloud Build triggers](https://docs.cloud.google.com/build/docs/triggers) Cloud Build 也可以建置 container、推送 Artifact Registry，再部署至 Cloud Run。[Deploying to Cloud Run](https://docs.cloud.google.com/build/docs/deploying-builds/deploy-cloud-run)

Cloud Deploy 是 preferred release implementation，但不是三週 MVP 的前置阻塞。若在第一週完成 pipeline、targets、service account 與 approval 權限，即納入展示；若設定阻塞，改用受控 Cloud Build promotion，仍然保留相同的 image digest、人工核准與 release receipt 契約。Cloud Deploy 支援 release promotion、approval 與 Cloud Run targets。[Cloud Deploy overview](https://docs.cloud.google.com/deploy/docs/overview)、[Cloud Run targets](https://docs.cloud.google.com/deploy/docs/run-targets)

### 文件與私有網路

Google Drive 與 Google Docs 是一種 Adapter，不是核心資料模型。Drive API 可以搜尋與取得檔案，Docs API 可以讀取文件內容；第一版只存取使用者選定的文件，不掃描整個公司 Drive。[Drive API search](https://developers.google.com/workspace/drive/api/guides/search-files)、[Docs API document](https://developers.google.com/workspace/docs/api/concepts/document)

公司內網資料可以由未來的 OnPremisesDocumentAdapter 在內網讀取，送出經過政策允許的片段或結構化需求。若公司需要 Cloud Run 存取內網服務，Google Cloud 提供 VPC、Cloud VPN 或 Cloud Interconnect 的混合連線方式；這是企業版架構，不列入三週網路建置範圍。[Cloud Run private networking](https://docs.cloud.google.com/run/docs/securing/private-networking?hl=en)、[Google Cloud hybrid connectivity](https://cloud.google.com/hybrid-connectivity)

## 10 資料保護與執行安全

- Demo 使用合成文件、合成角色與合成程式碼，不使用未授權的公司資料。
- Source Adapter 預設唯讀，使用者明確選擇來源，系統不做全域掃描。
- 每個 artifact 保存來源 ID、版本、hash 與 locator，讓 finding 能回到原文。
- Raw document、Gemini request、log 與 release receipt 分開管理，log 不保存 token、secret 或不必要的原文。
- 後端每個 API 入口驗證 Firebase token 與 project membership，不能只依靠前端狀態或 Firestore client rules。
- Agent 不得執行任意 command，不得寫入 production，不得自行核准或跳過 policy gate。
- Release controller 只接受固定的 image digest、target、approval record 與 release plan。
- Rollback 以已驗收的 Cloud Run revision 或同一 release controller 的上一個成功版本為單位。

Vertex AI 的資料治理說明指出，Google 不會在未取得客戶許可或指示的情況下使用資料訓練或微調模型；企業導入仍需依自己的資料分類、保存與網路政策決定是否送出原始片段。[Vertex AI data governance](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/vertex-ai-zero-data-retention)

## 11 Business Model Canvas

以下商業假設尚未由付費客戶驗證，應在賽後 design partner 試用中測量。

| 區塊 | 初版假設 |
| --- | --- |
| Customer Segments | 有內部 Web 系統與正式發布流程的軟體公司、企業 IT 團隊、平台工程團隊、QA 團隊，以及需要技術交付治理的導入與顧問公司 |
| Value Propositions | 在 production promotion 前，把需求、設計、程式碼、測試、權限與 Cloud Run 證據串成一份可追溯的 release decision；不要求客戶替換既有工具 |
| Channels | 由工程與 QA 團隊進行 design partner pilot；透過 Google Cloud 技術社群、Cloud partner、系統整合商與公開技術案例取得後續客戶 |
| Customer Relationships | 自助建立專案與連接器，搭配一次性的導入設定、release policy 範本、技術支援與企業 connector 服務 |
| Revenue Streams | 依 active product team、受管控的 release volume 或 connector 數量收費；企業版採年度合約，私有連接器與導入服務另計 |
| Key Resources | Gemini workflow、evidence model、Adapter SDK、release policy、測試案例庫、Cloud Run 與企業部署能力 |
| Key Activities | 維護資料來源 Adapter、降低 false block、建立企業 release policy、改善 evidence traceability、維持部署與稽核可靠性 |
| Key Partners | Google Cloud、Google Workspace、Git provider、Cloud partner、企業系統整合商與內部平台顧問 |
| Cost Structure | Gemini inference、Cloud Run、Cloud Build、Artifact Registry、Firestore、Storage、日誌、connector 維護、資安與客戶導入支援 |

### 定價假設

| 方案 | 服務內容 | 待驗證的收費方式 |
| --- | --- | --- |
| Pilot | 一個產品團隊、一條 release pipeline、有限案例數 | 免費 design partner 或一次性導入費 |
| Team | 多個專案、標準 Adapter、release history 與 team policy | 依 active product team 或 release volume 月費 |
| Enterprise | 私有連接器、SSO、稽核、政策與支援 | 年度合約、connector 與服務費 |

商業模型的核心不是把所有工具再包一次，而是替高風險 release decision 提供可追溯的證據與責任邊界。若試用者只需要文件問答而不需要 release gate，產品應判定為錯誤客群。

## 12 競爭與差異化

| 相鄰產品 | 已覆蓋的範圍 | Handoff Guard 的切入點 |
| --- | --- | --- |
| Figma | 設計交接、規格、Dev Mode | 驗證設計意圖是否出現在程式、測試與 runtime evidence |
| Harness | release orchestration、approval、audit | 針對內部 Web 系統的需求、角色、狀態與設計語意做 readiness 判斷 |
| Cloud Deploy | Cloud Run release、promotion、approval | 提供 promotion 前的跨來源證據與問題解釋 |
| Rocketlane | 客戶服務交付、專案與客戶成功 | 聚焦內部系統版本發布，而非客戶服務專案管理 |
| 一般 Gemini chatbot | 文件問答與摘要 | 將模型輸出放入有 policy、evidence、approval 與 release receipt 的流程 |

Handoff Guard 不以「支援最多整合」作為主要賣點。亮點是同一個 release decision 能回到需求段落、設計 frame、PR、CI build、Cloud Run revision 與核准紀錄。

### 比賽展示亮點

1. **Cross-artifact traceability**：同一個需求 ID 串起 Docs、設計、PR、測試、staging 與 production。
2. **Evidence-backed AI gate**：模型不是回傳一個沒有依據的分數，而是提出 finding 並附上來源。
3. **Internal-system semantics**：檢查角色、權限、loading、empty、error、unauthorized 與流程狀態。
4. **Agentic workflow with human authority**：Agent 可以平行分析與建立 remediation suggestion，但不能自行核准 production。
5. **Adapter architecture**：文件可以來自 Drive、本機或日後的私有系統，核心判斷不綁定單一來源。
6. **Real Cloud Run delivery**：展示 build、staging、approval、promotion、smoke test 與 release receipt 的完整鏈路。
7. **One click with a safety boundary**：一鍵是執行已通過檢查的受控 promotion，不是繞過檢查的自動部署。

## 13 評分策略與展示材料

| 評分項目 | 比重 | Handoff Guard 的證據 |
| --- | --- | --- |
| Technical Merit and Gen AI Implementation | 40% | Gemini bounded agents、structured output、Adapter contract、Cloud Run、CI、policy gate 與 release receipt |
| Problem Alignment and Impact | 25% | 內部系統發布痛點、角色與例外流程、人工與直接 Gemini baseline 比較 |
| Innovation and Creativity | 25% | 跨 artifact evidence graph、設計到部署的 readiness gate、可替換 connector |
| User Experience and Solution Design | 10% | 清楚的來源定位、Blocker 解釋、修正後重跑、單一受控發布流程 |

### 三分鐘影片分鏡

| 時間 | 畫面與敘事 |
| --- | --- |
| 0:00 至 0:20 | 說明內部系統版本在文件、設計與測試之間不一致的問題 |
| 0:20 至 0:45 | 載入需求文件與 approval workflow，顯示來源與版本 |
| 0:45 至 1:10 | 展示 feature branch、PR、Cloud Build 與 Cloud Run staging |
| 1:10 至 1:40 | Gemini agents 找到缺少的 reject state、角色測試或 error handling |
| 1:40 至 2:05 | Handoff Guard 顯示 Blocked，使用者查看證據並修正問題 |
| 2:05 至 2:30 | 重新檢查，取得 Ready 與人工核准，按下受控發布 |
| 2:30 至 2:50 | Cloud Run production smoke test、release tag 與 receipt |
| 2:50 至 3:00 | 說明 Adapter、Google Cloud 與未來企業私有連接器 |

### 英文摘要草稿

Handoff Guard is an evidence-backed release readiness layer for internal web systems. It connects requirements and design intent to feature branches, CI results, staging behavior, access-control coverage, and Cloud Run deployment evidence. Gemini extracts verifiable acceptance conditions and proposes findings, while deterministic policies and human approval control production promotion. The prototype demonstrates a complete path from a requirement document to a tested Cloud Run release, with source citations, release tags, and an auditable receipt. The architecture uses adapters so enterprise teams can connect Google Workspace, local files, design tools, source repositories, and deployment platforms without replacing their existing workflow.

## 14 評估方法

### 評估比較

使用相同的合成案例比較三種方式：人工檢查、直接使用 Gemini 的單次 prompt，以及 Handoff Guard 的完整 workflow。每個案例包含需求、角色、設計狀態、PR 摘要、CI 結果與 staging 證據。

### 初始目標

下列數字是 prototype 調校目標，不是已完成成果：

| 指標 | 初始目標 | 判定方式 |
| --- | --- | --- |
| Requirement traceability | 至少 90% 的必要需求有 evidence reference | 人工抽查案例 |
| Seeded issue detection | 至少 80% 的預先植入缺口被阻擋或標記 | holdout cases |
| False blocking | 不高於 20% | 人工確認 Blocked finding |
| Review time | 相對人工檢查減少至少 30% | 配對案例計時 |
| Unauthorized promotion | 0 件 | 權限與 API 測試 |
| Release receipt completeness | 100% | 檢查 commit、digest、revision、approval、結果 |

建立 12 至 15 個案例，包含 happy path、缺少 reject state、角色權限錯誤、沒有 error state、測試未覆蓋、文件版本過期、PR 與需求不一致及 staging 與設計不一致。案例數量可以隨第一週實測調整，但必須保留至少兩個未被模型調校看過的 holdout cases。

## 15 三週開發計畫

預估總容量為 150 人時，假設兩位成員每週各投入約 25 小時。工作估算包含 16 人時緩衝；若平台權限、模型或連線阻塞，先砍 P1，不犧牲 P0 垂直切片。

### 第一週建立可用垂直切片

| 時間 | 主要工作 | 出口條件 |
| --- | --- | --- |
| 9/9 至 9/10 | 決定 v2 scope、建立 README、LICENSE、.gitignore、資料契約與第一個 commit | repo 可重現，沒有把 v1 舊 scope 混入 |
| 9/11 至 9/12 | React、FastAPI、Firebase Auth、Firestore、Cloud Run skeleton | 可登入並建立 release review |
| 9/13 至 9/14 | DocumentAdapter、Gemini Requirement Agent、Evidence schema | 一份未預先寫死的文件能產生帶來源的需求 |
| 9/15 | 第一週驗收 G1 | Cloud Run URL 能完成一次真實 Gemini analysis |

### 第二週完成 readiness gate

| 時間 | 主要工作 | 出口條件 |
| --- | --- | --- |
| 9/16 至 9/17 | PR metadata、GitAdapter、CI result、QA Agent | PR 能連回 requirement 與 test plan |
| 9/18 至 9/19 | Cloud Build、Artifact Registry、Cloud Run staging、DevOps Agent | staging 有 image digest、revision 與 smoke test |
| 9/20 至 9/21 | Policy Gate、Readiness UI、阻塞與 remediation suggestion | 缺少角色或狀態測試時會阻擋 promotion |
| 9/22 | 第二週驗收 G2 | 從文件到 staging 的 evidence chain 可重現 |

### 第三週完成發布與交件

| 時間 | 主要工作 | 出口條件 |
| --- | --- | --- |
| 9/23 至 9/24 | production release controller、approval、rollback evidence | Ready 且核准後可受控 promotion |
| 9/25 | candidate tag、final tag、release receipt | tag、commit、digest、revision 一致 |
| 9/26 | holdout evaluation、資安與權限測試 | 測試證據與限制寫入結果 |
| 9/27 至 9/28 | 英文簡報、三分鐘影片、公開 repo 與第二裝置驗收 | 交件包可由另一位成員重現 |
| 9/29 | 內部 freeze 與提交檢查 | G4 版本固定，不再加入未驗證功能 |

### 工時配置

| 工作項目 | 人時 | 主要負責 |
| --- | --- | --- |
| Scope、repo 與資料契約 | 8 | 兩人共同 |
| UI 與操作流程 | 16 | PM、QA 角色 |
| Gemini workflow | 22 | 工程角色 |
| Adapter 與 evidence model | 16 | 工程角色 |
| Cloud Build、Cloud Run 與 container | 22 | 工程角色 |
| Readiness gate 與 release controller | 16 | 兩人共同 |
| 測試、案例與評估 | 18 | 兩人共同 |
| 英文材料與影片 | 16 | PM、QA 角色 |
| 緩衝 | 16 | 兩人共同 |
| 合計 | 150 | 兩人三週 |

## 16 風險與控制點

| 風險 | 早期觸發條件 | 控制方式 |
| --- | --- | --- |
| Scope drift | 新增 Drive、Figma、Cloud Deploy 或 Agent 功能但沒有刪減工作 | 每項新功能必須標記 P0、P1 或 P2，並指出替換的工時 |
| OAuth 阻塞 | Google Drive consent 或 test user 尚未通過 | 先用本機文件 Adapter，Drive 僅在 G2 前有餘裕才開啟 |
| 模型結果不穩 | 同一案例輸出不同、虛構來源或 schema 失敗 | 固定 schema、來源驗證、保存 prompt/model version，保留人工確認 |
| 部署權限失敗 | Cloud Build、Artifact Registry 或 Cloud Run service account 缺權限 | Day 1 建立最小 service account 與 smoke deployment |
| 發布誤操作 | Agent 可直接呼叫 gcloud 或 tag 觸發 production | deployment action 只接受固定 digest、approval 與 target |
| 內部文件外洩 | raw document 進入公開 repo、影片或 debug log | 全部使用合成資料，檢查 log、bucket、repo 與展示帳號 |
| 團隊合併衝突 | feature branch 長時間不合併或改動同一需求 | 小 PR、短命 branch、feature flag、每日整合 |
| 交件版本漂移 | tag、Cloud Run revision 與簡報使用不同 commit | freeze 後建立 release manifest 與 final receipt |

### 需要向主辦方書面確認

1. Claude Code、Codex 或其他 AI coding assistant 的使用是否符合全新開發與智慧財產條款。
2. 使用 Google Drive／Docs 的測試帳號、OAuth consent 與公開展示方式。
3. 公開 GitHub repository、Cloud Run URL 與三分鐘影片的最終截止時間及時區。
4. Cloud credits、GCP project billing 與評審期間的服務保存要求。

## 17 Repo 與交件結構

apps/web、services/api、packages/contracts、packages/policy、adapters/documents、adapters/design、adapters/git、adapters/ci、adapters/runtime、adapters/deployment、workflows/release_readiness、infra/cloudbuild.yaml、infra/cloudrun、tests/contract、tests/integration、tests/holdout。

第一次 commit 必須包含 .gitignore、README.md、LICENSE、最小 app skeleton、Cloud Build 設定範例、資料契約與本計畫書。未取得 remote 與隊友確認前，不自動 push 或建立公開 repository。

## 18 立即行動

1. 將本 v2 文件指定為目前的產品範圍，v1 保留但標記為歷史基線。
2. 寫出第一個 approval workflow 的合成需求、Figma snapshot 與三個故意植入的缺口。
3. 建立 repo 基礎檔案並完成第一個可重現 commit。
4. 取得 Google Cloud project、Gemini、Cloud Run、Cloud Build 與 Artifact Registry 權限。
5. 先做本機文件 Adapter 與 Cloud Run staging，不等待 Drive OAuth。
6. 第一次 Cloud Run smoke deployment 成功後，才開始加入 readiness gate 與 release controller。
7. 每日檢查 P0 完成度；P1 未在 G2 前完成時直接後移，不修改核心承諾。

## 19 來源

本計畫查核基準日為 2026 年 9 月 9 日。服務功能、價格、賽事截止時間與條款可能更新，正式提交前必須再次核對。

1. [AI Builder Cup Themes and Requirements](https://aibuildercup.com/themes.html)
2. [Cloud Run overview](https://docs.cloud.google.com/run/docs/overview/what-is-cloud-run)
3. [Deploying to Cloud Run with Cloud Build](https://docs.cloud.google.com/build/docs/deploying-builds/deploy-cloud-run)
4. [Cloud Build triggers](https://docs.cloud.google.com/build/docs/triggers)
5. [Cloud Deploy overview](https://docs.cloud.google.com/deploy/docs/overview)
6. [Cloud Deploy Cloud Run targets](https://docs.cloud.google.com/deploy/docs/run-targets)
7. [Google Drive API search](https://developers.google.com/workspace/drive/api/guides/search-files)
8. [Google Docs API document model](https://developers.google.com/workspace/docs/api/concepts/document)
9. [Private networking and Cloud Run](https://docs.cloud.google.com/run/docs/securing/private-networking?hl=en)
10. [Google Cloud hybrid connectivity](https://cloud.google.com/hybrid-connectivity)
11. [Vertex AI data governance](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/vertex-ai-zero-data-retention)
12. [Figma design handoff](https://www.figma.com/design-handoff/)
13. [Harness release orchestration](https://www.harness.io/products/continuous-delivery/release-orchestration)
14. [Rocketlane](https://www.rocketlane.com/)

