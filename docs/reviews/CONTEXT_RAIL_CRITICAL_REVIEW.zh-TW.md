# ContextRail v4.2 Critical Thinking Review

Review version 2.5  
日期 2026-09-13  
範圍 vision v3.3、五週 Project Plan v4.2、Roadmap v1.2、Architecture v1.2、Project Context Contract v1.0、One Pager v2.3、規劃 README 與兩個 HTML 預覽  
方法 同一撰寫者反證檢查、官方文件查核、零上下文 Fresh-eyes 對抗審查，以及本輪跨文件 second-pass

## 01 結論

**本輪 implementation readiness 修正為 NO-GO。** 文件方向仍可作為條件式 UI 規格驗證基線，但 Project → Change → Decision → Work Order → Evidence → staging → Receipt 的 UI 操作尚未實際跑通；在 [UI Flow & State Contract](../design/CONTEXT_RAIL_UI_FLOW_STATE_CONTRACT.zh-TW.md) 的成功、阻擋與 `zh-TW`／`en` 語系案例通過前，不應開始正式 Go／Cloud Run 實作。

建議有條件進入五週原型開發。v4.2 將主要 persona 明確收斂為 Solution Architect、Tech Lead 或 Delivery Architect，並把 FDE、Implementation Engineer 與 DevOps 保留為待驗證的相鄰使用者，而不是原型啟動前提。市場策略再收斂為「新創／中小企業優先、企業擴展預留」：先驗證 5 至 30 人、使用 Git／CI 與 AI coding assistant、沒有專職平台團隊且管理多環境的工程團隊，再決定是否值得投入大型企業的組織級權限、私有網路與多 provider 能力。市場定位維持 environment-aware AI Change Assurance for AI-assisted changes to cloud-native business systems，並鎖定「具有自訂多環境 promotion 路徑的 Project」這個 niche wedge。Cloud Run 只作第一個部署切口，第一個案例仍選內部系統；B2B SaaS 是下一個市場驗證 cohort。Release Bundle 允許一次 release 包含多個 Change，但不合併逐 Change 的責任與 gate。AINE assurance slice 保留為 P0 的技術架構描述，但不把它包裝成完整 AINE evolution。市場已有相當接近的 coding assistant、架構、成本、變更治理、環境部署及交付能力，因此目前可支持的結論仍是「有一個值得測試的切入點」，而非「競品很少」或「已找到明確護城河」。本版另把 vibe coding 能快速完成的窄版能力與不能宣稱的完整企業能力分開，避免用「不做」掩蓋可驗證的 P1 機會。

P0 保留受限 AINE assurance slice：Gemini、tenant-ready 資料契約、Project EnvironmentTopology／PromotionPolicy、同一份 DecisionRecord 產出的 Change Decision Brief 與 Agent Context Pack、Agent Work Order、一條真實外部 coding-agent run、PR／獨立 review、三次人工決策、Cloud Run staging、隔離 prod-demo smoke 與 Release Receipt。雙視圖共享 `decision_id`、版本、`source_snapshot_hash`、environment topology version、`policy_version` 與 `evidence_refs`；來源、環境政策或核准變更時一併失效，不各自重新解讀需求。只有 G4 PASS、總容量至少 300 人時、至少保留 40 人時風險緩衝，且契約／API／IAM／location 已驗證後，至多選一項窄版 P1：固定 remote agent adapter、FDE Handoff Pack、Application Design Center REST Adapter、單一 Drive／Docs 文件唯讀來源或 Cloud Run revision rollback；local folder 只作來源契約 fixture。完整多租戶營運、任意 Agent、autonomous FDE、真正內網 connector、跨資源 rollback、完整環境管理與 production readiness 仍不在本次宣稱；Cloud Assist MCP 仍是非核心的條件式 P2。此收斂不改變人工判斷權與發布權邊界。

五週相較原先三週，提供較合理的整合與驗證空間。它仍依賴兩位成員各每週約 25 小時、第一週取得 GCP／Git 權限，以及限制為一個受管應用。190 人時核心工作加 60 人時風險緩衝是規劃估算，不是實際生產力數據。

本次已將計畫缺陷改入 [v4.2 計畫書](../planning/CONTEXT_RAIL_PROJECT_PLAN_V4.2.zh-TW.md)、[vision.md](../../vision.md)、[Roadmap](../../roadmap.md)、[Architecture](../../architecture.md) 和 [One Pager v2.3](../product/CONTEXT_RAIL_ONE_PAGER_V2.3.zh-TW.md)。下文的「已修正」只指文件修正；產品尚未實作，沒有 Project CRUD／stale propagation、EnvironmentTopology policy enforcement、Agent provenance、工程驗收、Release Bundle gate、雙視圖一致性、非技術讀者理解度、付費意願或事故降低的實測結果。Fresh-eyes 是文件一致性 gate，不是產品的獨立第三方工程審查；這份 review 也不能作為產品本身的 Independent Review evidence。

本輪 second-pass 的判定是 **CONDITIONAL GO／文件可進入實作，但目前 P0 不應視為無條件可交付**。文件之間的方向大致一致，真正的風險已從「主題是否足夠 niche」轉移到三件事：小型團隊是否願意承擔另一個工作台、五週是否能同時完成 assurance path 與可展示的部署證據，以及 Release Bundle、M:N repository、獨立 review 與自訂環境的語義是否足夠精確。若 G1 至 G3 沒有按時通過，應縮成單一 Change 的 staging assurance demo，而不是保留所有功能名稱再以 mock 補齊。

## 02 證據基線

- 使用者已確認兩位成員，最新開發期為五週。
- 使用者已指定後端使用 Go；React 前端維持不變。
- Project Registry／Workspace CRUD、archive 語義與 Project update 導致下游 `STALE` 的行為已列入 P0 文件範圍；尚無實作或 API／UI 驗證證據。
- 目前 UI prototype 已加入 browser-local state 與 localStorage，可演練 Project／Change／Decision／Work Order／Evidence／Promotion 的部分 action；另加入 `zh-TW`／`en` locale switcher 與人類可讀的 status label，但尚無 Go backend 的 canonical persistence、真實 Git／Cloud Run evidence，Import 仍是 preview，environment reorder 尚未接上。
- UI Flow & State Contract v0.3 已建立，定義 UI-01 至 UI-20、角色、狀態、前置條件、文件 readiness、Context Pack lineage、locale、audit、evidence 與 P0 hard boundary；尚未通過互動驗證。
- 使用者表示本人是 Solution Architect；因此可用自身實際或匿名案例先驗證 workflow，但這不是外部市場或 FDE 共通性證據。
- 本機 develop 尚無 commit、無 remote；現有交付以規劃文件為主。
- 歷史 v2 的工時表正確合計為 150 人時，其中 16 人時緩衝；本次查核沒有發現加總錯誤。v4.2 將核心工作維持為 190 人時、風險緩衝維持 60 人時；Agent evidence 工作包仍為 24 人時，Project CRUD、EnvironmentTopology／PromotionPolicy 與 2–3 Change Release Bundle 的最小 gate 併入 P0，窄版 P1 不得消耗至少 40 人時的風險緩衝。
- 官方資料支持競品已有重疊功能。ADC 有可程式化 REST 資源與 assessment 路徑；Cloud Assist MCP 目前需要 private preview 存取。兩者尚未在本專案做 API、IAM 或同條件 hands-on comparison。
- AI Builder Cup 首頁目前列 prototype 截止 10/18；FAQ 的部分組隊日期仍為 10/4，與首頁 10/11 不一致。日期狀態維持 UNKNOWN；G0 必須以實際提交入口或主辦方確認，確認前以最早可能日期作保守準備。
- 尚無實際 GCP project inventory、billing、模型 quota、可驗證的 coding-agent run、使用者訪談或部署測試結果。

## 03 高優先發現

### CR01 競品差異尚未被證明

