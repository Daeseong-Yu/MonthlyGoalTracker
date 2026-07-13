#!/usr/bin/env bash
set -euo pipefail

mode="dry-run"
stack_name="${SERVERLESS_PRODUCTION_STACK_NAME:-MonthlyGoalTracker-production}"
prepare_policy_name="MonthlyGoalTrackerProductionPostDeploy"
migrate_policy_name="MonthlyGoalTrackerProductionMigrationAudit"

usage() {
  cat <<'USAGE'
Usage:
  AWS_REGION=... \
  SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN=... \
  SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN=... \
    scripts/setup-serverless-production-post-deploy-iam.sh [--check|--apply]

Options:
  --check  Resolve current production resources and simulate existing role policies.
  --apply  Update exact inline policies on the existing roles, then simulate them.

Default mode validates local inputs without calling AWS APIs. This script never
creates roles and never prints role or resource identifiers.
USAGE
}

while (($# > 0)); do
  case "$1" in
    --check)
      mode="check"
      ;;
    --apply)
      mode="apply"
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

fail() {
  echo "$1" >&2
  exit 1
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "Missing required command: $1"
  fi
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    fail "Missing required environment value: ${name}"
  fi
}

require_env AWS_REGION
require_env SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN
require_env SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN

if [[ ! "${AWS_REGION}" =~ ^[a-z]{2}(-gov)?-[a-z]+-[0-9]+$ ]]; then
  fail "AWS_REGION must be an AWS region name."
fi

if [[ ! "${stack_name}" =~ ^[A-Za-z][-A-Za-z0-9]{0,127}$ ]]; then
  fail "SERVERLESS_PRODUCTION_STACK_NAME must be a CloudFormation stack name."
fi

if [[ "${SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN}" =~ [[:space:]] ]]; then
  fail "SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN must be an IAM role ARN."
fi
if [[ ! "${SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN}" =~ ^arn:([^:]+):iam::([0-9]{12}):role/(.+)$ ]]; then
  fail "SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN must be an IAM role ARN."
fi
prepare_partition="${BASH_REMATCH[1]}"
prepare_account="${BASH_REMATCH[2]}"
prepare_role_name="${BASH_REMATCH[3]##*/}"

if [[ "${SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN}" =~ [[:space:]] ]]; then
  fail "SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN must be an IAM role ARN."
fi
if [[ ! "${SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN}" =~ ^arn:([^:]+):iam::([0-9]{12}):role/(.+)$ ]]; then
  fail "SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN must be an IAM role ARN."
fi
migrate_partition="${BASH_REMATCH[1]}"
migrate_account="${BASH_REMATCH[2]}"
migrate_role_name="${BASH_REMATCH[3]##*/}"

if [[ "${SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN}" == "${SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN}" ]]; then
  fail "Prepare and migrate roles must be different."
fi

if [[ "${prepare_partition}" != "${migrate_partition}" || "${prepare_account}" != "${migrate_account}" ]]; then
  fail "Prepare and migrate roles must belong to the same AWS account and partition."
fi

if [[ "${mode}" == "dry-run" ]]; then
  echo "Dry run: production post-deploy IAM input names and formats are valid."
  echo "No roles or policies were changed."
  exit 0
fi

require_command aws
if [[ "${mode}" == "apply" ]]; then
  require_command jq
fi

read_stack_value() {
  local category="$1"
  shift
  local value

  if ! value="$(aws "$@" 2>/dev/null)"; then
    fail "Production ${category} lookup failed."
  fi
  if [[ -z "${value}" || "${value}" == "None" || "${value}" == "null" ]]; then
    fail "Production ${category} lookup returned no value."
  fi
  printf '%s' "${value}"
}

stack_arn="$(read_stack_value "stack" \
  cloudformation describe-stacks \
  --stack-name "${stack_name}" \
  --region "${AWS_REGION}" \
  --query 'Stacks[0].StackId' \
  --output text)"

read_stack_resource() {
  local category="$1"
  local logical_id="$2"
  read_stack_value "${category}" \
    cloudformation describe-stack-resource \
    --stack-name "${stack_name}" \
    --logical-resource-id "${logical_id}" \
    --region "${AWS_REGION}" \
    --query 'StackResourceDetail.PhysicalResourceId' \
    --output text
}

frontend_bucket="$(read_stack_resource "frontend bucket" "FrontendBucket")"
distribution_id="$(read_stack_resource "frontend distribution" "FrontendDistribution")"
migration_function="$(read_stack_resource "migration function" "MigrationFunction")"
audit_function="$(read_stack_resource "audit function" "AuditFunction")"

expected_stack_prefix="arn:${prepare_partition}:cloudformation:${AWS_REGION}:${prepare_account}:stack/${stack_name}/"
if [[ "${stack_arn}" != "${expected_stack_prefix}"* ]]; then
  fail "Production stack identity does not match the configured roles and region."
fi
if [[ ! "${frontend_bucket}" =~ ^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$ ]]; then
  fail "Production frontend bucket identifier is invalid."
fi
if [[ ! "${distribution_id}" =~ ^[A-Z0-9]+$ ]]; then
  fail "Production frontend distribution identifier is invalid."
fi
if [[ ! "${migration_function}" =~ ^[A-Za-z0-9_-]+$ || ! "${audit_function}" =~ ^[A-Za-z0-9_-]+$ ]]; then
  fail "Production function identifier is invalid."
fi

