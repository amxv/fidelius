# Fidelius

Fidelius allows agents to ask humans to enter secrets and saves them to macOS Keychain.

An agent runs one blocking command, a small native Mac window appears, and the human can paste one or more requested keys. Fidelius reports only whether the prompt completed and the character count for each value. The values themselves are never printed by Fidelius.

## Install

```bash
curl -fsSL https://fidelius.ashray.xyz/install.sh | bash
```

Fidelius currently supports macOS. The installer downloads the latest universal release from GitHub, verifies its SHA-256 checksum, and installs `fidelius` plus `Fidelius.app` into `~/.local/bin` by default.

## Use

Ask for one key:

```bash
fidelius -s my-app OPENAI_API_KEY
```

Ask for several, with an optional message for the human:

```bash
fidelius -s scraper -m "I need the Maps key to finish the scrape." GOOGLE_MAPS_API_KEY FIRECRAWL_API_KEY
```

The command waits until the human saves or cancels the native prompt.

Afterward, use Apple's normal Keychain CLI:

```bash
security find-generic-password -s my-app -a OPENAI_API_KEY -w
```

That keeps Fidelius intentionally small: it handles human input and Keychain storage, while `security` handles retrieval and Unix composition.

## Development

```bash
cd docs && bun install && cd ..
make check
make build
make install-local
```

`make build` produces the local Go CLI and native Swift app under `dist/`. `make build-universal` produces universal macOS binaries suitable for release packaging.

The landing page is a small Astro site under `docs/`:

```bash
cd docs
bun run dev
bun run check
bun run build
```

The canonical installer lives at `scripts/install.sh` and is served at `https://fidelius.ashray.xyz/install.sh` by the Astro endpoint.

## Releases

Releases are tag-driven GitHub Releases only. There is no npm package.

```bash
make check
make release-tag VERSION=0.1.0
```

GitHub Actions builds a universal macOS CLI and `Fidelius.app`, ad-hoc signs the app, smoke-tests the installer, and publishes the tarball plus SHA-256 checksum.

## License

Apache-2.0
