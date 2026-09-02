# Fidelius

**Fidelius lets agents securely ask humans for secrets.**

An agent runs one blocking command. Fidelius opens a native desktop prompt, you paste the requested secrets, and the agent gets a private temporary directory with one file per secret. Fidelius never prints secret values, and the directory auto-deletes after 5 minutes by default.

<p align="center">
  <img src="docs/public/fidelius-window.png" alt="Fidelius asking for a Google Maps API key on macOS" width="720">
</p>

## Install

```bash
curl -fsSL https://fidelius.ashray.xyz/install.sh | bash
```

Supports macOS 13+ and desktop Linux on amd64 and arm64. Linux uses the native `zenity` GTK dialog when available, with `kdialog` as the KDE fallback.

## Use

```bash
secrets=$(fidelius ask \
  -m "I need the Google Maps API key to finish scraping restaurant menus." \
  GOOGLE_MAPS_API_KEY)
```

Fidelius prints only the temporary directory path. The secret is available as:

```text
$secrets/GOOGLE_MAPS_API_KEY
```

From there, use normal Unix tools. For example:

```bash
# GitHub Actions secret
gh secret set GOOGLE_MAPS_API_KEY \
  < "$secrets/GOOGLE_MAPS_API_KEY"

# .env.local
printf 'GOOGLE_MAPS_API_KEY=' >> .env.local
cat "$secrets/GOOGLE_MAPS_API_KEY" >> .env.local
printf '\n' >> .env.local

# macOS Keychain
{
  cat "$secrets/GOOGLE_MAPS_API_KEY"; printf '\n'
  cat "$secrets/GOOGLE_MAPS_API_KEY"; printf '\n'
} | security add-generic-password -U \
  -s restaurant-scraping-demo \
  -a GOOGLE_MAPS_API_KEY \
  -w
```

The files are private (`0700` directory, `0600` files), so an agent can safely retry a failed command without asking you for the secret again.

### Auto-delete timeout

```bash
fidelius timeout        # Auto-delete timeout: 5m
fidelius timeout 10m    # change it
fidelius timeout 30s
fidelius timeout 2h
```

Multiple secrets can be requested in one prompt, and multiple agents can run Fidelius independently at the same time.

[Development and release notes →](DEVELOPMENT.md)

Apache-2.0
