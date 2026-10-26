#!/usr/bin/env bash
# EB-019 / G3: one headless Claude Code run (DR-018 v2) changes the external
# Project github.com/williamlabdev/order-operations-portal under a ContextRail
# Work Order. The harness, not the agent, commits, pushes and opens the pull
# request; a second, read-only claude -p run on another model reviews it; the
# candidate is read back from GitHub and the reviewed head commit is promoted to
# a local staging instance (DR-017). Run from the repo root.
#
#   scripts/agent-run.sh              # run 1: branch agent/order-status-filter
#   RUN_N=2 scripts/agent-run.sh      # the one retry DR-018 allows: branch agent/order-status-filter-2
#
# Spends API credit: author claude-fable-5-1 capped at US$20, reviewer
# claude-opus-5-5 capped at US$5. The key is read from KEY_FILE into a shell
# variable that is never exported; it is placed on the two claude command lines
# only, and every transcript is checked for it before it is kept.
# Ports (override with env): RAIL_PORT=8094 STAGING_PORT=8095
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
RUN_N="${RUN_N:-1}"
RUN_ID="run-eb019-$RUN_N"
RAIL_PORT="${RAIL_PORT:-8094}"
STAGING_PORT="${STAGING_PORT:-8095}"
REPO_SLUG="williamlabdev/order-operations-portal"
REPO_URL="https://github.com/$REPO_SLUG"
TARGET_BRANCH="agent/order-status-filter"; [ "$RUN_N" = 1 ] || TARGET_BRANCH="$TARGET_BRANCH-$RUN_N"
AUTHOR_MODEL="claude-fable-5-1"; AUTHOR_BUDGET=20
REVIEW_MODEL="claude-opus-5-5"; REVIEW_BUDGET=5
# --bare exposes only Bash, Edit and Read (init event of a probe run, 2026-10-26), so Write, Glob and Grep are not listed.
AUTHOR_TOOLS="Read,Edit,Bash(go test:*),Bash(go vet:*),Bash(gofmt:*)"
REVIEW_TOOLS="Read"
# Keep Read away from where secrets live; the key file and gh credentials are outside the clone anyway.
DENY_TOOLS="Read(//$HOME/dev/source/core/**),Read(//$HOME/.config/**),Read(//$HOME/.ssh/**),Read(//$HOME/.claude/**)"
KEY_FILE="${KEY_FILE:-$HOME/dev/source/core/aine/orvena/.env}"
EB="evidence/EB-019"
OUT="$EB/agent-run-$RUN_N.txt"
WORK="$(cd "$(mktemp -d "${TMPDIR:-/tmp}/eb019-work.XXXXXX")" && pwd -P)"
PROJECT="order-operations-portal"
BASE="http://127.0.0.1:${RAIL_PORT}/v1/projects/${PROJECT}"
ACTOR="founder-001"
PIDS=()

mkdir -p "$EB" bin
: > "$OUT"
log() { printf '%s\n' "$*" | sed "s#$WORK#<scratch>#g" | tee -a "$OUT"; }
pretty() { python3 -I -c 'import json,sys; print(json.dumps(json.load(sys.stdin), indent=2, ensure_ascii=False))'; }
jget() { python3 -I -c "import json,sys; d=json.load(sys.stdin); print($1)"; }
die() { log "FATAL: $*"; exit 1; }
cleanup() {
  log ""
  log "## teardown"
  for pid in "${PIDS[@]:-}"; do [ -n "$pid" ] && kill "$pid" 2>/dev/null || true; done
  sleep 1
  for port in "$RAIL_PORT" "$STAGING_PORT"; do
    if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then log "port $port: STILL LISTENING"; else log "port $port: no listener"; fi
  done
  rm -rf "$WORK"
  log "work dir (clones, binaries, ContextRail state) removed: $([ -e "$WORK" ] && echo NO || echo yes)"
}
trap cleanup EXIT

