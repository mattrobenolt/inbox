# Getting started

This guide covers the first run, account setup, and basic customization.

Prerequisites
- Go 1.25+

Install
- `GOEXPERIMENT=jsonv2 go install go.withmatt.com/inbox@latest`
- Ensure `$(go env GOPATH)/bin` is on your `PATH`

Build and run
1. Build (optional)
   - `just build`
   - Or: `GOEXPERIMENT=jsonv2 go build .`
2. Add an account
   - `inbox accounts`
   - Choose "Add account" and complete the browser OAuth flow.
3. Launch the TUI
   - `inbox`

Configuration
- Location: `${XDG_CONFIG_HOME:-~/.config}/inbox/config.toml`
- Example file: `config.toml.example`

Common tweaks
- Inbox snippet lines: `ui.list_snippet_lines`
- Auto refresh interval (seconds): `ui.refresh_interval_seconds` (set `-1` to disable)
- Theme selection: `theme.name` (from go.withmatt.com/themes)
- Theme overrides: `theme.status`, `theme.list`, `theme.detail`, `theme.attachment`, `theme.image`, `theme.modal`
- Keymaps: `keys.list`, `keys.detail`, `keys.search`, `keys.attachment`, `keys.image`, `keys.attachments_modal`
- Account badges: `accounts[].badge_fg`, `accounts[].badge_bg`

Theme preview
- Use `inbox theme-preview --name Nord` to inspect palette and resolved colors

Notes
- OAuth tokens are stored at `${XDG_CONFIG_HOME:-~/.config}/inbox/tokens/`.
- Cache lives at `${XDG_CACHE_HOME:-~/.cache}/inbox/threads.json`.
- Image previews use the Kitty graphics protocol; in other terminals you will only see metadata.
