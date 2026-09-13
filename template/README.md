# ContextRail Project Context Template

這是一個可複製的 Project template repository，用來讓人、RAG、coding agent 與 release gate 使用同一份可驗證、可追溯的 Project Context。

它不是應用程式 starter，也不是要求所有 AI 專案遵循的業界標準。建立新 Project 時，請從本 repo 建立 GitHub repository，再替換所有 `<REPLACE_ME>` 內容。

## 使用方式

1. 在 GitHub 對這個 repository 選擇 **Use this template**，建立一個新的 Project repository。
2. 將 `project.yaml` 的 identity、owner、private repository、service 與 environments 填完整。
3. 填寫 `vision.md`、`architecture.md`、`docs/engineering/development.md`、`docs/operations/environments.md` 與 `AGENTS.md`。
4. 建立 Python virtual environment 並安裝 `requirements.txt`：`python3 -m venv .venv && .venv/bin/python -m pip install -r requirements.txt`。
5. 使用 venv 執行 `./.venv/bin/python scripts/validate_project_context.py --strict`。
6. 使用 venv 執行 `./.venv/bin/python scripts/rebuild_context_pack.py`，檢查 `docs/ai/context-pack.json` 的來源 hash。
7. 使用 venv 執行 `./.venv/bin/python -m unittest discover -s tests`，驗證 validator 與 Context Pack rebuild。

Windows 對應執行檔為 `.venv\\Scripts\\python.exe`；不要依賴系統 Python 或全域安裝的 PyYAML。
6. 從 `requests/REQ-000-template.md` 複製第一張 Request，讓 DecisionRecord、Agent Work Order、Engineering Evidence 與 Release Receipt 都沿用同一個 `project_id`／`request_id`。

## Template 內含內容

| 區域 | 用途 |
| --- | --- |
| `project.yaml` | Project identity、repository、service、environment 與文件 registry 的 SSOT |
| `README.md`、`vision.md`、`architecture.md` | 人類入口、產品意圖與架構基線 |
| `docs/engineering/development.md` | 本機開發、測試、build 與 debug contract |
| `docs/operations/environments.md` | environment topology、promotion 與 production gate |
| `AGENTS.md` | Agent scope、必要驗證、禁止動作與 escalation |
| `docs/ai/context-pack.json` | 可重建的 derived context，不是 SSOT |
| `requests/`、`decisions/`、`evidence/` | Request、DecisionRecord、雙版本摘要與 Evidence Bundle 的模板 |
| `integrations/spec-kit/` | Optional Spec Kit workflow 與 ContextRail artifact mapping |
| `scripts/` | Project Context validation 與 Context Pack rebuild |
| `SECURITY.md`、`.editorconfig`、`CONTRIBUTING.md` | 安全回報、格式與協作基線 |

## Governance boundary

這個 template 只建立脈絡與證據的文件邊界，不會自動核准需求、執行 coding agent、修改 IAM 或部署 production。缺少文件、來源、review、build、target 或人類核准時，應保持 `NEEDS_INPUT`、`UNKNOWN`、`STALE` 或 `BLOCKED`。

Spec Kit integration 是 optional。它可以協助產生 specification、plan 與 tasks，但不取代 ContextRail 的 DecisionRecord、policy、EvidenceBundle、environment gate 或 human approval。

## Source of truth

Markdown、`project.yaml`、Request、DecisionRecord 與外部 Git/CI/Cloud evidence 是來源。`docs/ai/context-pack.json` 是可刪除後重建的 derived projection；它必須保存 source path、version、content hash、effective time 與 access scope，不能把缺件補成事實。
