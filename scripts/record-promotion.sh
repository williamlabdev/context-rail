#!/usr/bin/env bash
# Record what Cloud Run actually deployed into ContextRail's release ledger.
#
# Reads the live revision from Cloud Run with gcloud (on the operator machine),
# runs the smoke check, and POSTs the observation to the release's deployment
# endpoint with an idempotency key. ContextRail then verifies digest, target
# and config against the approved manifest and issues the Release Receipt.
#
# Usage:
#   scripts/record-promotion.sh <context-rail-url> <project-id> <release-id> <gcp-project> <region> <service> [idempotency-key]
#
# Example:
#   scripts/record-promotion.sh https://context-rail-staging-xxxx.a.run.app order-operations-portal REL-001 \
#       my-gcp-project asia-east1 order-operations-portal-staging
set -euo pipefail

CTR_URL="${1:?context-rail url}"; PROJECT_ID="${2:?context-rail project id}"; RELEASE_ID="${3:?release id}"
GCP_PROJECT="${4:?gcp project}"; REGION="${5:?region}"; SERVICE="${6:?cloud run service}"
KEY="${7:-deploy-$(date -u +%Y%m%dT%H%M%SZ)-$$}"
ACTOR="${CONTEXT_RAIL_ACTOR:-$(whoami)}"

command -v gcloud >/dev/null || { echo "gcloud is required" >&2; exit 1; }
command -v python3 >/dev/null || { echo "python3 is required" >&2; exit 1; }

echo "== reading live revision of ${SERVICE} (${GCP_PROJECT}/${REGION})"
REVISION="$(gcloud run services describe "${SERVICE}" --project="${GCP_PROJECT}" --region="${REGION}" --format='value(status.latestReadyRevisionName)')"
URL="$(gcloud run services describe "${SERVICE}" --project="${GCP_PROJECT}" --region="${REGION}" --format='value(status.url)')"
IMAGE="$(gcloud run revisions describe "${REVISION}" --project="${GCP_PROJECT}" --region="${REGION}" --format='value(spec.containers[0].image)')"
DIGEST="${IMAGE##*@}"
[[ "${DIGEST}" == sha256:* ]] || { echo "revision image is not pinned by digest (${IMAGE}); refuse to record" >&2; exit 1; }
TARGET_REF="cloud-run/${SERVICE}"
DEPLOYED_AT="$(gcloud run revisions describe "${REVISION}" --project="${GCP_PROJECT}" --region="${REGION}" --format='value(metadata.creationTimestamp)')"

echo "== smoke ${URL}"
SMOKE="FAIL"
if curl -fsS --max-time 20 "${URL}/healthz" >/dev/null 2>&1; then SMOKE="PASS"; fi
echo "smoke: ${SMOKE}"

BODY="$(python3 - "$KEY" "$REVISION" "$URL" "$DIGEST" "$TARGET_REF" "$DEPLOYED_AT" "$SMOKE" "$ACTOR" <<'PY'
import json, sys
key, revision, url, digest, target, at, smoke, actor = sys.argv[1:]
print(json.dumps({
  "reason": "recorded by scripts/record-promotion.sh from gcloud run describe",
  "actor": actor, "idempotency_key": key, "operation_id": revision, "source": "cloud-run",
  "revision": revision, "service_url": url, "deployed_digest": digest, "deployed_target_ref": target,
  "deployed_at": at, "smoke": {"status": smoke, "evidence_ref": url + "/healthz"},
}))
PY
)"

echo "== recording deployment on ${CTR_URL}/v1/projects/${PROJECT_ID}/releases/${RELEASE_ID}/deployment (key ${KEY})"
curl -fsS -X POST -H "Content-Type: application/json" -H "X-ContextRail-Actor: ${ACTOR}" \
  "${CTR_URL}/v1/projects/${PROJECT_ID}/releases/${RELEASE_ID}/deployment" -d "${BODY}" \
  | python3 -c 'import json,sys; v=json.load(sys.stdin); r=v["release"]; print("release:", r["release_id"], r["status"]); rc=r.get("receipt"); print("receipt:", rc and (rc["receipt_id"], rc["status"], rc["receipt_hash"][:26])); print("gates:", [(g["gate"], g["status"]) for g in r["deployments"][-1]["gates"]])'
