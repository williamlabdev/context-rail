# ContextRail UI Flow & State Contract

版本：v0.3
日期：2026-09-13
狀態：開發前規格草案，尚未通過互動驗證
適用範圍：ContextRail P0 hero vertical slice

## 0. 本文件的目的

本文件定義 ContextRail 在開始 Go／React 實作前，必須先被驗證的 UI 操作流程與狀態契約。它補足 Vision、Roadmap、Architecture 與 Project Plan 只描述「應該發生什麼」、但沒有完整定義「使用者按下什麼、系統如何改變、產生什麼證據」的缺口。

### 判定

- **FACT：** 目前 repository 的 `develop` 尚無 commit，也沒有 Go、React application、API、Cloud Run 或 Git integration implementation。
- **FACT：** `context-rail-project-workspace.html` v2 已加入 client-side state、localStorage、Project／Change／Decision／Work Order／Evidence／Promotion 的 prototype action；它仍不是 backend persistence，也不產生真實 Cloud Run 或 Git evidence。
- **FACT：** Import 仍是 preview，environment reorder 尚未接上 action，部分 report／receipt 仍是示範資料或 simulated result。
- **FACT：** UI 已加入 `zh-TW`／`en` locale switcher；目前以同一份 client-side state 重新渲染後套用語系，尚未由 Go backend 提供正式 locale contract。
- **ASSUMPTION：** P0 hero case 採一個 Project、一個 primary repository、一個內部業務系統與一個主要 Change；M:N repository 只保留資料契約，不在 hero flow 做自動 discovery。
- **RECOMMENDATION：** 使用者可見文案分為繁體中文與 English；Project name、repository path、ID、commit、digest、Cloud Run 等 proper name／technical identifier 保留原值，狀態 enum 以可理解的人類標籤呈現。
- **RECOMMENDATION：** 在以下 UI flow 以正向與負向案例跑通前，實作 readiness 為 **NO-GO**；可以先做 UI prototype 修正與 contract test，但不要開始擴充後端功能。
- **UNKNOWN：** 本文件尚未由實際使用者在瀏覽器完成端到端點擊驗證；UI-01 至 UI-20 尚未取得 PASS，所有未標示為 FACT 的操作仍是待驗證規格。

## 1. P0 唯一驗證主線

```text
Projects
  → 建立或選擇 Project
  → Project Workspace
  → Documents / AI Context
  → 建立 Change
  → 缺資料時 NEEDS_INPUT
  → 補資料並重新評估
  → 人工接受 Decision
  → 建立 immutable Work Order
  → 開發者啟動 Agent／feature branch／PR
  → 匯入或讀回 run、diff、CI、review evidence
  → 人工接受 Candidate
  → staging preflight／deploy／smoke
  → Release Approver 核准下一個 transition
  → prod-demo promotion
  → Release Receipt
```

這條主線必須同時包含一個可成功完成的案例，以及至少一個被正確阻擋的案例。報告是附件與檢視層；真正的產品行為是改變 gate、建立下一步工作、要求補件、標記失效或阻擋 promotion。

## 2. UI 角色與權限邊界

| 角色 | 可以做的事 | 不能單獨完成的事 |
| --- | --- | --- |
| Project Admin | 建立／修改／封存 Project；管理環境 topology 與 policy；查看 audit | 不因為管理設定而自動核准 Change 或 Release |
| Decision Maker | 建立 Change；補充需求；接受、修改、延後或拒絕 Decision | 不取代 Candidate Review 或 Release Approval |
| Developer／Agent Operator | 查看已接受的 Work Order；啟動既有 coding agent；提交 PR／run evidence | 不修改已接受的 scope；不核准自己的 Candidate 或 Release |
| Independent Reviewer | 查看 diff、測試與 review scope；提交 review evidence 與結論 | 不直接部署；不能與 Candidate 作者使用同一個 review identity |
| Release Approver | 查看 promotion preflight；核准或拒絕指定 transition | 不改寫 DecisionRecord；不跳過必要 evidence |
| Viewer | 依 Project membership 查看狀態、報告與歷史證據 | 不可寫入任何治理狀態 |

