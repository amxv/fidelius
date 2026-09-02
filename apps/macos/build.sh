#!/usr/bin/env bash
set -euo pipefail

component_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${component_root}/../.." && pwd)"
output="${1:-${repo_root}/dist/Fidelius.app}"
mode="${2:-native}"
version="${FIDELIUS_VERSION:-0.0.0}"
bundle_version="${version}"
if [[ ! "${bundle_version}" =~ ^[0-9]+(\.[0-9]+)*$ ]]; then
  bundle_version="0.0.0"
fi
binary="${output}/Contents/MacOS/fidelius-ui"

rm -rf "${output}"
mkdir -p "$(dirname "${binary}")"
cp "${component_root}/Info.plist" "${output}/Contents/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${bundle_version}" "${output}/Contents/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion ${bundle_version}" "${output}/Contents/Info.plist"

sources=("${component_root}"/Sources/*.swift)
build_arch() {
  local arch="$1"
  local destination="$2"
  swiftc -O -target "${arch}-apple-macos13.0" "${sources[@]}" -o "${destination}"
}

if [[ "${mode}" == "universal" ]]; then
  temporary="$(mktemp -d "${TMPDIR:-/tmp}/fidelius-app.XXXXXX")"
  trap 'rm -rf "${temporary}"' EXIT
  build_arch arm64 "${temporary}/arm64"
  build_arch x86_64 "${temporary}/x86_64"
  lipo -create "${temporary}/arm64" "${temporary}/x86_64" -output "${binary}"
else
  arch="$(uname -m)"
  [[ "${arch}" == "arm64" || "${arch}" == "x86_64" ]] || {
    echo "unsupported macOS architecture: ${arch}" >&2
    exit 1
  }
  build_arch "${arch}" "${binary}"
fi

codesign --force --sign - "${output}" >/dev/null
