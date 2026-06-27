#!/usr/bin/env bash
set -euo pipefail

dry_run=false
require_ses_production_access=false

usage() {
  cat <<'USAGE'
Usage:
  AWS_REGION=... AWS_ROLE_TO_ASSUME=... SERVERLESS_EMAIL_FROM=... scripts/check-serverless-aws-readiness.sh [options]

Optional:
  SERVERLESS_DOMAIN_NAME=...
  SERVERLESS_CERTIFICATE_ARN=...

Options:
  --dry-run                        Validate local inputs without calling AWS APIs.
  --require-ses-production-access  Fail if SES production access is not enabled.

This script performs read-only AWS readiness checks and does not print account,
identity, certificate, or secret values.
USAGE
}

while (($# > 0)); do
  case "$1" in
    --dry-run)
      dry_run=true
      ;;
    --require-ses-production-access)
      require_ses_production_access=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "Missing required environment value: ${name}" >&2
    exit 1
  fi
}

validate_inputs() {
  if [[ ! "${AWS_REGION}" =~ ^[a-z]{2}(-gov)?-[a-z]+-[0-9]+$ ]]; then
    aws_readiness_error "AWS_REGION must be an AWS region name."
  fi

  if [[ "${AWS_ROLE_TO_ASSUME}" != arn:*:iam::*:role/* || "${AWS_ROLE_TO_ASSUME}" =~ [[:space:]] ]]; then
    aws_readiness_error "AWS_ROLE_TO_ASSUME must be an IAM role ARN."
  fi

  if [[ ! "${SERVERLESS_EMAIL_FROM}" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]; then
    aws_readiness_error "SERVERLESS_EMAIL_FROM must be a valid email address."
  fi

  if [[ -n "${SERVERLESS_DOMAIN_NAME:-}" || -n "${SERVERLESS_CERTIFICATE_ARN:-}" ]]; then
    if [[ -z "${SERVERLESS_DOMAIN_NAME:-}" || -z "${SERVERLESS_CERTIFICATE_ARN:-}" ]]; then
      aws_readiness_error "SERVERLESS_DOMAIN_NAME and SERVERLESS_CERTIFICATE_ARN must be configured together."
    fi
  fi

  if [[ -n "${SERVERLESS_DOMAIN_NAME:-}" ]]; then
    if [[ "${SERVERLESS_DOMAIN_NAME}" == *"://"* ||
      "${SERVERLESS_DOMAIN_NAME}" == *"/"* ||
      "${SERVERLESS_DOMAIN_NAME}" =~ [[:space:]] ]]; then
      aws_readiness_error "SERVERLESS_DOMAIN_NAME must be a DNS host name without scheme or path."
    fi
  fi

  if [[ -n "${SERVERLESS_CERTIFICATE_ARN:-}" &&
    "${SERVERLESS_CERTIFICATE_ARN}" != arn:*:acm:us-east-1:*:certificate/* ]]; then
    aws_readiness_error "SERVERLESS_CERTIFICATE_ARN must reference an ACM certificate in us-east-1."
  fi
}

aws_readiness_error() {
  echo "$1" >&2
  exit 1
}

require_env AWS_REGION
require_env AWS_ROLE_TO_ASSUME
require_env SERVERLESS_EMAIL_FROM
validate_inputs

if [[ "${dry_run}" == "true" ]]; then
  echo "Dry run: AWS readiness input names and formats are valid."
  exit 0
fi

require_command aws

if ! aws sts get-caller-identity --output json >/dev/null 2>&1; then
  aws_readiness_error "AWS credential readiness failed: configure the deployment role before staging deploy."
fi

if ! aws ssm get-parameter \
  --name /cdk-bootstrap/hnb659fds/version \
  --region "${AWS_REGION}" \
  --output json >/dev/null 2>&1; then
  aws_readiness_error "CDK bootstrap readiness failed: bootstrap the target AWS region before staging deploy."
fi

if [[ "${require_ses_production_access}" == "true" ]]; then
  production_access="$(aws sesv2 get-account \
    --region "${AWS_REGION}" \
    --query 'ProductionAccessEnabled' \
    --output text 2>/dev/null || true)"
  if [[ "${production_access}" != "True" && "${production_access}" != "true" ]]; then
    aws_readiness_error "SES account readiness failed: enable SES production access before production launch."
  fi
fi

sender_domain="${SERVERLESS_EMAIL_FROM#*@}"
identity_verified=false
for identity in "${SERVERLESS_EMAIL_FROM}" "${sender_domain}"; do
  verified_status="$(aws sesv2 get-email-identity \
    --email-identity "${identity}" \
    --region "${AWS_REGION}" \
    --query 'VerifiedForSendingStatus' \
    --output text 2>/dev/null || true)"
  if [[ "${verified_status}" == "True" || "${verified_status}" == "true" ]]; then
    identity_verified=true
    break
  fi
done

if [[ "${identity_verified}" != "true" ]]; then
  aws_readiness_error "SES sender readiness failed: verify the sender address or domain before staging deploy."
fi

if [[ -n "${SERVERLESS_CERTIFICATE_ARN:-}" ]]; then
  certificate_status="$(aws acm describe-certificate \
    --region us-east-1 \
    --certificate-arn "${SERVERLESS_CERTIFICATE_ARN}" \
    --query 'Certificate.Status' \
    --output text 2>/dev/null || true)"
  if [[ "${certificate_status}" != "ISSUED" ]]; then
    aws_readiness_error "Custom domain readiness failed: issue the CloudFront ACM certificate in us-east-1."
  fi
fi

echo "Serverless AWS readiness checks passed."