P0 可以由兩人團隊扮演多個角色，但每一筆 evidence 必須保存實際 actor。若同一人同時是候選作者、reviewer 與 release approver，系統必須顯示 separation 缺失並將 gate 設為 `NEEDS_REVIEW`，不能顯示為獨立通過。

## 3. 狀態模型

### 3.1 Project

```text
DRAFT → ACTIVE → PAUSED → ARCHIVED
```

- `DRAFT`：必要 identity、primary repository、owner、第一版 topology／policy 尚未完整。
- `ACTIVE`：可以建立 Change，但每次 action 仍受 Project policy 控制。
- `PAUSED`：保留歷史內容，但暫停新 Change 與 promotion。
- `ARCHIVED`：不可建立新 Change 或 promotion；歷史 DecisionRecord、Evidence Bundle、Release Receipt 與 audit 仍可讀。

Project update 必須產生新 `project_version` 與 AuditEvent。若 repository、環境 target、promotion policy、必要 evidence 或 owner 變更，依賴舊版本的 Decision、Work Order、Candidate 或 Approval 必須進入 `STALE`／`NEEDS_REVIEW`。

### 3.2 Change

建議的 P0 狀態如下；狀態名稱是 UI 與 API 的共同契約，不代表模型可以自行跳轉：

```text
DRAFT
  → NEEDS_INPUT
  → DECISION_READY
  → DECISION_ACCEPTED | MODIFICATION_REQUESTED | DEFERRED | REJECTED
  → WORK_ORDER_READY
  → IN_PROGRESS
  → CANDIDATE_READY
  → CANDIDATE_ACCEPTED | CANDIDATE_BLOCKED
  → STAGING_READY
  → STAGING_VERIFIED
  → RELEASE_READY
  → PROMOTION_IN_PROGRESS
  → RELEASED
```

任何需要重新檢查的版本變動都可使現有狀態進入 `STALE`；`STALE` 不得直接被按鈕改回原狀態，必須建立新的 Decision／Work Order／Approval version。

### 3.3 Gate 與 Evidence

Gate 與 Change status 分開保存：

```text
Gate: NOT_STARTED | NEEDS_INPUT | PASS | FAIL | UNKNOWN | STALE
Evidence: MISSING | OBSERVED | VERIFIED | CONFLICTING | EXPIRED
```

P0 至少有三個人工作業 gate：

1. `Decision Gate`：是否接受需求、架構方案、成本假設與目標 transition。
2. `Candidate Gate`：候選是否符合 Work Order、Git／CI、測試與獨立 review 證據。
3. `Promotion Gate`：指定 commit／digest／target／設定是否符合下一個環境的 policy。

`UNKNOWN`、`MISSING`、`CONFLICTING`、`EXPIRED` 或 `STALE` 不得被 UI 渲染成綠色 PASS。

## 4. 畫面與操作契約

### 4.1 Projects Registry

| 使用者操作 | 前置條件 | 成功狀態 | 必須留下的證據 | 失敗／阻擋 |
| --- | --- | --- | --- | --- |
| 建立 Project | key、名稱、primary repo、technical owner、business owner、data classification、至少一個合法 environment target | 建立 `DRAFT` 或 `ACTIVE`；取得穩定 `project_id` | Project version、actor、欄位 snapshot、AuditEvent | 缺必要欄位或 production 最低保護時不能 ACTIVE |
| 開啟 Workspace | Project 為 ACTIVE／PAUSED／ARCHIVED | URL／state 明確帶 `project_id`；顯示正確 Project context | workspace open event、project version | 不可把另一個 Project 的 Change 或 evidence 混入 |
| 修改 Project | 有 Project Admin 權限 | 新 project version；受影響資料顯示 STALE | before／after snapshot、reason、AuditEvent | 版本衝突、無權限、未知 target |
| 封存 Project | 二次確認；輸入封存原因 | Project 進入 ARCHIVED；selector 移除 active entry | archive actor、reason、timestamp | 已封存不可再次建立新 Change |
| 匯入設定 | 只接受明確格式的 manifest／fixture；不做未授權 crawling | 先顯示 import preview，再由人確認建立 | source hash、欄位差異、confirmation actor | 格式錯誤、repo／target 不可確認時顯示 NEEDS_INPUT |

