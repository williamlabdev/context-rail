#!/usr/bin/env bash
# EB-017 / G4: promote one locally built demo binary through local staging and
# local prod-demo with ContextRail's release gates reading each instance back
# over loopback (DR-017). Reproducible: run from the repo root.
#
#   scripts/local-promotion.sh            # writes evidence/EB-017/local-promotion-run.txt
#   OUT=/tmp/run.txt scripts/local-promotion.sh
#
# Ports (override with env): RAIL_PORT=8090 STAGING_PORT=8091 PROD_DEMO_PORT=8092 DRIFT_PORT=8093
# No API key is needed: the rule advisor is forced (CONTEXT_RAIL_ADVISOR=rule).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
RAIL_PORT="${RAIL_PORT:-8090}"
STAGING_PORT="${STAGING_PORT:-8091}"
PROD_DEMO_PORT="${PROD_DEMO_PORT:-8092}"
DRIFT_PORT="${DRIFT_PORT:-8093}"
OUT="${OUT:-evidence/EB-017/local-promotion-run.txt}"
STATE_DIR="${STATE_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/eb017-state.XXXXXX")}"
PROJECT="order-operations-portal"
BASE="http://127.0.0.1:${RAIL_PORT}/v1/projects/${PROJECT}"
ACTOR="founder-001"
PIDS=()

mkdir -p "$(dirname "$OUT")" bin
: > "$OUT"
log() { printf '%s\n' "$*" | tee -a "$OUT"; }
pretty() { python3 -I -c 'import json,sys; print(json.dumps(json.load(sys.stdin), indent=2, ensure_ascii=False))'; }
jget() { python3 -I -c "import json,sys; d=json.load(sys.stdin); print($1)"; }
cleanup() {
  log ""
  log "## teardown"
  for pid in "${PIDS[@]:-}"; do [ -n "$pid" ] && kill "$pid" 2>/dev/null || true; done
  sleep 1
  for port in "$RAIL_PORT" "$STAGING_PORT" "$PROD_DEMO_PORT" "$DRIFT_PORT"; do
    if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then log "port $port: STILL LISTENING"; else log "port $port: no listener"; fi
  done
  log "state_dir=<scratch> (mktemp or STATE_DIR, not committed)"
}
trap cleanup EXIT

