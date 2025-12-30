# inbox

A Gmail-focused terminal UI for triage, reading threads, and skimming attachments with a vim-style statusline and theme support.

Features
- Multi-account Gmail inbox with threaded views
- Fast list browsing with configurable snippet lines
- Message detail view with HTML and plain text rendering
- Attachment browser with text and image previews (Kitty protocol for images)
- Search/filter support and auto refresh with a notification bell on new mail
- Theme support via go.withmatt.com/themes with per-surface overrides
- Configurable keymaps and per-account badges

Commands
- `inbox` launches the TUI
- `inbox accounts` opens the account manager (add/remove)
- `inbox theme-preview --name <theme>` prints palette and resolved theme colors

Installation
- `GOEXPERIMENT=jsonv2 go install go.withmatt.com/inbox@latest`
- Make sure `$GOPATH/bin` (or `$(go env GOPATH)/bin`) is on your `PATH`

Configuration
- Config file: `${XDG_CONFIG_HOME:-~/.config}/inbox/config.toml`
- Tokens: `${XDG_CONFIG_HOME:-~/.config}/inbox/tokens/*.json`
- See `config.toml.example` for all options

Development
- Go 1.25+ with `GOEXPERIMENT=jsonv2`
- `just fmt lint` for formatting and linting
- `just run` to run the app

Getting started
- See `GETTING_STARTED.md` for setup and first run
