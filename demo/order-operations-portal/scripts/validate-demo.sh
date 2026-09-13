#!/usr/bin/env bash
set -euo pipefail

demo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$demo_root"

required=(project.yaml README.md vision.md architecture.md AGENTS.md docs/engineering/development.md docs/operations/environments.md docs/governance/policies.md docs/ai/context-pack.json requests/REQ-001-manual-order-review.md decisions/DR-001-manual-order-review.json)
for path in "${required[@]}"; do
  test -f "$path" || { echo "MISSING $path" >&2; exit 1; }
done

go test ./...
go build -o "${TMPDIR:-/tmp}/order-operations-portal-validate" ./...
echo "VALIDATION PASS: Project Context Contract files, tests and build are present."