# call METHOD PATH [JSON] -> logs status + pretty body, keeps body in $BODY
call() {
  local method="$1" path="$2" data="${3:-}" tmp
  tmp="$(mktemp)"
  if [ -n "$data" ]; then
    STATUS="$(curl -sS -o "$tmp" -w '%{http_code}' -X "$method" -H 'Content-Type: application/json' -H "X-ContextRail-Actor: $ACTOR" --data "$data" "$path" || echo 000)"
  else
    STATUS="$(curl -sS -o "$tmp" -w '%{http_code}' -X "$method" -H "X-ContextRail-Actor: $ACTOR" "$path" || echo 000)"
  fi
  BODY="$(cat "$tmp")"; rm -f "$tmp"
  log ""
  log "### $method ${path#http://127.0.0.1:${RAIL_PORT}}"
  [ -n "$data" ] && { log "request_body:"; printf '%s' "$data" | pretty | sed "s#$WORK#<scratch>#g" | tee -a "$OUT" >/dev/null; }
  log "http=$STATUS"
  printf '%s' "$BODY" | pretty 2>/dev/null | sed "s#$WORK#<scratch>#g" | tee -a "$OUT" >/dev/null || log "$BODY"
  case "$STATUS" in 2*) ;; *) [ "${ALLOW_FAIL:-0}" = 1 ] || die "unexpected http=$STATUS" ;; esac
}

log "# EB-019 real run $RUN_N: headless Claude Code under a ContextRail Work Order -> GitHub -> local staging (DR-018 v2, G3)"
log "captured_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
log "context-rail commit=$(git rev-parse HEAD)"
log "tracked changes not in that commit: $(git status --porcelain --untracked-files=no | tr '\n' ' ' | sed 's/ *$//')"
log "ports: context-rail=$RAIL_PORT staging=$STAGING_PORT"
log "author: claude -p --bare --model $AUTHOR_MODEL --max-budget-usd $AUTHOR_BUDGET --allowedTools \"$AUTHOR_TOOLS\""
log "reviewer: claude -p --bare --model $REVIEW_MODEL --max-budget-usd $REVIEW_BUDGET --allowedTools \"$REVIEW_TOOLS\""
log "both: --disallowedTools \"$DENY_TOOLS\"" 
log "claude version: $(claude --version 2>/dev/null | head -1)"

# The key must not already be in the environment: that would let every child
# process, not just the two claude runs, see it.
[ -z "${ANTHROPIC_API_KEY:-}" ] || die "ANTHROPIC_API_KEY is exported in the invoking shell; unset it and rerun"
[ -r "$KEY_FILE" ] || die "key file not readable"
KEY="$(sed -n 's/^ANTHROPIC_API_KEY=//p' "$KEY_FILE" | head -1 | sed 's/[[:space:]]#.*$//' | tr -d "\"'\r[:space:]")"
[ -n "$KEY" ] || die "no ANTHROPIC_API_KEY line in the key file"
log "env_check: ANTHROPIC_API_KEY exported in the invoking shell? NO; key read from the local .env (not printed)"
GH_TOKEN_VALUE="$(gh auth token 2>/dev/null || true)"
[ -n "$GH_TOKEN_VALUE" ] || die "gh is not authenticated"
# scrub FILE: refuse to keep anything that contains a secret.
scrub() {
  if grep -qF "$KEY" "$1" || grep -qF "$GH_TOKEN_VALUE" "$1"; then rm -f "$1"; die "a secret appeared in $(basename "$1"); file deleted, run aborted"; fi
}