# call METHOD PATH [JSON] -> prints http status line + pretty body, keeps body in $BODY
call() {
  local method="$1" path="$2" data="${3:-}" tmp
  tmp="$(mktemp)"
  if [ -n "$data" ]; then
    STATUS="$(curl -sS -o "$tmp" -w '%{http_code}' -X "$method" -H 'Content-Type: application/json' -H "X-ContextRail-Actor: $ACTOR" --data "$data" "$path")"
  else
    STATUS="$(curl -sS -o "$tmp" -w '%{http_code}' -X "$method" -H "X-ContextRail-Actor: $ACTOR" "$path")"
  fi
  BODY="$(cat "$tmp")"; rm -f "$tmp"
  log ""
  log "### $method ${path#http://127.0.0.1:${RAIL_PORT}}"
  [ -n "$data" ] && { log "request_body:"; printf '%s' "$data" | pretty | tee -a "$OUT" >/dev/null; }
  log "http=$STATUS"
  printf '%s' "$BODY" | pretty 2>/dev/null | tee -a "$OUT" >/dev/null || log "$BODY"
  case "$STATUS" in 2*) ;; *) [ "${ALLOW_FAIL:-0}" = 1 ] || { log "FATAL: unexpected http=$STATUS"; exit 1; } ;; esac
}
gates() { printf '%s' "$BODY" | python3 -I -c '
import json,sys
d=json.load(sys.stdin)
r=d.get("release",d)
deps=r.get("deployments") or []
a=deps[-1] if deps else {}
print("release", r.get("release_id"), "status", r.get("status"), "deployments", len(deps))
print("attempt", a.get("attempt_id"), "key", a.get("idempotency_key"), "outcome", a.get("outcome"), "revision", a.get("revision"), "digest", (a.get("deployed_digest") or "")[:19]+"...", "target", a.get("deployed_target_ref"))
for g in a.get("gates",[]): print("  gate", g["gate"], g["status"], "-", g.get("detail",""))
rc=r.get("receipt") or {}
if rc: print("  receipt", rc.get("receipt_id"), rc.get("status"), "previous", rc.get("previous_receipt_id",""), "hash", rc.get("receipt_hash"))
'; }

HEAD_COMMIT="$(git rev-parse HEAD)"
log "# EB-017 real run: local staging -> local prod-demo promotion with loopback read-back (DR-017, G4)"
log "captured_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
log "commit=$HEAD_COMMIT"
log "tracked changes not in that commit: $(git status --porcelain --untracked-files=no | tr '\n' ' ' | sed 's/ *$//')"
log "ports: context-rail=$RAIL_PORT staging=$STAGING_PORT prod-demo=$PROD_DEMO_PORT drift=$DRIFT_PORT"
log "advisor=rule (CONTEXT_RAIL_ADVISOR=rule; no API key read or used)"
log "env_check: ANTHROPIC_API_KEY exported in the invoking shell? $([ -n "${ANTHROPIC_API_KEY:-}" ] && echo YES || echo NO)"

log ""
log "## build"
log "\$ go build -o bin/order-portal ./demo/order-operations-portal"
(cd demo/order-operations-portal && go build -o "$ROOT/bin/order-portal" .)
DIGEST="sha256:$(shasum -a 256 bin/order-portal | cut -d' ' -f1)"
log "bin/order-portal digest: $DIGEST"
log "\$ go build -trimpath -o bin/order-portal-drift ./demo/order-operations-portal   (same source, different bytes: the drift binary)"
(cd demo/order-operations-portal && go build -trimpath -o "$ROOT/bin/order-portal-drift" .)
DRIFT_DIGEST="sha256:$(shasum -a 256 bin/order-portal-drift | cut -d' ' -f1)"
log "bin/order-portal-drift digest: $DRIFT_DIGEST"
[ "$DIGEST" != "$DRIFT_DIGEST" ] || { log "FATAL: drift binary has the same digest"; exit 1; }
log "\$ go build -o bin/context-rail ./cmd/context-rail"
go build -o bin/context-rail ./cmd/context-rail

log ""
log "## start three demo instances (same binary twice, drift binary once)"
PORT="$STAGING_PORT"   ORDER_PORTAL_ENV=staging   ./bin/order-portal       >"$STATE_DIR/staging.log" 2>&1 & PIDS+=($!)
PORT="$PROD_DEMO_PORT" ORDER_PORTAL_ENV=prod-demo ./bin/order-portal       >"$STATE_DIR/prod-demo.log" 2>&1 & PIDS+=($!)
PORT="$DRIFT_PORT"     ORDER_PORTAL_ENV=prod-demo ./bin/order-portal-drift >"$STATE_DIR/drift.log" 2>&1 & PIDS+=($!)
log "pids: staging=${PIDS[0]} prod-demo=${PIDS[1]} drift=${PIDS[2]}"

log ""
log "## start ContextRail"
log "\$ CONTEXT_RAIL_ADVISOR=rule PORT=$RAIL_PORT CONTEXT_RAIL_FIXTURE_ROOTS=demo/order-operations-portal,examples/support-insights CONTEXT_RAIL_STATIC_DIR=frontend/dist CONTEXT_RAIL_STATE_DIR=<scratch> ./bin/context-rail"
CONTEXT_RAIL_ADVISOR=rule PORT="$RAIL_PORT" CONTEXT_RAIL_FIXTURE_ROOTS=demo/order-operations-portal,examples/support-insights \
  CONTEXT_RAIL_STATIC_DIR=frontend/dist CONTEXT_RAIL_STATE_DIR="$STATE_DIR/rail" ./bin/context-rail >"$STATE_DIR/rail.log" 2>&1 & PIDS+=($!)
for i in $(seq 1 50); do curl -sf "http://127.0.0.1:${RAIL_PORT}/healthz" >/dev/null 2>&1 && break; sleep 0.2; done
for p in "$STAGING_PORT" "$PROD_DEMO_PORT" "$DRIFT_PORT"; do for i in $(seq 1 50); do curl -sf "http://127.0.0.1:${p}/healthz" >/dev/null 2>&1 && break; sleep 0.2; done; done
log "startup line: $(grep -m1 'listening' "$STATE_DIR/rail.log" | sed "s#$STATE_DIR#<scratch>#g")"

log ""
log "## what each instance says about itself (GET /healthz, read directly for the record)"
for p in "$STAGING_PORT" "$PROD_DEMO_PORT" "$DRIFT_PORT"; do log "127.0.0.1:$p -> $(curl -s "http://127.0.0.1:${p}/healthz")"; done

log ""
log "## topology: point staging at a local target and add prod-demo (runtime state; project.yaml untouched)"
call GET "$BASE/environments"
V="$(printf '%s' "$BODY" | jget 'd["current_version"]')"
call PATCH "$BASE/environments/staging" "{\"expected_version\":$V,\"actor\":\"$ACTOR\",\"reason\":\"EB-017: staging is a local process (DR-015/DR-017)\",\"target_ref\":\"local/order-operations-portal-staging\"}"
V="$(printf '%s' "$BODY" | jget 'd["current_version"]')"
call POST "$BASE/environments" "{\"expected_version\":$V,\"actor\":\"$ACTOR\",\"reason\":\"EB-017: isolated local prod-demo (DR-015/DR-017)\",\"id\":\"prod-demo\",\"display_name\":\"prod-demo\",\"type\":\"prod-demo\",\"sequence\":4,\"target_ref\":\"local/order-operations-portal-prod-demo\",\"owner\":\"founder-001\",\"required_evidence\":[\"release-approval\",\"staging-receipt\",\"smoke\",\"single-operator-controls\"],\"approver_policy\":\"distinct human approver\"}"

log ""
log "## change: request -> decision -> work order -> candidate -> accepted"
call POST "$BASE/changes" "$(cat <<JSON
{"actor":"$ACTOR","reason":"EB-017: the healthz identity change that this promotion ships",
 "request":{"title":"Report revision and build digest from /healthz","objective":"Let a local instance answer which build runs and which start of it answers, so a promotion gate can read it back without a container registry.",
 "owner_summary":"Each local instance reports its environment label, a per-start revision and the sha256 of its own executable on /healthz.",
 "scope_included":["healthz identity fields","unit tests"],"scope_excluded":["project.yaml","authentication","order review behaviour"],
 "acceptance_criteria":[{"text":"healthz reports revision, build_digest and started_at"},{"text":"build_digest equals the sha256 of the running executable"}],
 "allowed_paths":["main.go","main_test.go"],"forbidden_actions":["project.yaml edits","authentication"],
 "target_environment_id":"staging","business_constraints":{"data_classification":"internal","expected_monthly_volume":"500 reviews"},"requested_by":"$ACTOR"}}
JSON
)"
OPTION="$(printf '%s' "$BODY" | jget 'd["change"]["versions"][-1]["options"][0]["id"]')"
log "selected option: $OPTION"
call POST "$BASE/changes/CHG-001/decision" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017\",\"decision\":\"ACCEPT\",\"role\":\"founder\",\"selected_option\":\"$OPTION\",\"rationale\":\"Smallest change that gives the release gate something real to read back.\",\"risk_level\":\"low\"}"
call POST "$BASE/changes/CHG-001/work-order" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017\"}"
WO_ID="$(printf '%s' "$BODY" | jget 'd["work_order"]["work_order_id"]')"
WO_HASH="$(printf '%s' "$BODY" | jget 'd["work_order"]["work_order_hash"]')"
BASE_BRANCH="$(printf '%s' "$BODY" | jget 'd["work_order"].get("base_branch","develop")')"
TARGET_BRANCH="$(printf '%s' "$BODY" | jget 'd["work_order"].get("target_branch","g4/local-promotion")')"
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
call POST "$BASE/changes/CHG-001/candidates" "$(cat <<JSON
{"actor":"$ACTOR","reason":"EB-017: candidate from the agent run that produced the healthz change",
 "run":{"kind":"ContextRailRun","schema_version":"1","run_id":"run-eb017-1","work_order_id":"$WO_ID","work_order_hash":"$WO_HASH",
   "agent":{"name":"claude-code","provider":"anthropic","model":"claude-fable-5-1"},"adapter":"claude-code-cli","started_by":"$ACTOR","started_at":"$NOW"},
 "observation":{"source":"local-git","repository":"williamlabdev/context-rail","branch":"$TARGET_BRANCH","base_branch":"$BASE_BRANCH",
   "base_commit":"7d371b8","head_commit":"$HEAD_COMMIT","changed_paths":["main.go","main_test.go"],
   "checks":[{"name":"test","status":"PASS","evidence_ref":"evidence/EB-017/go-build-test-output.txt","observed_at":"$NOW"},
             {"name":"build","status":"PASS","evidence_ref":"evidence/EB-017/local-promotion-run.txt#build","observed_at":"$NOW"}],
   "reviews":[{"reviewer":"verifier subagent (Claude; model per agent definition: sonnet)","kind":"ai","verdict":"APPROVED","evidence_ref":"evidence/EB-017/ai-review-demo-healthz.md","at":"$NOW"}],
   "compensating_controls":{"declared":true,"evidence_ref":"evidence/EB-017/ai-review-demo-healthz.md + demo/order-operations-portal/docs/governance/policies.md (single_operator mode)","reason":"single operator: the only human is also the run starter; an independent AI review plus the local gate read-back stand in for a second human (DR-015 scope: local staging and prod-demo only)"},
   "observed_at":"$NOW"}}
