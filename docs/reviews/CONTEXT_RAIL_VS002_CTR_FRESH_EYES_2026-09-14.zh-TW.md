# VS-002：CTR 與 Fresh-eyes Review

- Review target: `dbe7eb94dbddd216b8e3fe1fdf2f65f31bb67984`
- Branch: `codex/001-manual-order-review`
- Date: 2026-09-14
- Scope: `REQ-003`、`DR-003`、VS-002 Spec Kit `spec.md`／`plan.md`／`tasks.md`、research、data model、API/UI contracts、quickstart，以及 VS-001 dependency boundary
- Review method: CTR（契約、範圍、規格與任務一致性）＋ Fresh-eyes（不依賴前文結論，從第一次使用者與執行者角度重讀）
- Command availability: repo 內沒有可直接執行的 `ctr` 或 `fresh-eyes` binary；本報告依既有 `DEMO_001_CTR_FRESH_EYES` 方法與 review 定義完成等價唯讀查核
- Review decision: `REQUEST_CHANGES`
- Review identity: AI review；不取代人類接受 `DR-003`、AWO issuance 或後續 implementation review

## 1. 結論

VS-002 已經是明確的前後端 slice：Go read API、React/TypeScript read-only workspace、UI state、browser journey 與 evidence 都有描述。相較 VS-001，這次 `spec.md` 已真正加入可被人操作的瀏覽器旅程。

目前仍不能進入 implementation，原因不是前後端範圍不清，而是兩個執行邊界尚未對齊：`DR-003` 的 allowed paths 沒有涵蓋 tasks/plan 使用的測試路徑；VS-001 尚未有 Go implementation，VS-002 的 dependency 雖在文字中提到，卻沒有成為明確的 blocking gate。另有 task formatting 與前端工具設定的可執行性缺口。

這些問題都可在建立 `AWO-003` 前修正。現階段不應建立 Work Order，也不應因既有 root/template 測試通過而宣稱 VS-002 runtime 或 browser journey 已完成。

## 2. 查核證據

| 查核 | 結果 |
| --- | --- |
| `git status --short` | 起始與收尾工作樹乾淨 |
| `git rev-parse HEAD` | `dbe7eb94dbddd216b8e3fe1fdf2f65f31bb67984` |
| `python3 -m json.tool decisions/DR-003-project-registry-context-ui.json` | PASS |
| `template/.venv/bin/python -m unittest discover -s tests -v` | 15 tests PASS |
| `template/.venv/bin/python -m unittest discover -s template/tests -v` | 8 tests PASS |
| VS-002 runtime／Go／React／browser execution | 尚未開始；不是 PASS，也不是產品缺陷結論 |

## 3. CTR findings

### F01 — HIGH：DR-003 allowed paths 與 plan/tasks 的測試路徑不一致

**證據**：

- `decisions/DR-003-project-registry-context-ui.json:22-32` 只列 `tests/registry/` 與 `tests/frontend/`；
- `specs/003-project-registry-context-ui/plan.md:100-104` 使用 `tests/http/` 與 `tests/browser/`；
- `specs/003-project-registry-context-ui/tasks.md:30-34,65,79,92` 也要求建立 `tests/http/projects_test.go` 與 `tests/browser/project-context.spec.ts`。

**影響**：若直接依目前 `DR-003` 建立 AWO，至少兩個必要測試路徑不在 allowed scope 內；若依 tasks 執行，則會違反 DecisionRecord 的 bounded path。

**建議**：在 `DR-003` 明確加入 `tests/http/`、`tests/browser/`，或統一將 tasks/plan 的路徑改到 DecisionRecord 已允許的目錄；更新後重新固定 source identity。

### F02 — HIGH：VS-001 尚未實作，但 VS-002 沒有真正的 blocking dependency gate

**證據**：

- `specs/003-project-registry-context-ui/plan.md:9-11,44-50,112-119` 說 VS-002 消費 VS-001 contract，且不能重複 normalization；
- `specs/003-project-registry-context-ui/tasks.md:35-36` 卻把「reuse VS-001 model 或記錄 adapter」列成 VS-002 Phase 2 task；
- `specs/003-project-registry-context-ui/tasks.md:109-113` 只用文字說 User Story 1 depends on VS-001，沒有要求 VS-001 core implementation 或明確 reusable package 先達成特定狀態。

**影響**：執行者可能在 VS-001 尚未完成時，於 VS-002 重新建立 `internal/projectregistry/`，造成兩份語義模型；也可能先寫 API/UI，再回頭發現 upstream contract 尚未可被 Go package 消費。

**建議**：二選一並寫入 tasks gate：

1. 先完成 VS-001 的 Go contract/package，VS-002 只允許消費它；或
2. 明確把 VS-001 Go package 作為 VS-002 的 prerequisite work package，並記錄不重複模型的驗收條件。

未完成此 gate 前，不應簽發 AWO-003。

### F03 — MEDIUM：Foundational phase 仍有 User Story labels

