#!/usr/bin/env bash
set -euo pipefail

environment_name="${1:-serverless-staging}"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

contains_name() {
  local needle="$1"
  local haystack="$2"

  grep -Fxq "${needle}" <<<"${haystack}"
}

variable_value() {
  local name="$1"

  gh api --paginate "repos/:owner/:repo/environments/${environment_name}/variables" \
    --jq ".variables[] | select(.name == \"${name}\") | .value"
}

validate_variable_values() {
  local aws_region="$1"
  local aws_role_to_assume="$2"
  local email_from="$3"
  local site_base_url="$4"
  local domain_name="$5"

  if [[ ! "${aws_region}" =~ ^[a-z]{2}(-gov)?-[a-z]+-[0-9]+$ ]]; then
    echo "AWS_REGION variable must be an AWS region name." >&2
    exit 1
  fi

  if [[ "${aws_role_to_assume}" != arn:*:iam::*:role/* || "${aws_role_to_assume}" =~ [[:space:]] ]]; then
    echo "AWS_ROLE_TO_ASSUME variable must be an IAM role ARN." >&2
    exit 1
  fi

  if [[ ! "${email_from}" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]; then
    echo "SERVERLESS_EMAIL_FROM variable must be a valid email address." >&2
    exit 1
  fi

  if [[ -n "${domain_name}" ]]; then
    if [[ "${domain_name}" == *"://"* || "${domain_name}" == *"/"* || "${domain_name}" =~ [[:space:]] ]]; then
      echo "SERVERLESS_DOMAIN_NAME variable must be a DNS host name without scheme or path." >&2
      exit 1
    fi
  fi

  if [[ -n "${site_base_url}" ]]; then
    if [[ "${site_base_url%/}" =~ ^https://([^/?#]+)$ ]]; then
      site_host="${BASH_REMATCH[1]}"
    else
      echo "SERVERLESS_SITE_BASE_URL variable must be an absolute HTTPS origin without path, query, or fragment." >&2
      exit 1
    fi

    if [[ -n "${domain_name}" && "${site_host,,}" != "${domain_name,,}" ]]; then
      echo "SERVERLESS_SITE_BASE_URL host must match SERVERLESS_DOMAIN_NAME when custom domain is configured." >&2
      exit 1
    fi
  fi
}

require_command gh

environment_names="$(gh api --paginate repos/:owner/:repo/environments --jq '.environments[].name')"
if ! contains_name "${environment_name}" "${environment_names}"; then
  echo "Missing GitHub environment: ${environment_name}" >&2
  exit 1
fi

variable_names="$(gh api --paginate "repos/:owner/:repo/environments/${environment_name}/variables" --jq '.variables[].name')"
secret_names="$(gh api --paginate "repos/:owner/:repo/environments/${environment_name}/secrets" --jq '.secrets[].name')"

missing=()
for required_variable in AWS_REGION AWS_ROLE_TO_ASSUME SERVERLESS_EMAIL_FROM; do
  if ! contains_name "${required_variable}" "${variable_names}"; then
    missing+=("variable:${required_variable}")
  fi
done

if contains_name "SERVERLESS_DOMAIN_NAME" "${variable_names}" &&
  ! contains_name "SERVERLESS_CERTIFICATE_ARN" "${secret_names}"; then
  missing+=("secret:SERVERLESS_CERTIFICATE_ARN")
fi

if contains_name "SERVERLESS_CERTIFICATE_ARN" "${secret_names}" &&
  ! contains_name "SERVERLESS_DOMAIN_NAME" "${variable_names}"; then
  missing+=("variable:SERVERLESS_DOMAIN_NAME")
fi

if contains_name "SERVERLESS_SMOKE_EMAIL" "${secret_names}" &&
  ! contains_name "SERVERLESS_SMOKE_PASSWORD" "${secret_names}"; then
  missing+=("secret:SERVERLESS_SMOKE_PASSWORD")
fi

if contains_name "SERVERLESS_SMOKE_PASSWORD" "${secret_names}" &&
  ! contains_name "SERVERLESS_SMOKE_EMAIL" "${secret_names}"; then
  missing+=("secret:SERVERLESS_SMOKE_EMAIL")
fi

if ((${#missing[@]} > 0)); then
  printf 'Missing serverless staging configuration names:\n' >&2
  printf -- '- %s\n' "${missing[@]}" >&2
  exit 1
fi

validate_variable_values \
  "$(variable_value AWS_REGION)" \
  "$(variable_value AWS_ROLE_TO_ASSUME)" \
  "$(variable_value SERVERLESS_EMAIL_FROM)" \
  "$(variable_value SERVERLESS_SITE_BASE_URL)" \
  "$(variable_value SERVERLESS_DOMAIN_NAME)"

echo "Serverless staging readiness names and variable formats are present."
