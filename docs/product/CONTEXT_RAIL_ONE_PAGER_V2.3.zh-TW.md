# ContextRail 產品一頁式說明

版本 2.3｜日期 2026-09-12｜定位 Environment-aware Cloud-native AI Change Assurance｜Startup/SMB-first Cloud Run prototype

ContextRail 核對 AI 輔助變更的需求、架構、成本、Project 環境拓撲與發布身份；P0 先服務新創／中小企業的 5–30 人工程團隊，用 Project Registry 建立、查詢、修改與封存受管 Project，再以 Cloud Run 驗證至隔離 prod-demo smoke／Release Receipt。使用者可以為每個 Project 定義 Development、Testing、UAT、Staging、Production 或自訂環境，並為每個 promotion transition 設定必要證據與人工責任。同一份版本化 DecisionRecord 產生給人的 **Change Decision Brief** 與給 Agent 的 **Agent Context Pack**，避免把自由文字報告當作雙重契約。封存不會刪除歷史決策、證據或 receipt；Project 更新會產生新版本並使受影響核准標記 `STALE`。大型企業的組織級權限、私有網路與多 provider 治理保留為後續 enterprise-ready extension，不是 P0 的必要條件。

## 目標使用者與問題

| 項目 | 說明 |
| --- | --- |
| 目標使用者 | **新創與中小企業優先**：內部系統或 B2B SaaS 的 5–30 人團隊；角色為 Solution Architect／Tech Lead／Delivery Architect；同一 Project 有兩個以上自訂交付環境，prototype 先用 Cloud Run。大型企業保留為後續擴展市場。 |
| 核心問題 | AI 可以快速產生程式，但團隊不容易確認變更是否超出範圍、破壞架構、增加成本，或部署到錯誤的版本與設定。 |
| 導入切口 | 先從新創／中小企業的低風險、可回溯功能變更開始；一個 release 可包含 2–3 個 feature、需求變更或 bug fix，先治理 staging promotion，再以證據驗證是否值得擴展到大型企業。 |

## 核心工作流

1. **管理 Project**：建立、查詢、修改或封存 Project；Project 是 Ticket、環境、政策、文件與證據的治理邊界，不是 Jira 替代品。
2. **定義 Project 環境**：建立環境名稱、標準類型、順序、target、allowed actions、必要證據與 approver policy。
3. **理解需求**：讀取已授權的需求、架構文件與政策來源，判斷變更影響哪些環境。
4. **評估變更**：Gemini 評估架構／成本／風險／未知項與 promotion path，產出可讀結論。
5. **人工接受**：人員接受或修改 Change Contract／DecisionRecord，同時產生雙視圖。
6. **限制實作**：Agent Context Pack 編譯 Work Order，限制 repo、branch、路徑、目標環境、驗收與禁止動作。
7. **驗證候選**：Claude Code 或其他 agent 建 feature branch／PR；核對 diff、測試、review、build identity 與環境證據。
8. **Staging gate**：核准 digest 與 staging 設定，產生 staging evidence；不等於正式發布。
9. **受控 promotion**：Release Approver 核對 policy 與設定差異，以同 digest 部署隔離 prod-demo；smoke 後產生 Receipt。Production 在 P0 只保持 read-only／blocked。

## Demo 情境與可見成果

| Demo | 可見成果 |
| --- | --- |
| 內部簽核系統新增附件功能 | Brief 說明方案與成本；Pack 保存 scope、path、test、policy，涵蓋 Cloud Storage／權限／保存規則；Topology 顯示目前只能由 Staging 進入隔離 prod-demo。 |
| Agent 修改超出允許路徑，或需求文件已更新 | Pack 阻擋候選；Brief 用一般語言說明原因、風險與決策人。 |
| 合法變更完成驗證 | 2–3 個 Change 組成的 Release Bundle 通過各自 gate 後，Project topology→需求→決策→PR→測試→digest→revision 可回查並產生 Receipt。 |

## 產品產出

| 產出 | 用途 |
| --- | --- |
| Project Registry／Workspace | Registry 管理受管 Project 的生命週期；Workspace 顯示該 Project 的 Ticket、環境、gate、風險與證據。封存保留歷史，Project 設定版本變更會觸發 stale propagation。 |
| Change Decision Pack | 同一份版本化 DecisionRecord 渲染 Brief 與 Pack；共享 `decision_id`、版本、`source_snapshot_hash`、`evidence_refs`。 |
| Environment Topology & Promotion Policy | Project 的環境定義、順序、target、必要證據、允許動作與 approver separation；不是 IaC 或 deployment source of truth。 |
| Agent Work Order | 將 Pack 編譯成有範圍、期限與禁止動作的契約；不含 secret，不授予核准、IAM 或發布權。 |
| Engineering Evidence Bundle | 彙整 PR、diff、測試、獨立 review、build 與執行聲明，連回同一決策。 |
| Release Bundle／Release Receipt | Release Bundle 聚合一次 release 的多個 Change，但保留各自 scope 與 gate；Receipt 關聯核准 commit、digest、環境差異與 Cloud Run revision，保留人類摘要與機器欄位。 |

## 產品亮點與技術架構

**產品亮點**：Project 是治理邊界，不是單純資料夾；同一決策雙視圖——人先看結論／影響／風險／決策人／下一步，Agent 取得 scope／path／test／policy 契約；同一 Project 可自訂環境，但每次 promotion 都必須滿足環境專屬證據與人工 gate，核准意圖至 revision 可回查；Release Bundle 讓小型團隊一次 release 包含多個 Change，卻不犧牲逐 Change 的責任與證據。P0 是受限 AINE assurance slice；技術 prototype 為 Go／React、Gemini、Cloud Run，不取代 coding agent、PR review 或 CI/CD。

真正的 niche 不是「有四個環境」，而是「把 AI 變更穿越企業自訂環境時的意圖、證據與責任綁在同一份決策記錄」；若只需要狀態儀表板，應使用既有 CI／Cloud Deploy 或把 ContextRail 縮成 policy connector。

## 核心範圍與延伸邊界

| 核心交付 | 延伸邊界 |
| --- | --- |
| P0：一個 Project、一個 repo、一個新創／SMB 內部系統主案例加一次案例驗證；Project CRUD／封存、四環境 topology／promotion policy、DecisionRecord／雙視圖、RAG、架構／成本、Work Order、PR／review evidence、2–3 Change Release Bundle、staging、prod-demo smoke、Release Receipt。 | P1 只有 G4 PASS、300 人時、保留 40 人時緩衝且契約／API／IAM／location 已驗證後選一項；大型企業組織級權限、完整環境管理、多租戶、任意 agent、內網 connector、hard delete、跨資源 rollback 不在本次。 |

## 文件同步

同步來源：Vision v3.2、Roadmap v1.1、Architecture v1.1、Project Plan v4.2、Critical Review v2.3；Markdown 是內容真相源，Word 版由 Markdown 再生。
