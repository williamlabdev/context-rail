#!/usr/bin/env bash
# Deploy the ContextRail read-only workspace to Cloud Run staging.
#
# Usage:
#   scripts/deploy-cloud-run.sh <gcp-project-id> [region] [service]
#   scripts/deploy-cloud-run.sh --smoke https://context-rail-staging-xxxx.run.app
#
# The script is idempotent: enabling APIs and creating the Artifact Registry
# repository are no-ops when they already exist. It never touches production
# and does not create IAM bindings beyond what `gcloud run deploy` needs for
# an unauthenticated demo service.
set -euo pipefail

smoke() {
  local base="${1%/}"
  echo "== smoke: ${base}"
  curl -fsS "${base}/healthz" | tee /dev/stderr | grep -q '"status":"ok"'
  echo
  curl -fsS "${base}/v1/projects" | python3 -c 'import json,sys; s=json.load(sys.stdin); ids=[p["project"]["id"] for p in s["projects"]]; print("projects:", ids); assert len(ids)>=2, "expected both fixture Projects"'
  curl -fsS -o /dev/null -w 'workspace / -> HTTP %{http_code}\n' "${base}/"
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "${base}/v1/projects")
  [ "$code" = "405" ] && echo "POST /v1/projects -> 405 (read-only boundary holds)" || { echo "expected 405 for POST, got $code"; exit 1; }
}

if [ "${1:-}" = "--smoke" ]; then
  smoke "${2:?service url required}"
  exit 0
fi

PROJECT_ID="${1:?usage: $0 <gcp-project-id> [region] [service]}"
REGION="${2:-asia-east1}"
SERVICE="${3:-context-rail-staging}"
AR_REPO="context-rail"

command -v gcloud >/dev/null || { echo "gcloud is required"; exit 1; }
cd "$(dirname "$0")/.."
SHORT_SHA="$(git rev-parse --short HEAD)"
if [ -n "$(git status --porcelain)" ]; then
  echo "WARNING: working tree is dirty; the deployed image will not match commit ${SHORT_SHA} exactly." >&2
fi

echo "== project ${PROJECT_ID} region ${REGION} service ${SERVICE} commit ${SHORT_SHA}"
gcloud config set project "${PROJECT_ID}" >/dev/null

echo "== enabling APIs"
gcloud services enable run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com >/dev/null

echo "== ensuring Artifact Registry repo ${AR_REPO}"
gcloud artifacts repositories describe "${AR_REPO}" --location="${REGION}" >/dev/null 2>&1 || \
  gcloud artifacts repositories create "${AR_REPO}" --location="${REGION}" --repository-format=docker \
    --description="ContextRail images"

echo "== submitting Cloud Build"
gcloud builds submit --config cloudbuild.yaml \
  --substitutions="_REGION=${REGION},_AR_REPO=${AR_REPO},_SERVICE=${SERVICE},SHORT_SHA=${SHORT_SHA}" .

URL="$(gcloud run services describe "${SERVICE}" --region="${REGION}" --format='value(status.url)')"
REVISION="$(gcloud run services describe "${SERVICE}" --region="${REGION}" --format='value(status.latestReadyRevisionName)')"
IMAGE="$(gcloud run revisions describe "${REVISION}" --region="${REGION}" --format='value(spec.containers[0].image)')"

echo
echo "== deployed"
echo "url:      ${URL}"
echo "revision: ${REVISION}"
echo "image:    ${IMAGE}"
echo "commit:   ${SHORT_SHA}"
echo
smoke "${URL}"
echo
echo "Record url / revision / image digest / commit in the evidence bundle for this deployment."
