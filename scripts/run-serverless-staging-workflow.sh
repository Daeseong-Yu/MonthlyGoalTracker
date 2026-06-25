#!/usr/bin/env bash
set -euo pipefail

workflow_name="Serverless Staging Deploy"
workflow_file="serverless-staging.yml"
ref=""
dispatch=false
run_migration=false
run_audit=false
audit_expected_counts_file=""
cleanup_files=()

cleanup() {
  if ((${#cleanup_files[@]} > 0)); then
    rm -f -- "${cleanup_files[@]}"
  fi
}

trap cleanup EXIT

usage() {
  cat <<'USAGE'
Usage:
  scripts/run-serverless-staging-workflow.sh [options]

Options:
  --dispatch                         Dispatch the GitHub Actions workflow.
  --ref REF                          Git ref to dispatch. Default: current branch.
  --run-migration                    Invoke the migration Lambda after deploy.
  --run-audit                        Invoke the audit Lambda after deploy or migration.
  --audit-expected-counts-file FILE  JSON object of expected audit row counts.

Default mode is dry-run. Workflow run identifiers and audit count values are not printed.
When dispatching with expected audit counts, the helper uses a mode-600 temporary
payload file and does not pass count values as command-line arguments.
USAGE
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

while (($# > 0)); do
  case "$1" in
    --dispatch)
      dispatch=true
      ;;
    --ref)
      shift
      if (($# == 0)); then
        echo "--ref requires a value" >&2
        exit 1
      fi
      ref="$1"
      ;;
    --run-migration)
      run_migration=true
      ;;
    --run-audit)
      run_audit=true
      ;;
    --audit-expected-counts-file)
      shift
      if (($# == 0)); then
        echo "--audit-expected-counts-file requires a value" >&2
        exit 1
      fi
      require_command jq
      require_command mktemp
      audit_expected_counts_file="$(mktemp "${TMPDIR:-/tmp}/serverless-audit-counts.XXXXXX.json")"
      cleanup_files+=("${audit_expected_counts_file}")
      chmod 600 "${audit_expected_counts_file}"
      jq -ce 'if type == "object" and all(.[]; type == "number") then . else error("expected JSON object with numeric values") end' "$1" >"${audit_expected_counts_file}"
      run_audit=true
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

if [[ -z "${ref}" ]]; then
  ref="$(git branch --show-current 2>/dev/null || true)"
fi
if [[ -z "${ref}" ]]; then
  ref="main"
fi

if [[ "${dispatch}" != "true" ]]; then
  echo "Dry run: would dispatch '${workflow_name}' on ref '${ref}'."
  echo "Input names:"
  echo "- run_migration=${run_migration}"
  echo "- run_audit=${run_audit}"
  if [[ -n "${audit_expected_counts_file}" ]]; then
    echo "- audit_expected_counts_json=provided"
  fi
  echo "Run again with --dispatch to request the workflow. Values and run identifiers are not printed."
  exit 0
fi

require_command gh
require_command jq
require_command mktemp

scripts/check-serverless-staging-readiness.sh >/dev/null

dispatch_payload_file="$(mktemp "${TMPDIR:-/tmp}/serverless-staging-dispatch.XXXXXX.json")"
cleanup_files+=("${dispatch_payload_file}")
chmod 600 "${dispatch_payload_file}"

if [[ -n "${audit_expected_counts_file}" ]]; then
  jq -n \
    --arg ref "${ref}" \
    --arg runMigration "${run_migration}" \
    --arg runAudit "${run_audit}" \
    --rawfile auditExpected "${audit_expected_counts_file}" \
    '{
      ref: $ref,
      inputs: {
        run_migration: $runMigration,
        run_audit: $runAudit,
        audit_expected_counts_json: $auditExpected
      }
    }' >"${dispatch_payload_file}"
else
  jq -n \
    --arg ref "${ref}" \
    --arg runMigration "${run_migration}" \
    --arg runAudit "${run_audit}" \
    '{
      ref: $ref,
      inputs: {
        run_migration: $runMigration,
        run_audit: $runAudit,
        audit_expected_counts_json: ""
      }
    }' >"${dispatch_payload_file}"
fi

gh api \
  --method POST \
  "repos/:owner/:repo/actions/workflows/${workflow_file}/dispatches" \
  --input "${dispatch_payload_file}" >/dev/null
echo "Serverless staging workflow dispatch requested."
