package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"go.withmatt.com/inbox/internal/config"
	"go.withmatt.com/inbox/internal/gmail"
	"go.withmatt.com/inbox/internal/oauth"
	"go.withmatt.com/inbox/internal/tui"
)

func runTUI(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()
	if debug {
		os.Setenv("INBOX_DEBUG", "1")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	// Check if we have accounts configured
	if len(cfg.Accounts) == 0 {
		return errors.New("no accounts configured. Run 'inbox accounts' to add an account")
	}

	// Create clients for all accounts
	var clients []*gmail.Client
	var accountNames []string
	var accountBadges []tui.AccountBadge
	for _, account := range cfg.Accounts {
		srv, err := getGmailService(ctx, account.Email)
		if err != nil {
			return fmt.Errorf("unable to create Gmail service for %s: %w", account.Email, err)
		}
		clients = append(clients, gmail.NewClient(srv))
		accountNames = append(accountNames, account.Name)
		badgeFg, err := config.ResolveColor(account.BadgeFg, cfg.Theme)
		if err != nil {
			return fmt.Errorf("unable to resolve badge_fg for %s: %w", account.Name, err)
		}
		badgeBg, err := config.ResolveColor(account.BadgeBg, cfg.Theme)
		if err != nil {
			return fmt.Errorf("unable to resolve badge_bg for %s: %w", account.Name, err)
		}
		accountBadges = append(accountBadges, tui.AccountBadge{
			Name: account.Name,
			Fg:   badgeFg,
			Bg:   badgeBg,
		})
	}

	theme, err := config.ResolveTheme(cfg.Theme)
	if err != nil {
		return fmt.Errorf("unable to resolve theme: %w", err)
	}
	uiConfig := cfg.UI.WithDefaults()
	if err := tui.Run(ctx, clients, accountNames, accountBadges, theme, uiConfig, cfg.Keys); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}
	return nil
}

func getGmailService(ctx context.Context, email string) (*gmailapi.Service, error) {
	// Get OAuth token
	client, err := oauth.GetClientQuiet(ctx, email)
	if err != nil {
		return nil, err
	}

	// Create Gmail service
	srv, err := gmailapi.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return srv, nil
}
