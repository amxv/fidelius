---
name: release
description: Validate Fidelius and push a tag for its GitHub Release workflow.
allowed-tools: Bash, Read
---

# Release Fidelius

Fidelius publishes GitHub Release archives for macOS and Linux. There is no npm package.

Before tagging:

```bash
make check
make site-build
make build-linux VERSION=x.y.z
git diff --check
git status --short --branch
```

Confirm `main` is pushed and clean, then:

```bash
make release-tag VERSION=x.y.z
```

The `v*` workflow builds, smoke-tests, and publishes:

- `fidelius-darwin-universal.tar.gz`
- `fidelius-linux-amd64.tar.gz`
- `fidelius-linux-arm64.tar.gz`
- one `.sha256` file for each archive

The macOS archive includes the ad-hoc-signed `Fidelius.app`. Linux uses Zenity or KDialog from the user's desktop environment and therefore ships only the static Go binary.

Watch both platform jobs and the final publish job to completion. Verify all six assets before announcing the release.
