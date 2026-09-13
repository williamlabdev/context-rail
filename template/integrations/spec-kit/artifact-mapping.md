# Spec Kit to ContextRail artifact mapping

Spec Kit 負責協助形成可執行的 specification；ContextRail 負責判斷變更是否符合 Project Context、policy 與 release evidence。兩者的 artifact 不可互相取代。

| Spec Kit artifact / phase | ContextRail artifact | Boundary |
| --- | --- | --- |
| constitution | Project principles、`docs/governance/policies.md` | principles 需由 Project owner 接受；不得覆寫既有 policy |
| specify | `Request` + `Change Slice` spec | 描述 what／why、user behavior、scope 與 acceptance criteria |
| clarify | `NEEDS_INPUT` questions / Request updates | 未解答的問題不能被模型自行補成事實 |
| plan | architecture impact + implementation plan | 技術方案仍是 candidate，重大選擇需進入 `DecisionRecord` |
| tasks | bounded execution tasks / `AgentWorkOrder` input | tasks 不會自行授予 repository、IAM 或 deployment 權限 |
| implement | `AgentRunRecord` | 只記錄執行聲明；實際 commit、diff、tests 由 Git／CI read-back 驗證 |
| analyze / checklist / converge | evidence reconciliation + review preparation | 不能取代獨立 review 或 human gate |
| release / deploy | ContextRail environment gate + receipt | Spec Kit 不代表 staging 或 production authorization |

## Minimum handoff references

每次由 Spec Kit 產生或更新的 feature work，至少要能回指：

`project_id`、`request_id`、`change_id`、`decision_id`、spec／plan／tasks paths、`source_snapshot_hash`、`policy_version`、status 與 human gate。

Template 內的 `work-orders/AWO-000-template.json` 與 `runs/ARR-000-template.json` 是可複製的結構化交接骨架。它們不會自動把 Request 變成 accepted，也不會把 `AgentRunRecord` 當成 Git／CI provenance；缺少人類 gate 時應保留 `PENDING`、`BLOCKED` 或 `NOT_STARTED`。

若某個 Spec Kit artifact 無法回指這些來源，狀態應為 `NEEDS_INPUT` 或 `STALE`，而不是直接進入 implementation。

## Pilot sequence

```text
REQ-001 manual order review
        ↓
Spec Kit specify / clarify
        ↓
Change Slice: stateless review endpoint + validation + UI result
        ↓
DecisionRecord DR-001
        ↓
Spec Kit plan / tasks
        ↓
bounded agent execution
        ↓
test + build + review evidence
        ↓
Cloud Run staging gate
```
