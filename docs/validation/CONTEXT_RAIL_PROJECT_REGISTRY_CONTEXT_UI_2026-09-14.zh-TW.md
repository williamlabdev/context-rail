# VS-002 Project Registry／Documents & AI Context 驗證

- Checked at: 2026-09-14 Asia/Taipei
- Source checkpoint: `25204a5`
- Scope: local read-only Go API、React workspace、browser journeys
- Python reference: `template/.venv/bin/python scripts/context_rail_registry.py`

## Result

Go 與 Python reference 對兩個明確 fixture roots 的 normalized snapshot 相等；比較時排除每次執行都會變動的 `observed_at`。完整結果見 [EB-003 oracle-comparison](../../evidence/EB-003/oracle-comparison.txt)。

## Boundary

- comparison 只涵蓋 checked-in local fixtures；不涵蓋 remote repository、provider crawling、Cloud Run、IAM 或 production。
- Go contract 另外測試 manifest 明確宣告的 `STALE`、`MISSING`、`CONFLICT`、`UNKNOWN`、`UNDECLARED`，確認不會被提升為 `CURRENT`。
- Python reference 只作為 VS-001 compatibility oracle；VS-002 runtime authority 是 Go contract/API。
- `observed_at` 是 observation metadata，不是 semantic equality 的一部分。