log ""
log "## clone the Project (the external repository is the single source, DR-018)"
git clone -q "$REPO_URL" "$WORK/root"
BASE_COMMIT="$(git -C "$WORK/root" rev-parse HEAD)"
log "\$ git clone $REPO_URL   -> main @ $BASE_COMMIT ($(git -C "$WORK/root" log -1 --format=%s))"
grep -q 'build_digest' "$WORK/root/main.go" || die "external main has no healthz identity; merge order-operations-portal PR #2 first (DR-018 precondition)"
grep -q 'local/order-operations-portal-staging' "$WORK/root/project.yaml" || die "external project.yaml does not declare local staging (DR-018 precondition)"
if git ls-remote --exit-code --heads "$REPO_URL" "$TARGET_BRANCH" >/dev/null 2>&1; then die "branch $TARGET_BRANCH already exists on GitHub; use RUN_N for a new run"; fi
git clone -q "$WORK/root" "$WORK/agent"
git -C "$WORK/agent" remote set-url origin "$REPO_URL"
git -C "$WORK/agent" checkout -q -b "$TARGET_BRANCH"
log "agent clone on $TARGET_BRANCH from $BASE_COMMIT; ContextRail reads the untouched clone"

log ""
log "## start ContextRail (rule advisor, GitHub read-back, the Project root is the external clone)"
go build -o bin/context-rail ./cmd/context-rail
log "\$ GITHUB_TOKEN=<gh auth token> CONTEXT_RAIL_ADVISOR=rule PORT=$RAIL_PORT CONTEXT_RAIL_FIXTURE_ROOTS=<scratch>/root,examples/support-insights CONTEXT_RAIL_STATIC_DIR=frontend/dist CONTEXT_RAIL_STATE_DIR=<scratch>/rail ./bin/context-rail"
GITHUB_TOKEN="$GH_TOKEN_VALUE" CONTEXT_RAIL_ADVISOR=rule PORT="$RAIL_PORT" CONTEXT_RAIL_FIXTURE_ROOTS="$WORK/root,examples/support-insights" \
  CONTEXT_RAIL_STATIC_DIR=frontend/dist CONTEXT_RAIL_STATE_DIR="$WORK/rail" ./bin/context-rail >"$WORK/rail.log" 2>&1 & PIDS+=($!)
for i in $(seq 1 50); do curl -sf "http://127.0.0.1:${RAIL_PORT}/healthz" >/dev/null 2>&1 && break; sleep 0.2; done
log "startup line: $(grep -m1 'listening' "$WORK/rail.log")"
grep -q 'read_back=github' "$WORK/rail.log" || die "GitHub read-back is not enabled"

log ""
log "## change -> decision -> Work Order"
call POST "$BASE/changes" "$(cat <<JSON
{"actor":"$ACTOR","reason":"EB-019: the change the headless agent run makes",
 "request":{"title":"Filter the order list by status","objective":"Let operations staff list only the orders in one status, so the review queue (PENDING_REVIEW) can be read without scanning reviewed orders.",
 "owner_summary":"GET /api/orders accepts an optional status query parameter; without it the endpoint behaves as before.",
 "scope_included":["status query parameter on GET /api/orders","validation of the status value","unit tests"],"scope_excluded":["authentication","order review behaviour","web UI","new dependencies","production environment"],
 "acceptance_criteria":[{"text":"GET /api/orders?status=<status> returns only the orders whose status equals <status>"},{"text":"GET /api/orders without a status parameter returns every order, as before"},{"text":"a status value that is not PENDING_REVIEW, APPROVED or REJECTED is answered with 400 and a JSON error body"},{"text":"main_test.go covers the three cases and go test ./... passes"}],
 "allowed_paths":["main.go","main_test.go"],"forbidden_actions":["authentication","production deployment","changing the order review endpoint","adding dependencies"],
 "target_environment_id":"staging","business_constraints":{"data_classification":"internal","expected_monthly_volume":"500 reviews"},"requested_by":"$ACTOR"}}