**證據**：`specs/003-project-registry-context-ui/tasks.md:35-36` 的 T012、T013 仍標有 `[US1]`，但兩者位於 `Phase 2: Foundational contracts`。

**影響**：違反 Spec Kit tasks 的格式語義：Foundational task 不應歸屬某一 User Story，會讓 story completion 與 dependency graph 產生誤讀。

**建議**：移除 T012、T013 的 `[US1]`；若它們其實是 US1 專屬工作，則移到 Phase 3，並保留 Phase 2 只做共享 contract setup。

### F04 — MEDIUM：前端 package/config 與 quickstart 的可執行設定尚未形成 task coverage

**證據**：

- `specs/003-project-registry-context-ui/plan.md:85-95` 只列 `package.json`、`package-lock.json`、source 與 tests，沒有 `tsconfig`、Vite config、Playwright config 或 build/test scripts 的明確位置；
- `specs/003-project-registry-context-ui/quickstart.md:24-32` 要求 `npm run typecheck`、`build`、`test`、`test:e2e`；
- `specs/003-project-registry-context-ui/tasks.md:21-22,99-102` 要求執行這些命令，但沒有一個 task 明確建立對應 scripts/config。

**影響**：`quickstart` 的命令名稱看似固定，但尚無文件可回答 scripts 如何存在、browser runner 如何啟動 service、frontend build 如何被 Go static host 消費。

**建議**：在 setup task 加入 `frontend/tsconfig.json`、Vite config、Playwright config 與 package scripts 的明確 task；在 quickstart 指定 browser test 的 service lifecycle、host/port 與 fixture configuration。這不是要求現在安裝依賴，而是讓後續 Work Order 可重現。

## 4. Fresh-eyes findings

### F05 — MEDIUM：第一次接手者可以理解「read-only」，但尚不能從 quickstart 得知完整執行入口

**觀察**：

- UI contract 清楚禁止 create、edit、archive、upload、approve、deploy；
- spec 的三個 user stories 清楚描述 registry、uncertain states 與 loading/empty/error；
- 但 quickstart 仍需要等 implementation 階段決定 service host/port 及 browser test 的 process lifecycle。

**影響**：讀者能理解要做什麼，卻不能在目前 checkout 直接從文件完成一條端到端操作；這是規格尚未進入 implementation-ready 的限制，不是目前產品執行失敗。

**建議**：與 F04 一起補齊固定的 local runner contract；若實作前仍未決定，將 quickstart 狀態標成 `PLANNED`，不要讓它看起來像已驗證的命令。

### F06 — LOW：DR-003 的 unknowns 與 research 的候選決策尚未標示清楚層級

**證據**：

- `DR-003:54-57` 仍把 frontend bundler/browser runner 與 static host/process split 列為 unknown；
- `specs/003-project-registry-context-ui/research.md` 已提出 React/TypeScript、Vite、Playwright 與 Go static host 的 candidate decisions。

**影響**：熟悉治理流程的讀者知道這可能是「候選尚未被人接受」；第一次接手者可能以為 plan 已定案，或反過來以為 research 不具參考價值。

**建議**：保留 DR-003 的 unknown 以表示尚未 human accepted，但在 plan/research 明確加上 `candidate pending DR-003` 的標籤，避免 candidate 與 accepted decision 混用。

## 5. Positive findings

- `spec.md` 已包含完整前端使用旅程，而不只是後端 snapshot contract。
- `plan.md` 明確把 Go source-reading、React presentation、browser evidence 分開。
- API contract 沒有加入 write route；UI contract 也禁止 mutation controls。
- `DERIVED` context status 已與 VS-001 實際輸出一致，沒有再出現先前的 data-model mismatch。
- Spec、plan、tasks 的 10 個 FR 均有 traceability；沒有發現零 coverage 的 functional requirement。
- `DR-003` 仍是 `PENDING_HUMAN_DECISION`，沒有偽造人類核准或部署證據。

## 6. Coverage summary

| 項目 | 結果 |
| --- | --- |
| Functional Requirements | 10 |
| User Stories | 3 |
| Tasks | 42 |
| FR coverage | 10/10 有明確 task mapping |
| Browser journey coverage | Positive、uncertain、loading/empty/error 均有 task |
| Mutation protection coverage | API/UI 禁寫、hash comparison、no mutation task 均有描述 |
| Constitution critical conflict | 0 |
| Review-blocking findings | F01、F02 |

## 7. Decision and next actions

目前判定：`REQUEST_CHANGES`。

修正順序：

1. 對齊 `DR-003` allowed paths 與 plan/tasks；
2. 把 VS-001 implementation/package 狀態變成 VS-002 的明確 prerequisite；
3. 移除 T012/T013 的 Foundational User Story labels；
4. 補前端 config/scripts、service lifecycle 與 browser host/port contract；
5. 重新執行 CTR/Fresh-eyes；
6. 只有 review 通過且人類接受 `DR-003` 後，才建立 AWO-003。

本報告是 AI review evidence，不是人類接受、獨立 reviewer 或 staging/production authorization。
