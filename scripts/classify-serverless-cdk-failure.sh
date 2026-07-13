#!/usr/bin/env bash
set -euo pipefail

if (($# != 1)); then
  echo "Usage: scripts/classify-serverless-cdk-failure.sh <deploy-log>" >&2
  exit 2
fi

deploy_log="$1"
if [[ ! -f "${deploy_log}" ]]; then
  echo "CDK deploy log is unavailable." >&2
  exit 2
fi

category="unknown"
if grep -Eiq 'backup retention period.*exceeds.*maximum.*free tier' "${deploy_log}"; then
  category="free-plan-backup-retention"
elif grep -Eiq 'accessdenied|not authorized to perform|unauthorizedoperation' "${deploy_log}"; then
  category="authorization"
elif grep -Eiq 'limitexceeded|quota.*exceed|insufficient.*capacity|capacity.*unavailable' "${deploy_log}"; then
  category="quota-or-capacity"
elif grep -Eiq 'alreadyexists|already exists' "${deploy_log}"; then
  category="resource-conflict"
elif grep -Eiq 'validationerror|invalidparameter|invalid request' "${deploy_log}"; then
  category="invalid-configuration"
elif grep -Eiq 'rollback_(in_progress|failed|complete)|rollback in progress' "${deploy_log}"; then
  category="cloudformation-rollback"
fi

printf 'Production CDK deploy failure category: %s.\n' "${category}"
