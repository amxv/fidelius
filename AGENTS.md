# AGENTS.md

Fidelius lets agents securely ask humans for secrets through a native GUI without printing those secret values into the terminal or chat.

## Product contract

Keep Fidelius intentionally tiny.

- `fidelius ask [-m MESSAGE] NAME [NAME...]` opens the human prompt.
- On success, stdout contains **only** the path to a private temporary directory.
- The directory contains one file per requested secret, named exactly after the requested secret.
- Secret values must never be written to Fidelius stdout or stderr.
- Directories are mode `0700`; secret files are mode `0600`.
- Temporary sessions auto-delete after the configured timeout. Default: 5 minutes.
- `fidelius timeout [DURATION]` is the only persistent preference surface.
- Fidelius is not a secret manager and must not learn provider-specific destinations such as Keychain, dotenv, GitHub, Vercel, etc. Agents use ordinary Unix tools after the handoff.
- Human interaction and downstream secret use are intentionally decoupled so agents can retry failed commands without asking the human again.
- Multiple concurrent invocations must remain independent.

## Architecture

- `cmd/fidelius/` — process entrypoint.
- `internal/app/` — command parsing, timeout config, private temporary sessions, auto-delete, and OS prompt adapters.
- `apps/macos/` — native AppKit GUI. Secret values return to Go through an inherited private file descriptor.
- Linux — uses Zenity when available, KDialog as fallback. The Go core remains toolkit-independent.
- `scripts/install.sh` — macOS/Linux installer served unchanged from the landing-page domain.
- `docs/` — self-contained Astro landing page.
- `.github/workflows/` — macOS + Linux CI and tag-driven releases.

Installed macOS layout:

```text
~/.local/bin/fidelius
~/.local/bin/Fidelius.app
```

Installed Linux layout:

```text
~/.local/bin/fidelius
```

## Commands

```bash
make check
make build
make build-universal
make build-linux
make install-local
make site-build
```

Before pushing, run `make check`, `make site-build`, relevant release builds, and `git diff --check`.

## Distribution

The macOS app is ad-hoc signed with `codesign --sign -`. Release archives are installed through `curl` + `tar` after SHA-256 verification. Do not disable Gatekeeper or strip quarantine attributes.

Linux release binaries are static Go binaries. `fidelius ask` requires either `zenity` or `kdialog` on a graphical Linux desktop.
