#!/usr/bin/env bash
set -euo pipefail

repo="amxv/fidelius"
version="${FIDELIUS_VERSION:-latest}"
install_dir="${FIDELIUS_INSTALL_DIR:-${HOME}/.local/bin}"

die() {
  printf '[fidelius install] ERROR: %s\n' "$*" >&2
  exit 1
}

log() {
  printf '[fidelius install] %s\n' "$*"
}

warn() {
  printf '[fidelius install] WARNING: %s\n' "$*" >&2
}

command -v curl >/dev/null 2>&1 || die "curl is required"
if command -v shasum >/dev/null 2>&1; then
  checksum_file() { shasum -a 256 "$1" | awk '{print $1}'; }
elif command -v sha256sum >/dev/null 2>&1; then
  checksum_file() { sha256sum "$1" | awk '{print $1}'; }
else
  die "shasum or sha256sum is required"
fi

os="$(uname -s)"
arch="$(uname -m)"
case "${os}" in
  Darwin)
    asset_name="fidelius-darwin-universal.tar.gz"
    ;;
  Linux)
    case "${arch}" in
      x86_64|amd64) asset_name="fidelius-linux-amd64.tar.gz" ;;
      aarch64|arm64) asset_name="fidelius-linux-arm64.tar.gz" ;;
      *) die "unsupported Linux architecture: ${arch}" ;;
    esac
    ;;
  *)
    die "Fidelius currently supports macOS and Linux"
    ;;
esac

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

download_file() {
  local url="$1"
  local destination="$2"
  curl \
    --fail \
    --location \
    --silent \
    --show-error \
    --connect-timeout 10 \
    --max-time 120 \
    --retry 3 \
    --retry-delay 1 \
    "${url}" \
    -o "${destination}"
}

log "downloading ${asset_url}"
download_file "${asset_url}" "${archive}" || die "download failed"
download_file "${checksum_url}" "${checksum}" || die "checksum download failed"

expected="$(awk 'NR==1 {print $1}' "${checksum}")"
actual="$(checksum_file "${archive}")"
[[ -n "${expected}" && "${actual}" == "${expected}" ]] || die "SHA-256 verification failed"

mkdir -p "${temporary}/payload"
tar -xzf "${archive}" -C "${temporary}/payload"
[[ -x "${temporary}/payload/fidelius" ]] || die "release is missing the fidelius binary"

mkdir -p "${install_dir}"
binary_temp="${install_dir}/.fidelius.install.$$"
rm -f "${binary_temp}"
install -m 0755 "${temporary}/payload/fidelius" "${binary_temp}"
mv -f "${binary_temp}" "${install_dir}/fidelius"
log "installed fidelius to ${install_dir}/fidelius"

if [[ "${os}" == "Darwin" ]]; then
  command -v codesign >/dev/null 2>&1 || die "codesign is required on macOS"
  [[ -x "${temporary}/payload/Fidelius.app/Contents/MacOS/fidelius-ui" ]] || die "release is missing Fidelius.app"
  codesign --verify --deep --strict "${temporary}/payload/Fidelius.app" || die "Fidelius.app signature verification failed"

  app_destination="${install_dir}/Fidelius.app"
  if [[ -x "${app_destination}/Contents/MacOS/fidelius-ui" ]] && pgrep -f -x "${app_destination}/Contents/MacOS/fidelius-ui" >/dev/null 2>&1; then
    die "Fidelius is currently open; close the prompt and run the installer again"
  fi
  app_temp="${install_dir}/.Fidelius.app.install.$$"
  rm -rf "${app_temp}"
  cp -R "${temporary}/payload/Fidelius.app" "${app_temp}"
  rm -rf "${app_destination}"
  mv "${app_temp}" "${app_destination}"
  log "installed Fidelius.app to ${app_destination}"
else
  if command -v zenity >/dev/null 2>&1; then
    log "Linux prompt: zenity"
  elif command -v kdialog >/dev/null 2>&1; then
    log "Linux prompt: kdialog"
  else
    warn "no desktop dialog helper found; install zenity (GTK) or kdialog (KDE) before running 'fidelius ask'"
  fi
fi

case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *)
    printf '\n%s is not currently on PATH. Add this to your shell profile:\n  export PATH="%s:$PATH"\n' "${install_dir}" "${install_dir}"
    ;;
esac
