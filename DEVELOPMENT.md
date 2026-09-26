# Developing Fidelius

Fidelius keeps the platform-specific GUI separate from the portable secret handoff:

- `cmd/fidelius/` and `internal/app/` — Go CLI, temporary files, auto-delete lifecycle, timeout configuration, and platform prompt adapters
- `apps/macos/` — native Swift/AppKit prompt and app icon
- `internal/app/prompt_linux.go` — Linux desktop prompt through Zenity, with KDialog fallback
- `internal/app/prompt_windows.go` and `prompt_windows.ps1` — native Windows Forms prompt through Windows PowerShell
- `docs/` — Astro landing page and `/install.sh` endpoint
- `scripts/install.sh` and `scripts/install.ps1` — platform installers

The Go core never stores secrets permanently. After the human submits, it creates a private temporary directory with one `0600` file per secret and starts an independent auto-delete process. Each new Fidelius invocation also removes stale sessions whose auto-delete time has passed.

On macOS, the AppKit helper sends values to Go through a dedicated inherited file descriptor, not stdout or stderr. On Linux and Windows, Fidelius captures the native dialog helper's output in a private pipe and never forwards it to the terminal. Windows session directories and files have an ACL that grants access only to the invoking user. PowerShell stderr is discarded so a helper error cannot expose a value.

## Local setup

```bash
cd docs
bun install
cd ..
make check
make build
make install-local
```

Useful build targets:

```bash
make build-universal   # universal macOS CLI + Fidelius.app
make build-linux       # Linux amd64 + arm64 binaries
make build-windows     # Windows amd64 binary
```

## Exercise the prompt

```bash
secrets=$(fidelius ask \
  -m "I need this secret to verify the local build." \
  TEST_SECRET)

cat "$secrets/TEST_SECRET" >/dev/null
```

Use a short isolated timeout when testing auto-delete behavior rather than changing your normal preference:

```bash
FIDELIUS_CONFIG_DIR="$(mktemp -d)" fidelius timeout 3s
```

## Linux

Desktop Linux uses the native dialog helper already appropriate for the user's environment:

1. `zenity` — GTK form with all requested password fields in one window
2. `kdialog` — KDE fallback; multiple secrets are requested sequentially

If neither is installed, `fidelius ask` explains what to install. The release binary itself is static and has no GUI toolkit dependency.

## Windows

The Windows binary embeds a Windows Forms prompt and uses the built-in Windows PowerShell 5.1 to display it. `fidelius ask` needs an interactive Windows desktop session. The PowerShell installer verifies the release ZIP against its SHA-256 file and installs `fidelius.exe` in `%USERPROFILE%\.local\bin`, adding that directory to the user PATH. Open a new PowerShell window after installation.

## Landing page

```bash
cd docs
bun run dev
bun run check
bun run build
```

The installer endpoints serve the checked-in scripts unchanged: `/install.sh` for macOS/Linux and `/install.ps1` for Windows.

The product screenshot used by the site and README is `docs/public/fidelius-window.png`. Capture it from the real native Mac app; do not recreate the app in HTML.

## Release

Before tagging:

```bash
make check
make site-build
make build-linux VERSION=x.y.z
make build-windows VERSION=x.y.z
git diff --check
```

Releases are tag-driven GitHub Releases:

```bash
make release-tag VERSION=x.y.z
```

The release workflow publishes:

- universal macOS CLI + `Fidelius.app`
- Linux amd64 CLI
- Linux arm64 CLI
- Windows amd64 CLI
- SHA-256 checksum for each archive
