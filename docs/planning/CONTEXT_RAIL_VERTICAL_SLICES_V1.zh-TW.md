# ContextRail Vertical Slice Map v1

Status: working baseline
Version: 1.0
Updated: 2026-09-14
Scope: ContextRail P0 core product

## 1. 定位

Vertical Slice 是一條可以被獨立提出、決策、實作、驗證與 review 的最小產品路徑。它不是單一技術模組，也不是把 roadmap 的名詞直接拆成 backlog；每個 slice 都必須能說明：

`使用者結果 → Request → Decision → Agent 執行範圍 → Evidence → Review → 下一個 environment gate`

Spec Kit 用來把已選定的 slice 形成 `spec.md`、`plan.md` 與 `tasks.md`。這些文件協助執行，但不取代 ContextRail 的 Request、DecisionRecord、human gate、EvidenceBundle 或 release receipt。

目前採用「先定義 slice，再用 Spec Kit 產出規格」的順序。尚未有實際產品 API 的 slice，可以先以 contract spike 或 fixture evidence 表示，不得宣稱已完成 runtime 能力。

## 2. P0 slice map

| Slice | 使用者結果 | 主要治理鏈 | 目前狀態 | 下一個明確出口 |
| --- | --- | --- | --- | --- |
| VS-001 Project Registry read-only import | 看見 Project、repo、service、environment、文件、context 與 readiness 的可追溯 snapshot | Request → Decision → Spec Kit → read-only importer → contract evidence | Contract spike PASS；core API 未實作 | 完成 `REQ-002` 的 core read-only API decision |
| VS-002 Documents / AI Context | 找到缺件、過期、衝突與衍生 Context Pack 的來源 | Request → Decision → Spec Kit → document registry/context view → evidence | NOT_STARTED | 定義 UI state/action contract |
| VS-003 Project lifecycle and topology | 建立、修改、封存 Project，並以 immutable topology version 管理 environments | Request → Decision → Spec Kit → API/UI → audit evidence | NOT_STARTED | 先完成 VS-001 的資料讀取契約 |
| VS-004 Change Decision Pack | 一份 accepted DecisionRecord 同時產生人類摘要與 Agent Context Pack | Request → architecture impact → Decision → Spec Kit → render evidence | Demo proven；core NOT_STARTED | 將 demo artifact mapping 移植到 core contract |
| VS-005 Agent handoff and candidate review | 受限 Work Order 產生 feature branch/PR，並由 evidence/review gate 判斷候選 | Decision → AWO → Agent Run → Evidence → Review | Demo proven；core NOT_STARTED | 定義外部 agent adapter 的最小 read-back contract |
| VS-006 Staging promotion and receipt | 只把通過 gate 的 commit/digest/config 推到 staging 並產生 receipt | Candidate review → environment gate → deploy → smoke → receipt | Demo pre-deploy gate proven；real runtime NOT_STARTED | 先完成 VS-003 的 target/config contract |

## 3. 本輪選擇：VS-001

### 使用者結果

Solution Architect 或 Project owner 可以用一個唯讀操作取得 Project Registry snapshot，知道：

- Project identity、status、classification 與 owner；
- 關聯 repository、service 與 environment；
- required documents 的狀態、來源與 readiness；
- Context Pack 是 `CURRENT`、`PARTIAL`、`STALE` 或 `MISSING`；
- 未宣告 runtime、文件 drift 或其他未知項仍以 `UNDECLARED`、`STALE`、`UNKNOWN` 呈現。

### Included

- 讀取一個或多個既有 Project root 的 `project.yaml`；
- 支援 singular `service` 與 plural `services` 的 consumer manifest 形狀；
- 正規化 Project、Repository、Service、Environment、ProjectDocument、Decision 與 readiness；
- 輸出 versioned `ProjectRegistrySnapshot`；
- 保持 read-only，不修改來源 Project、Context Pack 或治理 artifact；
- 以 `demo/order-operations-portal` 與 `examples/support-insights` 驗證不同 runtime 與 stale/undeclared 狀態。

### Excluded

- Project CRUD、archive、membership 或細粒度 authorization；
- Firestore、Cloud Storage、GitHub connector 或任意 URL crawling；
- Context Pack rebuild、RAG、Gemini、AI recommendation；
- Cloud Run、IAM、deployment 或 production promotion；
- 從觀察結果自動建立 ProjectRepository、ProjectService 或任何 human decision。

### Gate boundary

VS-001 的 PASS 只表示「可以安全地讀取與正規化契約 fixture」。它不表示 Project 已經能被建立、不表示 source 已確認、不表示 staging/production ready，也不授予任何 agent 或人員寫入權限。

## 4. 選擇理由與裁切

VS-001 是後續所有 slice 的共同讀取邊界：沒有穩定的 Project、source、document 與 readiness contract，VS-002 到 VS-006 會把資料對應重新寫在各自功能裡，最後產生多份不一致的 context。先完成唯讀 contract 也能在沒有 Cloud Run、Firestore 或付費 connector 的情況下驗證 Project Context Contract。

本輪不把既有 Python importer 直接升格為產品 runtime。它是 contract oracle／spike；`REQ-002` 的 plan 會決定 Go modular monolith 如何消費同一份 normalized contract，並保留 read-back 與 evidence boundary。

## 5. 對應 roadmap 與 architecture

- Roadmap P0-A：Project Registry、Repository/ProjectRepository、Service、ProjectDocument 與 Context Pack provenance。
- Roadmap Week 1：Go API skeleton、Project Registry、manifest validation 與 source contract。
- Architecture domain：Project 是治理邊界；Repository/Service 是可多對多關聯；ContextPack 是 derived projection。
- Project Context Contract：import → validate → rebuild 是明確動作；readiness 必須保留 missing/stale/conflict/derived 狀態。

## 6. 完成定義

VS-001 只有同時滿足以下條件才可標記為 core slice complete：

1. `REQ-002`、`DR-002`、Spec Kit `spec.md`、`plan.md`、`tasks.md` 可互相回指；
2. read-only API 或 CLI 的輸出契約已固定，且與 Python contract oracle 對照；
3. demo 與 support-insights 都能被讀取，stale/undeclared 不被抹平；
4. 測試、build、diff 與 source identity 有 EvidenceBundle；
5. 人類 review 完成前，不能建立 staging deploy work order，也不能宣稱 production readiness。