原先以「從需求到部署的組合少見」推導產品具有差異，證據不足。Infracost 已提供開發及 PR 階段的成本估算與政策；Gemini Cloud Assist 涵蓋架構、更新資源、部署與成本優化；Application Design Center 已提供 application、revision、assessment、generate 與 deploy 等 API；ServiceNow 有 Enterprise Architecture、決策紀錄與 DevOps Change Velocity；Harness 有治理、驗證與發布。[Infracost](https://www.infracost.io/docs/)、[Gemini Cloud Assist](https://cloud.google.com/products/gemini/cloud-assist)、[Application Design Center REST API](https://docs.cloud.google.com/application-design-center/docs/reference/rest)、[ServiceNow EA](https://www.servicenow.com/products/enterprise-architecture.html)、[ServiceNow DevOps](https://www.servicenow.com/standard/resource-center/data-sheet/ds-servicenow-devops.html)、[Harness](https://www.harness.io/products/continuous-delivery)

反證測試：讓目標團隊用自己的工具加一份結構化清單，處理同一案例。如果完成速度、錯漏與 setup cost 相近，獨立平台可能沒有足夠價值。

已修正於計畫 01 至 03、16 至 18 節、Roadmap 與 Architecture：客群改為新創／中小企業中已使用 coding assistant、維護 Cloud-native 業務系統且沒有專職平台團隊的小型工程團隊；第一個 cohort 是 Cloud Run 內部系統，B2B SaaS 是下一個驗證 cohort，大型企業保留為後續 extension。差異化假設改為 AI Change Assurance。Gemini、coding agent、ADC、Cloud Assist、Git、CI 與 Cloud Run 都是被整合的執行或證據來源，不是要被取代的產品。仍需實測，不能以目前 review 宣告通過。

### CR02 五週是日曆長度而不是工程容量

使用者沒有確認每週投入時數。若兩人各只有每週 15 小時，五週總容量仍然是 150 人時，低於 190 人時核心基線；其中一人不具備預期技術能力也會造成另一人超載。

已修正於計畫 19 節：列出 150／200／250／300 人時情境、A／B 工時與 G1 至 G5 出口。基準每人 125 人時，含 30 人時風險緩衝。實際開發開始後每週以工時和通過里程碑重估，不依靠「有 AI 所以一定更快」作估算。比賽日期另設 G0 確認 gate；10/18 目前只是工程窗口假設，不能覆蓋尚未解除的 10/4／10/11 日期衝突。

開放條件：容量與角色在 Week 1 核實；G3 未過就取消全部 stretch 與 P1。若 G4 未完成 prod-demo，交件改成 staging-only 並明示缺失，不用示意影片代替部署證據。

### CR03 成本資料不足會產生虛假的精確數字

模型與 RAG 不能決定每個 SKU 的正確單價、階梯、region、免費額度或實際使用量。單純計算 Run、Storage、Firestore 的部分費項，也不能宣稱已得到整個系統的總成本。

已修正於計畫 08、15 節：使用版本化價格、Decimal、情境、明確排除項；未知 total 不填零。計算示例採顯式 TEST_FIXTURE，官方 demo 另用查核價格。客戶應用成本與產品自營預算分開，USD 300 是預算分配而不是估算帳單。

驗收需要包含單位錯誤、region 不符、分級費率、缺 egress、重複分配免費額度、負增量與價格失效等案例。算術一致只驗證公式，不驗證使用量預測準確。

### CR04 RAG 累積也可能累積錯誤和過期權限

文件越多不保證答案越好。舊 ADR、新政策、未接受的模型建議與失敗案例若混在同一索引，可能放大錯誤；撤權未同步則可能洩漏原文。

已修正於計畫 09、10、18 節：文件與 chunk 版本、有效期間、前置權限、撤回與 cache 失效、決策候選分類。關鍵政策先以已知 ID 取得，避免把 top-k 沒檢索到誤判為不存在規則。

五週核心測試重複匯入、政策更新、來源版本失效及歷史決策定位；完整撤權、刪除與歷史回查列為 stretch。保留來源引用可定位與引用支持性兩套指標。RAG 本身不作為政策執行器或線上自我訓練機制。

### CR05 核准後變動與部分失敗可能破壞發布正確性

commit 相同不代表配置相同；digest 相同不代表身份、secret 或目標服務相同。deploy timeout 或 receipt 寫入失敗，也不能一律當成未發布而再次部署。

已修正於計畫 10、12、14 節：Approval 綁定 manifest hash 和有效期；變更使核准失效；部署前核對身份與設定；staging／prod-demo 各自保存設定 hash 與 allowed environment delta；idempotency key、operation 與 receipt 回讀。staging 與 prod-demo 分為不同 services，使用相同 digest 產生不同 revisions。Tag 與 rollback 保留在 stretch，不能改寫核心 release 結果。

核心驗收包括雙擊、核准後改設定、timeout 後實際成功，以及 receipt 暫時寫入失敗後回讀。若執行 stretch，再測 tag API 失敗、同名 tag 衝突或 rollback 失敗。P0 使用固定 controller 和最小權限；文件中的指令或 PR 提供的任意 YAML 不能繞過核准。

## 04 其他實質發現

### CR06 決策系統容易越權推論業務價值

架構與成本只構成需求決策的一部分。沒有營收、效率收益、期限或機會成本資料，系統不能合理斷言需求「值得」或「不值得」。

計畫 05、08、10 節維持決策支援：人員提供價值與約束，模型提出方案與取捨。避免未校準的綜合分數把重大安全缺口用低成本抵銷。v4.2 將需求接受、候選接受和發布核准分成三次決策，並把環境 promotion policy 作為每次 transition 的必要條件；Release Bundle 可以聚合多個 Change，但任何 AI 輸出都不能跨越下一個人工作業。

### CR07 三類成本模型與七種 catalog 容易被誤認成七種整合

描述服務用途、計算費用、取得現況及配置資源，是不同工程能力。泛稱「支援 7 個 GCP 服務」會掩蓋未實作範圍。

計畫 06 節已列支援矩陣。Cloud SQL 等選項可供架構討論，缺乏完整 cost evidence 時就不參與已知總價排名。Cloud Run controller 只操作事先建立的 target，沒有自動 IaC 或 IAM provisioner。

### CR08 月成本結果無法在發布當下驗證

Cloud Billing 匯出持續提供帳務資料；staging 的短期觀測既不等於 production 負載，也不足以證明整月成本。[Cloud Billing export](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery)

計畫 08 節把即時設定檢查、短期用量與後續帳務回饋分開。Release Receipt 不填寫尚未發生的月帳單或節省百分比。日後比較也要保留帳務時間窗口和資料完整度。

### CR09 Adapter 無法自行解決文件不得上雲

本機上傳本質上仍是資料出網。公司內網身份、資料駐留、模型輸出政策、文件撤回和私有網路連線需要個別設計。

計畫 09、11 節已明確區分來源介面與企業資料邊界。Drive／Docs 單文件唯讀匯入延至唯一 P1 選項之一，local folder 只作來源契約 fixture，內網 connector 延至 P2；MVP 全用合成或明確授權資料。不能把介面定義當作企業整合已完成。

### CR10 產品可能只是一項整合功能而非獨立 SaaS

若痛點低頻，或客戶已有 ServiceNow／Harness／Infracost，另一個工作台可能增加維護成本。RAG 與 Evidence Graph 是可複製的技術，歷史資料的價值也取決於品質與是否重複使用。

計畫 16、17、18 節改為先測固定範圍的付費 design-partner pilot、淨節省時間、導入／支援成本與現有工具 baseline，不再把 USD 199 當成既定月費。產品優先是一個 change assurance layer，而不是重新建立 architecture advisor、FinOps 或 CD 平台。若低成本配置已可解決，應轉成 GitHub Check、connector 或 plugin。不能預先把 API 成本低解讀成高毛利。

### CR11 小樣本評估無法支持可靠性或市場規模宣稱

8 個案例和 2 個 holdout 可以抓原型缺陷，不能推論所有公司或降低 production 事故率。由同一團隊設計與實作，也有認知偏差。

計畫 18 節要求報分子／分母、每案結果、prompt 調整邊界及限制。用相同資料量的 baseline，另請至少兩位目標使用者操作；若無法取得使用者，影響指標標記未驗證。

### CR12 GCP 配置與發布基礎仍未建立

本機 repo 無 remote，不能假定 PR／Cloud Build triggers 已可用。Embedding 維度、索引及模型 region 也需先實測。公開 demo、身份與帳單生命週期尚無工程證據。

計畫 11、19、21、22 節已把 GCP／Git／embedding spike 放入 Week 1。Day 1 不能僅完成 README 就宣稱雲端基礎已就緒。尚未取得的外部權限與帳戶是待建立條件，不是已完成資產。

### CR13 歷史文件與英文交件可能再次漂移

原本計畫、vision、Word 各有版本，對話又持續擴充功能。只改某一份容易讓實作和展示使用不同 scope。

本版以 Markdown v4.2 為範圍真相源，vision v3.2、Roadmap v1.1、Architecture v1.1 與 One Pager v2.3 同步五週、Go 後端、Solution Architect-first persona、startup/SMB-first 市場策略、environment-aware cloud-native AI Change Assurance、Project Registry／Workspace CRUD 與 archive 語義、同一 DecisionRecord 的雙視圖輸出、EnvironmentTopology／PromotionPolicy、Release Bundle、tenant-ready schema、窄版 P1 邊界、AINE assurance slice 與 provider 分級；Word 由 Markdown 再生，v1／v2／v3 存檔。正式提交的英文 README、deck 與影片還需要獨立交件驗收，中文企劃不能代替它們。

### CR14 Go 後端是可行選擇但不會自動降低交付風險

Cloud Run 支援 Go 服務，Google 也提供 Gemini、Firestore、Cloud Storage 與 Firebase Admin 的 Go client。這表示主要整合有官方路徑，但不證明選定版本、向量查詢、身份驗證、Cloud Build 或部署 controller 已在本專案正常運作。[Cloud Run Go quickstart](https://docs.cloud.google.com/run/docs/quickstarts/build-and-deploy/deploy-go-service)、[Google Gen AI SDK for Go](https://cloud.google.com/vertex-ai/generative-ai/docs/sdks/overview)、[Firestore Go client](https://docs.cloud.google.com/go/docs/reference/cloud.google.com/go/firestore/latest)、[Cloud Storage Go client](https://docs.cloud.google.com/go/docs/reference/cloud.google.com/go/storage/latest)、[Firebase Admin setup](https://firebase.google.com/docs/admin/setup)

Go 的 goroutine 不是持久工作佇列。Cloud Run request 結束、instance 回收或程序重啟後，記憶體中的工作不能作為 release evidence。長操作仍須保存 operation record、外部 operation ID 與冪等狀態。

已修正於 vision 12 節與計畫 01、06、11、13、19、20、22 節：產品 API 與 demo app 後端統一使用 Go，加入 SDK、身份、build、測試與可恢復操作契約。250 人時基線暫時不變，因為目前沒有證據能把語言選擇換算成節省工時；若團隊 Go 經驗不足，CR02 的容量風險會提高。

### CR15 Cloud Assist 與 ADC 可能增加證據，也可能稀釋差異化

導入 Google Cloud 原生 advisor 會讓展示更完整，但也有兩個反面風險。第一，若 ContextRail 只轉述 Cloud Assist 或 ADC，產品會退化成 wrapper；第二，Cloud Assist MCP 目前需要 private preview 與帳戶團隊協助，若放入關鍵路徑會讓五週交付受外部資格控制。[Cloud Assist MCP integration](https://docs.cloud.google.com/cloud-assist/configure-mcp)、[Application Design Center REST API](https://docs.cloud.google.com/application-design-center/docs/reference/rest)

Cloud Assist 的 Cloud Billing 對話功能提供一般帳務協助，但官方限制指出它不提供產品價格或特定 GCP 成本資訊，因此不能產生核准用 PriceRecord。架構或成本驅動因子建議都需要來源、版本、假設與人工確認；金額仍由 ContextRail 的確定性計算器產生。[Gemini Cloud Assist in Cloud Billing](https://docs.cloud.google.com/billing/docs/how-to/gemini/overview)

已修正於 vision 04、06、09 至 12 節與計畫 01、06 至 08、11、12、17 至 20、22 節：P0 不依賴外部 advisor，ADC REST 只在唯一 P1 gate（G4 PASS、總容量至少 300 人時、至少保留 40 人時緩衝、契約／API／IAM／location 已驗證）後評估，Cloud Assist MCP 保持 P2。所有 provider 輸出正規化為候選 assessment，只有 ContextRail 的 Change Contract、三次人工決策、stale invalidation 與 release identity 能控制候選接受與發布。新增整合只有在降低遺漏或增加可回溯證據，且沒有不合理增加設定時間時才保留。

## 05 AI Native Software Engineering 反證

### CR16 使用 coding assistant 不等於已實作 AINE

如果產品只是讓團隊在外部使用 Claude Code 或 Codex，最後匯入一個 PR，就仍然是一般 AI-assisted development。AINE assurance slice 的可驗收差異必須出現在 lifecycle：核准意圖被編譯成受限 work order、Agent 產出保持候選、候選有可核對 evidence，而且人工接受與發布分離。發布結果回到下一個 evidence-linked proposal 屬於 Evolution Engine stretch；未實作時不得宣稱完整 AINE evolution。

已修正於 vision 01、04 至 09 節及計畫 05、06、10、12、13、18、19 節。P0 要跑通一條真實 Agent-to-deployment 路徑；若 G3 沒有真實 run、Git／CI 交叉驗證與候選決定，對外只能稱 decision-to-deployment workflow，不能聲稱已實作 AINE assurance slice。Outcome／Evolution Proposal 只在宣稱 Evolution Engine 時才是必要條件；沒有它仍可在證據完整時宣稱受限的 AINE assurance slice，但不得宣稱完整 AINE evolution。

### CR17 Agent Run Record 可能只是可偽造的自述

由 Agent、CLI 或開發者提交的 provider、model、命令與結果可以遺漏或被修改；即使 hash 一致，也不能證明 Agent 是每一行程式的唯一作者。把一份 JSON 當成完整 provenance 會製造虛假確定性。

已修正於計畫 10、12、13、18 節：Agent Run Record 明確標為來源聲明，ContextRail 另從受管 Git、PR、CI 與 Cloud Build 讀回 commit、diff、changed paths、tests 與 build identity。無法驗證的欄位維持 UNKNOWN／NEEDS_REVIEW，報告保留 provenance limitation，不使用「不可否認」或「完整重建」的宣稱。

### CR18 Agent 執行擴大憑證與提示注入風險

文件、issue、PR 或測試 log 可能包含誘導 Agent 擴張任務的文字。若 ContextRail 在雲端直接持有個人 coding-agent 憑證、Git 廣泛寫入權或 deployment secrets，單一 prompt injection 就可能跨越 repo、IAM 或 production 邊界。

已修正於計畫 06、10 至 13 節：P0 不在 ContextRail Cloud Run 內託管任意 coding agent；開發者透過本機或 CI 的受限 adapter 明確啟動。Work order 綁定 repo、base branch、allowed paths、forbidden actions、acceptance IDs 與 expiry；coding agent 沒有 prod-demo、IAM 或核准權限，超範圍結果直接 BLOCKED 或 NEEDS_REVIEW。

### CR19 Agent 自我 review 可能造成虛假的獨立性

同一次生成執行要求模型「再檢查一次」不是 independent review。即使換模型或另開 session，也不能消除共同 prompt、資料與團隊偏誤；review PASS 更不代表可以 merge 或 deploy。

已修正於計畫 06、10、13、14、18 節：P0 的 review 要有不同 run_id、reviewer role 與人工 reviewer identity，reviewer 不得同時是候選作者或 work-order issuer，並由 CI 執行確定性檢查。人員仍作出 CandidateDecision；另一個明確的 Release Approver 才能 promotion。若兩人仍只能做到角色分離，需明示限制，不能宣稱組織級獨立審計。

### CR20 Outcome Observation 容易被誇大成自我演化

一次 prod-demo smoke、短期 latency 或合成使用者訊號不足以證明需求創造價值，也不能支持系統自動調整產品。若 observation 直接觸發 source、policy 或 deployment mutation，會繞過原本的人工作業。

已修正於 vision 04、05、09 至 11 節與計畫 06、10、14、19 節：核心 P0 停在可追溯 Release Receipt；Outcome Observation 與 Evolution Proposal 改為 G4 通過且風險緩衝足夠時的 stretch。若實作，提案保持候選並需要下一次人工決策；不自動修改需求、程式、政策、部署或 rollback。未取得足夠 outcome 時標記 INCONCLUSIVE。

### CR21 通用 Agent orchestration 會使五週計畫失控

直接支援多 repo、多 agent、遠端執行、長任務恢復、憑證託管與任意工具選擇，會把專案變成另一個 coding-agent 平台，也會增加安全、排程與競品風險。

已修正於計畫 06、10 至 12、19 節：P0 只有一個 repo、一份 work-order contract、一條由開發者啟動的 coding-agent path、一個 review path 與一組 Cloud Run target；tenant-ready schema 只提供有限 workspace 隔離，不宣稱完整 SaaS 營運。Agent evidence 保留 24 人時；核心總量降為 190 人時並保留 60 人時風險緩衝。固定 remote agent、FDE Handoff Pack、ADC／單一 Drive／Docs 唯讀來源與 Cloud Run revision rollback 都延到唯一 P1 gate 通過後至多選一項；local folder 只作 fixture。若容量不足，先保留 candidate-to-staging 證據鏈，不擴大 adapter 數量。

## 06 市場定位與範圍反證

### CR22 Control Plane 定位容易被理解成 CI CD 加強版

Control Plane for Cloud Run 會讓評審先聯想到 pipeline、平台工程或部署管理。即使內文有 Change Contract 和三次決策，第一層標題若錯誤分類，使用者仍可能拿它直接和 Harness、Cloud Build 或 Cloud Deploy 比較。

已修正於 vision 01、02、06、10 節與計畫 01 至 03、16 至 18、21 節：市場名稱改為 AI Change Assurance for AI-assisted changes to cloud-native business systems，AINE control layer 下移為技術架構，Cloud Run 保留為第一個 prototype 部署切口，並直接說明 CI/CD 負責執行、ContextRail 負責核對變更。Week 5 讓至少兩位目標使用者在未看功能清單前用一句話描述產品；若多數答案仍是 CI/CD、自動部署或 coding agent，定位測試失敗。

### CR23 十一項 P0 會稀釋英雄流程

RAG、架構、成本、Agent、review、staging、promotion、tag、rollback、outcome 和交件若都視為同等必要，兩人五週會把時間用在表面整合，評審也難辨識哪個結果證明差異。自動 Tag 和 outcome proposal 具有展示效果，但不是證明 AI Change Assurance 的最低條件。

已修正於計畫 06、10、14、18、19 節：核心改為 F01 至 F09，從一個內部系統變更走到 staging、prod-demo smoke 與 Release Receipt；2–3 個 Change 可組成 Release Bundle，但仍逐 Change 驗證；8 個情境含 2 個 holdout，並做 5 組同條件 baseline。tenant-ready schema 保持為 P0 的資料契約，不等於商業多租戶；固定 adapter、FDE Handoff Pack、ADC／單一 Drive／Docs 來源 adapter 與 Cloud Run revision rollback 改為唯一 P1 gate 通過後至多一項的窄版 P1，local folder 只作 fixture。完整跨資源 rollback、Outcome Evolution 與擴充文件生命週期仍是後續能力。沒有 P1 不降低核心驗收，也不能在簡報中假裝已完成。

### CR24 Solution Architect 經驗不能單獨證明 FDE 市場

團隊內有 Solution Architect 經驗，使本原型可以先用實際或去識別化歷史案例做 workflow dogfooding，驗證需求、架構、成本、Agent work order、證據與發布核准是否能形成一條可操作流程。這足以降低原型啟動障礙，但只證明 founder-problem fit，不能推論跨公司需求、FDE／Implementation 的共通性、買方身份或付費意願。FDE 是交付角色與 operating model，不應因市場出現 FDE 投資訊號就把產品改成 autonomous FDE agent。

已修正於 vision 01、02、03、09 節與計畫 01 至 03、16 至 19 節：Solution Architect／Tech Lead／Delivery Architect 是目前主要 persona；FDE、Implementation 與 DevOps 是外部驗證 cohort。第一個案例可以由團隊自身完成，第二個案例與 3 至 5 位外部相鄰角色用來測試可複製性。若外部證據不足，保留 FDE-compatible wedge，不提前宣稱 FDE-first 市場定位。

### CR25 Vibe coding 使窄版 prototype 可行，但不能取代企業證據

Vibe coding 可以快速產生 CRUD、UI、adapter boilerplate 與展示流程，讓 tenant-ready schema、固定來源匯入或 Cloud Run revision 操作等窄版能力在五週內具備實作可能。但它不能證明 workspace 隔離、遠端 agent 權限、私有網路邊界、rollback 語義、資料撤回或 production readiness；若因為「做得到」就把所有項目移入 P0，會讓範圍看似完整，卻削弱驗證可信度。

已修正於 vision 02、06、09、12 節與計畫 06、10 至 12、14、19、20 節：tenant-ready schema 留在 P0；固定 remote agent、FDE Handoff Pack、ADC／單一 Drive／Docs 唯讀來源與 Cloud Run revision rollback 只能在唯一 P1 gate 通過後至多選一項，local folder 只作 fixture；完整企業能力仍後移。Vibe coding 可以降低樣板實作時間，但不降低 adapter、權限、失敗處理、證據完整性與人工核准的驗收門檻；G4 未過時取消所有 P1／stretch。

### CR26 一份 AI 報告無法同時服務兩類讀者

如果只產生一份自由文字 AI 報告，非技術利害關係人很難快速回答「結論是什麼、影響是什麼、誰要決定、下一步是什麼」；Agent 則需要固定的 scope、path、test、policy 與 expiry 契約。若另做兩份獨立摘要，又可能在需求版本、成本假設或阻擋原因上分歧。

已修正於 vision 04 至 06 節、計畫 04 至 06、18、20 節與 One Pager：同一份版本化 Change Contract／DecisionRecord 是唯一真相，渲染 Change Decision Brief 與 Agent Context Pack；兩者共享 `decision_id`、版本、`source_snapshot_hash`、`policy_version` 與 `evidence_refs`，來源或核准變更時一併失效。人類視圖首屏使用一般語言，機器視圖採固定 schema，且 Agent Context Pack 不含 secret、不授予核准或發布權。此設計改善可讀性與 handoff，不增加 Agent 自主權；雙視圖一致性與非技術讀者理解度仍需實測。

### CR27 使用者自訂環境若只有 UI，仍不足以形成 niche

「Development／Testing／Staging／Production 可由使用者自行定義」本身只是設定畫面，容易被既有 CI/CD、Cloud Deploy 或平台工程工具視為一般 pipeline metadata。若 ContextRail 只顯示環境名稱與目前狀態，沒有把 target、promotion predecessor／successor、allowed actions、required evidence、approver separation 與 config hash 綁進決策，這個方向不會產生可辨識的差異，也會擴大產品範圍。

本版已修正於 vision 01、04 至 06 節、計畫 01、05、06、10 至 14、17 至 19 節與 One Pager：EnvironmentTopology 與 PromotionPolicy 成為 Project 的一級契約；環境名稱可自訂，但標準類型與 production 最低保護不可繞過；Code Review／Candidate Review／Release Approval 定義為 gate，不冒充環境；P0 只真實執行 staging 與隔離 prod-demo，Production 維持 read-only／blocked。這讓產品更 niche，但目前仍是待驗證假設，不是市場證明。

反證測試：用同一變更比較固定四環境清單、團隊自訂 topology＋policy、既有 Cloud Deploy／CI policy 與 ContextRail。若使用者只需要狀態儀表板，或既有工具能以低設定成本提供同等的「下一個允許環境＋缺少證據＋責任人」判斷，應把產品縮成 environment metadata／policy connector，而不是繼續擴大工作台。

### CR28 Project CRUD 若只做資料管理，仍無法形成產品價值

Project 的建立、查詢、修改與刪除在工程上容易完成，但若只提供名稱、Repo 與環境清單，ContextRail 會變成另一個 project catalog；它既不是 Jira 的工作追蹤，也沒有證明變更治理的必要性。尤其是修改 Project 的 repo、environment topology 或 promotion policy 後，若舊 DecisionRecord、Work Order 與 Approval 仍可繼續使用，系統會產生比沒有平台更危險的錯誤確定性。

已修正於 Vision v3.2、Project Plan v4.2、One Pager v2.3、Roadmap、Architecture 與 UI：Project Registry 管理 active／paused／archived Project，Workspace 管理其 Ticket；每張 Ticket 必須帶 `project_id`。Project update 產生新版本與 append-only audit event，並使受影響的決策、work order、candidate／release approval 進入 `STALE` 或 `NEEDS_REVIEW`。Delete 只做 archive，封存後停止新變更與 promotion，但保留 Decision Ledger、Evidence Bundle、Release Receipt 與歷史查詢。P0 API 以 create／list／detail／update／archive 為界，不延伸成 Jira、billing、SSO／SCIM 或完整租戶管理。

驗收必須包含：重複建立不產生兩個 Project、未授權身份不能查詢或修改、版本衝突不能靜默覆寫、更新 topology／policy 後既有核准不能放行、封存後新 Ticket／promotion 被阻擋但歷史 receipt 可讀。若使用者只需要 Project 清單，或現有 catalog 加 CI policy 已能以相近 setup cost 提供同一個「下一個允許 transition、缺少證據與責任人」判斷，應把 Project UI 縮成既有平台的 connector，而不是繼續增加管理功能。

### CR29 Prototype 不應以大型企業為第一個市場

「Future of Work & Enterprise Productivity」容易讓產品文件直接把大型企業、完整合規與組織級平台能力當成 P0 目標。但兩人五週的可驗證範圍只有一個 Cloud Run vertical slice；若先追求企業級 SSO／SCIM、私有網路、細粒度 ACL、多 provider inventory、跨雲與 production operations，setup cost 會超過目前可驗證的核心問題，並且無法知道小型團隊是否真的需要這個決策層。

已修正於 Vision v3.2、Roadmap v1.1、Architecture v1.1、Project Plan v4.2 與 One Pager v2.3：P0 先鎖定新創／中小企業的 5 至 30 人工程團隊，條件是已使用 Git／CI 與 AI coding assistant、管理兩個以上交付環境、沒有專職平台團隊；第一個案例仍是內部系統，B2B SaaS 是下一個 cohort。大型企業改列為 enterprise-ready extension space，只有在小型團隊能重複導入、持續使用並願意付費後，才驗證組織級擴展。Release Bundle 同時補足「一次 release 含多個 feature／需求變更／bug fix」的實務情境，但不引入完整 release-train 管理。

反證測試：用新創／中小企業的真實或去識別化案例，比較既有 PR template＋coding assistant＋Cloud Build 與 ContextRail；記錄首次 setup、每次淨 review、2–3 Change bundle 的整理負擔、錯誤阻擋與持續使用意願。如果產品只有在大型企業整合才顯得有價值，或小型團隊的導入成本不合理，就縮成既有 Git／CI／Cloud Deploy 的 policy／evidence connector，不宣稱獨立控制平面已成立。

## 07 下一階段實驗與停止條件

| 假設 | 最小實驗 | 保留方向的訊號 | 反證後處理 |
| --- | --- | --- | --- |
| 需求審查有痛點 | 3 至 5 次訪談，追問最近一次實際變更 | 能指出重查、返工或決策失聯 | 改問題定義，停止加功能 |
| Solution Architect 工作流可被產品化 | 以自身或匿名歷史案例完成一次完整 Change Contract → Release Receipt | 主要 artifact 能被另一人重做，且不用新增 P0 功能 | 保留為內部 workflow 工具，不宣稱市場成立 |
| 問題跨角色可重現 | 優先檢查 3 至 5 位新創／中小企業的 Solution Architect、Tech Lead、DevOps、Implementation 或 FDE 類似角色 | 至少 3 人描述近期重複痛點、至少 2 人認為需要核准／阻擋／交接 | 維持 Solution Architect-first，FDE 僅列為次要 persona；大型企業留到後續 extension 驗證 |
| FDE 是有效市場切入點 | 對 FDE／Implementation 角色做 concierge demo，追問真實交付與交接案例 | 至少 2 人願意提供去識別化案例、討論 pilot 或介紹買方 | 不擴大成 FDE agent；保留 FDE-compatible wedge |
| RAG 改善決策 | 同資料 baseline 與 2 個 holdout | 引用支持性及遺漏優於 baseline | 先改來源與檢索 |
| 成本建議可採用 | 人工基準與已知假設核對 | 能重算、缺項清楚、使用者願意確認 | 縮小可估算服務 |
| 額外工作台值得用 | 兩位新創／中小企業目標使用者計時操作一個含 2–3 個 Change 的 Release Bundle | 扣除導入後仍節省檢查時間，且願意重複使用 | 考慮 PR 整合或 plugin |
| Change Contract 有獨立價值 | 五組相同任務比較 PR template 加 coding assistant、Gemini 與 Cloud Build、GCP 原生工具及 ContextRail | mandatory 漏放行為零、candidate／release identity 完整，且淨檢查時間有改善 | 縮成 GitHub Check 或 evidence connector |
| Environment-aware wedge 有獨立價值 | 用同一變更比較固定四環境清單、團隊自訂 topology＋policy、既有 Cloud Deploy／CI policy 與 ContextRail | 使用者能看出下一個允許環境、缺少的環境證據與責任人；不會把環境清單誤當成 deployment 完成 | 若只是展示狀態或重複既有部署 policy，縮成 environment metadata／policy connector |
| AINE lifecycle 有增量價值 | 同一變更比較一般 coding assistant＋PR 與 work-order／candidate／promotion 三關卡 | 減少意圖、範圍或 release identity 漂移，且淨 review 時間可接受 | 保留一般 release assurance，不宣稱 AINE assurance slice |
| Agent evidence 可核對 | 注入 work-order hash、allowed-path、run metadata 與 review 缺陷 | 所有 mandatory mismatch 被阻擋，正常 run 不被誤阻擋 | 縮小可宣稱 provenance，只保留 Git／CI evidence |
| Outcome 能形成有用提案 | 僅在 stretch：一個明示時間窗的 runtime／product signal 產生 proposal，由人評分 | proposal 引用正確 outcome、限制完整且不自動採用 | 保留 receipt，不宣稱 evolution engine |
| 市場定位能被理解 | 兩位目標使用者先看一句定位與 30 秒 demo，再自行描述產品 | 能說出 AI 變更核對或放行，而非只說 CI/CD、自動部署或 coding agent | 重寫首頁與影片，不擴充功能 |
| ADC P1 提供增量證據 | Week 1 驗證 API／IAM／location；額外容量存在且 G4 通過後匯入一個 application、revision 與 assessment | 可回溯欄位或遺漏優於 Gemini P0，額外設定與維護可接受 | 不實作或移到後續版本 |
| Cloud Assist 值得整合 | Week 1 只確認 private preview／edition／MCP 存取；有權限才做隔離 spike | 提供 P0／ADC 沒有的可引用證據，且不取得發布權 | 維持 P2，不以 mock 取代實測 |
| 可持續收費 | 先以新創／中小企業團隊詢問固定範圍 pilot 條件 | 願意投入案例、時間或預算，並能說明由誰決策與付款 | 調整客群或停止 SaaS 假設；不先用大型企業需求合理化範圍 |
| Go 後端可完成核心整合 | Week 1 建置 Go API 並驗證 Gemini、Firestore、Storage、Firebase Auth 與 Cloud Run | 第二位成員可重現 build、test 與 deploy | 保留 Go，裁切 P1 並縮小整合面 |
| 五週可交付 | 每週依 G1 至 G5 實測 | 關鍵路徑如期通過 | G3 取消 stretch 與 P1；G4 必要時 staging-only |

這些是實驗設計，不表示訪談、pilot 或測試已經發生。對外聯繫、公開 repo 與實際雲端操作需在後續工程及交件流程執行。

## 08 Fresh-eyes Follow-up

本輪依 Fresh-eyes 流程，以三個零上下文反駁式 finder 和一個獨立 verifier 檢查 Vision、Project Plan、One Pager 與本 review；所有 finding 都要求命令輸出或 file:line 證據，且本輪沒有修改產品程式，只更新了規劃文件與 UI mockup。

### 已確認並已納入 v3.2／v4.2／v2.3

- **日期衝突：CONFIRMED／🔴**。Project Plan 仍記錄 prototype 10/18、首頁組隊 10/11、FAQ 10/4，且時區未明示。文件現在把日期狀態標為 UNKNOWN，新增 G0 確認 gate，並要求確認前採最早可能日期作保守準備。
- **AINE 宣稱條件：CONFIRMED／🟠**。Outcome／Evolution Proposal 是 Evolution Engine 的必要條件，但不是受限 AINE assurance slice 的 P0 必要條件。文件現在分開這兩個宣稱，避免用 P0 沒有 outcome proposal 就否定整條 assurance path，或反過來宣稱完整 AINE evolution。
- **Environment-aware wedge：DOCUMENTED／🟠**。文件現在把 Project EnvironmentTopology、PromotionPolicy、target environment 與 promotion evidence 寫成一級契約；但尚無實作或使用者證據，不能把「可自訂環境」說成已驗證差異化。
- **Startup/SMB-first market order：DOCUMENTED／🟠**。文件現在明確把新創／中小企業的 5 至 30 人工程團隊列為 P0 驗證對象，並把大型企業列為後續 enterprise-ready extension；尚無 setup、重複使用或付費意願證據，不能宣稱市場順序已被驗證。
- **Release Bundle：DOCUMENTED／🟠**。文件現在允許一次 release 包含多個 Change，並要求逐 Change 保留 DecisionRecord、evidence 與 gate；尚無實作或多變更操作證據。

### 已釐清的可能誤讀

- **staging 與 prod-demo：PLAUSIBLE／🟠**。原先 One Pager 的 staging 摘要可能讓人誤認為完整發布終點。現在 One Pager 明確拆成 staging evidence、Release Approver、同 digest promotion、prod-demo smoke 與 Release Receipt；Plan 與 Vision 也使用相同順序。
- **P1 gate 與來源 adapter：已收斂**。所有 P1／stretch 共用 G4 PASS、至少 300 人時、至少保留 40 人時緩衝及契約／API／IAM／location 驗證；Drive／Docs 僅限單一文件唯讀候選，local folder 只作 fixture。
- **文件同步鏈：已補齊**。One Pager 現在被 Vision、Project Plan、Critical Review 與 README 明確交叉引用；Word 仍由 Markdown 再生。
- **環境與 gate 的區分：已補齊**。Code Review／Candidate Review／Release Approval 是 gate，不是 environment；P0 的真實執行仍只到 staging／prod-demo，Production 維持 read-only／blocked。
- **Project CRUD 與治理語義：DOCUMENTED／🟠**。Project Registry／Workspace、`project_id`、版本化 update、`STALE` propagation 與 archive-only delete 已寫入文件與 UI；尚無實作、權限或交易一致性證據。

上述結果只關閉文件層的矛盾與誤讀，不代表產品功能、雲端部署、競品差異或市場需求已被驗證。

## 09 尚未解除的條件

本次可以完成文件一致性、工時計算、成本範例算術、來源查核及 Word 排版。以下不能由文件 review 解除：

1. 兩人每週實際時數、Go／React 經驗與主要負責人。
2. GCP billing、模型 quota、region、權限、Git remote，以及所選 Go／SDK 版本的端到端相容性。
3. ADC API／IAM／location／resource version 的可用性，以及 assessment 是否提供 Gemini P0 以外的增量證據。
4. Cloud Assist private preview／edition／MCP 資格；未取得時只能列為未測 P2。
5. 架構與價格資料的正確性，以及未建模費項。
6. 真實新創／中小企業使用者資料、導入負擔、Release Bundle 操作負擔、付費意願及競品實測；大型企業需求只屬後續 extension 假設。
7. Coding-agent CLI／授權／版本、work-order adapter、Agent Run Record 真實性、不同角色 review 與 Git／CI 交叉驗證。
8. 使用者是否能把產品理解為 AI Change Assurance，而不是 CI/CD、自動部署或 coding agent。
9. 若選擇 stretch，Outcome observation、Tag 或 Rollback 的工程證據與限制。
10. 工程完成度、pipeline 可靠性、授權測試與部署結果。
11. 主辦方尚有歧義的截止時區、組隊日期、第三方 coding assistant 與條款細節。
12. 自身 Solution Architect 案例與外部角色案例是否共享相同的重查、返工、交接或發布風險；以及 FDE／Implementation 是否真的把問題視為可購買的交付保證，而不只是摘要需求。
13. Change Decision Brief 的非技術讀者理解度、Agent Context Pack 的實際 schema／consumer 相容性，以及兩種視圖在來源或核准變更後是否能同步失效。
14. EnvironmentTopology 是否真的能降低跨環境重查、誤 promotion 或責任不清，而不是只增加一個環境設定畫面；需以既有 Cloud Deploy／CI policy 與固定清單做同條件比較。
15. Project CRUD 是否能降低建立與維護治理邊界的成本；需測試版本衝突、權限、archive、stale propagation 與歷史 evidence 回查，並與既有 catalog／CI policy 比較。

這些條件已列入里程碑與停止規則。團隊可以先執行 Week 1 的環境與資料驗證，再用結果決定後續投入。

## 10 本輪跨文件 Critical Thinking Review

### 10.1 文件一致性判定

| 檢查面 | 目前狀態 | 判定 | 仍需證明的事情 |
| --- | --- | --- | --- |
| 市場順序 | Vision、Roadmap、Project Plan、One Pager 都採 startup／SMB-first，企業能力後置 | **一致／🟠** | 具備兩個以上環境與 AI coding workflow 的小型團隊是否真的願意導入及付費 |
| 產品邊界 | ContextRail 是 Change Assurance layer，不取代 coding agent、CI/CD、Jira 或 FDE | **一致／🟠** | 使用者能否在 30 秒內理解「核對意圖、證據與放行責任」，而不是把它當成另一個 pipeline |
| 工程切片 | Go／React、Gemini、Firestore、Storage、Cloud Run、Git／CI、staging、prod-demo 都出現在 P0 | **表面一致／🔴** | 一個真實 vertical slice 是否能在兩人五週內完成並保留足夠失敗測試與證據 |
| Project／Repository | Architecture 支援 M:N；One Pager 的 P0 寫一個 repo；Roadmap 已預留多 repo schema | **需要縮限／🟠** | P0 是只驗證一個 primary repo，還是要驗證 dependency／path-scoped link |
| Change／Release Bundle | 每個 Change 各自有契約、決策與 evidence；Bundle 可一次 promotion | **概念正確但語義未完／🔴** | 部分 Change 失敗、共用 digest、重建與重試時，究竟是阻擋、拆分還是整包失敗 |
| Environment | 可自訂 topology／policy；P0 真實執行 staging 與隔離 prod-demo；production blocked | **一致／🟠** | UI 是否清楚區分「已設定」、「已驗證」與「可發布」，避免展示造成錯誤確定性 |
| AI／RAG | RAG 提供有版本引用；DecisionRecord 是 SSOT；Brief／Pack 由同一記錄產生 | **設計合理／🟠** | 來源更新、撤回、引用不足與 `NEEDS_INPUT` 是否在真實資料流中失效並被看懂 |
| AI Native 宣稱 | 文件描述受限 AINE assurance slice，不宣稱完整 autonomous FDE 或 Evolution Engine | **可接受／🟠** | Agent run、Git／CI、review 與人工接受是否能形成可回讀的 lifecycle 證據 |
| 商業模式 | 先做固定範圍 design-partner pilot，再評估訂閱；不預設月費 | **正確但未驗證／🟠** | 首次 setup、每次淨檢查時間與持續使用意願能否抵銷導入成本 |
| 交件與文件 | Markdown 為真相源，Word／HTML 是投影；review 已連回各文件 | **可追蹤／🟠** | HTML 與 Word 是否能在每次 scope 變更後自動或可重現地同步 |
| Project 文件契約 | 新增 Project Context Contract；ProjectDocument 是來源登錄，Context Pack 是可重建衍生物 | **方向一致／🟠** | 必要文件是否真的降低重查；欄位、狀態與 readiness 是否能被 Go API、RAG 與 UI 同步驗證 |

### 10.2 關鍵新發現

#### CR30 Startup／SMB-first 是驗證順序，不是已證明的最佳市場

**反證。** 新創／中小企業通常更容易快速採用工具，但不代表它們會願意建立另一個治理工作台。真正符合本案的不是所有 SMB，而是「5 至 30 人、已使用 Git／CI 與 AI coding assistant、管理兩個以上交付環境、沒有專職平台團隊」的行為型 cohort。這個條件組合本身已經很窄，可能同時排除最不需要治理工具的小團隊，以及最有預算但需要企業能力的團隊。

**目前判定：🟠 待驗證。** 保留 startup／SMB-first 作為 prototype design-partner 順序是合理的，但不能在對外材料中把它寫成市場規模或購買需求已成立。Enterprise Productivity 應被解釋為要改善的工作結果，而不是暗示 P0 已服務大型企業。

**最小實驗。** 找三個符合上述 operating characteristics 的團隊，不以公司營收或員工數作唯一條件；讓他們用既有 PR template、coding assistant、CI 與 ContextRail 完成同一個需求變更。記錄首次 setup、理解產品所需時間、人工修正次數、每次檢查淨時間、是否願意再次使用，以及誰能批准或付款。

**停止條件。** 若至少兩個團隊認為既有工具已足夠，或導入時間高於一次變更可節省的時間，停止擴大 UI／adapter，改驗證 Git／CI policy connector；大型企業需求不能用來合理化這個缺口。

#### CR31 比賽主題與客群敘事可能互相打架

**反證。** 參賽主題包含 Enterprise Productivity，但產品現在刻意不做大型企業 SSO／SCIM、私有網路、多 provider 與 production readiness。評審若把「Enterprise」理解成企業級部署能力，可能認為 scope 不足；若把它理解成生產力結果，startup／SMB-first 才成立。

**目前判定：🟠 需要統一對外語句。** 首頁、影片與簡報應使用「讓精簡工程團隊以企業級的可核對方式管理 AI 變更」之類的結果導向敘事，並立刻補一句「P0 以小型 Cloud Run 團隊驗證」。不要用「enterprise-grade platform」暗示已完成企業能力，也不要把 Cloud Run 說成長期市場限制。

**驗證方式。** 讓兩位未參與設計的讀者只看一句定位與 30 秒 demo，請他們回答：產品核對什麼、誰做決定、哪一步會被阻擋。若答案仍集中在 CI/CD、自動部署或 coding agent，定位驗證失敗，應重寫訊息而非增加功能。

#### CR32 P0 的完整性仍可能犧牲驗證深度

**反證。** Project CRUD、M:N repository schema、自訂 environment topology、RAG、架構建議、成本模型、雙視圖、Work Order、外部 agent run、PR／review／CI、Cloud Build、staging、prod-demo、drift、idempotency、Release Bundle、baseline、holdout 與交件材料都被放入同一個五週敘事。190 人時核心加 60 人時緩衝是估算，不是已取得的工程容量；外部 agent、權限與雲端失敗處理很容易消耗緩衝。

**目前判定：🔴 有條件可做，不能無條件承諾。** 文件雖有 G1 至 G5，但 P0 capability map 仍可能讓團隊在核心路徑尚未通過時繼續補表面功能。

**硬切線。**

1. **MUST：** 一個 Project、一個 primary repo、一個內部系統、單一 Change 的正向與負向案例、同一 DecisionRecord 的 Brief／Pack、真實 Gemini／RAG、受限 Work Order、PR／review／CI read-back、staging 與可回讀的 Receipt。
2. **SHOULD：** 自訂 topology／policy、隔離 prod-demo、設定／digest drift、idempotency 與第二個案例。
3. **CONDITIONAL：** 2–3 Change Release Bundle；只有單一 Change 在 G3 前已通過，才允許把 Bundle 放入 demo。
4. **DROP FIRST：** Cloud Assist、ADC、Drive／Docs、FDE Handoff Pack、tag、rollback、outcome proposal、一般化多 provider 與多 repo discovery。

若 G3 未通過，對外名稱改為「staging-only AI Change Assurance workflow」；不要保留完整 P0 名稱再用靜態畫面掩蓋缺失。

#### CR33 Release Bundle 的資料模型完整，不代表發布語義完整

**反證。** 「多個 Change 各自保留 gate」解決了責任追蹤，但沒有回答 bundle 是原子發布還是可部分成功。若兩個 Change 共用一次 build、其中一個在 staging 失敗，或一個 Change 的來源在等待期間變成 `STALE`，系統不能只顯示 bundle failed 而不說明哪個 Change 造成阻擋及是否已有任何資源變更。

**目前判定：🔴 P0 必須先縮成明確語義。** 建議 P0 定義為：Bundle 是一次 release attempt 的編排與追蹤容器；所有 Change gate 都 PASS、共用 manifest／digest 與 target 證據都一致時，才允許一次 promotion；任一 Change 失敗則整個 promotion blocked；P0 不提供已部署後的 selective rollback 或 partial commit。每個 Change 仍要有單獨阻擋原因，Receipt 要列出 bundle status 與每個 Change status。

若無法在 Week 4 實作上述語義，Bundle 退回「只做 release summary／清單」的展示，不宣稱多 Change promotion 已完成。

#### CR34 M:N Repository 關係應留在 schema，不應提前變成 P0 discovery 產品

**反證。** Architecture 以 M:N 反映真實企業關係是正確的，但 One Pager 又把 P0 定為一個 repo，Roadmap 也明確不承諾企業級 GitHub／GitLab crawling。若 P0 同時做多 repo UI、provider inventory、共享 repo ownership、path scope 與權限，便會引入大量同步與授權問題，且不直接證明 Change Assurance 價值。

**目前判定：🟠。** P0 可保留 `ProjectRepository` 與 `ProjectDependency` 的資料契約，但實作只驗證一個 primary repo；若需要展示多對多，最多加入一個明示的 dependency fixture，不能把 fixture 說成自動發現。

**必測案例。** 同一 repo 被兩個 Project 引用、primary owner 改變、path-scoped link 不在允許路徑，以及 disconnected private GitLab。後者必須輸出 `UNKNOWN`／`UNLINKED_REPOSITORY`，不能顯示成「沒有 repo」。

#### CR35 兩人團隊的「獨立 review」需要明確定義

**反證。** 文件要求 Agent run 與 review 使用不同 `run_id`、`reviewer role`，這能避免同一模型把自己的輸出直接當成 review；但兩人團隊仍可能由同一人寫程式、寫測試、產生 evidence，再由另一人只按下 PASS。這不等於組織層級的獨立審查。

**目前判定：🟠 可作原型證據，但不可過度命名。** P0 應明定 reviewer 必須是不同的人類 actor，不能是同一個 Agent run；UI 要顯示 author、reviewer、candidate decision maker 與 release approver。若角色無法分離，狀態應是 `NEEDS_REVIEW`，而非 `INDEPENDENT_REVIEW_PASS`。對外可稱「separated review identity」，不宣稱符合所有企業 code review／SoD 要求。

#### CR36 自訂環境的產品價值在 policy decision，不在環境 CRUD

**反證。** 使用者可以新增、修改、排序、退休環境是必要管理能力，但這些畫面本身不是差異化。真正的價值必須是：對目前 Change，系統指出下一個允許 transition、缺少哪項 evidence、哪個人負責，以及哪個 topology／policy version 使舊核准失效。

**目前判定：🟠。** Demo 必須先展示 decision gate，再展示設定頁；Production 的 read-only／blocked 要在 UI 和 Receipt 同時出現。環境只被設定但沒有真實 target evidence 時，狀態只能是 `CONFIGURED` 或 `NOT_EXECUTED`，不能用綠色 `READY` 取代。

**反證測試。** 與固定四環境清單、既有 Cloud Deploy／CI policy 做同條件比較。若使用者只要環境狀態，不需要下一步判斷或失效傳播，應把產品縮成 policy／evidence connector。

#### CR37 RAG 不是 DecisionRecord，DecisionRecord 也不是事實本身

**反證。** 文件正確地把 DecisionRecord 設為 SSOT，但它保存的是當時的人員接受、模型候選、來源 snapshot 與政策版本；若來源本身錯誤、價格已過期或使用者接受了不完整資料，DecisionRecord 只能保存「當時做了什麼決定」，不能自動證明決定正確。

**目前判定：🟠。** 人類 Brief 必須把「已知事實、假設、未知、來源日期、誰確認」分開；Agent Pack 必須拒絕沒有必要輸入的 work order。RAG 評估應分成 citation retrieval、citation support、source freshness、permission filtering 和 stale invalidation，不用一個相似度分數代表品質。

**必測案例。** 文件更新、來源撤回、同名文件衝突、缺少 RTO／RPO／資料分類與無法確認 repository。所有案例都應產生 `NEEDS_INPUT`／`STALE`／`UNKNOWN`，不能靠 Gemini 補出合理內容。

#### CR38 「完整報告」若沒有決策動作，仍會被視為 AI 摘要

**反證。** Brief、Architecture／Cost Report、Implementation Readiness 與 Release Receipt 能讓輸出看起來專業，但使用者不會只為另一份報告付費。產品價值必須落在報告驅動的可見動作：接受、補資料、重新分析、阻擋、要求 review、重建 candidate 或核准下一個環境。

**目前判定：🟠。** UI 首屏應展示結論、影響、阻擋原因、責任人與下一步 action；附件才放完整報告。每個 action 必須回寫同一 `decision_id`、版本與 audit event。若報告沒有對應 action 或 action 不改變 gate，報告只是展示層，不是決策系統。

#### CR39 Cloud Run 成功部署不等於產品成功，也不等於 production readiness

**反證。** Cloud Run URL、image digest、revision 與 smoke 是重要工程證據，但只證明指定 demo 的部署路徑。它不能證明小型團隊願意採用、成本預估準確、資料權限安全、回滾可靠或可服務大型企業。

**目前判定：🟠。** Release Receipt 必須分開標記：

| 證據 | 可以支持的結論 | 不能支持的結論 |
| --- | --- | --- |
| Cloud Run revision／smoke | 指定 demo revision 可被部署及基本操作 | production readiness、SLO、真實客戶流量安全 |
| image digest／commit | 本次候選與發布身份可回查 | Agent 是唯一作者、程式沒有未測缺陷 |
| 成本 estimate | 在指定 price snapshot、usage assumption 下可重算 | 真實月帳單、節省金額或毛利 |
| Release Receipt | 本次流程證據可回溯 | 發布後長期業務價值 |

#### CR40 商業模式仍應先賣「可驗證 pilot」，不是先賣 SaaS 平台

**反證。** 新創／SMB 可能沒有專職平台團隊，這同時代表它們可能沒有時間維護 topology、policy、source adapter 與 evidence。若產品必須先導入大量資料，固定訂閱的價值尚未成立。

**目前判定：🟠。** 先用 concierge design-partner pilot 驗證三件事：首次設定是否可接受、每次變更是否重複使用、是否有人願意提供預算或正式導入承諾。只有當團隊能在沒有 founder 介入下重跑案例，才評估 self-serve SaaS；大型企業導入服務與 connector 應視為後續產品線，不應在 P0 預支工程成本。

#### CR41 Project template 能降低脈絡缺口，也可能變成另一套文件負擔

**反證。** `project.yaml`、README、Vision、Architecture、development guide、environment policy、AGENTS.md、ADR、glossary 與 Context Pack 能讓人與 Agent 更快找到來源，但對沒有平台團隊的新創／SMB，這個集合也可能變成「先寫完文件才能開始」的門檻。公開 repo 可以提供 spec-driven workflow、agent instruction file、cloud starter pack 或 service catalog 的 pattern，卻沒有證據證明所有 AI 專案都應採用同一棵目錄樹。

**目前判定：🟠 建議保留契約，但只把最小集合放進 P0。** 必要文件限定為 manifest、README、產品意圖、架構基線、開發／測試、環境／promotion 與 Agent rules；Roadmap、ADR、glossary、完整 API／event schema、資安、成本與 runbook 依 Project 需要逐步加入。Context Pack 一律是 derived，不能被誤認為新的 SSOT。UI 應顯示「缺少哪一份文件會阻擋哪個 readiness」，而不是要求所有文件一開始都完成。

**最小實驗。** 用同一個 demo Project 比較空白匯入、只提供 manifest＋README、提供完整 P0 集合三種 setup；記錄首次可作出 Decision 的時間、人工補件次數、retrieval citation 命中率、Agent 是否能遵守 allowed paths，以及第二位使用者是否能重跑。若完整模板只增加維護時間而未降低錯誤或交接時間，縮回 manifest＋README＋架構／環境基線，不再增加文件種類。

**停止條件。** 任何衍生 Context Pack 不能回指來源、文件狀態不能讓 decision／development／staging readiness 產生可驗證差異，或使用者把它理解成另一個文件管理器時，停止擴大 template，改做既有 Git／CI／catalog 的小型 connector。

## 11 建議的 Go／No-Go 門檻

### GO：可以開始實作的條件

- G0 已確認實際交件日、GCP sandbox／billing、Go toolchain、Gemini／embedding、Git source 與一個可展示案例。
- P0 hard cut line 已寫入 issue／task board，並指定哪一位負責每個 human gate。
- 明確選擇 P0 的 repository 形態：一個 primary repo，其他 repository 只作 schema／fixture，不承諾 discovery。
- 明確定義 Release Bundle 是 all-pass atomic promotion；不能 partial promote，也不包含 rollback。
- Brief、Pack、Receipt 的欄位與 action 共用同一 DecisionRecord schema，並有至少一個 `NEEDS_INPUT` 負向案例。
- UI Flow & State Contract 的 UI-01 至 UI-20 已以可保存狀態跑通；所有主要 mutation 都能在重新整理／重新進入後回讀，並顯示 actor、版本、前後狀態、audit 與 evidence，且 `zh-TW`／`en` 切換不改變 canonical state。

### NO-GO：暫停擴充或立即縮 scope 的條件

- Week 1 仍無法取得真實 Cloud Run／Gemini／Firestore／Storage／身份證據。
- Week 2 結束仍只有自由文字報告，沒有可驗證的 DecisionRecord、Work Order、policy gate 或 stale propagation。
- Week 3 結束沒有一條真實 coding-agent／Git／CI read-back 到 staging 的流程。
- 兩個以上目標團隊無法在未讀完整文件下說明產品不是 CI/CD，或認為現有工具已足夠且不願重複使用。
- 團隊只能用同一人或同一 Agent run 充當 author、reviewer 與 approver。
- 為了展示 Release Bundle 而犧牲單一 Change 的 negative test、來源引用或 release identity。
- UI action 仍只顯示 toast／preview、Project／environment／Change 狀態無法重新查回，或 Workspace 導覽沒有對應內容與 gate action。

### 最終判定

目前的文件組合可以作為 **條件式 UI 規格驗證基線**，但不能作為「可以開始正式 Go／Cloud Run 實作」的基線，也不能作為「市場已證明、企業可直接導入、五週必定完成」的證據。最值得保留的 niche 假設是「小型但有多環境交付壓力的團隊，需要一個把 AI 變更的意圖、證據、責任與 promotion identity 綁在一起的決策層」；最需要優先反證的則是這些團隊是否願意為它多建立一個工作面，以及這條 UI flow 是否真的能降低決策與交接負擔。

因此建議順序固定為：**先完成並驗證 UI-01 至 UI-20，再用單一 Change 證明 decision-to-staging assurance，接著才驗證 Release Bundle，最後才討論 enterprise-ready extension 或 SaaS 擴張。** 若 UI flow 未通過，先停在規格／mockup 修正；若後續工程關鍵證據缺失，對外應誠實降級為 staging-only、evidence connector 或 founder-problem-fit demo。
