# Implementation Readiness Report

> **示例資料／非真實部署證據**

## 報告給誰

給 FDE、Implementation Consultant、PM、客戶窗口與需求提供者。這份報告的工作不是替使用者猜答案，而是把「還缺什麼、誰要提供、缺少後會阻擋什麼」說清楚。

## 識別

| 欄位 | 值 |
|---|---|
| `project_id`／版本 | `proj-internal-approval`／`3` |
| Project key | `APV` |
| `decision_id` | `CHG-2026-0017` |
| 目前決策版本 | `3` |
| 目前 staging 狀態 | `APPROVED_FOR_STAGING_ONLY` |
| production 狀態 | `BLOCKED` |
| 來源 | `sha256:example-source-snapshot-7f83...` |

## Project 環境資料

請先確認這個 Project 實際使用哪些環境、每個環境的負責人，以及從一個環境進入下一個環境需要哪些證據。示例目前採用 `Development → Testing → Staging → Production`；如果公司使用 `QA`、`UAT` 或其他名稱，請提供名稱與用途，不要只提供一個模糊的「測試環境」。

| 需要確認 | 為什麼需要 | 狀態 |
|---|---|---|
| 環境名稱與順序 | 決定變更下一步可以去哪裡 | 已提供示例，需由 Project owner 確認 |
| 每個環境的 target／owner | 避免把證據套到錯的服務 | 待確認 |
| 每個 promotion 的必要證據與批准人 | 決定何時阻擋、何時可以交接 | 待確認 |

## 需求摘要（白話版）

公司希望員工可以直接在內部簽核申請中附檔，讓正確的審核者可以查看；檔案不能公開給其他人。第一階段先在測試環境驗證，完成必要審查後才考慮正式上線。

## 如果資料還沒補齊，系統會怎麼做

| Gate | 結果 |
|---|---|
| 是否可以整理需求 | 可以 |
| 是否可以比較暫定方案 | 可以，但要標示假設 |
| 是否可以發出 Agent Work Order | 不可以，`BLOCKED` |
| 是否可以直接部署正式環境 | 不可以，`BLOCKED` |
| 是否可以自行補寫公司政策 | 不可以 |

## 需要補齊的資料

| 需要的資料 | 為什麼需要 | 應由誰提供 | 缺少時的影響 | 狀態 |
|---|---|---|---|---|
| 檔案保留幾天 | 決定刪除規則、儲存量與成本 | Compliance Owner | 無法確認 lifecycle 與正式成本 | 已確認：365 天 |
| 每月上傳／下載量 | 估算儲存、流量與服務負載 | Product Owner | 只能做部分成本估算 | 已確認：10,000／30,000 |
| 單檔大小與格式 | 判斷傳輸方式與測試範圍 | Product Owner | 方案選擇與驗收不完整 | 已確認：25 MB／五類格式 |
| 資料分類與地區 | 決定儲存地區與資安政策 | Security Owner | 無法完成資安 gate | 已確認：Internal Confidential／asia-east1 |
| 誰能查看附件 | 形成授權規則與測試案例 | Product Owner | 無法證明不會越權 | 待確認細節 |
| RTO／RPO | 決定備份與復原要求 | SRE／Service Owner | production gate 仍阻擋 | 未確認 |
| 病毒掃描／DLP 要求 | 確認是否需要額外服務與流程 | Security Owner | 可能造成新架構變更 | 未確認 |

## 提供資料時請使用這個格式

```text
retention_days: 365
monthly_uploads: 10000
monthly_downloads: 30000
max_file_size_mb: 25
data_classification: Internal Confidential
region_requirement: asia-east1
allowed_roles: applicant, reviewer
unrelated_user_behavior: deny
rto: <請提供>
rpo: <請提供>
virus_scan_required: <請提供>
```

請注意：`<請提供>` 不是建議值，也不是系統會自動填入的預設值。

## FDE／Implementation 下一步話術

可以這樣對需求方說：

> 我們已經可以先做測試環境驗證，但正式上線前還需要確認檔案保存多久、誰可以查看、發生故障時要恢復到什麼程度，以及是否需要病毒掃描或 DLP。這些資料會直接影響成本、資安審查與架構，所以我們不會先替公司猜答案。

## 版本規則

當需求方補上 RTO／RPO、授權細節或新增病毒掃描要求時，必須產生新的 `decision_version`，重新產生主管、架構、Agent 與 Release 報告。舊的 Work Order 不會因為資料補齊而自動恢復。

本報告的 Single Source of Truth 是 [Canonical Decision Record](./CANONICAL_DECISION_RECORD_HERO.example.json)。
