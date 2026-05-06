#!/usr/bin/env bash
set -euo pipefail

: "${DEPLOY_ARTIFACT_BUCKET:?DEPLOY_ARTIFACT_BUCKET is required}"
: "${RELEASE_ID:?RELEASE_ID is required}"
: "${AWS_REGION:?AWS_REGION is required}"

staging="/tmp/monthly-goal-tracker-${RELEASE_ID}-frontend"
release_root="/opt/monthly-goal-tracker/frontend/releases"
release_dir="${release_root}/${RELEASE_ID}"
current_dist="/opt/monthly-goal-tracker/frontend/dist"

cleanup() {
  rm -rf "${staging}"
}
trap cleanup EXIT

rm -rf "${staging}"
mkdir -p "${staging}"
aws s3 cp "s3://${DEPLOY_ARTIFACT_BUCKET}/releases/${RELEASE_ID}/frontend/frontend-dist.tar.gz" "${staging}/frontend-dist.tar.gz" --region "${AWS_REGION}"

install -d -m 0755 "${release_root}"
rm -rf "${release_dir}"
mkdir -p "${release_dir}"
tar -xzf "${staging}/frontend-dist.tar.gz" -C "${release_dir}"
test -f "${release_dir}/index.html"

if [ -d "${current_dist}" ] && [ ! -L "${current_dist}" ]; then
  mv "${current_dist}" "${release_root}/pre-cicd-$(date +%Y%m%d%H%M%S)"
fi

ln -sfn "${release_dir}" "${current_dist}.next"
mv -Tf "${current_dist}.next" "${current_dist}"

cd "${release_root}"
ls -1dt */ 2>/dev/null | tail -n +6 | while IFS= read -r old_release; do
  rm -rf -- "${release_root}/${old_release%/}"
done
