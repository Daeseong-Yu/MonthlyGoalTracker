#!/usr/bin/env bash
set -euo pipefail

: "${DEPLOY_ARTIFACT_BUCKET:?DEPLOY_ARTIFACT_BUCKET is required}"
: "${RELEASE_ID:?RELEASE_ID is required}"
: "${AWS_REGION:?AWS_REGION is required}"

case "$(uname -m)" in
  x86_64)
    artifact="monthly-goal-api-linux-amd64"
    ;;
  aarch64|arm64)
    artifact="monthly-goal-api-linux-arm64"
    ;;
  *)
    echo "unsupported server architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

staging="/tmp/monthly-goal-tracker-${RELEASE_ID}-backend"
release_root="/opt/monthly-goal-tracker/backend/releases"
release_dir="${release_root}/${RELEASE_ID}"
current_binary="/opt/monthly-goal-tracker/backend/monthly-goal-api"
api_env_file="/etc/monthly-goal-tracker/api.env"
previous_binary=""

cleanup() {
  rm -rf "${staging}"
}
trap cleanup EXIT

load_api_env() {
  if [ ! -r "${api_env_file}" ]; then
    echo "API environment file is not readable: ${api_env_file}" >&2
    exit 1
  fi

  unset APP_HOST APP_PORT DATABASE_URL APP_BASIC_AUTH_USERNAME APP_BASIC_AUTH_PASSWORD_HASH
  unset APP_SESSION_COOKIE_NAME APP_CSRF_COOKIE_NAME APP_SESSION_TTL_HOURS
  unset APP_COOKIE_SECURE APP_COOKIE_SAMESITE APP_SIGNUP_RATE_LIMIT_PER_MINUTE
  unset APP_LOGIN_RATE_LIMIT_PER_MINUTE APP_AUTH_RATE_LIMIT_MAX_BUCKETS
  unset APP_TRUSTED_PROXIES APP_LEGACY_CLAIM_TOKEN
  unset APP_EMAIL_VERIFICATION_TTL_HOURS APP_PASSWORD_RESET_TTL_HOURS
  unset APP_EMAIL_FROM APP_SMTP_HOST APP_SMTP_PORT APP_SMTP_USERNAME APP_SMTP_PASSWORD
  unset APP_EMAIL_VERIFICATION_BASE_URL APP_PASSWORD_RESET_BASE_URL
  unset PGHOST PGPORT PGDATABASE PGUSER PGPASSWORD PGPASSFILE PGSERVICE PGSERVICEFILE
  unset PGSSLMODE PGSSLCERT PGSSLKEY PGSSLROOTCERT PGSSLPASSWORD PGAPPNAME
  unset PGCONNECT_TIMEOUT PGSSLSNI PGTARGETSESSIONATTRS

  while IFS= read -r line || [ -n "${line}" ]; do
    case "${line}" in
      ""|\#*)
        continue
        ;;
    esac

    key="${line%%=*}"
    value="${line#*=}"

    case "${key}" in
      APP_HOST|APP_PORT|DATABASE_URL|APP_BASIC_AUTH_USERNAME|APP_BASIC_AUTH_PASSWORD_HASH|APP_SESSION_COOKIE_NAME|APP_CSRF_COOKIE_NAME|APP_SESSION_TTL_HOURS|APP_COOKIE_SECURE|APP_COOKIE_SAMESITE|APP_SIGNUP_RATE_LIMIT_PER_MINUTE|APP_LOGIN_RATE_LIMIT_PER_MINUTE|APP_AUTH_RATE_LIMIT_MAX_BUCKETS|APP_TRUSTED_PROXIES|APP_LEGACY_CLAIM_TOKEN|APP_EMAIL_VERIFICATION_TTL_HOURS|APP_PASSWORD_RESET_TTL_HOURS|APP_EMAIL_FROM|APP_SMTP_HOST|APP_SMTP_PORT|APP_SMTP_USERNAME|APP_SMTP_PASSWORD|APP_EMAIL_VERIFICATION_BASE_URL|APP_PASSWORD_RESET_BASE_URL)
        export "${key}=${value}"
        ;;
    esac
  done < "${api_env_file}"

  if [ -z "${DATABASE_URL:-}" ]; then
    echo "DATABASE_URL is required in ${api_env_file}" >&2
    exit 1
  fi
  if [ "${APP_COOKIE_SECURE:-}" != "true" ]; then
    echo "APP_COOKIE_SECURE=true is required in ${api_env_file}" >&2
    exit 1
  fi
}

rm -rf "${staging}"
mkdir -p "${staging}"
aws s3 cp "s3://${DEPLOY_ARTIFACT_BUCKET}/releases/${RELEASE_ID}/backend/${artifact}" "${staging}/monthly-goal-api" --region "${AWS_REGION}"

install -d -m 0755 "${release_root}"
rm -rf "${release_dir}"
mkdir -p "${release_dir}"
install -m 0755 "${staging}/monthly-goal-api" "${release_dir}/monthly-goal-api"

load_api_env
"${release_dir}/monthly-goal-api" --migrate-only

if [ -L "${current_binary}" ]; then
  previous_binary="$(readlink -f "${current_binary}")"
elif [ -f "${current_binary}" ]; then
  previous_dir="${release_root}/pre-cicd-$(date +%Y%m%d%H%M%S)"
  mkdir -p "${previous_dir}"
  cp "${current_binary}" "${previous_dir}/monthly-goal-api"
  chmod 0755 "${previous_dir}/monthly-goal-api"
  previous_binary="${previous_dir}/monthly-goal-api"
fi

ln -sfn "${release_dir}/monthly-goal-api" "${current_binary}.next"
mv -Tf "${current_binary}.next" "${current_binary}"

if systemctl restart monthly-goal-api; then
  for _ in $(seq 1 10); do
    if curl -fsS http://127.0.0.1:8080/api/health >/dev/null; then
      cd "${release_root}"
      ls -1dt */ 2>/dev/null | tail -n +6 | while IFS= read -r old_release; do
        rm -rf -- "${release_root}/${old_release%/}"
      done
      exit 0
    fi
    sleep 2
  done
fi

if [ -n "${previous_binary}" ] && [ -x "${previous_binary}" ]; then
  ln -sfn "${previous_binary}" "${current_binary}.next"
  mv -Tf "${current_binary}.next" "${current_binary}"
  systemctl restart monthly-goal-api || true
fi

systemctl status monthly-goal-api --no-pager || true
exit 1
