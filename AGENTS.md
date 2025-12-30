# AGENTS.md

> **Context:** This is `inbox`, a TUI Gmail client written in Go. It allows users to triage, read, and search emails from the terminal.

## 1. Build & Run Commands

The environment is assumed to be pre-configured with all necessary dependencies (Go, `just`, `golangci-lint`, etc.).

We **strictly** use `just` as a command runner. Do not use manual `go` commands.

- **Build:**
  ```bash
  just build
  ```

- **Run:**
  ```bash
  just run
  ```

## 2. Verification

**There are currently no tests.**

After making any changes, you **must** run the formatter and linter to verify your work:

```bash
just fmt lint
```

## 3. Project Structure

- **`cmd/`**: CLI command definitions (Cobra).
  - `root.go`: Main entry point.
  - `tui.go`: Launches the main TUI application.
  - `accounts.go`: Account management CLI.
- **`internal/`**: Private application code.
  - **`tui/`**: The Bubble Tea application logic (models, updates, views).
  - **`gmail/`**: Gmail API client and type definitions.
  - **`config/`**: Configuration loading and theme definitions.
- **`main.go`**: Shim that calls `cmd.Execute()`.

## 4. Development Standards

- **Linting:** Strict adherence to `golangci-lint`.
- **Formatting:** Standard `gofmt` style.
- **UI Framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea).
- **Styling:** [Lip Gloss](https://github.com/charmbracelet/lipgloss).
