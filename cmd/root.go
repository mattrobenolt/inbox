package cmd

import (
	"fmt"
	"os"

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

func Execute() {
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
