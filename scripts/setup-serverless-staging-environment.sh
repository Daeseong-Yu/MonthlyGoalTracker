#!/usr/bin/env bash
set -euo pipefail

environment_name="serverless-staging"
apply=false

usage() {
  cat <<'USAGE'
Usage:
  AWS_REGION=... \
  AWS_ROLE_TO_ASSUME=... \
  SERVERLESS_EMAIL_FROM=... \
    scripts/setup-serverless-staging-environment.sh [--apply]

Optional custom domain:
  SERVERLESS_DOMAIN_NAME=...
  SERVERLESS_CERTIFICATE_ARN=...

Optional app base URL:
  SERVERLESS_SITE_BASE_URL=...

When SERVERLESS_SITE_BASE_URL and SERVERLESS_DOMAIN_NAME are both omitted,
the staging workflow uses a placeholder app base URL for a first-pass generated
CloudFront deployment. Configure the generated HTTPS origin afterward and
rerun the workflow before email action URL smoke.

Optional authenticated smoke:
  SERVERLESS_SMOKE_EMAIL=...
  SERVERLESS_SMOKE_PASSWORD=...

Options:
  --apply                 Create/update the GitHub environment variables and secrets.
  --environment NAME      Target GitHub environment name. Default: serverless-staging.

Default mode is dry-run. Values are never printed.
USAGE
}

while (($# > 0)); do
  case "$1" in
    --apply)
      apply=true
      ;;
    --environment)
      shift
      if (($# == 0)); then
        echo "--environment requires a value" >&2
        exit 1
      fi
      environment_name="$1"
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

env_present() {
  local name="$1"
  [[ -n "${!name:-}" ]]
}

missing=()
for required_name in AWS_REGION AWS_ROLE_TO_ASSUME SERVERLESS_EMAIL_FROM; do
  if ! env_present "${required_name}"; then
    missing+=("${required_name}")
  fi
done

if env_present SERVERLESS_DOMAIN_NAME && ! env_present SERVERLESS_CERTIFICATE_ARN; then
  missing+=("SERVERLESS_CERTIFICATE_ARN")
fi

if env_present SERVERLESS_CERTIFICATE_ARN && ! env_present SERVERLESS_DOMAIN_NAME; then
  missing+=("SERVERLESS_DOMAIN_NAME")
fi

if env_present SERVERLESS_SMOKE_EMAIL && ! env_present SERVERLESS_SMOKE_PASSWORD; then
  missing+=("SERVERLESS_SMOKE_PASSWORD")
fi

if env_present SERVERLESS_SMOKE_PASSWORD && ! env_present SERVERLESS_SMOKE_EMAIL; then
  missing+=("SERVERLESS_SMOKE_EMAIL")
fi

if ((${#missing[@]} > 0)); then
  printf 'Missing required environment values:\n' >&2
  printf -- '- %s\n' "${missing[@]}" >&2
  exit 1
fi

if [[ ! "${SERVERLESS_EMAIL_FROM}" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]; then
  echo "SERVERLESS_EMAIL_FROM must be a valid email address." >&2
  exit 1
fi

if env_present SERVERLESS_DOMAIN_NAME; then
  if [[ "${SERVERLESS_DOMAIN_NAME}" == *"://"* || "${SERVERLESS_DOMAIN_NAME}" == *"/"* ]]; then
    echo "SERVERLESS_DOMAIN_NAME must be a DNS host name without scheme or path." >&2
    exit 1
  fi
  if [[ "${SERVERLESS_CERTIFICATE_ARN}" != arn:*:acm:us-east-1:*:certificate/* ]]; then
    echo "SERVERLESS_CERTIFICATE_ARN must reference an ACM certificate in us-east-1 for CloudFront." >&2
    exit 1
  fi
fi

if env_present SERVERLESS_SITE_BASE_URL; then
  site_base_url="${SERVERLESS_SITE_BASE_URL%/}"
  if [[ "${site_base_url}" =~ ^https://([^/?#]+)$ ]]; then
    site_host="${BASH_REMATCH[1]}"
  else
    echo "SERVERLESS_SITE_BASE_URL must be an absolute HTTPS origin without path, query, or fragment." >&2
    exit 1
  fi
  if env_present SERVERLESS_DOMAIN_NAME && [[ "${site_host,,}" != "${SERVERLESS_DOMAIN_NAME,,}" ]]; then
    echo "SERVERLESS_SITE_BASE_URL host must match SERVERLESS_DOMAIN_NAME when custom domain is configured." >&2
    exit 1
  fi
fi

variable_names=(AWS_REGION AWS_ROLE_TO_ASSUME SERVERLESS_EMAIL_FROM)
if env_present SERVERLESS_SITE_BASE_URL; then
  variable_names+=(SERVERLESS_SITE_BASE_URL)
fi
if env_present SERVERLESS_DOMAIN_NAME; then
  variable_names+=(SERVERLESS_DOMAIN_NAME)
fi

secret_names=()
if env_present SERVERLESS_CERTIFICATE_ARN; then
  secret_names+=(SERVERLESS_CERTIFICATE_ARN)
fi
if env_present SERVERLESS_SMOKE_EMAIL; then
  secret_names+=(SERVERLESS_SMOKE_EMAIL SERVERLESS_SMOKE_PASSWORD)
fi

if [[ "${apply}" != "true" ]]; then
  echo "Dry run: would ensure GitHub environment '${environment_name}'."
  printf 'Variable names:\n'
  printf -- '- %s\n' "${variable_names[@]}"
  if ((${#secret_names[@]} > 0)); then
    printf 'Secret names:\n'
    printf -- '- %s\n' "${secret_names[@]}"
  fi
  if ! env_present SERVERLESS_SITE_BASE_URL && ! env_present SERVERLESS_DOMAIN_NAME; then
    echo "SERVERLESS_SITE_BASE_URL omitted: staging first-pass deploy will use a placeholder app base URL."
  fi
  echo "Run again with --apply to create/update names. Values are not printed."
  exit 0
fi

require_command gh

gh api --method PUT "repos/:owner/:repo/environments/${environment_name}" --silent

for variable_name in "${variable_names[@]}"; do
  printf '%s' "${!variable_name}" | gh variable set "${variable_name}" --env "${environment_name}" >/dev/null
  echo "Set GitHub environment variable name: ${variable_name}"
done

for secret_name in "${secret_names[@]}"; do
  printf '%s' "${!secret_name}" | gh secret set "${secret_name}" --env "${environment_name}" >/dev/null
  echo "Set GitHub environment secret name: ${secret_name}"
done

scripts/check-serverless-staging-readiness.sh "${environment_name}"
