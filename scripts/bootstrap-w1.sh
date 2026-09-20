#!/usr/bin/env bash
# One-shot for the credential-bound Week 1 steps, run on the owner's machine:
#   1. verify the current slice locally (go + frontend);
#   2. create the private GitHub remote and push every branch;
#   3. (optional) create/select the GCP project, link billing, deploy staging.
#
# Usage:
#   scripts/bootstrap-w1.sh                    # steps 1–2 only
#   scripts/bootstrap-w1.sh <gcp-project-id>   # steps 1–3
#
# Idempotent: re-running skips what already exists. Nothing here touches
# production or makes the repository public.
set -euo pipefail
cd "$(dirname "$0")/.."

BRANCH="${BRANCH:-docs/w4-submission-prep}"
REPO_NAME="${REPO_NAME:-context-rail}"
PROJECT_ID="${1:-}"
REGION="${REGION:-asia-east1}"

step() { printf '\n\033[1;36m== %s\033[0m\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1 — $2" >&2; exit 1; }; }

# ---------------------------------------------------------------- 1. verify
step "checkout ${BRANCH}"
git checkout -q "${BRANCH}"
git log --oneline -1

need go "install Go 1.22+ (brew install go)"
need node "install Node 22 (brew install node@22)"
step "go vet + go test"
go vet ./...
go test ./...

step "frontend typecheck + unit tests + build"
( cd frontend && npm ci --no-audit --no-fund >/dev/null && npm run typecheck && npm test && npm run build )

# ---------------------------------------------------------------- 2. github
step "GitHub private remote"
need gh "brew install gh && gh auth login"
if ! gh auth status >/dev/null 2>&1; then
  echo "gh is not logged in; opening login..."
  gh auth login
fi
if git remote get-url origin >/dev/null 2>&1; then
  echo "origin already set: $(git remote get-url origin)"
else
  OWNER="$(gh api user --jq .login)"
  if gh repo view "${OWNER}/${REPO_NAME}" >/dev/null 2>&1; then
    git remote add origin "https://github.com/${OWNER}/${REPO_NAME}.git"
    echo "attached existing repo ${OWNER}/${REPO_NAME}"
  else
    gh repo create "${REPO_NAME}" --private --source . --remote origin \
      --description "ContextRail — environment-aware AI Change Assurance (Google Cloud AI Builder Cup 2026 prototype)"
    echo "created private repo ${OWNER}/${REPO_NAME}"
  fi
fi
git push -u origin develop feat/cloud-run-baseline feat/vs-003-topology feat/vs-004-decision-pack feat/vs-005-candidate-gate feat/vs-006-promotion feat/vs-007-prod-demo feat/ui-20-context-pack feat/ui-15-19-locale docs/w4-submission-prep 2>&1 | tail -6
git push origin codex/001-manual-order-review 2>/dev/null || true
echo "remote: $(git remote get-url origin)  (private — switch to public only at submission time)"

# ---------------------------------------------------------------- 3. gcp
if [ -z "${PROJECT_ID}" ]; then
  step "GCP deploy skipped"
  echo "re-run with a project id to deploy:  scripts/bootstrap-w1.sh <gcp-project-id>"
  exit 0
fi

step "GCP project ${PROJECT_ID}"
need gcloud "install the Google Cloud CLI (brew install --cask google-cloud-sdk) and run: gcloud auth login"
if ! gcloud auth list --filter=status:ACTIVE --format='value(account)' | grep -q .; then
  echo "gcloud has no active account; opening login..."
  gcloud auth login
fi
if gcloud projects describe "${PROJECT_ID}" >/dev/null 2>&1; then
  echo "project exists"
else
  gcloud projects create "${PROJECT_ID}" --name="ContextRail" --set-as-default
fi
gcloud config set project "${PROJECT_ID}" >/dev/null

step "billing"
if gcloud billing projects describe "${PROJECT_ID}" --format='value(billingEnabled)' 2>/dev/null | grep -q True; then
  echo "billing already enabled"
else
  ACCOUNTS="$(gcloud billing accounts list --filter=open=true --format='value(name)')"
  COUNT="$(printf '%s\n' "${ACCOUNTS}" | grep -c . || true)"
  if [ "${COUNT}" = "1" ]; then
    gcloud billing projects link "${PROJECT_ID}" --billing-account="${ACCOUNTS#billingAccounts/}"
    echo "linked billing account ${ACCOUNTS}"
  else
    echo "found ${COUNT} open billing accounts; link one manually then re-run:" >&2
    gcloud billing accounts list >&2 || true
    echo "  gcloud billing projects link ${PROJECT_ID} --billing-account=<ACCOUNT_ID>" >&2
    exit 1
  fi
fi

step "deploy Cloud Run staging (${REGION})"
scripts/deploy-cloud-run.sh "${PROJECT_ID}" "${REGION}"

cat <<EOF

Done. Paste the url / revision / image / commit printed above into
evidence/EB-004/README.md (Deployment receipt) and commit it.
EOF