JSON
)"
printf '%s' "$BODY" > "$WORK/change.json"
OPTION="$(printf '%s' "$BODY" | jget 'd["change"]["versions"][-1]["options"][0]["id"]')"
log "selected option: $OPTION"
call POST "$BASE/changes/CHG-001/decision" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-019\",\"decision\":\"ACCEPT\",\"role\":\"founder\",\"selected_option\":\"$OPTION\",\"rationale\":\"Small read-only API addition; the agent run is the point of the exercise.\",\"risk_level\":\"low\"}"
call POST "$BASE/changes/CHG-001/work-order" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-019\",\"target_branch\":\"$TARGET_BRANCH\"}"
printf '%s' "$BODY" > "$WORK/work-order.json"
WO_ID="$(printf '%s' "$BODY" | jget 'd["work_order"]["work_order_id"]')"
WO_HASH="$(printf '%s' "$BODY" | jget 'd["work_order"]["work_order_hash"]')"
[ "$(printf '%s' "$BODY" | jget 'd["work_order"]["target_branch"]')" = "$TARGET_BRANCH" ] || die "work order target branch differs"
[ "$(printf '%s' "$BODY" | jget 'd["work_order"]["repository"]')" = "github.com/$REPO_SLUG" ] || die "work order repository is not github.com/$REPO_SLUG"

# render TEMPLATE OUTFILE [key=file ...]: fills {placeholders} from the change and the Work Order.
render() {
  python3 -I - "$WORK/change.json" "$WORK/work-order.json" "$@" <<'PY'
import json, sys
change, order, template, out = sys.argv[1:5]
extra = dict(a.split("=", 1) for a in sys.argv[5:])
c = json.load(open(change))["change"]["versions"][-1]["request"]
w = json.load(open(order))["work_order"]
values = {
    "repository": w["repository"], "base_branch": w["base_branch"], "target_branch": w["target_branch"],
    "work_order_id": w["work_order_id"], "work_order_hash": w["work_order_hash"],
    "title": c["title"], "objective": c["objective"],
    "acceptance": "\n".join("- " + a["text"] for a in c["acceptance_criteria"]),
    "scope_included": ", ".join(c["scope_included"]), "scope_excluded": ", ".join(c["scope_excluded"]),
    "allowed_paths": ", ".join(w["allowed_paths"]), "forbidden_actions": ", ".join(w["forbidden_actions"]),
}
for key, path in extra.items():
    values[key] = open(path).read().rstrip("\n") if key == "diff" else path
text = open(template).read()
for key, value in values.items():
    text = text.replace("{" + key + "}", value)
open(out, "w").write(text)
PY
}
# summarize TRANSCRIPT: result line of a stream-json transcript plus the tools the agent called.
summarize() {
  python3 -I - "$1" <<'PY'
import json, sys
result, tools = {}, []
for line in open(sys.argv[1]):
    try: event = json.loads(line)
    except ValueError: continue
    if event.get("type") == "result": result = event
    if event.get("type") == "assistant":
        for block in event.get("message", {}).get("content", []):
            if block.get("type") == "tool_use":
                i = block.get("input", {})
                tools.append(block["name"] + ": " + str(i.get("command") or i.get("file_path") or i.get("pattern") or ""))
print("is_error=%s subtype=%s turns=%s duration_ms=%s total_cost_usd=%s" % (result.get("is_error"), result.get("subtype"), result.get("num_turns"), result.get("duration_ms"), result.get("total_cost_usd")))
print("models=%s" % ",".join(sorted((result.get("modelUsage") or {}).keys())))
for t in tools: print("tool " + t)
PY
}

log ""
log "## author run (headless Claude Code, $AUTHOR_MODEL)"
render scripts/agent-run-prompt.md "$EB/author-prompt-$RUN_N.md"
scrub "$EB/author-prompt-$RUN_N.md"
log "prompt rendered from $WO_ID: $EB/author-prompt-$RUN_N.md"
AUTHOR_STARTED="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
AUTHOR_EXIT=0
(cd "$WORK/agent" && ANTHROPIC_API_KEY="$KEY" claude -p --bare --no-session-persistence --model "$AUTHOR_MODEL" \
  --max-budget-usd "$AUTHOR_BUDGET" --allowedTools "$AUTHOR_TOOLS" --disallowedTools "$DENY_TOOLS" --output-format stream-json --verbose \
  < "$ROOT/$EB/author-prompt-$RUN_N.md") > "$WORK/author.jsonl" 2> "$WORK/author.err" || AUTHOR_EXIT=$?
