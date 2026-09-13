# Technical Architecture & Cost Report

> **示例資料／非真實部署證據**

## 報告識別

| 欄位 | 值 |
|---|---|
| `project_id`／版本 | `proj-internal-approval`／`3` |
| Project key | `APV` |
| `decision_id` | `CHG-2026-0017` |
| `decision_version` | `3` |
| `source_snapshot_hash` | `sha256:example-source-snapshot-7f83...` |
| `policy_version` | `internal-cloudrun-v1` |
| 技術狀態 | `APPROVED_FOR_STAGING_ONLY` |

## Project Environment Topology

本 Project 的環境順序是 `Development → Testing → Staging → Production`。ContextRail 只保存 target、promotion policy、必要 evidence 與 config hash；Cloud Run／Cloud Deploy／Terraform 仍是實際資源與部署狀態的來源。

| 目標環境 | 允許的 promotion | 本次狀態 |
|---|---|---|
| Development | feature branch／本機檢查 | 已定義 |
| Testing | PR、CI、整合測試 | 已定義 |
| Staging | 已接受 digest、設定核對、smoke test | **目前目標** |
| Production | 需要 staging verified、分離職責與正式核准 | **Blocked** |

## 技術結論

既有 Go API 繼續部署在 Cloud Run，Firestore 儲存附件 metadata；檔案內容放入 private Cloud Storage。Cloud Run 先驗證使用者與申請單角色，再發出五分鐘有效的 signed URL，讓檔案直接傳輸，不經過 Cloud Run proxy。

這個方案只代表 staging candidate，不代表 production approval。

## 資料流

```text
申請人／審核者
       │  request attachment URL / view attachment
       ▼
Go API on Cloud Run
       │  驗證登入者與申請單角色
       ├── Firestore：metadata、owner、申請單關聯、audit event
       └── signed URL
               ▼
        Private Cloud Storage
        asia-east1／365 天 lifecycle
```

## 方案比較

| 方案 | 優點 | 風險 | 結論 |
|---|---|---|---|
| Cloud Run proxy | 授權與資料流集中 | 大檔案增加服務負載、timeout 與成本 | 作為 fallback |
| Signed URL 直傳 | Cloud Run 不承載檔案內容；可限制有效期 | 必須正確處理 URL、CORS、撤銷與 audit | **staging 選用** |
| Public bucket | 開發看似簡單 | 可能暴露 Internal Confidential 檔案 | **禁止** |

## 服務影響

| 服務 | 影響 | staging 行動 |
|---|---|---|
| Cloud Run | URL 產生、授權、metadata API | 使用既有 Go service |
| Firestore | 新增 attachment metadata 與 audit event | 更新 schema／index |
| Cloud Storage | private bucket、lifecycle、CORS | 建立 staging bucket |
| Cloud Build | build、test、provenance | 使用既有 pipeline |
| Artifact Registry | 儲存 immutable image | 以 digest 追蹤 |
| Cloud Deploy | staging rollout／approval | 只驗證 staging target |
| Cloud KMS | CMEK、key rotation、IAM | MVP 不引入，需另案決策 |

## 安全與可靠性約束

- bucket 禁止 public IAM binding。
- signed URL 有效期為 5 分鐘，不作為長期權限。
- API 必須先驗證申請單角色，再發出 URL。
- object path 使用不可猜測的 `request_id/attachment_id`。
- 365 天後由 lifecycle rule 刪除檔案。
- staging 與 production 使用不同 bucket、service account 與設定快照。
- production 不接受只有 tag、沒有 immutable digest 的 image。
- 病毒掃描、DLP、CMEK 若成為政策要求，必須建立新的 decision version。

## 成本模型

本報告不把估算寫成虛假的精準報價。真正成本至少受以下因素影響：

| 驅動因子 | 目前示例輸入 | 仍需驗證 |
|---|---:|---|
| 每月上傳 | 10,000 次 | 平均檔案大小與實際分布 |
| 每月下載 | 30,000 次 | 使用者地區與 egress |
| 保留期限 | 365 天 | 實際累積儲存量 |
| Cloud Run | 產生 URL 與 metadata | staging request、CPU、memory |
| Firestore | metadata 讀寫 | query pattern、index 數量 |
| Build／Deploy | 每次 PR 與 rollout | 現有 pipeline 用量 |

**成本狀態：`PARTIAL`。** 正式上線前必須用 staging 的實際觀測資料與當期 Google Cloud pricing calculator 重新估算。

## Staging 驗收條件

| 類別 | 驗收 |
|---|---|
| 授權 | 申請人可上傳；審核者可查看；無關人員收到 403 |
| 安全 | public bucket、過長 URL TTL、未授權路徑均被 policy test 拒絕 |
| 保留 | 365 天 lifecycle 規則存在且可由設定快照證明 |
| 可觀測性 | request、attachment、actor、decision_id 可串成 audit trail |
| 發佈 | image digest、build provenance、staging config hash 可對應 |

## 來源一致性

本報告的事實來源是 [Canonical Decision Record](./CANONICAL_DECISION_RECORD_HERO.example.json)。如果架構方案、region、資料分類或成本假設改變，技術報告、主管報告、FDE 報告與 Agent Context Pack 必須同步失效並重新產生。