JSON
)"
call POST "$BASE/changes/CHG-001/candidates/CAND-001/decision" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017\",\"decision\":\"ACCEPT\",\"role\":\"founder\",\"rationale\":\"Checks pass, AI review approved, waiver declared.\"}"

log ""
log "## release 1: CHG-001 -> staging (local instance on :$STAGING_PORT)"
call POST "$BASE/releases" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017\",\"change_ids\":[\"CHG-001\"],\"target_environment_id\":\"staging\"}"
call POST "$BASE/releases/REL-001/build" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017\",\"image_digest\":\"$DIGEST\",\"image_ref\":\"file://bin/order-portal\",\"build_id\":\"local-build-1\",\"source_commit\":\"$HEAD_COMMIT\",\"includes_commits\":[\"$HEAD_COMMIT\"],\"evidence_ref\":\"evidence/EB-017/local-promotion-run.txt#build\"}"
call POST "$BASE/releases/REL-001/approval" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017\",\"decision\":\"APPROVE\",\"role\":\"release_manager\",\"rationale\":\"Local staging under single-operator controls.\"}"
call POST "$BASE/releases/REL-001/deployment" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017 staging deploy\",\"idempotency_key\":\"k-staging-1\",\"source\":\"local\",\"service_url\":\"http://127.0.0.1:${STAGING_PORT}\"}"
log "--- gates:"; gates | tee -a "$OUT"
log ""
log "### replay: same idempotency_key again (must not re-read the instance or issue a new receipt)"
call POST "$BASE/releases/REL-001/deployment" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017 staging deploy (retry)\",\"idempotency_key\":\"k-staging-1\",\"source\":\"local\",\"service_url\":\"http://127.0.0.1:${STAGING_PORT}\"}"
log "--- gates:"; gates | tee -a "$OUT"