AUTHOR_FINISHED="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
scrub "$WORK/author.jsonl"; scrub "$WORK/author.err"
sed "s#$WORK#<scratch>#g" "$WORK/author.jsonl" > "$EB/author-transcript-$RUN_N.jsonl"
log "exit=$AUTHOR_EXIT started=$AUTHOR_STARTED finished=$AUTHOR_FINISHED transcript=$EB/author-transcript-$RUN_N.jsonl (checked: no key, no GitHub token)"
[ -s "$WORK/author.err" ] && log "stderr: $(head -c 600 "$WORK/author.err")"
summarize "$WORK/author.jsonl" | tee "$WORK/author-summary.txt" | while IFS= read -r line; do log "$line"; done
[ "$AUTHOR_EXIT" = 0 ] || die "author run failed"
grep -q "^is_error=False subtype=success" "$WORK/author-summary.txt" || die "author run did not end in success"

log ""
log "## scope check before anything is committed (Work Order allowed_paths: main.go, main_test.go)"
CHANGED="$(git -C "$WORK/agent" status --porcelain --untracked-files=all | awk '{print $2}' | sort)"
log "changed paths: $(printf '%s' "$CHANGED" | tr '\n' ' ')"
[ -n "$CHANGED" ] || die "the agent changed nothing"
for path in $CHANGED; do case "$path" in main.go|main_test.go) ;; *) die "out-of-scope edit: $path (nothing committed or pushed)" ;; esac; done
log "scope ok"
git -C "$WORK/agent" diff > "$WORK/agent.diff"; scrub "$WORK/agent.diff"
log "diff checked: no key, no GitHub token"
log "\$ git diff --stat"; git -C "$WORK/agent" diff --stat | while IFS= read -r line; do log "  $line"; done

log ""
log "## harness re-runs the checks itself (not the agent's word)"
log "\$ gofmt -l main.go main_test.go -> '$(cd "$WORK/agent" && gofmt -l main.go main_test.go)'"
[ -z "$(cd "$WORK/agent" && gofmt -l main.go main_test.go)" ] || die "files are not gofmt-clean"
(cd "$WORK/agent" && go vet ./... && go test -count=1 -v ./... && scripts/validate-demo.sh) > "$WORK/checks.txt" 2>&1 || { log "$(cat "$WORK/checks.txt")"; die "local checks failed"; }
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|PASS|FAIL|VALIDATION)' "$WORK/checks.txt" | while IFS= read -r line; do log "  $line"; done

log ""
log "## harness commits, pushes and opens the pull request"
git -C "$WORK/agent" add main.go main_test.go
git -C "$WORK/agent" commit -q -m "Filter the order list by status" -m "Made by a headless Claude Code run ($AUTHOR_MODEL) under ContextRail Work Order $WO_ID; committed by the harness (context-rail scripts/agent-run.sh)." \
  -m "Agent-Run: $RUN_ID
Work-Order: $WO_ID $WO_HASH"
HEAD_COMMIT="$(git -C "$WORK/agent" rev-parse HEAD)"
log "head commit: $HEAD_COMMIT"
GIT_TERMINAL_PROMPT=0 git -C "$WORK/agent" -c credential.helper= -c credential.helper='!gh auth git-credential' push -q origin "$TARGET_BRANCH" > "$WORK/push.txt" 2>&1 || { log "$(cat "$WORK/push.txt")"; die "push failed"; }
log "pushed $TARGET_BRANCH"
PR_URL="$(gh pr create --repo "$REPO_SLUG" --base main --head "$TARGET_BRANCH" --title "Filter the order list by status" \
  --body "$(printf 'Made by a headless Claude Code run (`%s`, run `%s`) under ContextRail Work Order `%s` (`%s`).\nThe harness, not the agent, committed and opened this pull request; the agent could edit only main.go and main_test.go and run go test, go vet and gofmt.\n\nEvidence: context-rail `evidence/EB-019/`. Not merged by the run.' "$AUTHOR_MODEL" "$RUN_ID" "$WO_ID" "$WO_HASH")")"
