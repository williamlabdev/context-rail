# EB-001 人工審查包

這是 `Order Operations Portal` 第一個 Change Slice 的人工審查入口。Reviewer 不需要先逐份閱讀所有 JSON；先讀本頁，再依需要打開原始 evidence。

## Review target

- Project: `order-operations-portal`
- Request: `REQ-001` 訂單人工覆核
- Decision: `DR-001`
- Branch: `codex/001-manual-order-review`
- Baseline: `develop`
- Review commit: 執行 `git rev-parse HEAD`，並把完整 SHA 寫入 `code-review.md`
- Review independence: `multi_operator` 或高風險變更需要獨立人類 reviewer；本低風險 `single_operator` staging path 可使用明確標示的 AI review 與補償控制，但不能取代 production 的人類核准

## 一句話結論

這是一個使用合成資料的 stateless Go demo，讓 operations staff 對例外訂單作 `APPROVED`／`REJECTED` 人工決定並填寫理由。實作、測試與治理 gate 已補齊；目前仍不能宣稱 staging 或 production 已通過。

## 這次改了什麼

- API 嚴格拒絕空白／全形空白 note、trailing JSON、未知訂單與不支援的 decision。
- Review 成功時保存並顯示 `reviewedAt`。
- Browser UI 使用安全 DOM API 顯示 note，避免把訂單輸入插入 `innerHTML`；同時保留其他卡片的草稿。
- Context Pack 依 manifest 宣告的 source-of-truth 文件產生，並驗證每個 source hash、source coverage 與 aggregate snapshot hash。
- Readiness 將 request acceptance、testing evidence、staging deploy eligibility 與 post-deploy verification 分開。
- Staging script 依 operating mode 要求 review／commit／Context Pack lineage；本 demo 的低風險 single-operator path 另外要求 AI review、補償控制、不可變 image digest，並使用 tagged no-traffic revision 做 smoke。
- 新增 `AWO-001` 與 `ARR-001`；它們明確記錄目前 human gate pending、尚未宣稱 agent execution。

## 明確沒有做什麼

- 沒有加入 authentication、authorization、persistence、payment 或 fulfillment。
- 沒有修改 IAM、production target 或 Cloud Run credentials。
- 沒有執行 Cloud Run deployment，也沒有宣稱 container digest、revision 或 staging PASS。
- 沒有把 brownfield implementation 倒填成已存在的 Agent provenance。

## Acceptance Criteria 對照

| Criteria | Reviewer 要確認 | 目前 evidence／結果 |
| --- | --- | --- |
| AC-001 | 至少兩筆 synthetic `PENDING_REVIEW` orders | `main.go`、`main_test.go`、`test-output.txt` |
| AC-002 | approve／reject 都需要非空理由 | `main.go`、whitespace tests、`local-smoke.txt` |
| AC-003 | unknown order 回 `404` 且不改 state | `main_test.go`、`local-smoke.txt` |
| AC-004 | JSON 與 UI 都顯示結果及 review time | `main.go`、`web/index.html`、`main_test.go` |
| AC-005 | setup、test、build 可重現 | `docs/engineering/development.md`、`test-output.txt`、`vet-output.txt`、`build-output.txt` |
| AC-006 | staging 必須綁定 approved image digest | `check-staging-gate.py`、`deploy-staging.sh`；目前尚未部署，應保持 NEEDS_INPUT |

## Reviewer 必須回答的問題

- 實作是否只涵蓋 REQ-001 的 Change Slice，沒有偷偷加入 production capability？
- API 的 validation、state transition、trailing JSON 與 UI output 是否符合 spec？
- UI 是否不再將使用者輸入當作 HTML 解譯？
- Context Pack 的來源集合與 hash 是否真的能阻擋 stale／coverage drift？
- `ready_for_staging` 是否是部署前 gate，而 `staging_verified` 是否只代表部署後 evidence？
- staging script 是否會依 operating mode 阻擋 pending decision、錯誤 review identity、缺少補償控制、mutable image 與未綁定的 revision smoke？

## 建議驗證命令

在 repository root 執行：

```sh
./demo/order-operations-portal/scripts/validate-demo.sh
template/.venv/bin/python -m unittest discover -s tests -v
template/.venv/bin/python -m unittest discover -s template/tests -v
template/.venv/bin/python template/scripts/validate_project_context.py \
  --root demo/order-operations-portal --strict
template/.venv/bin/python scripts/context_rail_readiness.py \
  --root demo/order-operations-portal --output summary
```

預期：測試與 validation PASS；目前 readiness 應維持 `local=NEEDS_INPUT`、`ready_for_staging=NEEDS_INPUT`、`staging_verified=NEEDS_INPUT`、`production=BLOCKED`。這些阻擋是資料不足與 human gate，不是測試失敗。

## 深入 evidence

- [Request](../../requests/REQ-001-manual-order-review.md)
- [DecisionRecord](../../decisions/DR-001-manual-order-review.json)
- [Spec](../../specs/001-manual-order-review/spec.md)
- [Plan](../../specs/001-manual-order-review/plan.md)
- [Agent Work Order](../../work-orders/AWO-001-manual-order-review.json)
- [Agent Run Record](../../runs/ARR-001-manual-order-review.json)
- [Context Pack](../../docs/ai/context-pack.json)
- [Evidence index](README.md)
- [CTR／Fresh-eyes report](../../../../docs/reviews/DEMO_001_CTR_FRESH_EYES_2026-09-13.zh-TW.md)

## Review outcome

Reviewer should update `code-review.md` only after completing the checks above:

```text
Status: `PASS` or `REQUEST_CHANGES`
Reviewer: `<human identity>`
Review timestamp: `<UTC timestamp>`
Reviewed commit: `<full SHA from git rev-parse HEAD>`
Decision: `<ACCEPTED or CHANGES_REQUIRED>`
Notes: `<short evidence-backed conclusion>`
```

`PASS` means code review passed. It does not approve staging or production; those remain separate human decisions.
