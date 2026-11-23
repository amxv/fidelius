# Developing Fidelius

Fidelius keeps the platform-specific GUI separate from the portable secret handoff:

- `cmd/fidelius/` and `internal/app/` — Go CLI, temporary files, auto-delete lifecycle, timeout configuration, and platform prompt adapters
- `apps/macos/` — native Swift/AppKit prompt and app icon
- `internal/app/prompt_linux.go` — Linux desktop adapter for the GTK 4 and Qt 6 frontends in `apps/linux/`
- `internal/app/prompt_windows.go` and `apps/windows/` use Win32 controls and embedded icon/manifest resources
- `docs/` — Astro landing page and `/install.sh` endpoint
- `scripts/install.sh` and `scripts/install.ps1` — platform installers

The Go core never stores secrets permanently. After the human submits, it creates a private temporary directory with one `0600` file per secret and starts an independent auto-delete process. Each new Fidelius invocation also removes stale sessions whose auto-delete time has passed.

On macOS, the AppKit helper sends values to Go through a dedicated inherited file descriptor, not stdout or stderr. Linux launches a packaged helper with request metadata on stdin and receives the submitted values on inherited FD 3. Windows collects values directly from password edits in the CLI process. Windows session directories and files grant access only to the invoking user.

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

Linux packages contain a static Go CLI and two dynamically linked helpers. Build the host package with `make build` on Linux with a C/C++ compiler, pkg-config, GTK 4 and Qt 6 Widgets development packages. `make build-linux` cross-builds both Go binaries. `make build-linux-ui` builds the helpers for the host architecture. Release jobs build native amd64 and arm64 packages against Debian 12 system libraries. Each tarball contains `fidelius` and `fidelius-ui/` together.

At runtime only one toolkit and its display backend are required. For example, on Debian 12 install `libgtk-4-1` for GTK or `libqt6widgets6` and `qt6-wayland` for Plasma Wayland. The helpers honor the toolkit's Wayland/X11 selection. The installer switches the CLI symlink to a checksum-specific package under the selected bin directory and installs desktop identity files under `XDG_DATA_HOME`. Earlier packages remain available to active prompts and cleanup processes. Pinned CLI-only releases still install.

## Windows

The Windows 10/11 amd64 console executable opens a Win32 dialog on the calling thread. `make build-windows` first generates the manifest and icon resource object with the pinned rsrc tool. The multi-size ICO derives from the macOS lock artwork. An interactive desktop is required for `ask`. The PowerShell installer verifies the ZIP checksum, installs the EXE in `%USERPROFILE%\.local\bin`, and adds that directory to the user PATH.

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
- Linux amd64 CLI and GUI helpers
- Linux arm64 CLI and GUI helpers
- Windows amd64 CLI
- SHA-256 checksum for each archive
