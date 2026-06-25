#!/usr/bin/env bash
set -euo pipefail

dist_dir="${1:-frontend/dist}"
index_file="${dist_dir}/index.html"
assets_dir="${dist_dir}/assets"

fail() {
  echo "$1" >&2
  exit 1
}

if [[ ! -s "${index_file}" ]]; then
  fail "Frontend dist is missing index.html."
fi

if [[ ! -d "${assets_dir}" ]]; then
  fail "Frontend dist is missing assets directory."
fi

if ! grep -q 'id="root"' "${index_file}"; then
  fail "Frontend index.html does not look like the app shell."
fi

if ! grep -Eq 'src="[^"]*/assets/[^"]+\.js"' "${index_file}"; then
  fail "Frontend index.html does not reference a JavaScript asset."
fi

if ! grep -Eq 'href="[^"]*/assets/[^"]+\.css"' "${index_file}"; then
  fail "Frontend index.html does not reference a CSS asset."
fi

shopt -s nullglob
js_assets=("${assets_dir}"/*.js)
css_assets=("${assets_dir}"/*.css)

if ((${#js_assets[@]} == 0)); then
  fail "Frontend dist is missing JavaScript assets."
fi

if ((${#css_assets[@]} == 0)); then
  fail "Frontend dist is missing CSS assets."
fi

for asset in "${js_assets[@]}" "${css_assets[@]}"; do
  if [[ ! -s "${asset}" ]]; then
    fail "Frontend dist contains an empty asset."
  fi
done

echo "Frontend dist artifact check passed."
