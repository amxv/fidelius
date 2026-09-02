---
name: release
description: Validate Fidelius and push a tag for its GitHub Release workflow.
allowed-tools: Bash, Read
---

# Release Fidelius

Fidelius releases are GitHub Releases only. There is no npm publishing step.

Before tagging:

```bash
make check
make site-build
git diff --check
git status --short --branch
```

Confirm `main` is pushed and clean, then:

```bash
make release-tag VERSION=x.y.z
```

The `v*` workflow builds the universal macOS Go binary and native app, ad-hoc signs `Fidelius.app`, smoke-tests `scripts/install.sh`, and publishes:

- `fidelius-darwin-universal.tar.gz`
- `fidelius-darwin-universal.tar.gz.sha256`

Watch the workflow and verify the release assets before announcing the release. Anonymous installer downloads require the GitHub repository to be public.
