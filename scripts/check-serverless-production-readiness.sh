#!/usr/bin/env bash
set -euo pipefail

mode="prepare"
environment_name=""
dry_run=false

usage() {
  cat <<'USAGE'
Usage:
  scripts/check-serverless-production-readiness.sh --mode prepare|migrate|public [options]

Options:
  --mode MODE          Readiness contract to validate. Default: prepare.
  --environment NAME   Override the default GitHub environment name.
  --dry-run            Validate shell environment values without GitHub API calls.

Default GitHub environments:
  prepare  serverless-production-prepare
  migrate  serverless-production-migrate
  public   serverless-production-public

Values and secret contents are never printed.
USAGE
}

while (($# > 0)); do
  case "$1" in
    --mode)
      shift
      if (($# == 0)); then
        echo "--mode requires a value" >&2
        exit 1
      fi
      mode="$1"
      ;;
    --environment)
      shift
      if (($# == 0)); then
        echo "--environment requires a value" >&2
        exit 1
      fi
      environment_name="$1"
      ;;
    --dry-run)
      dry_run=true
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

case "${mode}" in
  prepare|migrate|public) ;;
  *)
    echo "--mode must be prepare, migrate, or public" >&2
    exit 1
    ;;
esac

if [[ -z "${environment_name}" ]]; then
  environment_name="serverless-production-${mode}"
fi

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

env_present() {
  local name="$1"
  [[ -n "${!name:-}" ]]
}

contains_name() {
  local needle="$1"
  local haystack="$2"
  grep -Fxq "${needle}" <<<"${haystack}"
}

validate_region() {
  [[ "$1" =~ ^[a-z]{2}(-gov)?-[a-z]+-[0-9]+$ ]] || {
    echo "AWS_REGION must be an AWS region name." >&2
    exit 1
  }
}

validate_role() {
  [[ "$1" == arn:*:iam::*:role/* && ! "$1" =~ [[:space:]] ]] || {
    echo "AWS_ROLE_TO_ASSUME must be an IAM role ARN." >&2
    exit 1
  }
}

validate_https_origin() {
  local value="${1%/}"
  if [[ ! "${value}" =~ ^https://([^/?#]+)$ ]]; then
    echo "SERVERLESS_SITE_BASE_URL must be an absolute HTTPS origin without path, query, or fragment." >&2
    exit 1
  fi
  printf '%s' "${BASH_REMATCH[1]}"
}

validate_domain() {
  if [[ -z "$1" || "$1" == *"://"* || "$1" == *"/"* || "$1" =~ [[:space:]] ]]; then
    echo "SERVERLESS_DOMAIN_NAME must be a DNS host name without scheme or path." >&2
    exit 1
  fi
}

validate_prepare_values() {
  local region="$1"
  local role="$2"
  local email_from="$3"
  local site_base_url="$4"
  local domain_name="$5"
  local signup_disabled="$6"
  local certificate_arn="${7:-}"
  local require_certificate_value="${8:-false}"

  validate_region "${region}"
  validate_role "${role}"
  if [[ ! "${email_from}" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]; then
    echo "SERVERLESS_EMAIL_FROM must be a valid email address." >&2
    exit 1
  fi
  validate_domain "${domain_name}"
  site_host="$(validate_https_origin "${site_base_url}")"
  if [[ "${site_host,,}" != "${domain_name,,}" ]]; then
    echo "SERVERLESS_SITE_BASE_URL host must match SERVERLESS_DOMAIN_NAME." >&2
    exit 1
  fi
  if [[ "${signup_disabled}" != "false" ]]; then
    echo "SERVERLESS_SIGNUP_DISABLED must be false for production launch readiness." >&2
    exit 1
  fi
  if [[ "${require_certificate_value}" == "true" && "${certificate_arn}" != arn:*:acm:us-east-1:*:certificate/* ]]; then
    echo "SERVERLESS_CERTIFICATE_ARN must reference an ACM certificate in us-east-1." >&2
    exit 1
  fi
}

validate_migrate_values() {
  validate_region "$1"
  validate_role "$2"
}

validate_public_values() {
  local site_base_url="$1"
  local domain_name="$2"
  validate_domain "${domain_name}"
  site_host="$(validate_https_origin "${site_base_url}")"
  if [[ "${site_host,,}" != "${domain_name,,}" ]]; then
    echo "SERVERLESS_SITE_BASE_URL host must match SERVERLESS_DOMAIN_NAME." >&2
    exit 1
  fi
}

if [[ "${dry_run}" == "true" ]]; then
  case "${mode}" in
    prepare)
      required=(AWS_REGION AWS_ROLE_TO_ASSUME SERVERLESS_EMAIL_FROM SERVERLESS_SITE_BASE_URL SERVERLESS_DOMAIN_NAME SERVERLESS_SIGNUP_DISABLED SERVERLESS_CERTIFICATE_ARN)
      for name in "${required[@]}"; do
        if ! env_present "${name}"; then
          echo "Missing required environment value: ${name}" >&2
          exit 1
        fi
      done
      validate_prepare_values \
        "${AWS_REGION}" \
        "${AWS_ROLE_TO_ASSUME}" \
        "${SERVERLESS_EMAIL_FROM}" \
        "${SERVERLESS_SITE_BASE_URL}" \
        "${SERVERLESS_DOMAIN_NAME}" \
        "${SERVERLESS_SIGNUP_DISABLED}" \
        "${SERVERLESS_CERTIFICATE_ARN}" \
        true
      ;;
    migrate)
      : "${AWS_REGION:?Missing required environment value: AWS_REGION}"
      : "${AWS_ROLE_TO_ASSUME:?Missing required environment value: AWS_ROLE_TO_ASSUME}"
      validate_migrate_values "${AWS_REGION}" "${AWS_ROLE_TO_ASSUME}"
      ;;
    public)
      : "${SERVERLESS_SITE_BASE_URL:?Missing required environment value: SERVERLESS_SITE_BASE_URL}"
      : "${SERVERLESS_DOMAIN_NAME:?Missing required environment value: SERVERLESS_DOMAIN_NAME}"
      validate_public_values "${SERVERLESS_SITE_BASE_URL}" "${SERVERLESS_DOMAIN_NAME}"
      if env_present AWS_ROLE_TO_ASSUME; then
        echo "Public smoke readiness must not include AWS_ROLE_TO_ASSUME." >&2
        exit 1
      fi
      if env_present SERVERLESS_SMOKE_EMAIL && ! env_present SERVERLESS_SMOKE_PASSWORD; then
        echo "SERVERLESS_SMOKE_EMAIL and SERVERLESS_SMOKE_PASSWORD must be configured together." >&2
        exit 1
      fi
      if env_present SERVERLESS_SMOKE_PASSWORD && ! env_present SERVERLESS_SMOKE_EMAIL; then
        echo "SERVERLESS_SMOKE_EMAIL and SERVERLESS_SMOKE_PASSWORD must be configured together." >&2
        exit 1
      fi
      ;;
  esac
  echo "Serverless production ${mode} dry-run readiness passed."
  exit 0
fi

require_command gh

environment_names="$(gh api --paginate repos/:owner/:repo/environments --jq '.environments[].name')"
if ! contains_name "${environment_name}" "${environment_names}"; then
  echo "Missing GitHub environment for serverless production ${mode}." >&2
  exit 1
fi

variable_names="$(gh api --paginate "repos/:owner/:repo/environments/${environment_name}/variables" --jq '.variables[].name')"
secret_names="$(gh api --paginate "repos/:owner/:repo/environments/${environment_name}/secrets" --jq '.secrets[].name')"

variable_value() {
  local name="$1"
  gh api --paginate "repos/:owner/:repo/environments/${environment_name}/variables" \
    --jq ".variables[] | select(.name == \"${name}\") | .value"
}

require_variable_name() {
  if ! contains_name "$1" "${variable_names}"; then
    echo "Missing required GitHub environment variable name: $1" >&2
    exit 1
  fi
}

case "${mode}" in
  prepare)
    for name in AWS_REGION AWS_ROLE_TO_ASSUME SERVERLESS_EMAIL_FROM SERVERLESS_SITE_BASE_URL SERVERLESS_DOMAIN_NAME SERVERLESS_SIGNUP_DISABLED; do
      require_variable_name "${name}"
    done
    if ! contains_name SERVERLESS_CERTIFICATE_ARN "${secret_names}"; then
      echo "Missing required GitHub environment secret name: SERVERLESS_CERTIFICATE_ARN" >&2
      exit 1
    fi
    validate_prepare_values \
      "$(variable_value AWS_REGION)" \
      "$(variable_value AWS_ROLE_TO_ASSUME)" \
      "$(variable_value SERVERLESS_EMAIL_FROM)" \
      "$(variable_value SERVERLESS_SITE_BASE_URL)" \
      "$(variable_value SERVERLESS_DOMAIN_NAME)" \
      "$(variable_value SERVERLESS_SIGNUP_DISABLED)"
    ;;
  migrate)
    require_variable_name AWS_REGION
    require_variable_name AWS_ROLE_TO_ASSUME
    validate_migrate_values "$(variable_value AWS_REGION)" "$(variable_value AWS_ROLE_TO_ASSUME)"
    ;;
  public)
    require_variable_name SERVERLESS_SITE_BASE_URL
    require_variable_name SERVERLESS_DOMAIN_NAME
    validate_public_values "$(variable_value SERVERLESS_SITE_BASE_URL)" "$(variable_value SERVERLESS_DOMAIN_NAME)"
    if contains_name AWS_ROLE_TO_ASSUME "${variable_names}"; then
      echo "Public smoke GitHub environment must not define AWS_ROLE_TO_ASSUME." >&2
      exit 1
    fi
    email_secret=false
    password_secret=false
    contains_name SERVERLESS_SMOKE_EMAIL "${secret_names}" && email_secret=true
    contains_name SERVERLESS_SMOKE_PASSWORD "${secret_names}" && password_secret=true
    if [[ "${email_secret}" != "${password_secret}" ]]; then
      echo "SERVERLESS_SMOKE_EMAIL and SERVERLESS_SMOKE_PASSWORD must be configured together." >&2
      exit 1
    fi
    ;;
esac

echo "Serverless production ${mode} readiness names and variable formats are present."
