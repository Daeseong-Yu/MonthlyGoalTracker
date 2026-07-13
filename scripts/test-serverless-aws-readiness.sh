#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_SCRIPT="${ROOT_DIR}/scripts/check-serverless-aws-readiness.sh"
STAGING_WORKFLOW="${ROOT_DIR}/.github/workflows/serverless-staging.yml"

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

run_failure_without_leak() {
  local name="$1"
  local expected="$2"
  local forbidden="$3"
  shift 3

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

  if [[ "${output}" == *"${forbidden}"* ]]; then
    echo "Failure message leaked a sensitive identifier for ${name}" >&2
    exit 1
  fi
}

valid_role="arn:aws:iam::000000000000:role/monthly-goal-tracker-serverless-staging"
valid_cert="arn:aws:acm:us-east-1:000000000000:certificate/example"

# Deployment workflows define these at job scope; each test case supplies its own values.
unset \
  AWS_REGION \
  AWS_ROLE_TO_ASSUME \
  SERVERLESS_EMAIL_FROM \
  SERVERLESS_DOMAIN_NAME \
  SERVERLESS_CERTIFICATE_ARN

fake_aws_dir="$(mktemp -d "${TMPDIR:-/tmp}/monthly-goal-tracker-fake-aws.XXXXXX")"

cleanup() {
  rm -rf -- "${fake_aws_dir}"
}

trap cleanup EXIT

cat > "${fake_aws_dir}/aws" <<'AWS'
#!/usr/bin/env bash
set -euo pipefail

mode="${FAKE_AWS_MODE:-success}"
command="${1:-} ${2:-}"

case "${command}" in
  "sts get-caller-identity")
    if [[ "${mode}" == "sts_fail" ]]; then
      exit 1
    fi
    echo '{"UserId":"example","Account":"000000000000","Arn":"arn:aws:sts::000000000000:assumed-role/example/session"}'
    ;;
  "ssm get-parameter")
    if [[ "${mode}" == "ssm_fail" ]]; then
      exit 1
    fi
    echo '{"Parameter":{"Name":"/cdk-bootstrap/hnb659fds/version"}}'
    ;;
  "sesv2 get-account")
    if [[ "${mode}" == "ses_account_fail" ]]; then
      echo "False"
    else
      echo "True"
    fi
    ;;
  "sesv2 get-email-identity")
    if [[ "${mode}" == "ses_identity_fail" ]]; then
      echo "False"
    else
      echo "True"
    fi
    ;;
  "acm describe-certificate")
    if [[ "${mode}" == "acm_fail" ]]; then
      echo "PENDING_VALIDATION"
    else
      echo "ISSUED"
    fi
    ;;
  *)
    echo "Unexpected fake aws command: ${command}" >&2
    exit 2
    ;;
esac
AWS
chmod +x "${fake_aws_dir}/aws"

run_success "valid minimal dry-run inputs" env \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  "${CHECK_SCRIPT}" --dry-run

run_failure "missing role ARN" "Missing required environment value: AWS_ROLE_TO_ASSUME" env \
  AWS_REGION=us-east-1 \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  "${CHECK_SCRIPT}" --dry-run

run_failure "invalid role ARN" "AWS_ROLE_TO_ASSUME must be an IAM role ARN." env \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME=monthly-goal-tracker-role \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  "${CHECK_SCRIPT}" --dry-run

run_failure "domain without certificate" "SERVERLESS_DOMAIN_NAME and SERVERLESS_CERTIFICATE_ARN must be configured together." env \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  SERVERLESS_DOMAIN_NAME=serverless.example.invalid \
  "${CHECK_SCRIPT}" --dry-run

run_failure "certificate without domain" "SERVERLESS_DOMAIN_NAME and SERVERLESS_CERTIFICATE_ARN must be configured together." env \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  SERVERLESS_CERTIFICATE_ARN="${valid_cert}" \
  "${CHECK_SCRIPT}" --dry-run

run_failure "invalid domain name" "SERVERLESS_DOMAIN_NAME must be a DNS host name without scheme or path." env \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  SERVERLESS_DOMAIN_NAME=https://serverless.example.invalid \
  SERVERLESS_CERTIFICATE_ARN="${valid_cert}" \
  "${CHECK_SCRIPT}" --dry-run

run_success "valid custom domain dry-run inputs" env \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  SERVERLESS_DOMAIN_NAME=serverless.example.invalid \
  SERVERLESS_CERTIFICATE_ARN="${valid_cert}" \
  "${CHECK_SCRIPT}" --dry-run

run_success "non dry-run readiness success" env \
  PATH="${fake_aws_dir}:${PATH}" \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  SERVERLESS_DOMAIN_NAME=serverless.example.invalid \
  SERVERLESS_CERTIFICATE_ARN="${valid_cert}" \
  "${CHECK_SCRIPT}"

run_failure_without_leak "credential readiness failure" \
  "AWS credential readiness failed" \
  "${valid_role}" \
  env \
  PATH="${fake_aws_dir}:${PATH}" \
  FAKE_AWS_MODE=sts_fail \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  "${CHECK_SCRIPT}"

run_failure_without_leak "CDK bootstrap readiness failure" \
  "CDK bootstrap readiness failed" \
  "${valid_role}" \
  env \
  PATH="${fake_aws_dir}:${PATH}" \
  FAKE_AWS_MODE=ssm_fail \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  "${CHECK_SCRIPT}"

run_failure_without_leak "SES sender readiness failure" \
  "SES sender readiness failed" \
  "no-reply@example.invalid" \
  env \
  PATH="${fake_aws_dir}:${PATH}" \
  FAKE_AWS_MODE=ses_identity_fail \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  "${CHECK_SCRIPT}"

run_failure_without_leak "custom domain readiness failure" \
  "Custom domain readiness failed" \
  "${valid_cert}" \
  env \
  PATH="${fake_aws_dir}:${PATH}" \
  FAKE_AWS_MODE=acm_fail \
  AWS_REGION=us-east-1 \
  AWS_ROLE_TO_ASSUME="${valid_role}" \
  SERVERLESS_EMAIL_FROM=no-reply@example.invalid \
  SERVERLESS_DOMAIN_NAME=serverless.example.invalid \
  SERVERLESS_CERTIFICATE_ARN="${valid_cert}" \
  "${CHECK_SCRIPT}"

if ! grep -Fq "scripts/check-serverless-aws-readiness.sh" "${STAGING_WORKFLOW}"; then
  echo "Expected serverless staging workflow to run scripts/check-serverless-aws-readiness.sh before deploy." >&2
  exit 1
fi

echo "Serverless AWS readiness helper tests passed."
