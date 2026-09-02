# AGENTS.md

Fidelius is a tiny macOS utility for one job: an agent can ask a human for secrets in a native GUI, then Fidelius saves those values to macOS Keychain.

## Product boundaries

Keep Fidelius intentionally small.

- It is not a secret manager.
- It does not expose `get`, `show`, `pipe`, provider-specific integrations, or its own secret store.
- Retrieval and composition belong to Apple's `/usr/bin/security` CLI and normal Unix tools.
- Fidelius must never print secret values to stdout or stderr.
- The CLI blocks until the GUI is saved or cancelled and may report only safe metadata such as names and character counts.
- Multiple requested keys should share one normal app window that remains available through Dock/Cmd-Tab and does not float above other apps.
- `-m/--message` may provide a short human-readable explanation for why the keys are needed.

## Architecture

- `cmd/fidelius/`: Go CLI entrypoint.
- `internal/app/`: argument parsing, helper discovery, and safe status output.
- `apps/macos/`: native AppKit prompt; values are handed to Apple’s `/usr/bin/security` over stdin for Keychain storage.
- `scripts/install.sh`: release installer served unchanged from the landing-page domain.
- `docs/`: self-contained minimal Astro landing page. There is no multi-page docs site.
- `.github/workflows/`: macOS CI and tag-driven GitHub Release publishing.

The installed layout keeps the components adjacent:

```text
~/.local/bin/fidelius
~/.local/bin/Fidelius.app
```

The Go binary launches `Fidelius.app/Contents/MacOS/fidelius-ui`, waits for it to exit, then returns safe status to the caller.

## Commands

```bash
make check
make build
make build-universal
make install-local
cd docs && bun run dev
cd docs && bun run build
```

Before pushing, run `make check`, `make site-build`, and `git diff --check`.

## Distribution

The app is ad-hoc signed with `codesign --sign -`, packaged with the Go CLI in a GitHub Release tarball, and installed through `curl` + `tar` without disabling Gatekeeper or modifying quarantine attributes. The installer verifies the release SHA-256 checksum before installation.

The repository may remain private during development. Anonymous installation from GitHub Releases only works after the repository is made public and a release is published.