stack_policy_arn="arn:${prepare_partition}:cloudformation:${AWS_REGION}:${prepare_account}:stack/${stack_name}/*"
bucket_arn="arn:${prepare_partition}:s3:::${frontend_bucket}"
object_arn="${bucket_arn}/*"
object_probe_arn="${bucket_arn}/__iam_simulation_probe__"
distribution_arn="arn:${prepare_partition}:cloudfront::${prepare_account}:distribution/${distribution_id}"
migration_function_arn="arn:${prepare_partition}:lambda:${AWS_REGION}:${prepare_account}:function:${migration_function}"
audit_function_arn="arn:${prepare_partition}:lambda:${AWS_REGION}:${prepare_account}:function:${audit_function}"

echo "Resolved current production IAM targets."

policy_dir=""
cleanup() {
  if [[ -n "${policy_dir}" && -d "${policy_dir}" ]]; then
    rm -rf -- "${policy_dir}"
  fi
}
trap cleanup EXIT

if [[ "${mode}" == "apply" ]]; then
  policy_dir="$(mktemp -d "${TMPDIR:-/tmp}/monthly-goal-tracker-production-iam.XXXXXX")"
  chmod 700 "${policy_dir}"
  prepare_policy_file="${policy_dir}/prepare.json"
  migrate_policy_file="${policy_dir}/migrate.json"

  jq -n \
    --arg stack "${stack_policy_arn}" \
    --arg bucket "${bucket_arn}" \
    --arg objects "${object_arn}" \
    --arg distribution "${distribution_arn}" \
    '{
      Version: "2012-10-17",
      Statement: [
        {
          Sid: "DescribeProductionStackResources",
          Effect: "Allow",
          Action: "cloudformation:DescribeStackResource",
          Resource: $stack
        },
        {
          Sid: "ReadProductionFrontendBucket",
          Effect: "Allow",
          Action: ["s3:ListBucket", "s3:GetBucketLocation"],
          Resource: $bucket
        },
        {
          Sid: "SynchronizeProductionFrontendObjects",
          Effect: "Allow",
          Action: ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
          Resource: $objects
        },
        {
          Sid: "ReadAndInvalidateProductionDistribution",
          Effect: "Allow",
          Action: [
            "cloudfront:GetDistribution",
            "cloudfront:CreateInvalidation",
            "cloudfront:GetInvalidation"
          ],
          Resource: $distribution
        }
      ]
    }' > "${prepare_policy_file}"

  jq -n \
    --arg stack "${stack_policy_arn}" \
    --arg migration "${migration_function_arn}" \
    --arg audit "${audit_function_arn}" \
    '{
      Version: "2012-10-17",
      Statement: [
        {
          Sid: "DescribeProductionStackResources",
          Effect: "Allow",
          Action: "cloudformation:DescribeStackResource",
          Resource: $stack
        },
        {
          Sid: "InvokeProductionMigrationAndAudit",
          Effect: "Allow",
          Action: "lambda:InvokeFunction",
          Resource: [$migration, $audit]
        }
      ]
    }' > "${migrate_policy_file}"

  chmod 600 "${prepare_policy_file}" "${migrate_policy_file}"

  if ! aws iam put-role-policy \
    --role-name "${prepare_role_name}" \
    --policy-name "${prepare_policy_name}" \
    --policy-document "file://${prepare_policy_file}" \
    --region "${AWS_REGION}" >/dev/null 2>&1; then
    fail "Prepare inline policy update failed."
  fi
  echo "Updated prepare inline policy."

  if ! aws iam put-role-policy \
    --role-name "${migrate_role_name}" \
    --policy-name "${migrate_policy_name}" \
    --policy-document "file://${migrate_policy_file}" \
    --region "${AWS_REGION}" >/dev/null 2>&1; then
    fail "Migrate inline policy update failed."
  fi
  echo "Updated migrate inline policy."
fi

simulate_permission() {
  local role_label="$1"
  local role_arn="$2"
  local action="$3"
  local resource_arn="$4"
  local decision

  if ! decision="$(aws iam simulate-principal-policy \
    --policy-source-arn "${role_arn}" \
    --action-names "${action}" \
    --resource-arns "${resource_arn}" \
    --region "${AWS_REGION}" \
    --query 'EvaluationResults[0].EvalDecision' \
    --output text 2>/dev/null)"; then
    fail "IAM simulation failed for ${role_label} ${action}."
  fi
  if [[ "${decision,,}" != "allowed" ]]; then
    fail "IAM simulation failed for ${role_label} ${action}."
  fi
}

simulate_permission "prepare" "${SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN}" \
  "cloudformation:DescribeStackResource" "${stack_arn}"
for action in s3:ListBucket s3:GetBucketLocation; do
  simulate_permission "prepare" "${SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN}" "${action}" "${bucket_arn}"
done
for action in s3:GetObject s3:PutObject s3:DeleteObject; do
  simulate_permission "prepare" "${SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN}" "${action}" "${object_probe_arn}"
done
for action in cloudfront:GetDistribution cloudfront:CreateInvalidation cloudfront:GetInvalidation; do
  simulate_permission "prepare" "${SERVERLESS_PRODUCTION_PREPARE_ROLE_ARN}" "${action}" "${distribution_arn}"
done

simulate_permission "migrate" "${SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN}" \
  "cloudformation:DescribeStackResource" "${stack_arn}"
simulate_permission "migrate" "${SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN}" \
  "lambda:InvokeFunction" "${migration_function_arn}"
simulate_permission "migrate" "${SERVERLESS_PRODUCTION_MIGRATE_ROLE_ARN}" \
  "lambda:InvokeFunction" "${audit_function_arn}"

echo "Production post-deploy IAM simulations passed."
