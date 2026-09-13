# Optional Spec Kit integration

這是 Project Context Contract 的 optional Spec Kit integration。它把 Spec-Driven Development 的規格、澄清、技術計畫與 tasks 接到 ContextRail 的 Request、Change Slice、DecisionRecord、Agent execution 與 Evidence 流程，但不把 Spec Kit 設為 template 的必要依賴。

## Boundary

- Template 的 Python validator、Context Pack rebuild 與 tests 仍使用 Project 自己的 `.venv`。
- Spec Kit CLI 是獨立的 optional developer tool；consumer 可以依照自己的 agent integration 安裝並 pin 版本。
- Spec Kit 產生的 `spec.md`、`plan.md` 與 `tasks.md` 是工作產物，不是 ContextRail 的治理 SSOT。
- ContextRail 的 `DecisionRecord`、policy、human gate、EvidenceBundle、environment gate 與 deployment receipt 仍是治理邊界。
- 小型修正可以沿用一般 issue／PR／test 流程，不必強制走完整 Spec Kit workflow。

## Optional setup

官方 CLI 安裝方式與 agent integration 請以 [Spec Kit 官方文件](https://github.com/github/spec-kit) 為準。安裝時應 pin 一個已驗證的 release version，並將 CLI 安裝視為開發者工具，不要加入 Project runtime dependencies。

在 Project root 啟用後，可依序使用對應的 Spec Kit commands：

```text
constitution → specify → clarify → plan → tasks → implement → converge
```

不同 agent 可能使用 `/speckit.*` 或 `$speckit-*` 命令名稱；以實際 integration 提供的形式為準。

## Pilot

第一個 pilot 使用 `demo/order-operations-portal` 的「訂單人工覆核」Change Slice。執行前先閱讀 [artifact mapping](artifact-mapping.md)，並確認任何生成的 spec／plan／tasks 都回寫到正確的 Request 與 DecisionRecord references。
