package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"go.withmatt.com/inbox/internal/config"
	"go.withmatt.com/inbox/internal/oauth"
)

type accountAction string

const (
	accountActionAdd    accountAction = "add"
	accountActionRemove accountAction = "remove"
	accountActionQuit   accountAction = "quit"
)

func RunAccounts(ctx context.Context, cfg *config.Config) error {
	if ctx == nil {
		ctx = context.Background()
	}

	status := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		action, err := runAccountsMenu(ctx, cfg, status)
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}

		var nextStatus string
		switch action {
		case accountActionAdd:
			nextStatus, err = runAddAccount(ctx, cfg)
		case accountActionRemove:
			nextStatus, err = runRemoveAccount(ctx, cfg)
		case accountActionQuit:
			return nil
		default:
			nextStatus = "Unknown action."
		}
		if err != nil {
			return err
		}
		status = nextStatus
	}
}

func runAccountsMenu(
	ctx context.Context,
	cfg *config.Config,
	status string,
) (accountAction, error) {
	action := accountActionAdd
	options := []huh.Option[accountAction]{
		huh.NewOption("Add account", accountActionAdd),
	}
	if len(cfg.Accounts) > 0 {
		options = append(options, huh.NewOption("Remove account", accountActionRemove))
	}
	options = append(options, huh.NewOption("Quit", accountActionQuit))

	fields := make([]huh.Field, 0, 3)
	if status != "" {
		fields = append(fields,
			huh.NewNote().Title("Status").Description(status),
		)
	}
	fields = append(fields,
		huh.NewNote().Title("Accounts").Description(formatAccountsNote(cfg)),
		huh.NewSelect[accountAction]().
			Title("Action").
			Options(options...).
			Value(&action),
	)

	form := huh.NewForm(huh.NewGroup(fields...)).
		WithProgramOptions(tea.WithAltScreen())
	if err := form.RunWithContext(ctx); err != nil {
		return "", err
	}
	return action, nil
}

func runAddAccount(ctx context.Context, cfg *config.Config) (string, error) {
	var name string
	var email string
	validateNotBlank := func(value string) error {
		if strings.TrimSpace(value) == "" {
			return errors.New("required")
		}
		return nil
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Placeholder("Personal").
				Validate(validateNotBlank).
				Value(&name),
			huh.NewInput().
				Title("Email").
				Placeholder("you@example.com").
				Validate(validateNotBlank).
				Value(&email),
		),
	).WithProgramOptions(tea.WithAltScreen())

	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "Add canceled.", nil
		}
		return "", err
	}

	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" || email == "" {
		return "Name and email are required.", nil
	}
	if findAccount(cfg, email) != nil {
		return "Account already exists.", nil
	}

	fmt.Fprintln(os.Stderr, "Opening browser for Gmail authentication...")
	tokenPath, err := config.TokenPath(email)
	if err != nil {
		return "", err
	}
	if _, err := oauth.GetClientQuiet(ctx, tokenPath, email); err != nil {
		return fmt.Sprintf("Auth failed: %v", err), nil
	}

	cfg.Accounts = append(cfg.Accounts, config.Account{
		Name:  name,
		Email: email,
	})
	if err := config.Save(cfg); err != nil {
		return fmt.Sprintf("Save failed: %v", err), nil
	}

	return "Added " + email, nil
}

func runRemoveAccount(ctx context.Context, cfg *config.Config) (string, error) {
	if len(cfg.Accounts) == 0 {
		return "No accounts configured.", nil
	}

	selected := cfg.Accounts[0].Email
	removeToken := true
	confirm := false

	options := make([]huh.Option[string], 0, len(cfg.Accounts))
	for _, account := range cfg.Accounts {
		label := account.Email
		if strings.TrimSpace(account.Name) != "" {
			label = fmt.Sprintf("%s <%s>", account.Name, account.Email)
		}
		options = append(options, huh.NewOption(label, account.Email))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Account").
				Options(options...).
				Value(&selected),
			huh.NewConfirm().
				Title("Delete token file?").
				Affirmative("Yes").
				Negative("No").
				Value(&removeToken),
			huh.NewConfirm().
				Title("Remove this account?").
				Affirmative("Remove").
				Negative("Cancel").
				Value(&confirm),
		),
	).WithProgramOptions(tea.WithAltScreen())

	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "Remove canceled.", nil
		}
		return "", err
	}

	if !confirm {
		return "Remove canceled.", nil
	}

	if err := removeAccount(cfg, selected, removeToken); err != nil {
		return fmt.Sprintf("Remove failed: %v", err), nil
	}
	return "Removed " + selected, nil
}

func formatAccountsNote(cfg *config.Config) string {
	if len(cfg.Accounts) == 0 {
		return "No accounts configured."
	}

	lines := make([]string, 0, len(cfg.Accounts))
	for _, account := range cfg.Accounts {
		label := account.Email
		if strings.TrimSpace(account.Name) != "" {
			label = fmt.Sprintf("%s <%s>", account.Name, account.Email)
		}
		lines = append(lines, "- "+label)
	}
	return strings.Join(lines, "\n")
}

func findAccount(cfg *config.Config, email string) *config.Account {
	for i := range cfg.Accounts {
		if strings.EqualFold(cfg.Accounts[i].Email, email) {
			return &cfg.Accounts[i]
		}
	}
	return nil
}

func removeAccount(cfg *config.Config, email string, removeToken bool) error {
	filtered := make([]config.Account, 0, len(cfg.Accounts))
	for _, account := range cfg.Accounts {
		if strings.EqualFold(account.Email, email) {
			continue
		}
		filtered = append(filtered, account)
	}
	cfg.Accounts = filtered
	if err := config.Save(cfg); err != nil {
		return err
	}

	if removeToken {
		tokenPath, err := config.TokenPath(email)
		if err == nil {
			if _, err := os.Stat(tokenPath); err == nil {
				if err := os.Remove(tokenPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
