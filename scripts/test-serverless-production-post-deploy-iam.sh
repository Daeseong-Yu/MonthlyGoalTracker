#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
SETUP_SCRIPT="${ROOT_DIR}/scripts/setup-serverless-production-post-deploy-iam.sh"

prepare_role_arn="arn:aws:iam::000000000000:role/example-production-prepare"
migrate_role_arn="arn:aws:iam::000000000000:role/example-production-migrate"

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

assert_output_does_not_leak() {
  local forbidden
  for forbidden in \
    "${prepare_role_arn}" \
    "${migrate_role_arn}" \
    "example-production-stack-id" \
    "example-frontend-bucket" \
    "EXAMPLEDISTRIBUTION" \
    "example-migration-function" \
    "example-audit-function"; do
    if [[ "${output}" == *"${forbidden}"* ]]; then
      echo "Output leaked a fixture identifier." >&2
      exit 1
    fi
  done
}

fake_aws_dir="$(mktemp -d "${TMPDIR:-/tmp}/monthly-goal-tracker-post-deploy-iam.XXXXXX")"
fake_aws_log="${fake_aws_dir}/aws.log"
fake_policy_dir="${fake_aws_dir}/policies"
mkdir -p "${fake_policy_dir}"

cleanup() {
  rm -rf -- "${fake_aws_dir}"
}

trap cleanup EXIT

cat > "${fake_aws_dir}/aws" <<'AWS'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >> "${FAKE_AWS_LOG}"

