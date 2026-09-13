# ContextRail Hero Demo：角色化報告索引

> **示例資料／非真實部署證據**

這一組檔案展示「一個 Single Source of Truth，產生多份不同身份的報告」。各報告不是互相複製後分別維護，而是共用同一組決策識別欄位、Project environment topology 與證據引用。

## 先看這裡

| 讀者 | 報告 | 用途 |
|---|---|---|
| 主管／PM | [Executive Change Decision Brief](./01_EXECUTIVE_CHANGE_DECISION_BRIEF_HERO.zh-TW.md) | 用白話回答要不要做、影響、風險與需要誰決定 |
| Solution Architect／Tech Lead | [Technical Architecture & Cost Report](./02_TECHNICAL_ARCHITECTURE_COST_REPORT_HERO.zh-TW.md) | 查看架構方案、Cloud Run 服務影響、成本驅動因子與 NFR |
| FDE／Implementation／業務窗口 | [Implementation Readiness Report](./03_IMPLEMENTATION_READINESS_HERO.zh-TW.md) | 查看客戶／內部團隊還要補什麼資料、缺資料會阻擋什麼 |
| Developer／AI Agent | [Agent Context Pack](./AGENT_CONTEXT_PACK_HERO.example.json) | 讀取 allowed paths、forbidden actions、驗收條件與權限邊界 |
| DevOps／SRE／Release Owner | [Engineering Evidence & Release Receipt](./RELEASE_RECEIPT_HERO.example.json) | 核對 source、digest、provenance、設定、rollout 與 drift gate |
| 系統／稽核 | [Canonical Decision Record](./CANONICAL_DECISION_RECORD_HERO.example.json) | 唯一的決策事實來源，不直接當作任何一個角色的閱讀版 |

## 共用識別欄位

所有報告都必須帶著以下欄位，才能確認它們描述的是同一個變更：

| 欄位 | 示例值 | 作用 |
|---|---|---|
| `project_id`／`project_version` | `proj-internal-approval`／`3` | 確認變更屬於哪個 Project，以及依照哪一版 Project contract |
| `project_key` | `APV` | 給人辨識 Project 的穩定短碼 |
| `decision_id` | `CHG-2026-0017` | 串起同一個需求／變更 |
| `decision_version` | `3` | 確認依照哪一版決策 |
| `source_snapshot_hash` | `sha256:example-source-snapshot-7f83...` | 確認輸入需求與架構快照 |
| `policy_version` | `internal-cloudrun-v1` | 確認使用哪一組規則 |
| `environment_topology_id`／`version` | `internal-approval-envs:v1` | 確認這次變更依照哪一組環境順序、target 與 promotion policy |
| `target_environment_id` | `staging` | 確認目前允許進入哪個環境；不能把 staging PASS 當成 production approval |
| `evidence_refs` | `evidence://CHG-2026-0017/...` | 連到測試、build、provenance、rollout 證據 |
| `status`／`expiry` | `APPROVED_FOR_STAGING_ONLY` | 確認決策狀態與有效期限 |

## 什麼情況會讓報告失效

需求、資料分類、保留期限、流量假設、Cloud Run／Storage 架構、IAM 邊界、environment topology、promotion policy、target、agent scope、policy version 或 evidence digest 任何一項改變，都必須重新產生所有角色視圖。舊版報告要標成 `STALE`，不能只更新其中一份。環境名稱可以由 Project owner 自訂，但 production 的最低保護不能以改名或重排繞過。

## 角色與權限分離

- 主管／PM 報告可以做業務決策，但不能直接改 Cloud Run 或批准 production deploy。
- 架構報告可以提出方案與成本模型，但不能把估算當成正式報價。
- FDE 報告可以向使用者索取缺少的輸入，但不能代替客戶決定 retention、region 或資安政策。
- Agent Context Pack 可以限制 agent 的修改與測試行為，但不能授予 merge、approval 或 production deploy 權限。
- Release Receipt 只描述證據是否存在；沒有證據時必須保持 `NOT_EXECUTED`、`REQUIRED_BUT_NOT_ATTACHED` 或 `BLOCKED`。

EnvironmentTopology 是治理 metadata，不是 Terraform、Cloud Run 或 Cloud Deploy 的 infrastructure／deployment source of truth；`Code Review`、`Candidate Review` 與 `Release Approval` 是 gate，不是環境。

完整的整合示例仍可參考 [Engineering Decision Pack Hero](./CONTEXT_RAIL_ENGINEERING_DECISION_PACK_HERO.zh-TW.md)。
