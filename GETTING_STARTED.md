# Getting Started with inbox

Welcome to `inbox`! This guide will walk you through setting up your accounts, configuring the UI, and mastering the keybindings.

## 1. Installation

### Prerequisites
- Go 1.25 or later
- **Nix users:** You can use the included `flake.nix` to get a fully configured environment by running `nix develop`.

### Install via Go
```bash
GOEXPERIMENT=jsonv2 go install go.withmatt.com/inbox@latest
```

Ensure `$(go env GOPATH)/bin` is in your system `PATH` so you can run the `inbox` command from anywhere.

## 2. Setting up Accounts

`inbox` uses Google's OAuth2 implementation to securely connect to your Gmail accounts. Your password is never seen or stored by the application.

### Adding your first account

1. Run the account manager:
   ```bash
   inbox accounts
   ```
2. Use the arrow keys to select **[ Add Account ]** and press `Enter`.
3. A browser window will open asking you to sign in with Google.
4. Allow `inbox` access to your Gmail account.
5. Once successful, return to the terminal. You should see your new account listed.

Repeat this process for as many accounts as you need (e.g., Personal, Work).

### Removing an account

To remove an account, run `inbox accounts`, select the account you wish to remove, and press `Enter`. Confirm the deletion when prompted.

## 3. Configuration

`inbox` looks for a configuration file at:
- **macOS:** `~/Library/Application Support/inbox/config.toml`
- **Linux:** `~/.config/inbox/config.toml`

If this file doesn't exist, `inbox` uses sensible defaults. To customize your experience, create this file. You can use the provided example as a base:

```bash
# For macOS:
mkdir -p "~/Library/Application Support/inbox"
cp config.toml.example "~/Library/Application Support/inbox/config.toml"

# For Linux:
mkdir -p ~/.config/inbox
cp config.toml.example ~/.config/inbox/config.toml
```

### Common Settings

**Refresh Interval:**
Control how often `inbox` checks for new mail (in seconds).
```toml
[ui]
refresh_interval_seconds = 300 # 5 minutes
```

**List Snippets:**
Adjust how many lines of the email body are shown in the main list.
```toml
[ui]
list_snippet_lines = 2
```

**Account Badges:**
Customize the color of the account tags in the unified inbox to easily distinguish them.
```toml
[[accounts]]
name = "Work"
email = "me@work.com"
badge_fg = "white"
badge_bg = "red"
```

## 4. Usage & Keybindings

`inbox` is designed to be navigated entirely with the keyboard.

### Global
- **`Ctrl+c`**: Force quit the application anywhere.

### Thread List (Main View)
| Key | Action |
| :--- | :--- |
| `j` / `Down` | Move selection down |
| `k` / `Up` | Move selection up |
| `Enter` | Open selected thread |
| `Space` | Toggle read/unread status |
| `x` | Select thread (for bulk actions) |
| `X` | Clear all selections |
| `d` | Delete thread (or selected threads) |
| `/` | Start searching |
| `r` | Refresh inbox |
| `q` | Quit |

### Thread Detail (Reading View)
| Key | Action |
| :--- | :--- |
| `j` / `Down` | Scroll down |
| `k` / `Up` | Scroll up |
| `Enter` | Toggle expand/collapse of a message |
| `t` | Toggle between HTML (rendered) and Plain Text view |
| `a` | Open attachments menu |
| `Esc` / `q` | Return to thread list |

### Search
- Type your query and press `Enter` to search.
- Press `Esc` to cancel and return to the inbox.

### Attachments
- Use `j`/`k` to navigate the list.
- Press `Enter` or `d` to download the file to your Downloads folder.
- Press `v` to view (if supported).
- Press `Esc` to close.

## 5. Themes

`inbox` supports custom themes. You can specify a theme in your `config.toml`:

```toml
[theme]
name = "Nord" 
```

You can also override specific colors if you want to tweak an existing theme. See `config.toml.example` for the full structure of theme overrides.

To preview how a theme looks, use the CLI tool:
```bash
inbox theme-preview --name Nord
```