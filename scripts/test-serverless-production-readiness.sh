#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_SCRIPT="${ROOT_DIR}/scripts/check-serverless-production-readiness.sh"
valid_prepare_role="arn:aws:iam::000000000000:role/monthly-goal-tracker-serverless-production-prepare"
valid_migrate_role="arn:aws:iam::000000000000:role/monthly-goal-tracker-serverless-production-migrate"
valid_certificate="arn:aws:acm:us-east-1:000000000000:certificate/example"

# Deployment workflows define these at job scope; each test case supplies its own values.
unset \
  AWS_REGION \
  AWS_ROLE_TO_ASSUME \
  SERVERLESS_EMAIL_FROM \
  SERVERLESS_SITE_BASE_URL \
  SERVERLESS_DOMAIN_NAME \
  SERVERLESS_SIGNUP_DISABLED \
  SERVERLESS_CERTIFICATE_ARN \
  SERVERLESS_SMOKE_EMAIL \
  SERVERLESS_SMOKE_PASSWORD

run_success() {
  local name="$1"
  shift
  if ! output="$("$@" 2>&1)"; then
    echo "Expected success: ${name}" >&2
    echo "${output}" >&2
    exit 1
  fi
}

run_failure() {
  local name="$1"
  local expected="$2"
  shift 2
  set +e
  output="$("$@" 2>&1)"
  status=$?
  set -e
  if [[ "${status}" -eq 0 ]]; then
    echo "Expected failure: ${name}" >&2
    exit 1
  fi
  if [[ "${output}" != *"${expected}"* ]]; then
    echo "Unexpected failure message for ${name}" >&2
    echo "Expected to contain: ${expected}" >&2
    echo "${output}" >&2
    exit 1
  fi
}

prepare_env=(
  AWS_REGION=us-east-1
  AWS_ROLE_TO_ASSUME="${valid_prepare_role}"
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid
  SERVERLESS_SITE_BASE_URL=https://portfolio.example.invalid
  SERVERLESS_DOMAIN_NAME=portfolio.example.invalid
  SERVERLESS_SIGNUP_DISABLED=false
  SERVERLESS_CERTIFICATE_ARN="${valid_certificate}"
)

run_success "valid prepare contract" env "${prepare_env[@]}" \
  "${CHECK_SCRIPT}" --mode prepare --dry-run

run_failure "production signup disabled" \
  "SERVERLESS_SIGNUP_DISABLED must be false" \
  env "${prepare_env[@]}" SERVERLESS_SIGNUP_DISABLED=true \
  "${CHECK_SCRIPT}" --mode prepare --dry-run

run_failure "site and custom domain mismatch" \
  "SERVERLESS_SITE_BASE_URL host must match SERVERLESS_DOMAIN_NAME" \
  env "${prepare_env[@]}" SERVERLESS_SITE_BASE_URL=https://different.example.invalid \
  "${CHECK_SCRIPT}" --mode prepare --dry-run

run_failure "missing certificate" \
  "Missing required environment value: SERVERLESS_CERTIFICATE_ARN" \
  env "${prepare_env[@]}" SERVERLESS_CERTIFICATE_ARN= \
  "${CHECK_SCRIPT}" --mode prepare --dry-run

run_success "valid migration contract" env \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_migrate_role}" \
  "${CHECK_SCRIPT}" --mode migrate --dry-run

run_success "valid public smoke contract" env \
  SERVERLESS_SITE_BASE_URL=https://portfolio.example.invalid \
  SERVERLESS_DOMAIN_NAME=portfolio.example.invalid \
  "${CHECK_SCRIPT}" --mode public --dry-run

run_failure "public smoke AWS role forbidden" \
  "Public smoke readiness must not include AWS_ROLE_TO_ASSUME" \
  env \
  SERVERLESS_SITE_BASE_URL=https://portfolio.example.invalid \
  SERVERLESS_DOMAIN_NAME=portfolio.example.invalid \
  AWS_ROLE_TO_ASSUME="${valid_prepare_role}" \
  "${CHECK_SCRIPT}" --mode public --dry-run

run_failure "partial public smoke credential pair" \
  "SERVERLESS_SMOKE_EMAIL and SERVERLESS_SMOKE_PASSWORD must be configured together" \
  env \
  SERVERLESS_SITE_BASE_URL=https://portfolio.example.invalid \
  SERVERLESS_DOMAIN_NAME=portfolio.example.invalid \
  SERVERLESS_SMOKE_EMAIL=smoke@example.invalid \
  "${CHECK_SCRIPT}" --mode public --dry-run

echo "Serverless production readiness helper tests passed."
