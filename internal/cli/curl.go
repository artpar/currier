package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/importer"
)

// NewCurlCommand creates a command that parses a curl command and opens TUI.
func NewCurlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "curl [curl command arguments...]",
		Short: "Import a curl command and open in TUI",
		Long: `Parse a curl command and open the TUI with the request ready to send.

Examples:
  currier curl https://httpbin.org/get
  currier curl -X POST https://httpbin.org/post -H "Content-Type: application/json" -d '{"name": "test"}'
  currier curl -u admin:secret https://api.example.com/protected`,
		DisableFlagParsing: true, // Pass all args to curl parser
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no curl arguments provided")
			}

			// Reconstruct curl command from args, properly quoting args that need it
			curlCmd := "curl " + quoteArgs(args)

			// Parse using existing importer
			curlImporter := importer.NewCurlImporter()
			collection, err := curlImporter.Import(context.Background(), []byte(curlCmd))
			if err != nil {
				return fmt.Errorf("failed to parse curl command: %w", err)
			}

			// Start TUI with the collection
			return runTUI([]*core.Collection{collection}, nil)
		},
	}
	return cmd
}

// quoteArgs properly quotes arguments that contain spaces, quotes, or other special chars
func quoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		// Check if arg needs quoting (contains space, quotes, or shell special chars)
		needsQuotes := strings.ContainsAny(arg, " \t\n\"'\\{}[]<>|&;$`")
		if needsQuotes {
			// Escape any existing double quotes and wrap in double quotes
			escaped := strings.ReplaceAll(arg, `"`, `\"`)
			quoted[i] = `"` + escaped + `"`
		} else {
			quoted[i] = arg
		}
	}
	return strings.Join(quoted, " ")
}
