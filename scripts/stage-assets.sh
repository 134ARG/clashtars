#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mkdir -p "${root}/internal/assets/subconverter"
mkdir -p "${root}/internal/assets/ui"
rm -f \
  "${root}/internal/assets/mihomo/mihomo" \
  "${root}/internal/assets/subconverter/subconverter_linux64.tar.gz" \
  "${root}/internal/assets/ui/compressed-dist.tgz"

"${root}/scripts/fetch-mihomo.sh" "${root}/internal/assets/mihomo/mihomo"
curl -fL --retry 3 --retry-delay 2 \
  -o "${root}/internal/assets/subconverter/subconverter_linux64.tar.gz" \
  "https://github.com/tindy2013/subconverter/releases/latest/download/subconverter_linux64.tar.gz"

curl -fL --retry 3 --retry-delay 2 \
  -o "${root}/internal/assets/ui/compressed-dist.tgz" \
  "https://github.com/MetaCubeX/metacubexd/releases/latest/download/compressed-dist.tgz"
