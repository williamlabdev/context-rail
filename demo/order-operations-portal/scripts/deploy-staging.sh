#!/usr/bin/env bash
set -euo pipefail

demo_root="$(cd "$(dirname "$0")/.." && pwd)"
receipt="$demo_root/evidence/EB-001/cloud-run-staging-receipt.json"
service="order-operations-portal-staging"
staging_tag="${STAGING_TAG:-manual-review}"

write_blocked_receipt() {
  local reason="$1"
  python3 - "$receipt" "$reason" <<'PY'
import json
import sys

path, reason = sys.argv[1:]
with open(path, "w", encoding="utf-8") as handle:
    json.dump(
        {
            "receipt_id": "CR-STAGING-001",
            "status": "NEEDS_INPUT",
            "environment": "staging",
            "target": "cloud-run/order-operations-portal-staging",
            "decision_id": "DR-001",
            "reason": reason,
            "production": {"status": "BLOCKED_IN_DEMO", "deployed": False},
        },
        handle,
        indent=2,
    )
    handle.write("\n")
PY
}

if ! command -v gcloud >/dev/null 2>&1 || ! command -v python3 >/dev/null 2>&1 || [[ -z "${GCP_PROJECT_ID:-}" || -z "${CLOUD_RUN_REGION:-}" || -z "${IMAGE_URI:-}" || -z "${APPROVED_COMMIT:-}" ]]; then
  write_blocked_receipt "Set GCP_PROJECT_ID, CLOUD_RUN_REGION, IMAGE_URI, APPROVED_COMMIT and install/authenticate gcloud before a real staging deploy."
  echo "STAGING BLOCKED: cloud access is not configured; receipt written to $receipt"
  exit 2
fi

if [[ "$IMAGE_URI" != *@sha256:* ]]; then
  write_blocked_receipt "IMAGE_URI must use an immutable @sha256 digest."
  echo "STAGING BLOCKED: mutable image identity; receipt written to $receipt"
  exit 2
fi

if ! gate_output="$(python3 "$demo_root/scripts/check-staging-gate.py" "$demo_root" "$APPROVED_COMMIT" 2>&1)"; then
  write_blocked_receipt "$gate_output"
  echo "$gate_output"
  exit 2
fi
echo "$gate_output"

gcloud run deploy "$service" \
  --project "$GCP_PROJECT_ID" \
  --region "$CLOUD_RUN_REGION" \
  --image "$IMAGE_URI" \
  --no-traffic \
  --tag "$staging_tag"

tag_url="$(gcloud run services describe "$service" --project "$GCP_PROJECT_ID" --region "$CLOUD_RUN_REGION" --format="value(status.traffic[?tag=$staging_tag].url)")"
revision="$(gcloud run services describe "$service" --project "$GCP_PROJECT_ID" --region "$CLOUD_RUN_REGION" --format='value(status.latestReadyRevisionName)')"
if [[ -z "$tag_url" ]]; then
  write_blocked_receipt "Cloud Run did not return a URL for tagged revision $staging_tag."
  echo "STAGING BLOCKED: tagged revision URL is missing; receipt written to $receipt"
  exit 2
fi
if ! curl -fsS "$tag_url/healthz" >/dev/null; then
  write_blocked_receipt "Staging smoke failed for tagged revision $staging_tag."
  echo "STAGING BLOCKED: tagged revision smoke failed; receipt written to $receipt"
  exit 2
fi
export STAGING_TAG_URL="$tag_url"
export CLOUD_RUN_REVISION="$revision"

python3 - "$receipt" <<'PY'
import json
import os
import sys

path = sys.argv[1]
with open(path, "w", encoding="utf-8") as handle:
    json.dump(
        {
            "receipt_id": "CR-STAGING-001",
            "status": "PASS",
            "environment": "staging",
            "gcp_project_id": os.environ["GCP_PROJECT_ID"],
            "region": os.environ["CLOUD_RUN_REGION"],
            "service": "order-operations-portal-staging",
            "target": "cloud-run/order-operations-portal-staging",
            "image_uri": os.environ["IMAGE_URI"],
            "approved_commit": os.environ["APPROVED_COMMIT"],
            "staging_tag": os.environ.get("STAGING_TAG", "manual-review"),
            "tag_url": os.environ["STAGING_TAG_URL"],
            "cloud_run_revision": os.environ["CLOUD_RUN_REVISION"],
            "smoke": {"path": "/healthz", "status": "PASS", "target": "tagged-revision"},
            "production": {"status": "BLOCKED_IN_DEMO", "deployed": False},
        },
        handle,
        indent=2,
    )
    handle.write("\n")
PY
echo "STAGING PASS: receipt written to $receipt"