PR_NUM="${PR_URL##*/}"
log "pull request: $PR_URL"

log ""
log "## external CI on the pull request"
# CI can take a while to register; --watch exits at once on "no checks reported".
for i in $(seq 1 36); do
  pending="$(gh pr checks "$PR_NUM" --repo "$REPO_SLUG" 2>&1 || true)"
  case "$pending" in *"no checks reported"*) sleep 5 ;; *) break ;; esac
done
gh pr checks "$PR_NUM" --repo "$REPO_SLUG" --watch --interval 15 > "$WORK/ci.txt" 2>&1 || true
CI_FINAL="$(gh pr checks "$PR_NUM" --repo "$REPO_SLUG" 2>&1 || true)"
log "$CI_FINAL"
printf '%s\n' "$CI_FINAL" | grep -qE '^test[[:space:]]+pass' || die "CI test did not pass"
printf '%s\n' "$CI_FINAL" | grep -qE '^build[[:space:]]+pass' || die "CI build did not pass"

log ""
log "## independent AI review (separate read-only run, $REVIEW_MODEL)"
gh pr diff "$PR_NUM" --repo "$REPO_SLUG" > "$WORK/pr.diff"
scrub "$WORK/pr.diff"
render scripts/agent-review-prompt.md "$EB/review-prompt-$RUN_N.md" "diff=$WORK/pr.diff" "head_commit=$HEAD_COMMIT" "pr_url=$PR_URL"
scrub "$EB/review-prompt-$RUN_N.md"
REVIEW_EXIT=0
(cd "$WORK/agent" && ANTHROPIC_API_KEY="$KEY" claude -p --bare --no-session-persistence --model "$REVIEW_MODEL" \
  --max-budget-usd "$REVIEW_BUDGET" --allowedTools "$REVIEW_TOOLS" --disallowedTools "$DENY_TOOLS" --output-format stream-json --verbose \
  < "$ROOT/$EB/review-prompt-$RUN_N.md") > "$WORK/review.jsonl" 2> "$WORK/review.err" || REVIEW_EXIT=$?
scrub "$WORK/review.jsonl"; scrub "$WORK/review.err"
sed "s#$WORK#<scratch>#g" "$WORK/review.jsonl" > "$EB/review-transcript-$RUN_N.jsonl"
log "exit=$REVIEW_EXIT transcript=$EB/review-transcript-$RUN_N.jsonl (checked: no key, no GitHub token)"
summarize "$WORK/review.jsonl" | while IFS= read -r line; do log "$line"; done
[ "$REVIEW_EXIT" = 0 ] || die "review run failed"
REVIEW_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
python3 -I - "$WORK/review.jsonl" "$EB/ai-review-$RUN_N.md" "$REVIEW_MODEL" "$PR_URL" "$HEAD_COMMIT" "$RUN_N" "$REVIEW_AT" <<'PY' | while IFS= read -r line; do log "$line"; done
import json, sys
transcript, out, model, pr, head, n, at = sys.argv[1:8]
result = {}
for line in open(transcript):
    try: event = json.loads(line)
    except ValueError: continue
    if event.get("type") == "result": result = event
import re
text = (result.get("result") or "").strip()
found = re.findall(r'\{\s*"verdict"\s*:.*?\}(?=[^{}]*$)', text, re.S)
try:
    verdict = json.loads(found[-1])
    assert verdict["verdict"] in ("APPROVED", "CHANGES_REQUESTED")
except Exception as error:
    print("verdict=UNPARSEABLE (%s)" % error); sys.exit(0)
with open(out, "w") as f:
    f.write("# EB-019 AI review, run %s\n\n" % n)
    f.write("- reviewer: separate headless `claude -p --bare` run, model `%s`, read-only tools (Read, Glob, Grep)\n" % model)
    f.write("- reviewed: %s at %s\n- at: %s\n- cost: US$%s\n- transcript: `review-transcript-%s.jsonl`\n\n" % (pr, head, at, result.get("total_cost_usd"), n))
    f.write("## Verdict: %s\n\n" % verdict["verdict"])
    for r in verdict.get("reasons", []): f.write("- %s\n" % r)
    f.write("\n## Full reply\n\n%s\n" % text)
