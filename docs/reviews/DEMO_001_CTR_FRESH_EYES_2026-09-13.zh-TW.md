# 第一個 Demo：CTR 與 Fresh-eyes Review

- Review target: `bb1aa0ee452ef52dd83007918ad15afe08a810ea`
- Branch: `codex/001-manual-order-review`
- Date: 2026-09-13
- Review completion: `COMPLETE`；兩位 finder 與一位獨立 verifier 均已完成。中途功能 finder/verifier 因額度中斷，使用者要求繼續後沿用相同任務完成。
- Scope: `demo/order-operations-portal/` 的完整 REQ-001 slice、支援它的 readiness inspector、Context Pack 工具與交接契約；不限於 `develop..HEAD`。
- Review decision: `REQUEST_CHANGES`；屬於 AI review 證據，沒有取代人工接受需求或 release approval。
- 本輪交付：review 報告。產品修正、DecisionRecord、EB-001 的 reviewer 狀態及部署狀態均未變更。

## 結論

本機測試與 build 可以重現，但現有測試沒有證明所有驗收條件或治理 gate 成立。第一個 demo 可繼續用於本機開發與修正；完整功能驗收和治理流程尚不能標為完成。Cloud Run 的缺席是已知後續工作，不是本機 review 的前置障礙。

Context Pack 中已列入的七份來源 hash 與 aggregate hash 均符合目前原始檔；應用程式及測試在 `08e3d47..bb1aa0e` 沒有 diff。這支持既有 local evidence 的程式版本連續性，但不會替未涵蓋的來源、人工決策或 UI 行為背書。

去重後保留 **15 個 `CONFIRMED` finding**：🔴 4／🟠 8／🟡 3。另有 C01 的 `PLAUSIBLE` 設計疑慮，以及主審提出的 C02 範圍選擇，均留待設計裁決。

| 分層 | 已確認項目 |
| --- | --- |
| 🔴 設計級 | F01 人工核准解析、F02 evidence 接受、F04 deploy 前置 gate、F06 policy snapshot 缺漏 |
| 🟠 中 | F03 context 失效傳播、F05 revision smoke 綁定、F07 strict hash 驗證、F08 決策狀態矛盾、F09 handoff 缺漏、F11 備註 HTML 執行、F12 非法 JSON、F13 UI 缺時間 |
| 🟡 小 | F10 branch 描述漂移、F14 未送出草稿遺失、F15 whitespace 測試缺口 |

影響範圍：readiness 目前只產生唯讀報告，原 demo 仍顯示 local/testing/staging `NEEDS_INPUT`。F01–F03 證明錯誤判讀，沒有證明目前會自動賦權；F04–F05 是未來呼叫 staging helper 時的缺陷，所有反例均在本機隔離環境，沒有實際部署。F08–F09 也不構成歷史未授權執行的證明。

## 方法與獨立性

使用 `/ctr`、`/fresh-eyes` 及兩者引用的 `critical-thinking-review.md`。先核對產出與規格，再檢查前提、偏誤與流程設計。

- 主審：直接讀原始檔、執行基礎驗證與 readiness 反例；部署只在 repo 外副本以假 `gcloud`、假 `curl` 執行。
- Finder A：Darwin，`01a099bf-bbbd-7633-8a82-4393d73bb1d5`；功能、API、UI、測試鏡頭。
- Finder B：Russell，`01a099bf-bc27-7f60-9bca-38f5815afbc6`；跨檔契約、狀態、證據來源鏡頭。
- Verifier：Kepler，`01a099c1-d1db-74f0-9374-78597a2eb6ad`；逐項依原始檔/重現判定 `CONFIRMED`、`PLAUSIBLE`、`REFUTED`。
- Finder 以 `fork_context=false` 啟動，僅收到檔案路徑、檢查目標與工具要求；沒有帶入本對話的完成結論。Verifier 收到候選 finding，沒有繼承對話。

## 主審查核紀錄

以下路徑相對 repository root；Go 指令在 `demo/order-operations-portal/` 執行。

