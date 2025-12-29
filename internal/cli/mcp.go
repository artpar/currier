package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/artpar/currier/internal/mcp"
)

// NewMCPCommand creates the mcp subcommand for starting the MCP server.
func NewMCPCommand() *cobra.Command {
	var dataDir string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for AI assistant integration",
		Long: `Start the Model Context Protocol (MCP) server to enable AI assistants
like Claude to use Currier for API testing and development.

The MCP server communicates over stdio and provides tools for:
- Sending HTTP requests
- Managing collections and environments
- Running collection tests
- Viewing request history
- Managing cookies

Configure in Claude Code's MCP settings:
  {
    "mcpServers": {
      "currier": {
        "command": "currier",
        "args": ["mcp"]
      }
    }
  }`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServer(dataDir)
		},
	}

	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Data directory (default: ~/.config/currier)")

	return cmd
}

func runMCPServer(dataDir string) error {
	// Create MCP server
	server, err := mcp.NewServer(mcp.ServerConfig{
		DataDir: dataDir,
	})
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}
	defer server.Close()

	// Set up context with cancellation on signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Run the server
	return server.Run(ctx, os.Stdin, os.Stdout)
}