### 4.2 Project Settings／Environment Topology

環境列必須可執行，而不只是顯示名稱：

| 操作 | 必要 UI | 成功狀態 | 失效規則 |
| --- | --- | --- | --- |
| Add | 名稱、standard type、順序、target、owner、required evidence、approver policy | 新 topology version；環境為 `ACTIVE` | 受影響舊 Decision／Approval 標記 STALE |
| Edit | 可編輯欄位與變更影響預覽 | 新 topology version；顯示 diff | display-only 變更可標 informational；target、order、policy、approver 或 evidence 變更必須失效 |
| Reorder | 拖曳／上移／下移後顯示 predecessor／successor | 新 topology version | 受影響 promotion path 必須重新驗證 |
| Retire | 確認沒有未處理的新 Change，或明示影響範圍 | `RETIRED`；保留歷史 evidence | 新 Change 不可選用；歷史 receipt 仍可讀 |
| Restore | 顯示原 topology 與重新驗證的 target／policy | 新 topology version；環境回到 `ACTIVE` | 不得恢復成舊版本而繞過新的 policy |

P0 真正執行的 target 只有 staging 與隔離的 prod-demo；Development／Testing 可以顯示 evidence／gate，不能把 fixture 或其他環境的 PASS 複製成 staging PASS。Production 可在 topology 中存在，但 P0 UI 必須明確顯示 `READ_ONLY`／`BLOCKED`。

### 4.3 Project Workspace

Workspace 首屏必須回答五個問題：

1. 這是哪一個 Project、哪一個 Change、哪一個 DecisionRecord version？
2. 目前可以做哪一個下一步？
3. 為什麼可以、不能或尚不能進入下一個環境？
4. 缺少哪一項資料或 evidence？誰要補？
5. 這個結論由哪些來源、policy version、commit、digest 或 receipt 支持？

Workspace 的導覽項必須是真實內容或明確 disabled，不可以點擊後仍停在同一個 overview 只顯示 toast：

| 區域 | P0 最小內容 | 主要 action |
| --- | --- | --- |
| Overview | 狀態、下一步、風險、目前 target、最新 gate | 開啟 Change、補資料、檢視 gate |
| Documents / AI Context | ProjectDocument 清單、必要／選配、受眾、SSOT／derived、來源版本、索引狀態、文件 readiness 與 Context Pack lineage | 驗證文件集合、查看缺件原因、重建 Context Pack、查看來源 metadata |
| Changes | Project 內 Change 清單、類型、狀態、target、阻擋原因 | 建立、選擇、封存／取消未開始的 Change |
| Impact | 現況基線、受影響服務／資料／IAM、方案比較、未知項 | 查看來源、要求補件、接受／修改／延後／拒絕 |
| Evidence | PR、diff、run、測試、CI、review、build、staging、runtime evidence | 查看原始證據、標記 conflict／needs review |
| Reports | 從同一 DecisionRecord 產生的 Brief、Technical Report、Agent Pack、Receipt 附件 | 預覽、下載、查看 version 與 evidence refs |
| Policy gate | 目前 transition、必要 evidence、每項 PASS／FAIL／UNKNOWN／STALE | 檢視 gate；不得用畫面按鈕繞過規則 |
| Audit trail | 版本、actor、action、reason、時間與前後 hash | 只讀回查 |

### 4.4 建立 Change 與需求決策

建立 Change 的必要欄位：

- Change type：`NEW_FEATURE`、`REQUIREMENT_CHANGE`、`BUG_FIX`、`HIGH_RISK_BUG_FIX`。
- 需求目標與使用者影響。
- primary Project 與 primary repository／path。
- 受影響服務與預計 target environment／transition。
- 驗收條件與禁止範圍。
- 資料分類、保存期限、預計用量、可靠性與安全限制；未知欄位要明示未知。

送出後，畫面必須先顯示 `Decision Gate`：

