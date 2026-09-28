# Spec: Desktop app via Wails

## What

A native desktop application that wraps the xpo board UI in an OS-native window using [Wails v2](https://wails.io). Users launch the app, pick a repository via a native file dialog, and get the full xpo board experience in a dedicated window — no browser tab, no port juggling.

## Why

The current `xpo board` command opens the web UI in the default browser on a semi-random port (8080–8089 via `FindFreePort`). This works but has friction:

- The board is a browser tab among dozens — easy to lose, no dock/taskbar presence.
- Port varies between launches, so bookmarks and PWA installs break.
- No native OS integration (file dialogs, menu bar, window management).
- Doesn't feel like a standalone tool.

A Wails app solves all of these with minimal new code because Wails is Go-native and the project already has a Go backend + React frontend.

## Why Wails (over alternatives)

| Option | Pros | Cons |
|--------|------|------|
| **Wails** | Go-native, reuses existing code directly, OS webview (~10MB binary), active community | Smaller ecosystem than Electron/Tauri |
| **Tauri** | Mature, OS webview, strong ecosystem | Backend is Rust — requires sidecar for Go binary, adds build complexity |
| **Electron** | Most mature, largest ecosystem | Bundles Chromium (~150MB+), overkill for this use case |

Wails wins because xpo is already a Go project with a React frontend — the integration is nearly direct.

## Architecture

### Key decision: embedded HTTP server vs. Wails bindings

**Option A — Embedded HTTP server (recommended for v1):**
Start the existing `internal/server` HTTP server on a localhost port inside the Wails app process. Point the webview at `http://localhost:<port>`. The React frontend works unchanged.

**Option B — Wails Go↔JS bindings:**
Replace HTTP `fetch()` calls in the frontend with Wails' `runtime.Call()` bindings to Go methods. Eliminates the HTTP server entirely.

**Decision: Option A** for the first iteration.

Rationale:
- Zero changes to the React frontend — every API call, SSE subscription, and asset request works as-is.
- The HTTP server is already tested and production-hardened.
- Option B requires refactoring every `fetch()` call in the frontend and reimplementing SSE over the Wails event system — high effort, low incremental value.
- Option B can be explored later as an optimization if the localhost roundtrip becomes a bottleneck (unlikely for a local tool).

### Process model

```
┌─────────────────────────────────────┐
│  Wails App Process                  │
│                                     │
│  ┌─────────────┐  ┌──────────────┐  │
│  │  Go Backend  │  │  OS Webview  │  │
│  │             │  │              │  │
│  │  HTTP Server │◄─│  React App   │  │
│  │  (localhost) │  │  (embedded)  │  │
│  │             │  │              │  │
│  │  xpo core   │  │              │  │
│  └─────────────┘  └──────────────┘  │
└─────────────────────────────────────┘
```

- The Wails app starts the HTTP server on a random available port (reusing `FindFreePort`).
- The webview loads `http://localhost:<port>`.
- All API calls go through the local HTTP server — the same code path as `xpo board`.
- The Go backend also exposes a few Wails-bound methods for native-only features (file dialogs, window title).

### Project structure

```
cmd/
  exponential/          # existing CLI (unchanged)
  exponential-desktop/  # new Wails desktop entry point
    main.go             # Wails app bootstrap
    app.go              # App struct with Wails-bound methods
internal/
  server/               # existing — reused as-is
  desktop/              # new — desktop-specific logic
    project.go          # project discovery, recent projects list
```

### New entry point: `cmd/exponential-desktop/main.go`

1. Show a project picker on launch (recent projects list + "Open…" button).
2. On project selection, load the xpo config from that directory.
3. Start the HTTP server (reusing `server.NewServer` + `server.Bind`).
4. Create the Wails window pointing at `http://localhost:<port>`.
5. Set the window title to `xpo — <project-name>`.

### Wails-bound Go methods (exposed to JS)

Minimal surface — only things the web app can't do via HTTP:

- `OpenProject() string` — native file dialog, returns selected directory path.
- `GetRecentProjects() []Project` — list of recently opened repositories.
- `GetAppVersion() string` — version string for about/title bar.

### Project picker

On first launch (or when no project is loaded), show a simple landing screen:

- List of recently opened projects (persisted in `~/.config/xpo/desktop.json`).
- "Open Project…" button that triggers the native directory picker.
- Each recent project shows: directory name, full path, last opened timestamp.

This is a React component rendered in the same webview, shown before the board loads. It does not require the HTTP server to be running yet.

### Build integration

Add a `Makefile` target:

```makefile
desktop: frontend
	cd cmd/exponential-desktop && wails build
```

Wails embeds the frontend assets the same way the CLI does — via `embed.FS`. The existing `internal/server/static/` embedded assets are reused; additionally, Wails needs its own `frontend/dist` for the project picker screen, or the picker can be part of the same React app with a route guard.

**Simpler approach:** Add a `/picker` route to the existing React app that renders the project picker. The Wails app loads `/picker` initially; selecting a project starts the HTTP server and navigates to `/`. This keeps everything in one frontend build.

### Desktop metadata

- **App name:** `Exponential`
- **App icon:** Generated from `web/public/xpo.svg` (need 1024×1024 PNG for macOS `.icns`, 256×256 for Windows `.ico`)
- **Bundle ID (macOS):** `io.palarix.exponential`
- **Window size:** 1200×800 default, resizable, min 800×600

## Scope

### In scope (v1)
- Wails app skeleton with project picker
- Embedded HTTP server serving the existing React frontend
- Native "Open Project" file dialog
- Recent projects list (persisted locally)
- macOS build (primary development platform)
- Makefile target for building

### Out of scope (future)
- Windows and Linux builds (Wails supports them, but testing/packaging deferred)
- Multi-window / multi-project simultaneously
- Auto-update mechanism
- Replacing HTTP with Wails bindings (Option B)
- System tray / menu bar integration
- Native notifications

## Acceptance criteria

- [ ] `make desktop` produces a working macOS `.app` bundle.
- [ ] Launching the app shows a project picker with an "Open…" button.
- [ ] Selecting a repository starts the xpo board in a native window.
- [ ] The full board UI works identically to `xpo board` in a browser (issues, drag-drop, keyboard shortcuts, SSE updates).
- [ ] Recently opened projects are remembered across app restarts.
- [ ] Window title shows the project name.
- [ ] `make test` still passes (no regressions to CLI or web).

## Open questions

1. **Wails v2 vs v3?** — Wails v3 is in alpha. v2 is stable and well-documented. Recommend v2 for now.
2. **Single React app or separate picker app?** — Recommend single app with a `/picker` route to avoid maintaining two frontends.
3. **Where to persist recent projects?** — `~/.config/xpo/desktop.json` (XDG-compatible). Could also use the OS keychain/preferences but that's overkill.
