# inbox

> A fast, beautiful, and distraction-free Gmail client for your terminal.

<img width="810" height="765" alt="CleanShot 2025-12-30 at 14 36 50@2x" src="https://github.com/user-attachments/assets/2a989a31-20fb-4ac4-92e6-c76d3dc520ef" />


`inbox` is a TUI built for power users who live in the terminal and want a keyboard-centric way to triage and read email. It supports multiple Gmail accounts, unified searching, and renders HTML emails beautifully.

## Features

- **vim-style Navigation:** Navigate your inbox without ever touching the mouse.
- **HTML Rendering:** Rich text emails are rendered cleanly to the terminal, with a plain-text fallback toggle.
- **Multiple Accounts:** Unified interface for all your Gmail accounts with color-coded badges.
- **Attachment Support:** Browse attachments and preview images directly in the terminal (Kitty protocol support).
- **Archive & Delete:** Archive or trash threads with confirmation and bulk selection.
- **Themable:** First-class theme support with per-element overrides.
- **Search:** Fast, server-side search integration.

## Installation

### Install via Homebrew
```bash
brew install --cask mattrobenolt/stuff/inbox
```

### Install via GitHub Releases
Download the latest release for your platform from:
https://github.com/mattrobenolt/inbox/releases

### Install via curl (GitHub Releases)
```bash
curl -sSfL https://raw.githubusercontent.com/mattrobenolt/inbox/main/install.sh | sh
```
To customize:
- `VERSION=0.1.0` to pin a version
- `BIN_DIR=$HOME/.local/bin` to choose an install dir

### Install via Go

```bash
GOEXPERIMENT=jsonv2 go install go.withmatt.com/inbox@latest
```
Ensure your `$GOPATH/bin` is in your `$PATH` to run the `inbox` command.

## Quick Start

1. **Install** `inbox` using Homebrew or GitHub Releases.
2. **Add an account:**
   ```bash
   inbox accounts
   ```
   Select "Add account" and follow the browser prompts to authenticate with Google.
3. **Launch:**
   ```bash
   inbox
   ```

For detailed configuration, keybindings, and advanced usage, check out the [Getting Started Guide](GETTING_STARTED.md).

## Configuration

Configuration is stored in:
- **macOS:** `~/Library/Application Support/inbox/config.toml`
- **Linux:** `~/.config/inbox/config.toml`

You can configure:
- Custom themes and colors
- Keybindings
- Interface density (snippet lines)
- Account badges

See [`config.toml.example`](config.toml.example) for a full reference.

## Development

If you'd like to contribute or build from source:

### Using Nix (Recommended)
This project includes a Nix flake that sets up a complete development environment with Go 1.25+, `golangci-lint`, and `just`.

```bash
nix develop
# or if you use direnv
direnv allow
```

### Manual Setup
```bash
git clone https://github.com/mattrobenolt/inbox
cd inbox
just build
./inbox
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
