package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command.
func NewRootCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "currier",
		Short:   "Currier - A TUI API client",
		Long:    "Currier is a vim-modal TUI API client for developers and AI agents.",
		Version: version,
	}

	// Add subcommands
	cmd.AddCommand(NewSendCommand())

	return cmd
}
