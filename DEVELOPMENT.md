# Developing Fidelius

Fidelius keeps the platform-specific GUI separate from the portable secret handoff:

- `cmd/fidelius/` and `internal/app/` — Go CLI, temporary files, auto-delete lifecycle, timeout configuration, and platform prompt adapters
- `apps/macos/` — native Swift/AppKit prompt and app icon
- `internal/app/prompt_linux.go` — Linux desktop prompt through Zenity, with KDialog fallback
- `docs/` — Astro landing page and `/install.sh` endpoint
- `scripts/install.sh` — canonical macOS/Linux installer

The Go core never stores secrets permanently. After the human submits, it creates a private temporary directory with one `0600` file per secret and starts an independent auto-delete process. Each new Fidelius invocation also removes stale sessions whose auto-delete time has passed.

On macOS, the AppKit helper sends values to Go through a dedicated inherited file descriptor, not stdout or stderr. On Linux, Fidelius captures the native dialog helper's output directly and never forwards it to the terminal.

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

## Landing page

```bash
cd docs
bun run dev
bun run check
bun run build
```

The canonical installer is `scripts/install.sh`. `docs/src/pages/install.sh.ts` serves that exact file at `https://fidelius.ashray.xyz/install.sh`.

The product screenshot used by the site and README is `docs/public/fidelius-window.png`. Capture it from the real native Mac app; do not recreate the app in HTML.

## Release

Before tagging:

```bash
make check
make site-build
make build-linux VERSION=x.y.z
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
- SHA-256 checksum for each archive
