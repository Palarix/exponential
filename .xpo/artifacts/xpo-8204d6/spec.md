# Telemetry: anonymous daily heartbeat with an update check

## What

Every xpo process (CLI, MCP server, board) sends at most one small heartbeat per install per day to `https://exponential.dev/api/ping`. The heartbeat carries the version, OS, architecture, the surface that sent it, and a random install ID. The response carries the latest released version, and the board shows "Update available" when it is newer. Telemetry is opt-out. xpo shows a notice before it sends the first heartbeat.

## Why

Adoption is growing, and we know little about who uses which version. The August draft of this spec used the web UI's logo as a beacon. That draft had three gaps:

- **Blind spots:** it counted only the web UI. Users who use only the CLI, or whose agents use only MCP, were never counted.
- **No install count:** every page load was one request, and the browser cached the logo for 24 h. Access logs gave versions and rough activity, not the number of installs.
- **Privacy claims:** CDN access logs keep IP addresses, so "no personal data" was not true by default.

A Go-side heartbeat fixes all three. It reaches every surface, counts installs, and sends only fields we choose. We store only what we choose to store.

## Acceptance Criteria

- [ ] On the first run on a machine, xpo prints the telemetry notice to stderr and sends no heartbeat.
- [ ] Each later run, from any surface, sends at most one heartbeat per UTC day per install.
- [ ] The heartbeat runs in the background, times out after 3 s, and never delays or fails a command.
- [ ] The heartbeat body contains only `install_id`, `version`, `os`, `arch` and `surface`.
- [ ] `XPO_TELEMETRY=false`, `DO_NOT_TRACK=1`, or `telemetry: false` in `~/.config/xpo/config.yaml` disables telemetry.
- [ ] `telemetry: false` in `.xpo/config.yaml` disables telemetry for everyone working on that project.
- [ ] xpo sends no heartbeat when `CI` is set.
- [ ] With telemetry disabled, xpo opens no network connection to exponential.dev.
- [ ] `xpo telemetry` prints the status, the reason for the status, the install ID and the last heartbeat date.
- [ ] `xpo telemetry off` and `xpo telemetry on` write the user-level setting.
- [ ] The board shows "Update available: vX.Y.Z" in the sidebar footer when the latest release is newer, with a link to the release notes.
- [ ] Preferences has a **Telemetry** section that shows the status and the opt-out commands.
- [ ] The README has a Telemetry section: what is sent, what is stored, what is not, and how to opt out.
- [ ] `xpo init` prints the notice.
- [ ] The Worker stores no IP address and no request headers.

## Flow

### Exponential repo

1. Add `Telemetry *bool` (`telemetry`) to `config.Config` in `internal/config/config.go`. Nil means enabled.
2. Add `internal/telemetry/telemetry.go` with `Enabled(cfg) (bool, reason string)`. It checks `DO_NOT_TRACK`, `CI`, `XPO_TELEMETRY` and the config, in that order.
3. Add `InstallID`, `NoticeShown` and `LastPing` to `UserConfig` in `internal/config/user.go`. These values live in `~/.config/xpo/user.yaml`.
4. Add `telemetry.Heartbeat(surface string)`. It returns at once when telemetry is disabled or `LastPing` is today.
5. In `Heartbeat`, show the notice on stderr when `NoticeShown` is false. Then set `NoticeShown`, and do not send.
6. In `Heartbeat`, create a random UUIDv4 `InstallID` when it is empty.
7. Send the POST in a goroutine with a 3 s timeout. On a 2xx response, store `LastPing` and the returned `latest` version.
8. Call `Heartbeat("cli")` from the root command's `PersistentPreRun` in `cmd/exponential`. Skip the `telemetry` subcommand.
9. Call `Heartbeat("mcp")` when the MCP server starts in `internal/mcpserver/server.go`.
10. Call `Heartbeat("board")` when the board server starts. Repeat the call every 24 h while the board runs.
11. Add `telemetry` and `latest_version` to the config response in `internal/server/handlers.go`.
12. Show "Update available" in the sidebar footer in `web/src/components/Layout/Sidebar.tsx` when `latest_version` is newer.
13. Add the Telemetry section to `web/src/components/Preferences/Preferences.tsx`. It is read-only.
14. Add the `xpo telemetry [on|off]` command.
15. Print the notice in `cmd/exponential/init.go`.
16. Add the README Telemetry section.

### Landing repo (`palarix/expo-landing`)

17. Add a Cloudflare Pages Function at `functions/api/ping.ts`.
18. In the Function, validate the body and write one data point to Workers Analytics Engine: `install_id`, `version`, `os`, `arch`, `surface`.
19. In the Function, return `{ "latest": "<tag>" }`. Cache the GitHub `releases/latest` response for 1 h.

## Decisions

- **A Go heartbeat, not the logo beacon.** It counts CLI and MCP users, counts installs, and keeps browser headers out of the data. The logo goes back to being a local file.
- **A random install ID, not a hash of machine data.** A random ID cannot be traced back to a machine or a person. Deleting `user.yaml` resets it. The README says this.
- **One heartbeat per day.** That is enough for daily and monthly active installs and version spread. It keeps traffic and data small.
- **A notice before the first heartbeat.** Nobody is counted before they could have seen how to opt out.
- **Opt-out at user level and project level.** A user opts out on their machine. A team can opt out a whole project through the committed `.xpo/config.yaml`.
- **`DO_NOT_TRACK` and `CI` are respected.** These are common conventions, and CI runs would inflate the install count.
- **Workers Analytics Engine, not access logs.** Analytics Engine stores only the fields we write. Pages Functions do not keep IP logs unless Logpush is set up.
- **We do not store country.** It is useful, but we can add it later with a README update.
- **The update check rides on the heartbeat.** With telemetry off, there is no update check. A separate check would be a second network call that users did not opt out of.

## Edge Cases

- **HIGH:** an MCP server writes the notice to stdout and breaks the protocol. The notice always goes to stderr.
- **HIGH:** the endpoint is down or slow, and commands hang. The send runs in a goroutine with a 3 s timeout, and the CLI does not wait for it on exit.
- **MEDIUM:** several processes start at the same moment and each sends a heartbeat. Accept duplicates. The analysis counts distinct install IDs per day.
- **MEDIUM:** `user.yaml` is not writable. xpo then sends no heartbeat, because without a saved `NoticeShown` it would show the notice every run.
- **LOW:** dev builds. They send their version string as-is, and the analysis filters them out.

## Assumptions

- The landing site runs on Cloudflare Pages at `exponential.dev`, and we can deploy Functions there.
- GitHub Releases is the source of truth for the latest version.
- Releases are public, so the update link points to `github.com/palarix/exponential/releases`.

## Open Questions

1. Is a random install ID acceptable to you, or do you want counts only, with no ID at all?
2. Should we also store a coarse country from Cloudflare's `cf.country`?
3. Should the CLI also print "update available" (for example in `xpo version` and `xpo doctor`), or only the board?
4. The landing repo is outside this worktree. Should its part be a separate issue?