| 查核 | 結果與限制 |
| --- | --- |
| `git status --short`、`git rev-parse HEAD` | 起始工作樹乾淨；review target 如上 |
| `./scripts/validate-demo.sh` | PASS；此腳本檢查檔案存在、Go tests/build，不驗證人類核准 |
| `template/.venv/bin/python -m unittest discover -s tests -v` | 4 tests PASS；現有測試主要覆蓋 fixture 與 pending 狀態 |
| `template/.venv/bin/python template/scripts/validate_project_context.py --root demo/order-operations-portal --strict` | `PROJECT CONTEXT STRICT PASS`；不等於 approval、review 或所有來源 lineage 已驗證 |
| `git diff 08e3d47..bb1aa0e -- demo/order-operations-portal/main.go demo/order-operations-portal/main_test.go demo/order-operations-portal/web/index.html demo/order-operations-portal/go.mod` | 無 diff |
| 對目前 Context Pack 的每個 source 重新算 SHA-256，再算 aggregate | 七份來源與 aggregate 全部相符；兩個 commit 的 snapshot 相同 |
| readiness 函數反例（附錄 A） | 未核准可 READY、已核准可被其他環境阻擋、FAIL review 可通過、缺文件不阻擋 local gate |
| repo 外部署副本 + PATH 中的假 gcloud/curl | 無 DecisionRecord、review 未完成、任意 image tag，仍呼叫 deploy 並寫 PASS；mock 健康回應來自舊 revision，receipt 卻記新 revision |
| 主審讀取 finder 反例測試並比對 Go/UI 原始檔 SHA-256 | repo 外副本的 main.go/main_test.go/web/index.html 與目標完全相同 |
| `go test -race -count=1 -run '^TestFresh' -v .`，repo 外副本 | invalid JSON 反例 FAIL（HTTP 200 且修改訂單）；空白/未知訂單/非法決策不改狀態 PASS；64 reviews + 64 lists 並發案例 PASS |

## Fresh-eyes findings

### F01 — 🔴 CONFIRMED：人工核准以整份物件的字串黑名單判斷

證據：`scripts/context_rail_readiness.py:77-82`、`demo/order-operations-portal/decisions/DR-001-manual-order-review.json:18-21`。

將 `accept_request` 改為 `ACCEPTED`，保留 staging pending 和 production blocked，`decision_is_human_accepted` 仍回傳 `False`；反之，刪除整個 `human_decisions` 或設 `accept_request: REJECTED`，在頂層 accepted status 下會回傳 `True`，local gate 成為 READY。

定級理由：核准缺席能被誤認為核准，而 production 的保護狀態也會阻擋不相關的本機動作，破壞以 action 分開判斷的核心設計。建議對特定 action 使用明確 accepted 狀態和必要核准欄位，其他環境的狀態不參與該 action 的接受判斷。

### F02 — 🔴 CONFIRMED：Evidence gate 沒有明確驗證 review PASS 與候選身份

證據：`scripts/context_rail_readiness.py:85-119`、`demo/order-operations-portal/evidence/EB-001/code-review.md:15`、`demo/order-operations-portal/project.yaml:46-49`。

只排除含 `REVIEW_REQUIRED` 的 review 文件，所以 `Status: FAIL` 也能通過。receipt 只查 `status == PASS`，不核對 Project、Decision、目標、commit、digest。主審以真實 local test/build 文件、FAIL review 和指向其他環境/映像的 PASS receipt 做記憶體注入，在 request accepted 下得到 testing/staging READY。

定級理由：不合格或不屬於這個候選的證據能使 gate 顯示通過。建議使用明確結構化結果，核對 reviewer、reviewed commit、decision 與 target/image 身份；本機測試結果不能被宣稱為已觀察的 CI run。

### F03 — 🟠 CONFIRMED：Local gate 沒有傳播缺文件或失效 context

證據：`scripts/context_rail_readiness.py:128-147`、`docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md:187-191`。

