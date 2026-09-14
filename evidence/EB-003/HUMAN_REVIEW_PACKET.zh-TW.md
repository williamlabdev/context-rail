# EB-003 人類易讀 Evidence Packet

這頁是 EB-003 的入口。先看結論與範圍，再依需要回查同一個 Evidence Bundle 裡的原始輸出；不需要先閱讀整份測試 log。

## 目前結論

| 項目 | 狀態 |
| --- | --- |
| Feature | VS-002：Project Registry／Documents & AI Context 唯讀工作區 |
| DecisionRecord | `DR-003 / ACCEPTED_FOR_DEVELOPMENT` |
| Work Order | `AWO-003 / ISSUED` |
| Reviewed source | `25204a5`（實作與 evidence capture checkpoint） |
| Scope | 本機 development/testing；不含 Cloud Run、IAM、production |
| VS-002 implementation | `IMPLEMENTATION_COMPLETE_PENDING_HUMAN_REVIEW` |
| VS-002 final review | 尚未宣告；這頁提供人審入口，不代替人類決定 |

## 這個 slice 實際展示什麼

1. 啟動 local service，指定兩個明確 Project fixture roots。
2. 讀取 Project Registry，列出 `Order Operations Portal` 與 `Support Insights`。
3. 選取 Project，顯示 repositories、services、environments、decisions、documents、Context 與 readiness。
4. `Support Insights` 的 Context 顯示 `STALE`，service runtime 顯示 `UNDECLARED`，並列出 stale source `README.md`。
5. 畫面標示 `READ-ONLY`；目前只有 GET API，沒有 create/edit/archive/upload/approve/deploy 控制。

## 人審時先回答這四件事

| 問題 | 目前可觀察結果 | 回查位置 |
| --- | --- | --- |
| 是否真的讀到兩個不同 Project？ | 是；切換後 detail 與 status 會替換 | `browser-output.txt`、`tests/browser/project-context.spec.ts` |
| 不確定狀態是否被保留？ | Go contract 與 HTTP contract 覆蓋 `STALE`、`MISSING`、`CONFLICT`、`UNKNOWN`、`UNDECLARED` | `tests/registry/contract_test.go`、`tests/http/projects_test.go` |
| 是否可能把 observation 當 authorization？ | Readiness 明確標示 `Informational; not authorization`；沒有 mutation route | `frontend/src/components/ReadinessSummary.tsx`、`frontend/src/components/WorkspaceState.tsx` |
| 是否有跨 Project 或寫入風險？ | browser test 檢查切換與所有 request method；consumer fixtures 需做 before/after hash | `tests/browser/project-context.spec.ts`、`mutation-check.txt` |

## 重要的非目標

- 這不是 Project CRUD。
- 這不是 DecisionRecord approval、Work Order issuance 或 deployment gate 的執行器。
- 這不是 Cloud Run staging deployment，也沒有 production release approval。
- fixture root 是明確設定值；不做 arbitrary filesystem traversal 或 remote URL crawling。

## 驗證導覽

| 層級 | 看什麼 | 結果欄位 |
| --- | --- | --- |
| Go contract | snapshot、錯誤、path safety、status preservation、zero mutation | `go-test-output.txt` |
| HTTP contract | list/detail、四類服務錯誤、GET-only boundary、status passthrough | `go-test-output.txt` |
| Frontend contract | API parsing、LOADING/READY/EMPTY/ERROR、uncertain status rendering | `frontend-quality-output.txt` |
| Browser journey | 兩個 Project、切換、STALE/UNDECLARED、無 mutation controls | `browser-output.txt` |
| Provenance | source-vs-derived document、stale source reason、readiness reasons | `frontend/src/components/DocumentStatusList.tsx`、`ReadinessSummary.tsx` |

各項原始 output 已放在本 bundle；若要核對，請先看下表，再打開對應檔案，不需要從 log 第一行開始閱讀。

## 判讀規則

- `PASS` 只代表該項檢查的輸出已存在且可重跑，不代表 production readiness。
- `STALE`、`MISSING`、`CONFLICT`、`UNKNOWN`、`UNDECLARED` 都是需要保留的觀察狀態，不可自動升級為 `CURRENT` 或 `READY`。
- 本 packet 方便人審，不取代原始測試輸出，也不新增 deployment authorization。