print("verdict=" + verdict["verdict"])
PY
VERDICT="$(grep -m1 '^## Verdict:' "$EB/ai-review-$RUN_N.md" 2>/dev/null | sed 's/## Verdict: //' || true)"
[ "$VERDICT" = APPROVED ] || die "the reviewer requested changes; the candidate is not submitted (see $EB/ai-review-$RUN_N.md)"

log ""
log "## candidate: GitHub read-back + declared AI review and single-operator controls"
CANDIDATE="$(python3 -I - "$WORK/author.jsonl" <<PY
import json, sys
commands = []
for line in open(sys.argv[1]):
    try: event = json.loads(line)
    except ValueError: continue
    if event.get("type") == "assistant":
        for block in event.get("message", {}).get("content", []):
            if block.get("type") == "tool_use" and block["name"] == "Bash":
                commands.append(block["input"].get("command", ""))
print(json.dumps({
  "actor": "$ACTOR", "reason": "EB-019: candidate from headless agent run $RUN_ID",
  "run": {"kind": "ContextRailRun", "schema_version": "1", "run_id": "$RUN_ID", "work_order_id": "$WO_ID", "work_order_hash": "$WO_HASH",
          "agent": {"name": "claude-code", "provider": "anthropic", "model": "$AUTHOR_MODEL"}, "adapter": "claude-code-cli-headless",
          "started_by": "$ACTOR", "started_at": "$AUTHOR_STARTED", "finished_at": "$AUTHOR_FINISHED",
          "declared_commands": commands, "declared_changed_paths": "$(printf '%s' "$CHANGED" | tr '\n' ' ')".split()},
  "observation": {
    "reviews": [{"reviewer": "headless claude -p, $REVIEW_MODEL, separate read-only run", "kind": "ai", "verdict": "APPROVED",
                 "evidence_ref": "$EB/ai-review-$RUN_N.md", "at": "$REVIEW_AT"}],
    "compensating_controls": {"declared": True, "evidence_ref": "$EB/ai-review-$RUN_N.md + docs/governance/policies.md (single_operator mode)",
                              "reason": "single operator: the only human is also the run starter; an AI review on a different model from the author, the harness scope check and the local gate read-back stand in for a second human (DR-015 scope: local staging only)"}},
  "read_back": {"source": "github", "repository": "$REPO_SLUG", "base": "main", "head": "$TARGET_BRANCH"}}))
PY
)"
call POST "$BASE/changes/CHG-001/candidates" "$CANDIDATE"
printf '%s' "$BODY" | python3 -I -c '
import json,sys
d=json.load(sys.stdin)
c=d["change"]["candidates"][-1]
o=c.get("observation",{})
print("candidate", c.get("candidate_id"), "verdict", c.get("verdict"), "source", o.get("source"), "head", o.get("head_commit"))
print("  observed changed_paths", o.get("changed_paths"))
for k in o.get("checks",[]): print("  check", k.get("name"), k.get("status"), k.get("evidence_ref",""))
for g in c.get("gates",[]): print("  gate", g["gate"], g["status"], "-", g.get("detail",""))
' 2>&1 | while IFS= read -r line; do log "$line"; done
CAND_ID="$(printf '%s' "$BODY" | jget 'd["change"]["candidates"][-1]["candidate_id"]')"
CAND_VERDICT="$(printf '%s' "$BODY" | jget 'd["change"]["candidates"][-1].get("verdict")')"
[ "$CAND_VERDICT" = CANDIDATE_ACCEPTABLE ] || die "candidate verdict $CAND_VERDICT; not accepted"
call POST "$BASE/changes/CHG-001/candidates/$CAND_ID/decision" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-019\",\"decision\":\"ACCEPT\",\"role\":\"founder\",\"rationale\":\"Read back from GitHub, CI green, AI review on a different model approved, waiver declared.\"}"