`decision_ready` 與 `local_development_ready` 分開計算，後者只看任一 accepted DecisionRecord。主審把必要 architecture 文件狀態模擬為 MISSING，request accepted 時得到 decision NEEDS_INPUT、local READY。`requiredFor: development` 的文件也未被此 gate 檢查，decision 的 snapshot 未與當前 context 比對。

定級理由：來源失效後，Agent 的可執行範圍仍可能被表示為有效。建議依目標 action 檢查所需文件、綁定的 decision/source/policy 版本，並傳播失效；不需要把無關的雲端資料加入本機 gate。

### F04 — 🔴 CONFIRMED：部署入口未執行宣告的核准與 digest gate

證據：`demo/order-operations-portal/scripts/deploy-staging.sh:7-27`、`demo/order-operations-portal/specs/001-manual-order-review/quickstart.md:54-56`、`demo/order-operations-portal/requests/REQ-001-manual-order-review.md:20`。

腳本只檢查 gcloud 是否存在及三個非空環境變數，之後直接 deploy。隔離副本沒有 DecisionRecord、review 仍未完成，`IMAGE_URI=unreviewed:latest` 仍進入假 gcloud，最後寫 PASS。沒有核對核准 image digest、reviewed commit、release decision 或授權 target。

定級理由：實際副作用入口沒有落實宣告的治理邊界。這是未來啟用 staging 前必修的缺口，並不表示本輪發生了未授權部署。建議部署前強制驗證同一候選的 scope、人工核准、review、digest 與目標。

### F05 — 🟠 CONFIRMED：Staging smoke 可能驗證舊 revision，receipt 卻標記新 revision

證據：`demo/order-operations-portal/scripts/deploy-staging.sh:23-43`。

