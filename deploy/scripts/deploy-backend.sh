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
previous_binary=""

cleanup() {
  rm -rf "${staging}"
}
trap cleanup EXIT

rm -rf "${staging}"
mkdir -p "${staging}"
aws s3 cp "s3://${DEPLOY_ARTIFACT_BUCKET}/releases/${RELEASE_ID}/backend/${artifact}" "${staging}/monthly-goal-api" --region "${AWS_REGION}"

install -d -m 0755 "${release_root}"
rm -rf "${release_dir}"
mkdir -p "${release_dir}"
install -m 0755 "${staging}/monthly-goal-api" "${release_dir}/monthly-goal-api"

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