| Action | 允許條件 | 結果 |
| --- | --- | --- |
| 要求補資料 | 任一必要輸入缺失、衝突或無法確認 | Change=`NEEDS_INPUT`；建立補件清單與 owner；不得產生可執行 Work Order |
| 重新評估 | 補件已保存且建立新 source snapshot | 新 DecisionRecord version；重新計算架構／成本／風險；保留舊版 |
| 接受 | 必要輸入已確認；Decision gate PASS；人類 Decision Maker | Change=`DECISION_ACCEPTED`；可建立 Work Order |
| 修改 | 人類改變方案、scope、成本假設或 target | 新版本；舊核准不沿用；回到 `DECISION_READY` 或 `NEEDS_INPUT` |
| 延後 | 人類提供原因與重新檢查條件 | Change=`DEFERRED`；不產生可執行候選 |
| 拒絕 | 人類提供原因 | Change=`REJECTED`；保留完整歷史 |

`Change Decision Brief` 與 `Agent Context Pack` 必須由同一個 DecisionRecord version 產生，並共享：

```text
decision_id
decision_version
project_id / project_version
source_snapshot_hash
policy_version
environment_topology_version
evidence_refs
expires_at
```

### 4.5 Work Order 與工程交接

Work Order 只有在 `DECISION_ACCEPTED` 後才能建立，不能以 staging gate 作為建立條件。這是目前 mockup 與 Vision 流程的明確順序差異，必須在 UI 修正。

Work Order 頁面必須顯示：

- decision_id、版本、source snapshot 與 policy version。
- repository、base branch、feature branch 規則與 allowed paths。
- accepted scope、out-of-scope、forbidden actions。
- acceptance test IDs、必要檢查與預計 target transition。
- work-order hash、建立人、建立時間與 expiry。
- 「建立」後的 immutable 狀態；修改必須建立新版本。

P0 的 ContextRail 不託管任意 remote agent。UI 應提供開發者啟動既有 Claude Code／Codex 流程後，匯入或讀回 Agent Run Record、PR 與證據的入口。

### 4.6 Candidate／Review／Evidence

Candidate 頁面必須把四個身份分開顯示：

```text
candidate_author
independent_reviewer
candidate_decision_maker
release_approver
```

最小 evidence checklist：

- Work Order hash matches。
- 只修改 allowed paths。
- branch／PR／commit 可回讀。
- Agent Run Record 有 provider、model、version、時間與執行檢查。
- 測試與 build evidence 可回讀。
- review evidence 來自不同 reviewer identity。
- candidate decision 有人類 actor、理由與時間。

缺任何必要項目時，Candidate 只能是 `CANDIDATE_BLOCKED` 或 `NEEDS_REVIEW`。Review PASS 不等於 merge 或 deploy authorization；UI 必須分開顯示「review 通過」與「候選已被人接受」。

### 4.7 Promotion 與 Release Receipt

Promotion 頁面以 transition 為中心，而不是只列出環境：

```text
current environment → next environment
```

送出前 preflight 必須核對：

- Change／Decision／Work Order 版本一致。
- 所有必要 gate PASS。
- commit、build artifact 與 image digest 可回查。
- target、region、service、configuration hash 符合該環境 policy。
- approval actor 與候選作者／reviewer 符合 separation 規則。
- staging 與 prod-demo 使用獨立 target evidence；不能複製 PASS。

成功 promotion 後，Receipt 必須保存：

```text
release_id
decision_id / change_ids
bundle_id (若有)
approval_actor / approval_time
approved_commit
image_digest
target_ref / config_hash
cloud_run_revision
smoke_result
operation_id / idempotency_key
evidence_refs
receipt_status
```

`receipt_status=SIMULATED_NOT_EXECUTED`、`UNKNOWN` 或失敗不可渲染成已發布。P0 Production 維持 blocked；P0 不提供跨資料庫、物件、IAM 與外部系統的完整 rollback。

### 4.8 Release Bundle

Release Bundle 是一次 release attempt 的容器，不是把多個 Change 合併成一個決策：

