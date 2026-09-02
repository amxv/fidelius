#!/usr/bin/env bash
set -euo pipefail

repo="amxv/fidelius"
version="${FIDELIUS_VERSION:-latest}"
install_dir="${FIDELIUS_INSTALL_DIR:-${HOME}/.local/bin}"
asset_name="fidelius-darwin-universal.tar.gz"

die() {
  printf '[fidelius install] ERROR: %s\n' "$*" >&2
  exit 1
}

log() {
  printf '[fidelius install] %s\n' "$*"
}

[[ "$(uname -s)" == "Darwin" ]] || die "Fidelius currently requires macOS"
command -v curl >/dev/null 2>&1 || die "curl is required"
command -v shasum >/dev/null 2>&1 || die "shasum is required"
command -v codesign >/dev/null 2>&1 || die "codesign is required"

if [[ -n "${FIDELIUS_ASSET_URL:-}" ]]; then
  asset_url="${FIDELIUS_ASSET_URL}"
elif [[ "${version}" == "latest" ]]; then
  asset_url="https://github.com/${repo}/releases/latest/download/${asset_name}"
else
  [[ "${version}" == v* ]] || version="v${version}"
  asset_url="https://github.com/${repo}/releases/download/${version}/${asset_name}"
fi
checksum_url="${FIDELIUS_CHECKSUM_URL:-${asset_url}.sha256}"

temporary="$(mktemp -d "${TMPDIR:-/tmp}/fidelius-install.XXXXXX")"
trap 'rm -rf "${temporary}"' EXIT
archive="${temporary}/${asset_name}"
checksum="${archive}.sha256"

log "downloading ${asset_url}"
curl --fail --location --silent --show-error "${asset_url}" -o "${archive}" || die "download failed"
curl --fail --location --silent --show-error "${checksum_url}" -o "${checksum}" || die "checksum download failed"

expected="$(awk 'NR==1 {print $1}' "${checksum}")"
actual="$(shasum -a 256 "${archive}" | awk '{print $1}')"
[[ -n "${expected}" && "${actual}" == "${expected}" ]] || die "SHA-256 verification failed"

mkdir -p "${temporary}/payload"
tar -xzf "${archive}" -C "${temporary}/payload"
[[ -x "${temporary}/payload/fidelius" ]] || die "release is missing the fidelius binary"
[[ -x "${temporary}/payload/Fidelius.app/Contents/MacOS/fidelius-ui" ]] || die "release is missing Fidelius.app"
codesign --verify --deep --strict "${temporary}/payload/Fidelius.app" || die "Fidelius.app signature verification failed"

mkdir -p "${install_dir}"
app_destination="${install_dir}/Fidelius.app"
if [[ -x "${app_destination}/Contents/MacOS/fidelius-ui" ]] && pgrep -f -x "${app_destination}/Contents/MacOS/fidelius-ui" >/dev/null 2>&1; then
  die "Fidelius is currently open; close the prompt and run the installer again"
fi

binary_temp="${install_dir}/.fidelius.install.$$"
app_temp="${install_dir}/.Fidelius.app.install.$$"
rm -f "${binary_temp}"
rm -rf "${app_temp}"
install -m 0755 "${temporary}/payload/fidelius" "${binary_temp}"
cp -R "${temporary}/payload/Fidelius.app" "${app_temp}"
mv -f "${binary_temp}" "${install_dir}/fidelius"
rm -rf "${app_destination}"
mv "${app_temp}" "${app_destination}"

log "installed fidelius to ${install_dir}/fidelius"
log "installed Fidelius.app to ${app_destination}"

case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *)
    printf '\n%s is not currently on PATH. Add this to your shell profile:\n  export PATH="%s:$PATH"\n' "${install_dir}" "${install_dir}"
    ;;
esac
