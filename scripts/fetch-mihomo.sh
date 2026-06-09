#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dest="${1:-${root}/internal/assets/mihomo/mihomo}"

release_api="${MIHOMO_RELEASE_API:-https://api.github.com/repos/MetaCubeX/mihomo/releases/latest}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

json="${tmp_dir}/release.json"
curl -fL --retry 3 --retry-delay 2 -o "${json}" "${release_api}"

asset_url="$(
  sed -n 's/.*"browser_download_url": "\(.*\)".*/\1/p' "${json}" |
    grep -E '/mihomo-linux-amd64-v1-v[0-9][^/]*\.gz$' |
    head -n 1
)"

if [[ -z "${asset_url}" ]]; then
  asset_url="$(
    sed -n 's/.*"browser_download_url": "\(.*\)".*/\1/p' "${json}" |
      grep -E '/mihomo-linux-amd64-compatible-v[0-9][^/]*\.gz$|/mihomo-linux-amd64-v[0-9][^/]*\.gz$' |
      head -n 1
  )"
fi

if [[ -z "${asset_url}" ]]; then
  echo "ERROR: unable to find a linux amd64 Mihomo gzip asset in latest release" >&2
  exit 1
fi

gzip_path="${tmp_dir}/mihomo.gz"
binary_path="${tmp_dir}/mihomo"

curl -fL --retry 3 --retry-delay 2 -o "${gzip_path}" "${asset_url}"
gzip -dc "${gzip_path}" > "${binary_path}"
install -D -m 0755 "${binary_path}" "${dest}"