- 每個 Change 保留自己的 DecisionRecord、Work Order、Candidate、Evidence 與 gate。
- Bundle 必須列出所有 Change 的狀態與阻擋原因。
- P0 語義採 all-pass atomic promotion：任一 Change gate 失敗、過期或 UNKNOWN，整個 promotion blocked。
- 共用 digest、manifest、target 與設定證據必須一致。
- P0 不支援 partial promotion 或 selective rollback。
- 若 Week 4 無法實作以上語義，Bundle 只能做 release summary，不得宣稱多 Change promotion 已完成。

## 5. UI 驗證案例

以下案例是開發前的 acceptance matrix；每個案例都要在 UI 中留下可重現的狀態與 evidence，不接受只看畫面截圖。

| ID | 案例 | 預期結果 |
| --- | --- | --- |
| UI-01 | 從 Projects 開啟 APV | Workspace、breadcrumb、Project key、repo、environment 與 decision_id 全部一致 |
| UI-02 | 建立缺少 owner／target 的 Project | 保持 DRAFT；欄位錯誤可理解；不可顯示 ACTIVE |
| UI-03 | 編輯 Project repo 或 policy | 產生新 project version；受影響 Decision／Work Order／Approval 顯示 STALE |
| UI-04 | 封存 Project 後返回 Registry | Project 從 active selector 移除；歷史可讀；不能建立 Change 或 promotion |
| UI-05 | 新增、修改、排序、退休、恢復環境 | 每項操作產生 topology version；受影響 transition 正確失效；退休環境不能被新 Change 選用 |
| UI-06 | 建立正常 Change | 進入 Decision Gate；顯示方案、成本、未知項與下一個 transition |
| UI-07 | 建立資料不足的 Change | `NEEDS_INPUT`；顯示欄位、owner、原因；不允許建立 Work Order |
| UI-08 | 補資料後重新評估 | source snapshot 與 DecisionRecord version 更新；Brief／Pack 同步更新；舊版仍可回查 |
| UI-09 | 接受 Decision 後建立 Work Order | Work Order 可建立且有 hash；不需要先通過 staging |
| UI-10 | Agent 修改超出 allowed paths | Candidate blocked；顯示違規 path、原始 evidence 與下一步 |
| UI-11 | 缺少 independent review | Candidate blocked／NEEDS_REVIEW；不能進 staging |
| UI-12 | 正常 candidate 進入 staging | commit、digest、target、config hash、smoke 與 revision 可回查 |
| UI-13 | 兩個 Change 組成 Bundle，其中一個失敗 | 整個 promotion blocked；逐 Change 顯示失敗原因；不允許 partial promotion |
| UI-14 | staging 後 target config drift | Promotion blocked；原核准標記 STALE；要求重新評估或重新核准 |
| UI-15 | 重新整理、返回、重新開啟同一 Project | 所有已保存狀態、版本、角色與 gate 不消失、不混入其他 Project |
| UI-16 | 在 Workspace 切換 `zh-TW`／`en` | 使用者可見文案完整切換；除允許保留的 proper name／technical identifier 外，不出現任意中英混排 |
| UI-17 | 切換語系後重新整理或直接返回同一 Project | locale、Project／Change context、狀態與版本都維持不變 |
| UI-18 | 顯示機器狀態碼，例如 `PROMOTION_SIMULATED` | 主要畫面顯示人類標籤；technical detail 仍可回查原始 code |
| UI-19 | 切換語系時檢查 Project name、repo path、ID 與使用者輸入 | 資料值、URL、path、commit、digest 與 evidence refs 不被翻譯或污染 |
| UI-20 | 開啟 Documents / AI Context，驗證一份缺件並重建 Context Pack | 缺件對應到 decision／development／staging readiness；Context Pack 顯示 `PARTIAL`、source path／version／hash 與 audit；AI 不會把缺件補成事實 |

## 6. UI action 的共通回應契約

每個會改變治理狀態的 action 都必須回傳或能查詢：

```text
action_id
actor_id / actor_role
project_id / project_version
change_id / decision_id / version
before_status / after_status
precondition_results
evidence_refs
audit_event_id
created_at
error_code (若失敗)
```

UI 顯示必須區分：