command="${1:-} ${2:-}"
case "${command}" in
  "cloudformation describe-stacks")
    echo "arn:aws:cloudformation:us-east-1:000000000000:stack/MonthlyGoalTracker-production/example-production-stack-id"
    ;;
  "cloudformation describe-stack-resource")
    logical_id=""
    while (($# > 0)); do
      if [[ "$1" == "--logical-resource-id" ]]; then
        shift
        logical_id="${1:-}"
        break
      fi
      shift
    done
    case "${logical_id}" in
      FrontendBucket) echo "example-frontend-bucket" ;;
      FrontendDistribution) echo "EXAMPLEDISTRIBUTION" ;;
      MigrationFunction) echo "example-migration-function" ;;
      AuditFunction) echo "example-audit-function" ;;
      *) exit 2 ;;
    esac
    ;;
  "iam put-role-policy")
    role_name=""
    policy_name=""
    policy_document=""
    while (($# > 0)); do
      case "$1" in
        --role-name)
          shift
          role_name="${1:-}"
          ;;
        --policy-name)
          shift
          policy_name="${1:-}"
          ;;
        --policy-document)
          shift
          policy_document="${1:-}"
          ;;
      esac
      shift
    done
    printf 'put|%s|%s\n' "${role_name}" "${policy_name}" >> "${FAKE_AWS_LOG}"
    cp -- "${policy_document#file://}" "${FAKE_POLICY_DIR}/${role_name}.json"
    ;;
  "iam simulate-principal-policy")
    action=""
    while (($# > 0)); do
      if [[ "$1" == "--action-names" ]]; then
        shift
        action="${1:-}"
        break
      fi
      shift
    done
    if [[ -n "${FAKE_DENY_ACTION:-}" && "${action}" == "${FAKE_DENY_ACTION}" ]]; then
      echo "implicitDeny"
    else
      echo "allowed"
    fi
    ;;
  *)
    echo "Unexpected fake aws command: ${command}" >&2
    exit 2
    ;;
esac
AWS
chmod +x "${fake_aws_dir}/aws"

base_env=(
  AWS_REGION=us-east-1
  SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN="${prepare_role_arn}"
  SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN="${migrate_role_arn}"
)

run_success "local dry run" env "${base_env[@]}" "${SETUP_SCRIPT}"
if [[ -s "${fake_aws_log}" ]]; then
  echo "Dry run must not call AWS." >&2
  exit 1
fi
assert_output_does_not_leak

run_failure "same role rejected" "Prepare and migrate roles must be different" env \
  AWS_REGION=us-east-1 \
  SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN="${prepare_role_arn}" \
  SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN="${prepare_role_arn}" \
  "${SETUP_SCRIPT}"

run_success "read-only policy check" env \
  "${base_env[@]}" \
  PATH="${fake_aws_dir}:${PATH}" \
  FAKE_AWS_LOG="${fake_aws_log}" \
  FAKE_POLICY_DIR="${fake_policy_dir}" \
  "${SETUP_SCRIPT}" --check
if grep -Fq "iam put-role-policy" "${fake_aws_log}"; then
  echo "Read-only check must not update IAM policies." >&2
  exit 1
fi
assert_output_does_not_leak

: > "${fake_aws_log}"
run_success "apply and verify policies" env \
  "${base_env[@]}" \
  PATH="${fake_aws_dir}:${PATH}" \
  FAKE_AWS_LOG="${fake_aws_log}" \
  FAKE_POLICY_DIR="${fake_policy_dir}" \
  "${SETUP_SCRIPT}" --apply

if [[ "$(grep -c '^put|' "${fake_aws_log}")" -ne 2 ]]; then
  echo "Expected exactly two inline policy updates." >&2
  exit 1
fi
if grep -Fq "iam create-role" "${fake_aws_log}"; then
  echo "Post-deploy IAM setup must not create roles." >&2
  exit 1
fi

jq -e '
  def actions:
    [.Statement[].Action | if type == "array" then .[] else . end] | sort;
  def resources:
    [.Statement[].Resource | if type == "array" then .[] else . end];
  (.Statement | length == 4) and
  (actions == ([
    "cloudformation:DescribeStackResource",
    "s3:ListBucket",
    "s3:GetBucketLocation",
    "s3:GetObject",
    "s3:PutObject",
    "s3:DeleteObject",
    "cloudfront:GetDistribution",
    "cloudfront:CreateInvalidation",
    "cloudfront:GetInvalidation"
  ] | sort)) and
  (resources | index("*") == null) and
  ([.Statement[] | select(.Action == "cloudformation:DescribeStackResource") | .Resource] == [
    "arn:aws:cloudformation:us-east-1:000000000000:stack/MonthlyGoalTracker-production/*"
  ]) and
  ([.Statement[] | select(.Action | arrays) | select(.Action | index("s3:ListBucket")) | .Resource] == [
    "arn:aws:s3:::example-frontend-bucket"
  ]) and
  ([.Statement[] | select(.Action | arrays) | select(.Action | index("s3:GetObject")) | .Resource] == [
    "arn:aws:s3:::example-frontend-bucket/*"
  ]) and
  ([.Statement[] | select(.Action | arrays) | select(.Action | index("cloudfront:CreateInvalidation")) | .Resource] == [
    "arn:aws:cloudfront::000000000000:distribution/EXAMPLEDISTRIBUTION"
  ])
' "${fake_policy_dir}/example-production-prepare.json" >/dev/null

jq -e '
  def actions:
    [.Statement[].Action | if type == "array" then .[] else . end] | sort;
  def resources:
    [.Statement[].Resource | if type == "array" then .[] else . end];
  (.Statement | length == 2) and
  (actions == ([
    "cloudformation:DescribeStackResource",
    "lambda:InvokeFunction"
  ] | sort)) and
  (resources | index("*") == null) and
  ([.Statement[] | select(.Action == "cloudformation:DescribeStackResource") | .Resource] == [
    "arn:aws:cloudformation:us-east-1:000000000000:stack/MonthlyGoalTracker-production/*"
  ]) and
  ([.Statement[] | select(.Action == "lambda:InvokeFunction") | .Resource[]] | sort) == ([
    "arn:aws:lambda:us-east-1:000000000000:function:example-migration-function",
    "arn:aws:lambda:us-east-1:000000000000:function:example-audit-function"
  ] | sort)
' "${fake_policy_dir}/example-production-migrate.json" >/dev/null

assert_output_does_not_leak

run_failure "denied action blocks verification" "IAM simulation failed for migrate lambda:InvokeFunction" env \
  "${base_env[@]}" \
  PATH="${fake_aws_dir}:${PATH}" \
  FAKE_AWS_LOG="${fake_aws_log}" \
  FAKE_POLICY_DIR="${fake_policy_dir}" \
  FAKE_DENY_ACTION="lambda:InvokeFunction" \
  "${SETUP_SCRIPT}" --check
assert_output_does_not_leak

echo "Serverless production post-deploy IAM helper tests passed."
