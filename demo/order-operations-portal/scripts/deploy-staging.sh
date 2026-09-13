#!/usr/bin/env bash
set -euo pipefail

demo_root="$(cd "$(dirname "$0")/.." && pwd)"
receipt="$demo_root/evidence/EB-001/cloud-run-staging-receipt.json"

if ! command -v gcloud >/dev/null 2>&1 || [[ -z "${GCP_PROJECT_ID:-}" || -z "${CLOUD_RUN_REGION:-}" || -z "${IMAGE_URI:-}" ]]; then
  cat > "$receipt" <<'JSON'
{
  "receipt_id": "CR-STAGING-001",
  "status": "NEEDS_INPUT",
  "environment": "staging",
  "target": "cloud-run/order-operations-portal-staging",
  "decision_id": "DR-001",
  "reason": "Set GCP_PROJECT_ID, CLOUD_RUN_REGION and IMAGE_URI and install/authenticate gcloud before a real staging deploy.",
  "production": {"status": "BLOCKED_IN_DEMO", "deployed": false}
}
JSON
  echo "STAGING BLOCKED: cloud access is not configured; receipt written to $receipt"
  exit 2
fi

gcloud run deploy order-operations-portal-staging \
  --project "$GCP_PROJECT_ID" \
  --region "$CLOUD_RUN_REGION" \
  --image "$IMAGE_URI" \
  --no-traffic

url="$(gcloud run services describe order-operations-portal-staging --project "$GCP_PROJECT_ID" --region "$CLOUD_RUN_REGION" --format='value(status.url)')"
revision="$(gcloud run services describe order-operations-portal-staging --project "$GCP_PROJECT_ID" --region "$CLOUD_RUN_REGION" --format='value(status.latestReadyRevisionName)')"
curl -fsS "$url/healthz" >/dev/null

cat > "$receipt" <<JSON
{
  "receipt_id": "CR-STAGING-001",
  "status": "PASS",
  "environment": "staging",
  "gcp_project_id": "$GCP_PROJECT_ID",
  "region": "$CLOUD_RUN_REGION",
  "service": "order-operations-portal-staging",
  "image_uri": "$IMAGE_URI",
  "cloud_run_revision": "$revision",
  "smoke": {"path": "/healthz", "status": "PASS"},
  "production": {"status": "BLOCKED_IN_DEMO", "deployed": false}
}
JSON
echo "STAGING PASS: receipt written to $receipt"