- **已保存：** 後端已成功寫入並可重新查詢。
- **預覽：** 尚未寫入治理狀態；不能顯示成功或 PASS。
- **待補資料：** 缺少必要輸入；不是模型失敗。
- **未知：** 觀測範圍或權限不足；不是不存在。
- **失效：** 依賴的版本已變更；不可沿用舊核准。
- **已阻擋：** 規則明確拒絕下一步；必須顯示原因與責任人。

所有 mutation action 需要：

1. 防止 double-submit 的 idempotency key。
2. 版本衝突提示與重新載入選項。
3. 成功後重新讀取 canonical record，而不是只更新前端記憶體。
4. 失敗後不偽造新狀態；保留原狀態與錯誤 evidence。
5. 重新整理或直接開啟 deep link 後能恢復正確 Project／Change context。

### 6.1 多國語系契約

語系不是兩份獨立資料或兩個決策來源，而是同一個 canonical state 的 presentation layer：

- locale 只允許 `zh-TW` 與 `en`，預設為 `zh-TW`，且重新整理後保留選擇。
- `zh-TW` 介面的 `Projects` 統一顯示為「專案列表」；`en` 介面顯示 `Projects`，不得出現「專案s」這類混合文案。
- 同一畫面不得把中文與英文 UI 文案任意混排；必要的產品名、角色名、Cloud Run、Git、API、RTO／RPO、ID、path、commit、digest 與其他 technical identifier 可以保留原文。
- `PROMOTION_SIMULATED`、`NEEDS_INPUT`、`BLOCKED` 等機器狀態碼在主要視圖顯示人類可理解的語意；原始 enum 只應在 technical detail、audit 或 debug context 可回查。
- Project name、repository URL／path、Change title、使用者輸入與 evidence payload 不因切換語系而被翻譯或改寫。
- 切換 locale 不得改變 Project、Change、DecisionRecord、版本、gate、actor、audit 或 evidence refs。

## 7. P0 明確不做

為保持這份 UI 契約可在五週內驗證，以下不是 P0 acceptance：

- 自動掃描所有 GitHub／GitLab／self-hosted GitLab repository。
- 任意 remote agent orchestration 或自主 FDE。
- 完整 Jira／Backstage／PM project management。
- Production deployment、完整 rollback、跨資源補償交易。
- 多租戶商業化、SSO／SCIM、細粒度企業 ACL。
- Cloud Assist MCP、完整 ADC integration、內網 connector。
- 以 RAG 相似度取代 policy engine 或人類決策。

上述功能若出現在 UI，必須標為 `P1`／`P2` 或 `NOT AVAILABLE`，不能放在 P0 主流程中造成已完成的錯覺。

## 8. 開發前的通過門檻

只有全部條件成立，才把 implementation readiness 從 `NO-GO` 改成 `CONDITIONAL GO`：

1. UI-01 至 UI-09 以真實可保存狀態跑通。
2. UI-10 至 UI-14 的負向案例能正確阻擋，且能回查原因與 evidence。
3. UI-15 重新整理與 deep link 不會遺失或混淆 context。
4. UI-16 至 UI-20 的語系切換、status label、文件 readiness、Context Pack lineage、資料保留與 locale persistence 通過驗證。
5. Brief、Agent Pack、Evidence Bundle 與 Receipt 的共享欄位能從同一 canonical record 查回。
6. 每個 mutation 都能看到 actor、版本、前後狀態與 audit event。
7. 至少一位非實作者能依照畫面完成 UI-07、UI-12 與 UI-20，並說明為什麼一個被阻擋、一個可以進 staging，以及為什麼缺件不能由 AI 補造。
8. 若任一項只能用靜態 fixture 或 toast 模擬，結果標記為 `NOT VERIFIED`，不得宣稱 UI flow 完成。

## 9. 下一步

1. 用 UI-01 至 UI-20 在目前 client-side prototype 完成正向與負向互動演練，先不接真實 Go API。
2. 修正所有只能 toast／preview、沒有明確保存結果或沒有阻擋原因的 action。
3. 完成互動驗證後，凍結 UI state／action／evidence contract。
4. 再以凍結的 contract 設計 Go API、Firestore schema 與 Cloud Run handler。
5. 每一個 UI acceptance case 都要對應一個後端 API／state transition／test，不以畫面完成取代 domain logic。
