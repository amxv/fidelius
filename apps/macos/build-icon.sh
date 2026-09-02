#!/usr/bin/env bash
set -euo pipefail

component_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source_svg="${component_root}/Resources/AppIcon.svg"
output="${component_root}/Resources/AppIcon.icns"

command -v rsvg-convert >/dev/null 2>&1 || {
  echo "rsvg-convert is required to regenerate AppIcon.icns" >&2
  exit 1
}

tmp="$(mktemp -d "${TMPDIR:-/tmp}/fidelius-icon.XXXXXX")"
trap 'rm -rf "${tmp}"' EXIT
iconset="${tmp}/AppIcon.iconset"
mkdir -p "${iconset}"

render() {
  local pixels="$1"
  local filename="$2"
  rsvg-convert -w "${pixels}" -h "${pixels}" "${source_svg}" > "${iconset}/${filename}"
}

render 16 icon_16x16.png
render 32 icon_16x16@2x.png
render 32 icon_32x32.png
render 64 icon_32x32@2x.png
render 128 icon_128x128.png
render 256 icon_128x128@2x.png
render 256 icon_256x256.png
render 512 icon_256x256@2x.png
render 512 icon_512x512.png
render 1024 icon_512x512@2x.png

iconutil -c icns "${iconset}" -o "${output}"
echo "Wrote ${output}"
