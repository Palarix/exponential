# Lightweight Telemetry via CDN Logo + Version Header

## Overview

The web UI loads its logo from the Cloudflare-hosted landing page instead of the local bundle. The response includes an `X-Latest` header with the latest release version (fetched from the GitHub API and cached). The web app compares it against its own version and can show an "update available" indicator.

No cookies, no persistent identifiers, no PII. Standard CDN access logs provide adoption data (version distribution, rough usage frequency). Telemetry is opt-out.

## Architecture

```
Web UI ──GET /beacon/xpo.svg?v=1.0.3──> CF Pages Function
                                           │
                                           ├─ Serve xpo.svg from static assets
                                           ├─ Add X-Latest: <version> header
                                           └─ Cache GitHub API response (~1h)
                                                │
                                                └─ github.com/repos/palarix/exponential/releases/latest
```

## Work Items

### Landing Page Repo (`palarix/expo-landing`)

#### 1. Add the logo asset

Copy `xpo.svg` into `public/beacon/xpo.svg` so it's served as a static asset.

#### 2. Create a Cloudflare Pages Function

Create `functions/beacon/xpo.svg.ts` — a Pages Function that:

1. Fetches the static `xpo.svg` from the origin (the built Pages output)
2. Calls `https://api.github.com/repos/palarix/exponential/releases/latest` to get `tag_name`
3. Caches the GitHub response in the Worker Cache API with `max-age=3600` (1 hour)
4. Returns the SVG with headers:
   - `X-Latest: <tag_name>` (e.g. `v1.0.3`)
   - `Access-Control-Allow-Origin: *` (so the web UI can read the header)
   - `Access-Control-Expose-Headers: X-Latest` (required for cross-origin header access)
   - `Cache-Control: public, max-age=86400` (client caches the SVG for 24h — the version header is a bonus, not critical)
   - `Content-Type: image/svg+xml`

The function intercepts requests to `/beacon/xpo.svg`, so the static file at that path serves as fallback if Functions are unavailable.

#### 3. CORS headers

Add `X-Latest` to `Access-Control-Expose-Headers` in the `_headers` file as a belt-and-suspenders fallback:

```
/beacon/*
  Access-Control-Allow-Origin: *
  Access-Control-Expose-Headers: X-Latest
```

### Exponential Repo (`palarix/exponential`)

#### 4. Config: telemetry opt-out

Add `Telemetry bool` field to `config.Config` (default `true`). Also respect `EXPONENTIAL_DISABLE_TELEMETRY` env var.

The server's `/api/config` response should include:
- `telemetry: true/false`
- `telemetry_url: "https://exponential.dev/beacon/xpo.svg"` (only when telemetry is enabled)

#### 5. Web UI: load logo from CDN

In `Layout.tsx`, when telemetry is enabled:
- Change the logo `<img src="/xpo.svg">` to `<img src="https://exponential.dev/beacon/xpo.svg?v={version}">` with `crossOrigin="anonymous"`
- On `onLoad`, read the `X-Latest` response header via a parallel `fetch()` (can't read headers from `<img>`)
- Compare `X-Latest` against the app's current version
- If newer, show a subtle indicator (e.g. a dot on the logo, or a small "Update available" text in the sidebar footer)

When telemetry is disabled, keep using the local `/xpo.svg`.

Fallback: if the CDN fetch fails (network error, timeout), silently fall back to the local logo. The telemetry ping is best-effort.

#### 6. `xpo init` messaging

After project initialization, print a message:

```
Telemetry: Exponential collects anonymous usage data (version + platform)
via a CDN image request. No personal data is collected.
Opt out: set EXPONENTIAL_DISABLE_TELEMETRY=1 or add 'telemetry: false'
to .xpo/config.yml
```

#### 7. README update

Add a "Telemetry" section explaining:
- What is collected (version string appended to a CDN image URL, standard HTTP access logs)
- What is NOT collected (no PII, no cookies, no fingerprinting, no tracking across sessions)
- How to opt out (env var or config)

## Opt-out Mechanism

Telemetry is disabled when ANY of these are true:
- `EXPONENTIAL_DISABLE_TELEMETRY` env var is set (any value)
- `telemetry: false` in `.xpo/config.yml`

## Acceptance Criteria

- [ ] Logo loads from CDN when telemetry is enabled
- [ ] `X-Latest` header is present and contains latest GitHub release tag
- [ ] Web UI shows "update available" indicator when a newer version exists
- [ ] Falls back to local logo on CDN failure
- [ ] Telemetry is disabled by env var or config
- [ ] `xpo init` prints telemetry notice
- [ ] README documents telemetry