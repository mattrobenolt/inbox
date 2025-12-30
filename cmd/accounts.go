package cmd

import (
	"github.com/spf13/cobra"

	"go.withmatt.com/inbox/internal/config"
	"go.withmatt.com/inbox/internal/tui"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage accounts",
	Long:  "Launch an interactive account manager to add or remove Gmail accounts.",
	RunE:  runAccounts,
}

func init() {
	rootCmd.AddCommand(accountsCmd)
}

func runAccounts(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	return tui.RunAccounts(cmd.Context(), cfg)
}
