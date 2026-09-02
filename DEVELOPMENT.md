# Developing Fidelius

Fidelius has four small pieces:

- `cmd/fidelius/` and `internal/` — Go CLI and argument/status handling
- `apps/macos/` — native Swift/AppKit prompt and app icon
- `docs/` — Astro landing page and the `/install.sh` endpoint
- `scripts/install.sh` — canonical release installer

## Local setup

```bash
cd docs
bun install
cd ..
make check
make build
make install-local
```

`make build` creates the local Go CLI and `Fidelius.app` under `dist/`. `make build-universal` creates universal Intel + Apple Silicon release artifacts.

To exercise the real prompt:

```bash
fidelius \
  -s fidelius-test \
  -m "I need this secret to verify the local build." \
  TEST_SECRET
```

Delete the test value afterward:

```bash
security delete-generic-password -s fidelius-test -a TEST_SECRET
```

## Landing page

```bash
cd docs
bun run dev
bun run check
bun run build
```

The canonical installer is `scripts/install.sh`. `docs/src/pages/install.sh.ts` serves that exact file at `https://fidelius.ashray.xyz/install.sh`.

The product screenshot used by the site and README is `docs/public/fidelius-window.png` and should be captured from the real native app, not recreated in HTML.

## Release

Before tagging:

```bash
make check
make site-build
git diff --check
```

Releases are tag-driven GitHub Releases only:

```bash
make release-tag VERSION=x.y.z
```

GitHub Actions builds the universal CLI and app, ad-hoc signs `Fidelius.app`, smoke-tests the installer, and publishes the archive plus SHA-256 checksum.