log ""
log "## release 2: promote REL-001 -> prod-demo"
call POST "$BASE/releases" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017\",\"source_release_id\":\"REL-001\",\"target_environment_id\":\"prod-demo\"}"
call POST "$BASE/releases/REL-002/approval" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017\",\"decision\":\"APPROVE\",\"role\":\"release_manager\",\"rationale\":\"Promote the staged build unchanged.\"}"
log ""
log "### attempt A: the drift binary answers on :$DRIFT_PORT (expected: digest_drift BLOCKED)"
call POST "$BASE/releases/REL-002/deployment" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017 prod-demo attempt A (drift)\",\"idempotency_key\":\"k-prod-demo-a\",\"source\":\"local\",\"service_url\":\"http://127.0.0.1:${DRIFT_PORT}\"}"
log "--- gates:"; gates | tee -a "$OUT"
log ""
log "### attempt B: the staging instance answers for prod-demo on :$STAGING_PORT (expected: target_drift + new_revision BLOCKED)"
call POST "$BASE/releases/REL-002/deployment" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017 prod-demo attempt B (wrong instance)\",\"idempotency_key\":\"k-prod-demo-b\",\"source\":\"local\",\"service_url\":\"http://127.0.0.1:${STAGING_PORT}\"}"
log "--- gates:"; gates | tee -a "$OUT"
log ""
log "### attempt C: the real prod-demo instance on :$PROD_DEMO_PORT (expected: PROMOTED, receipt chained to RR-001)"
call POST "$BASE/releases/REL-002/deployment" "{\"actor\":\"$ACTOR\",\"reason\":\"EB-017 prod-demo attempt C\",\"idempotency_key\":\"k-prod-demo-c\",\"source\":\"local\",\"service_url\":\"http://127.0.0.1:${PROD_DEMO_PORT}\"}"
log "--- gates:"; gates | tee -a "$OUT"

log ""
log "## final state"
call GET "$BASE/releases"
log ""
log "## demo instance logs (what each instance saw ContextRail do)"
for f in staging prod-demo drift; do log "--- $f: $(grep -c 'GET /healthz' "$STATE_DIR/$f.log") GET /healthz"; done