腳本使用 `--no-traffic`，卻對 service 的 `status.url` 做 smoke，接著將 `latestReadyRevisionName` 寫入 receipt。既有服務仍將流量送給舊 revision 時，舊版健康可以替未受測新版寫 PASS；主審的離線 mock 已重現這組輸出。Google 官方說明確認 `--no-traffic` 不將流量送至新 revision。[gcloud run deploy](https://docs.cloud.google.com/sdk/gcloud/reference/run/deploy)

定級理由：receipt 的測試身份不可靠。建議對新 revision 的可識別 URL 測試並核對實際 revision/image，不能只把 service-level health 與 latest-ready 名稱拼在一起。

### F06 — 🔴 CONFIRMED：Context snapshot 沒有包含宣告為必要來源的治理政策

證據：`demo/order-operations-portal/project.yaml:81-84`、`template/scripts/rebuild_context_pack.py:16-24`、`demo/order-operations-portal/docs/ai/context-pack.json:9-65`。

manifest 有八份 SSOT，pack 固定只收七份，漏掉 `docs/governance/policies.md`。因此只改該檔中的核准分離政策，不會改變 source snapshot，DR/EB 的引用仍保持相同。Finder B 的記憶體反例與主審的來源集合比對均確認這個缺口。

定級理由：治理決策的必要來源可變更卻不使引用身份改變。建議依 manifest 宣告及 action 所需來源產生 snapshot，並驗證覆蓋範圍；補齊後要保留舊 snapshot 的歷史，不能把新 hash 倒填成舊執行當時的輸入。

### F07 — 🟠 CONFIRMED：Strict validator 不核對實際 hash 或來源覆蓋

證據：`template/scripts/validate_project_context.py:87-99`、`docs/validation/CONTEXT_RAIL_TEMPLATE_CONSUMER_VALIDATION.md:30`。

validator 只檢查來源存在和 `sha256:` 前綴，沒有重新計算 source/aggregate hash，也沒有核對必要來源是否都入 pack。Finder B 與主審分別在記憶體將 source hash 和 snapshot hash 換成 `sha256:` 加 64 個零，原函數 `validate(root, strict=True)` 仍回傳 `[]`。

定級理由：`STRICT PASS` 不能支持 validation 文件所稱的「source hash lineage 已驗證」。這不是目前七份 hash 算錯，而是驗證器未能拒絕錯誤資料。建議補上確定性的 hash 與 coverage 驗證及反例。

### F08 — 🟠 CONFIRMED：同一決策的接受狀態與投影相互矛盾

證據：`demo/order-operations-portal/decisions/DR-001-manual-order-review.json:6,19`、`demo/order-operations-portal/decisions/architecture-impact.json:6`、`demo/order-operations-portal/specs/001-manual-order-review/plan.md:33`。

頂層 `ACCEPTED_FOR_DEVELOPMENT`、plan 勾選 accepted Request，卻同時存在 `accept_request: PENDING_HUMAN_CONFIRMATION` 與 `CANDIDATE_REQUIRES_HUMAN_ACCEPTANCE`。pending fixture 的存在是刻意的；缺口是沒有把候選狀態與正式接受狀態統一，使接手者可讀出相反結論。

定級理由：授權語義會隨讀取欄位而不同。建議明示為待接受候選，或連到可核實的特定人工決策後更新各投影。不能由此推定對話或 repo 外不存在授權，也不能在 review 中代填核准。

### F09 — 🟠 CONFIRMED：已提交 slice 缺少可重建的 Work Order／Run Record

證據：`docs/decisions/ADR-001-structured-agent-handoff.zh-TW.md:18-25`、`template/integrations/spec-kit/artifact-mapping.md:11-22`、`demo/order-operations-portal/specs/001-manual-order-review/plan.md:56-62`。

ADR 要求 accepted `AgentWorkOrder`；Spec Kit mapping 要求 `change_id` 與交接 references。完整 tracked demo 中只有規則/流程文字提及，沒有對應的 Work Order／AgentRunRecord 實體，plan 也直接把 plan/tasks 接到 AgentRunRecord。DR allowed paths 和 EB commit 提供部分資訊，但尚不能從提交內容重建哪份工作契約連到哪次執行。

定級理由：這是完整 governance demo 的交接證據缺口，不妨礙先 review 既有 brownfield 程式。建議未來執行前建立明確工作契約；歷史驗證若整理為 run record，須標明事後重建與可觀察來源，不偽裝成當時已存在的核准。

### F10 — 🟡 CONFIRMED：Feature branch 的描述已過時

證據：`demo/order-operations-portal/specs/001-manual-order-review/spec.md:3`、`demo/order-operations-portal/specs/001-manual-order-review/plan.md:3`。

spec 仍聲稱目前 checkout 在 develop；實際 review branch 是 `codex/001-manual-order-review`。plan 的未加 `codex/` 名稱明確標為 planned，本身不另外構成缺陷。定級理由：spec 的當前狀態描述漂移，可能使接手者找錯分支，但沒有改變已確認的 source commit。建議將歷史規劃與目前實際分支分欄記錄。

### F11 — 🟠 CONFIRMED：覆核備註被當作 HTML 執行

證據：`demo/order-operations-portal/web/index.html:37`、`demo/order-operations-portal/main.go:97`。

`order.note` 經 API 存入程序記憶體，再直接插入 `card.innerHTML`。Finder A 實際在瀏覽器送出 `<img src=x onerror="document.querySelector('#message').textContent='FRESH_EYES_XSS_EXECUTED'">`，結果訊息變為該 marker，重新載入仍再執行。主審讀原始檔確認從 note 儲存到 innerHTML 的未轉義資料路徑。

定級理由：即使使用 synthetic data，備註仍可改寫操作者所見的覆核結果。只驗證本機頁面 marker，沒有把外傳或其他行為列為已觀察影響。建議以 textContent/安全 DOM 節點呈現所有輸入文字並補 browser regression。

### F12 — 🟠 CONFIRMED：完整 body 不是合法 JSON，仍會修改訂單

證據：`demo/order-operations-portal/main.go:77`、`demo/order-operations-portal/specs/001-manual-order-review/contracts/http.md:38`。

handler 只 Decode 一次，接受 `{"decision":"APPROVED","note":"valid prefix"}garbage`，或兩個相連 JSON 物件；只處理第一個物件而回 HTTP 200。Finder A 與主審分別實跑反例：`json.Valid=false`，但 order 被更新。契約要求 invalid JSON 回 400。

定級理由：非法輸入不僅未被拒絕，還產生 state change。建議成功 decode 後確認剩餘內容只有空白及 EOF，所有驗證完成前不得更新訂單。

### F13 — 🟠 CONFIRMED：UI 未顯示 FR-004 必需的覆核時間

證據：`demo/order-operations-portal/specs/001-manual-order-review/spec.md:48`、`demo/order-operations-portal/web/index.html:45-47`、`demo/order-operations-portal/main.go:98`。

API 回傳 `reviewedAt`，UI 成功訊息卻只組合 order ID/decision，之後重新載入不含時間的列表。Finder A 的 browser snapshot 沒有覆核時間；主審搜尋確認 UI 未引用 reviewedAt。

定級理由：明列 MUST 的 acceptance condition 未滿足。建議至少在覆核成功結果呈現 API 回傳時間，若卡片需要持續顯示，採 process-local 儲存即可，不必擴大成持久化需求。

### F14 — 🟡 CONFIRMED：送出一張訂單會清空另一張訂單的未送出備註

證據：`demo/order-operations-portal/web/index.html:34,47`。

Finder A 先在兩張卡片填入不同備註，送出第一張後，第二張的 `DRAFT-KEEP-1002` 消失。每次 review 都呼叫 load 清空整個 orders DOM，因此其他 textarea 的輸入遺失。

定級理由：造成使用者輸入遺失，但草稿保留不是本 slice 明訂 MUST，單筆流程仍能進行。建議局部更新被覆核卡片，或重繪前保留其他訂單草稿。

### F15 — 🟡 CONFIRMED：T008 已勾完成，但缺少 whitespace-only note 自動測試

證據：`demo/order-operations-portal/specs/001-manual-order-review/tasks.md:45`、`demo/order-operations-portal/main_test.go:23-32`。

TestReviewRequiresNote 只測未設定 Note 的空字串。Finder A 在另一份 repo 外副本把 `strings.TrimSpace(req.Note) == ""` 改成 `req.Note == ""`，原有測試仍全通過，新增 whitespace 反例才抓到退化。主審確認現有測試沒有 whitespace-only payload，且原始實作對 ASCII/全形空白的反例仍正確回 400、不改狀態。

定級理由：缺少宣稱已完成的回歸保護，並非原始驗證功能已壞。建議增加 approve/reject、空白 note 及 state unchanged 的斷言，讓 task 完成狀態對得上自動測試。

## CTR：前提、偏誤與待裁決事項

### C01 — 🔴 PLAUSIBLE：部署資格與部署結果可能混用同一個狀態

`docs/validation/CONTEXT_RAIL_TEMPLATE_CONSUMER_VALIDATION.md:55` 將 `ready_for_staging` 稱為 deployment gate，`scripts/context_rail_readiness.py:93-100,153-159` 卻要求既有部署 receipt。若日後將它接成首次部署前 gate，會形成先有部署結果才能開始部署的循環；若它只是「staging 已驗證」狀態，則是名稱與文件需要釐清。

Finder B 另查到 `specs/001-manual-order-review/tasks.md:77-79` 排為 deploy → revision/smoke → human staging approval；同檔第 105 行卻要求部署前 human gate。T023 可能指部署後驗收，不能確定兩者必為同一批准，因此與前述疑慮合併標記 PLAUSIBLE。文件應明確區分「准許執行部署」與「接受部署驗證結果」。

獨立 verifier 將「目前自動部署已經死鎖」的說法判為 **REFUTED**：deploy 腳本尚未消費 readiness。🔴 定級針對若要把這個模型用作強制 gate 的設計風險，不是宣稱已觀察到執行死鎖。

供 William 裁決的選項：保留部署後驗證的單一指標並重新命名；或分成部署前資格與部署後結果。建議後者，先用本機假 executor 驗證狀態順序，再接真實 Cloud Run。這是建議，沒有在本輪改動狀態契約。

### C02 — 🟠 本機、Testing/CI 與 Cloud Run 仍被模型化成固定鏈

`ready_for_cloud_testing` 只讀本機 evidence 文件，沒有觀察 CI run；inspector 也未依 manifest 的環境拓撲決定是否存在 testing stage。`examples/support-insights/project.yaml` 只有 development/staging/production，卻同樣收到 cloud-testing gate。

供 William 裁決的選項：把 inspector 明確限定為 Order demo 的固定 profile；或讓 readiness 依 manifest 的 environment/action 產生。通用 template 的方向較適合後者，可先支持 local 與可選 testing，Cloud Run 維持後續整合。

### C03 — 完成度的先前結論需要更正

「基礎測試通過，所以本機功能 slice 已完成」把測試覆蓋範圍等同全部驗收；這個判斷我錯了。本輪保留錯誤與更正紀錄：測試可重現，但未覆蓋的 UI/治理行為必須依實作和反例判斷。改為「已有可執行候選，review 要求修正」。

偏誤掃描：不以舊文件中的 Cloud Run 目標否定目前 local-first 選擇；不把 review agent 的結果視為人工 release approval；不因同一作者產生 spec、test、evidence 且彼此一致，就推論整條治理路徑已被驗證。

Synthetic、process-local、無真實付款/履約的範圍仍足以支援這個本機互動案例；沒有據此要求增加完整 CRUD、資料庫、登入或 A2A。

## 已修正與待處理

- 已修正：本報告更正前述完成度判斷，保留測試範圍與反例證據。
- 程式修正：本輪無；所有確認缺陷仍開啟。
- 待裁決：C01 的部署前/後狀態模型、C02 的通用 topology 或固定 profile 範圍。
- 人工 gate：未代填 human acceptance 或獨立人類 reviewer；report 的 AI 審查身份可追溯，但不等於人類核准。

建議後續批次：先修 F11–F15 的本機行為與測試；再修 F01–F03、F06–F07 的 readiness／lineage 反例；整理 F08–F10 的候選狀態與真實交接紀錄。C01/C02 決定後再處理部署流程，F04/F05 必須在啟用 staging helper 前修正。每個批次保留獨立 commit 和重跑結果，最後針對修正進行 fresh-eyes 複審。

## 附錄 A：主審 readiness 反例輸出

用 repository `.venv` Python 匯入 `scripts.context_rail_readiness`。在記憶體複製 DR-001 改變 `human_decisions`；以 `unittest.mock.patch` 只替換讀入結果，原始檔和歷史 evidence 均未修改。

```text
accept_request ACCEPTED + staging PENDING + production BLOCKED => False; local NEEDS_INPUT
no human_decisions => True; local READY
accept_request REJECTED => True; local READY
FAIL review + unrelated PASS receipt => evidence_state ('READY', [])
FAIL review + existing local test/build => cloud_testing_evidence_state ('READY', [])
request ACCEPTED + FAIL review + unrelated receipt => local/testing/staging READY
required architecture MISSING + request ACCEPTED => decision NEEDS_INPUT; local READY
```

部署 probe 在 `/tmp/context-rail-review.BILyNV/` 的腳本副本執行，PATH 第一順位均為自建 shell mocks；未使用真實 gcloud 或 curl：

```text
MOCK gcloud: run deploy ... --image unreviewed:latest --no-traffic
MOCK curl (old revision response): -fsS https://old-service.invalid/healthz
STAGING PASS
receipt.image_uri = unreviewed:latest
receipt.cloud_run_revision = new-revision-not-served
receipt.smoke.status = PASS
```

上述為反例測試證據，不能作為 Cloud Run deployment receipt。

## 附錄 B：Finder 工具證據摘要

### Finder A — 功能、API、UI

- 依 skill 讀指定來源、AGENTS 與協定，執行 `rg -n 'FR-|SC-|AC-|note|reject|approve|order|mutex|method|PENDING_REVIEW' demo/order-operations-portal`。
- `go test -race ./...`：PASS；`go vet ./...`：exit 0；validate-demo：PASS；相同來源副本 `go build ./...`：exit 0。
- Browser 實際 setValue/click/getAXState/reload：F11 marker 可執行且 reload 再現；F13 未顯示時間；F14 第二張草稿被清空。服務與頁面於收尾關閉。
- 副本反例 `TestFreshRejectWholeInvalidJSON` 失敗；`TestFreshValidationLeavesStateUnchanged`、`TestFreshConcurrentReviewsAndLists` 通過。主審已再執行同組案例。
- Whitespace mutation：原有 tests 仍 PASS，新反例抓到退化。測試原文位於 `/tmp/context-rail-fresh-a.QqT9ip/fresh_eyes_test.go`，mutation 位於其 `whitespace-mutant/`；均未寫入產品程式。

### Finder B — 治理、文件、來源

- 讀指定來源並執行 `rg -n 'ACCEPTED|PENDING|REVIEW_REQUIRED|source_snapshot|source_snapshot_hash|work.order|Work Order|Source commit|Status|PASS' demo/order-operations-portal template/integrations/spec-kit`。
- 以 `git show bb1aa0e:<path>` 查目標版本，`git ls-files`/`git grep` 查交接實體、reference、branch 和 task 順序。
- 在記憶體重算政策文字/來源集合：`declared_sourceOfTruth=8`、`snapshot_sources=7`、missing=`docs/governance/policies.md`，policy-only change 不改 snapshot。主審已另外確認集合差異。
- 對 validator 的記憶體 mock：零 source/snapshot hash 仍得到 `counterexample_strict_errors=[]`；主審以 repo venv 再次得到相同結果。Finder 的此項原始命令使用 `python3 -B`，主審使用 `template/.venv/bin/python`。
- 未成立的反駁：舊 evidence commit 並非自行失效，因應用與既有七份來源無 diff；review/receipt 本身仍如實標 pending；machine-local `.specify/feature.json` 未追蹤符合其 ignore 設計。這些不列缺陷。

### Verifier — 獨立覆核與去重

- 親自讀原始檔並執行隔離 readiness cases：缺/空/rejected human decision 誤為 READY；staging pending 或 production blocked 誤阻 local；FAIL/空 review 與錯誤 identity 的 receipt 可被接受；MISSING/STALE/obsolete snapshot 未阻 local。
- 以假 gcloud/curl 執行部署腳本：pending decisions/未完成 review/未核准 mutable tag 仍進 deploy；candidate health=500、舊服務 health=200，仍替 candidate revision 寫 PASS。未接觸真實雲端。
- policy-only 改動後重建，policy hash 改變而 snapshot 不變；零 hash 和 `sources=[]` 兩種反例均 `errors=[]`、exit 0、STRICT PASS。
- 讀 HEAD 的 66 個 tracked demo 檔案確認 handoff 缺口；比對 DR/plan/task/branch 的原文。缺少 tracked 記錄不推定 repo 外歷史。
- 獨立 browser 驗證 F11 的提交/reload marker、F13 的未呈現時間、F14 的草稿遺失；原碼/測試及 mutation 支持 F12/F15。也核對 baseline 與並發/拒絕案例的 PASS，不把正面結果抵銷反例。
- 將主審 R1/R2 合併為 F01；R3–R6 對應 F02–F05；文件 finder B1–B4/B6 對應 F06–F10；功能 finder A1–A5 對應 F11–F15。這 15 項均為 CONFIRMED。
- 主審 R7 與文件 finder B5 合併為 C01、判 PLAUSIBLE；排除現有自動部署死鎖與已發生未授權執行的推論。C02 是主審基於 manifest 的設計建議，沒有冒列為 verifier 確認的實作缺陷。
