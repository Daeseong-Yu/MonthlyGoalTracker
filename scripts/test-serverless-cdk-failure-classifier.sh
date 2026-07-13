#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
CLASSIFIER="${ROOT_DIR}/scripts/classify-serverless-cdk-failure.sh"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/monthly-goal-tracker-cdk-failure.XXXXXX")"

cleanup() {
  rm -rf -- "${TMP_DIR}"
}

trap cleanup EXIT

run_case() {
  local name="$1"
  local expected_category="$2"
  local log_content="$3"
  local deploy_log="${TMP_DIR}/${name}.log"
  local output

  printf '%s\n' "${log_content}" > "${deploy_log}"
  if ! output="$(bash "${CLASSIFIER}" "${deploy_log}")"; then
    echo "Expected CDK failure classifier to succeed: ${name}." >&2
    exit 1
  fi

  if [[ "${output}" != "Production CDK deploy failure category: ${expected_category}." ]]; then
    echo "Unexpected CDK failure category: ${name}." >&2
    exit 1
  fi
}

run_case \
  "free-plan-backup-retention" \
  "free-plan-backup-retention" \
  "Resource handler returned message: The specified backup retention period exceeds the maximum available to free tier customers."
run_case \
  "authorization" \
  "authorization" \
  "AccessDenied: the deployment role is not authorized to perform the requested action."
run_case \
  "quota-or-capacity" \
  "quota-or-capacity" \
  "Resource creation failed because the service quota was exceeded."
run_case \
  "resource-conflict" \
  "resource-conflict" \
  "ResourceAlreadyExists: the requested resource already exists."
run_case \
  "invalid-configuration" \
  "invalid-configuration" \
  "InvalidParameterValue: the supplied resource configuration is not valid."
run_case \
  "cloudformation-rollback" \
  "cloudformation-rollback" \
  "The stack entered ROLLBACK_COMPLETE after resource creation failed."
run_case \
  "unknown" \
  "unknown" \
  "The deployment failed without a recognized provider message."

echo "Serverless CDK failure classifier tests passed."