log ""
log "## build the reviewed head commit and run it as local staging (:$STAGING_PORT)"
[ "$(git -C "$WORK/agent" rev-parse HEAD)" = "$HEAD_COMMIT" ] || die "agent clone moved off the head commit"
(cd "$WORK/agent" && go build -o "$WORK/order-portal" .)
DIGEST="sha256:$(shasum -a 256 "$WORK/order-portal" | cut -d' ' -f1)"
log "\$ go build -o <scratch>/order-portal .   (at $HEAD_COMMIT) -> $DIGEST"
PORT="$STAGING_PORT" ORDER_PORTAL_ENV=staging "$WORK/order-portal" >"$WORK/staging.log" 2>&1 & PIDS+=($!)
for i in $(seq 1 50); do curl -sf "http://127.0.0.1:${STAGING_PORT}/healthz" >/dev/null 2>&1 && break; sleep 0.2; done
log "127.0.0.1:$STAGING_PORT/healthz -> $(curl -s "http://127.0.0.1:${STAGING_PORT}/healthz")"

log ""
log "## release: CHG-001 -> staging"
call POST "$BASE/releases" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-019\",\"change_ids\":[\"CHG-001\"],\"target_environment_id\":\"staging\"}"
call POST "$BASE/releases/REL-001/build" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-019\",\"image_digest\":\"$DIGEST\",\"image_ref\":\"file://<scratch>/order-portal\",\"build_id\":\"local-build-$RUN_N\",\"source_commit\":\"$HEAD_COMMIT\",\"includes_commits\":[\"$HEAD_COMMIT\"],\"evidence_ref\":\"$OUT#build\"}"
call POST "$BASE/releases/REL-001/approval" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-019\",\"decision\":\"APPROVE\",\"role\":\"release_manager\",\"rationale\":\"Local staging under single-operator controls.\"}"
call POST "$BASE/releases/REL-001/deployment" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-019 staging deploy\",\"idempotency_key\":\"k-staging-$RUN_N\",\"source\":\"local\",\"service_url\":\"http://127.0.0.1:${STAGING_PORT}\"}"
printf '%s' "$BODY" | python3 -I -c '
import json,sys
d=json.load(sys.stdin)
r=d.get("release",d)
a=(r.get("deployments") or [{}])[-1]
print("release", r.get("release_id"), "status", r.get("status"), "attempt", a.get("attempt_id"), "outcome", a.get("outcome"), "revision", a.get("revision"), "target", a.get("deployed_target_ref"))
for g in a.get("gates",[]): print("  gate", g["gate"], g["status"], "-", g.get("detail",""))
rc=r.get("receipt") or {}
if rc: print("  receipt", rc.get("receipt_id"), rc.get("status"), "hash", rc.get("receipt_hash"))
' | while IFS= read -r line; do log "$line"; done
OUTCOME="$(printf '%s' "$BODY" | jget '((d.get("release",d).get("deployments") or [{}])[-1]).get("outcome")')"
[ "$OUTCOME" = PROMOTED ] || die "staging deployment outcome $OUTCOME"

log ""
log "## the staged build does what the change asked (read directly from :$STAGING_PORT)"
for q in "" "?status=PENDING_REVIEW" "?status=APPROVED" "?status=NOT_A_STATUS"; do
  code="$(curl -s -o "$WORK/smoke.json" -w '%{http_code}' "http://127.0.0.1:${STAGING_PORT}/api/orders$q")"
  log "GET /api/orders$q -> http=$code $(head -c 400 "$WORK/smoke.json")"
done
[ "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${STAGING_PORT}/api/orders?status=NOT_A_STATUS")" = 400 ] || die "unknown status is not answered with 400"

log ""
log "## result"
log "run=$RUN_ID pr=$PR_URL head=$HEAD_COMMIT candidate=$CAND_ID staging=$OUTCOME review=$VERDICT"
log "the pull request stays open; merging it and syncing the demo copy need the founder's command (DR-018)"
