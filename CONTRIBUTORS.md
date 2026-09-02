# Contributing

Fidelius is deliberately narrow. Changes should preserve its role as a human-input bridge into macOS Keychain rather than growing a second credential-management interface.

## Local checks

```bash
cd docs && bun install && cd ..
make check
make site-build
git diff --check
```

For a local interactive build:

```bash
make install-local
fidelius -s fidelius-test TEST_API_KEY SECOND_TEST_KEY
```

Delete test entries afterward with Apple's normal CLI:

```bash
security delete-generic-password -s fidelius-test -a TEST_API_KEY
security delete-generic-password -s fidelius-test -a SECOND_TEST_KEY
```

## Release

Push a `v*` tag only after `main` is ready. GitHub Actions builds and publishes the universal macOS release assets and generated release notes.
