package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"go.withmatt.com/inbox/internal/log"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "inbox",
	Short:   "A unified inbox reader for Gmail",
	Long:    `inbox is a terminal-based unified inbox reader for multiple Gmail accounts.`,
	Version: version,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	RunE: runTUI,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		debug, _ := cmd.PersistentFlags().GetBool("debug")
		return log.Setup(debug)
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		return log.Close()
	},
}

func Execute() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
