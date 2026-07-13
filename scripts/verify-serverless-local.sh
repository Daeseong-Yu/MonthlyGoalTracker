#!/usr/bin/env bash
set -euo pipefail

include_docker=false
run_e2e=false
run_static_preview=false

usage() {
  cat <<'USAGE'
Usage:
  scripts/verify-serverless-local.sh [options]

Options:
  --docker          Also build Lambda Docker images for API, migration, and audit.
  --e2e             Also run local Playwright e2e. Deployed serverless smoke remains skipped locally.
  --static-preview  Also run preview smoke against the built frontend dist via vite preview.

Runs local checks for the serverless Lambda/PostgreSQL migration without
deploying AWS resources or printing secret values.
USAGE
}

while (($# > 0)); do
  case "$1" in
    --docker)
      include_docker=true
      ;;
    --e2e)
      run_e2e=true
      ;;
    --static-preview)
      run_static_preview=true
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

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
SYNTH_DIR="${TMPDIR:-/tmp}/monthly-goal-tracker-serverless-verify"
AUDIT_COUNTS_DRY_RUN_FILE="$(mktemp "${TMPDIR:-/tmp}/monthly-goal-tracker-audit-counts.XXXXXX.json")"

cleanup() {
  rm -f -- "${AUDIT_COUNTS_DRY_RUN_FILE}"
}

trap cleanup EXIT

cd "${ROOT_DIR}/backend"
go test ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o /tmp/monthly-goal-api-linux-amd64 ./cmd/api
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o /tmp/monthly-goal-api-linux-arm64 ./cmd/api
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o /tmp/monthly-goal-lambda-linux-arm64 ./cmd/lambda
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o /tmp/monthly-goal-migrate-linux-arm64 ./cmd/migrate
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o /tmp/monthly-goal-audit-linux-arm64 ./cmd/audit

if [[ "${include_docker}" == "true" ]]; then
  for app_command in lambda migrate audit; do
    image_name="monthly-goal-tracker-${app_command}:verify"
    docker build \
      --platform linux/arm64 \
      --build-arg "APP_COMMAND=${app_command}" \
      -f Dockerfile.lambda \
      -t "${image_name}" \
      .

    image_platform="$(docker image inspect "${image_name}" --format '{{.Os}}/{{.Architecture}}')"
    if [[ "${image_platform}" != "linux/arm64" ]]; then
      echo "Unexpected Lambda image platform for ${app_command}: ${image_platform}" >&2
      exit 1
    fi

    image_command="$(docker image inspect "${image_name}" --format '{{json .Config.Cmd}}')"
    if [[ "${image_command}" != '["bootstrap"]' ]]; then
      echo "Unexpected Lambda image command for ${app_command}." >&2
      exit 1
    fi
  done
fi

cd "${ROOT_DIR}/frontend"
pnpm test
pnpm build
"${ROOT_DIR}/scripts/check-frontend-dist.sh" "${ROOT_DIR}/frontend/dist"
if [[ "${run_e2e}" == "true" ]]; then
  pnpm test:e2e
fi
if [[ "${run_static_preview}" == "true" ]]; then
  PLAYWRIGHT_BASE_URL="http://127.0.0.1:4173" \
  PLAYWRIGHT_WEB_SERVER_COMMAND="pnpm preview --host 127.0.0.1 --port 4173" \
    pnpm exec playwright test e2e/smoke-preview.spec.ts
fi

cd "${ROOT_DIR}/infra"
npm test
rm -rf "${SYNTH_DIR}"
mkdir -p "${SYNTH_DIR}"
./node_modules/.bin/cdk synth --output "${SYNTH_DIR}/default" > "${SYNTH_DIR}/default.yaml"
./node_modules/.bin/cdk synth \
  --output "${SYNTH_DIR}/custom-domain" \
  -c domainName=serverless.example.invalid \
  -c certificateArn=arn:aws:acm:us-east-1:000000000000:certificate/example \
  > "${SYNTH_DIR}/custom-domain.yaml"

cd "${ROOT_DIR}"
bash -n \
  scripts/verify-serverless-local.sh \
  scripts/check-frontend-dist.sh \
  scripts/check-serverless-aws-readiness.sh \
  scripts/test-serverless-aws-readiness.sh \
  scripts/check-serverless-staging-readiness.sh \
  scripts/run-serverless-staging-workflow.sh \
  scripts/setup-serverless-staging-environment.sh \
  scripts/check-serverless-production-readiness.sh \
  scripts/test-serverless-production-readiness.sh

AWS_REGION=us-east-1 \
AWS_ROLE_TO_ASSUME=arn:aws:iam::000000000000:role/monthly-goal-tracker-serverless-staging \
SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
SERVERLESS_SITE_BASE_URL=https://serverless.example.invalid \
SERVERLESS_DOMAIN_NAME=serverless.example.invalid \
SERVERLESS_CERTIFICATE_ARN=arn:aws:acm:us-east-1:000000000000:certificate/example \
SERVERLESS_SMOKE_EMAIL=smoke@example.invalid \
SERVERLESS_SMOKE_PASSWORD=example-password \
  scripts/setup-serverless-staging-environment.sh >/dev/null
scripts/run-serverless-staging-workflow.sh >/dev/null
printf '{"users":0,"sessions":0,"goals":0,"memos":0,"checks":0}\n' > "${AUDIT_COUNTS_DRY_RUN_FILE}"
scripts/run-serverless-staging-workflow.sh \
  --run-migration \
  --audit-expected-counts-file "${AUDIT_COUNTS_DRY_RUN_FILE}" >/dev/null
AWS_REGION=us-east-1 \
AWS_ROLE_TO_ASSUME=arn:aws:iam::000000000000:role/monthly-goal-tracker-serverless-staging \
SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
SERVERLESS_DOMAIN_NAME=serverless.example.invalid \
SERVERLESS_CERTIFICATE_ARN=arn:aws:acm:us-east-1:000000000000:certificate/example \
  scripts/check-serverless-aws-readiness.sh --dry-run >/dev/null
scripts/test-serverless-aws-readiness.sh >/dev/null
scripts/test-serverless-production-readiness.sh >/dev/null

if command -v actionlint >/dev/null 2>&1; then
  actionlint .github/workflows/*.yml
fi
git diff --check
python3 .ai/scripts/validate_workflow.py

echo "Serverless local verification passed."
