# ContextRail 規劃文件

目前基準為五週 Project Plan v4.2、Vision v3.2、Critical Review v2.3、One Pager v2.3、Roadmap v1.1 與 Architecture v1.1，更新日期 2026-09-12。市場定位為 environment-aware AI Change Assurance for AI-assisted changes to cloud-native business systems；prototype 明確以新創／中小企業的 5 至 30 人工程團隊為優先驗證對象，Cloud Run 是第一個部署切口，主要 persona 是 Solution Architect／Tech Lead／Delivery Architect；FDE、Implementation 與 DevOps 是待驗證的相鄰使用者，大型企業保留為後續 enterprise-ready extension，AINE control layer 是技術架構。前端採 React，產品 API 與示範應用後端採 Go；P0 以 Project Registry／Workspace CRUD 與 archive 語義、一個 Project 的自訂環境拓撲與 promotion policy、一個內部系統變更、同一 DecisionRecord 產出的 Change Decision Brief 與 Agent Context Pack、tenant-ready 資料契約、受限 Agent Work Order、一條外部 coding-agent run、PR／獨立 review、2 至 3 個 Change 的 Release Bundle、三次人工決策、staging／prod-demo smoke 及 Release Receipt 完成核心 assurance path。G4 PASS、總容量至少 300 人時、至少保留 40 人時風險緩衝且契約／API／IAM／location 已驗證後，至多選一項窄版 P1；local folder 只作 fixture。完整企業能力與 production readiness 仍維持邊界。開發與市場驗證並行，先用團隊自身 SA 案例，再用第二個匿名案例與新創／中小企業外部角色檢查可複製性。

品牌已由 Handoff Guard 更名為 ContextRail。本目錄的現行文件、輸出與 UI 檔名已同步改為 ContextRail；歷史 v1/v2 文件則保留原名稱，僅作背景參考。

- [Vision](../../vision.md)：產品方向、目標客群與決策邊界。
- [Roadmap](../../roadmap.md)：五週 P0、環境管理語義、source coverage、里程碑與開始前的確認項目。
- [Architecture](../../architecture.md)：Project／Repository／GCP Project 邊界、多對多關聯、RAG／Evidence Graph／Decision Engine 與 Go／Cloud Run 架構。
- [Project Plan v4.2](CONTEXT_RAIL_PROJECT_PLAN_V4.2.zh-TW.md)：兩人五週、Go 後端、startup/SMB-first 市場策略、Project Registry／Workspace CRUD、environment-aware cloud-native AI Change Assurance、Project environment topology／promotion policy、Solution Architect-first persona、雙視圖決策輸出、Release Bundle、AINE assurance slice、架構、成本、RAG、發布、Business Model Canvas 與驗收。
- [Critical Thinking Review](../reviews/CONTEXT_RAIL_CRITICAL_REVIEW.zh-TW.md)：反證、文件修正及仍待實測的條件。
- [One Pager](../product/CONTEXT_RAIL_ONE_PAGER_V2.3.zh-TW.md)：同步的對外摘要與 demo workflow。
- [Word 計畫書](ContextRail_Project_Plan_V4.2.zh-TW.docx)：由 v4.2 Markdown 產生的閱讀版；更新 Markdown 後需重新產生並完成 bundled LibreOffice render 的逐頁檢查。
- [Word 一頁式說明](../product/ContextRail_One_Pager_V2.3.zh-TW.docx)：由 v2.3 Markdown 產生的閱讀版；更新 Markdown 後需重新產生並完成 render 檢查。
- [UI mockup](../../output/ui/context-rail-project-workspace.html)：包含 Project Registry、Project Workspace 與 Project Settings 的互動示意。
- [Figma 匯入 SVG](../../output/figma/context-rail-project-crud.svg)：可直接拖曳至 Figma；SVG 會保留文字與向量形狀，但不包含 Prototype interaction 或 Auto Layout。
- [Figma 匯入說明](../design/CONTEXT_RAIL_FIGMA_IMPORT.zh-TW.md)：說明匯入後可編輯範圍與正式元件化邊界。

Markdown 為內容真相源。更新後透過 [Word 產生器](../../scripts/build_planning_docx.py) 再生 Word，並完成 render 和逐頁檢查；不要單獨編輯 Word 內容。

產生器使用兩個參數：來源 Markdown 與輸出 DOCX，執行時使用工作區依賴提供的 Python。本文中文字型為 Arial Unicode MS；本機 bundled LibreOffice 渲染時，需將 `FONTCONFIG_FILE` 指向 [QA 字型設定](../../scripts/qa-fontconfig.conf)，才能明確載入 macOS Supplemental 中文字型。其他環境須先確認相同字型可用，或調整產生器的 `CJK_FONT` 後重新檢查。

Word 閱讀版每次再生後都必須重新渲染並逐頁檢查；驗證範圍包含中文字型、表格、主要章節、內容一致性、工時與預算加總。這些是文件檢查，不是產品功能測試。

歷史文件包括 [計畫 v1](HANDOFF_GUARD_PROJECT_PLAN.zh-TW.md)、[計畫 v2](HANDOFF_GUARD_PROJECT_PLAN_V2.zh-TW.md) 和 [Vision v1](history/VISION_V1.md)。歷史文件保留原有三週排程，不作為目前實作依據。

本次交付為規劃與 review，未建立產品實作或執行雲端部署。
