package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/artpar/currier/internal/proxy"
	"github.com/spf13/cobra"
)

// ProxyOptions holds options for the proxy command.
type ProxyOptions struct {
	ListenAddr   string
	EnableHTTPS  bool
	ExportCA     string
	Verbose      bool
	BufferSize   int
	ExcludeHosts []string
	IncludeHosts []string
}

// NewProxyCommand creates the proxy command.
func NewProxyCommand() *cobra.Command {
	opts := &ProxyOptions{}

	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Start HTTP/HTTPS proxy server for traffic capture",
		Long: `Start an HTTP/HTTPS proxy server that captures all traffic passing through it.

Configure applications to use this proxy to capture their traffic.
For HTTPS interception, export and install the CA certificate.

Examples:
  # Start proxy (auto-assigns available port)
  currier proxy

  # Start proxy on specific port
  currier proxy --port 8080

  # Export CA certificate for HTTPS interception
  currier proxy --export-ca ~/currier-ca.crt

  # Start proxy with HTTPS disabled
  currier proxy --no-https

  # Only capture traffic to specific hosts
  currier proxy --include api.example.com --include *.test.com
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxy(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ListenAddr, "port", "p", ":0", "Port to listen on (:0 for auto, :8080 for specific port)")
	cmd.Flags().BoolVar(&opts.EnableHTTPS, "https", true, "Enable HTTPS interception (requires CA cert)")
	cmd.Flags().StringVar(&opts.ExportCA, "export-ca", "", "Export CA certificate to specified path and exit")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose logging")
	cmd.Flags().IntVar(&opts.BufferSize, "buffer", 1000, "Maximum number of captures to keep in memory")
	cmd.Flags().StringArrayVar(&opts.ExcludeHosts, "exclude", nil, "Hosts to exclude from capture (supports wildcards like *.example.com)")
	cmd.Flags().StringArrayVar(&opts.IncludeHosts, "include", nil, "Only capture traffic to these hosts (supports wildcards)")

	return cmd
}

func runProxy(cmd *cobra.Command, opts *ProxyOptions) error {
	// Build proxy configuration
	config := proxy.NewConfig(
		proxy.WithListenAddr(opts.ListenAddr),
		proxy.WithHTTPS(opts.EnableHTTPS),
		proxy.WithBufferSize(opts.BufferSize),
		proxy.WithVerbose(opts.Verbose),
	)

	if len(opts.ExcludeHosts) > 0 {
		config.ExcludeHosts = opts.ExcludeHosts
	}
	if len(opts.IncludeHosts) > 0 {
		config.IncludeHosts = opts.IncludeHosts
	}

	// Create proxy server
	server, err := proxy.NewServer(
		proxy.WithListenAddr(opts.ListenAddr),
		proxy.WithHTTPS(opts.EnableHTTPS),
		proxy.WithBufferSize(opts.BufferSize),
		proxy.WithVerbose(opts.Verbose),
		proxy.WithExcludeHosts(opts.ExcludeHosts...),
		proxy.WithIncludeHosts(opts.IncludeHosts...),
	)
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	// Handle CA export
	if opts.ExportCA != "" {
		if server.TLSConfig() == nil {
			return fmt.Errorf("HTTPS is disabled, no CA certificate to export")
		}
		if err := server.ExportCACert(opts.ExportCA); err != nil {
			return fmt.Errorf("failed to export CA certificate: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "CA certificate exported to: %s\n", opts.ExportCA)
		fmt.Fprintf(cmd.OutOrStdout(), "\nTo enable HTTPS interception, install this certificate as a trusted CA:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  macOS: security add-trusted-cert -d -r trustRoot -k ~/Library/Keychains/login.keychain \"%s\"\n", opts.ExportCA)
		fmt.Fprintf(cmd.OutOrStdout(), "  Linux: sudo cp \"%s\" /usr/local/share/ca-certificates/ && sudo update-ca-certificates\n", opts.ExportCA)
		fmt.Fprintf(cmd.OutOrStdout(), "  Windows: certutil -addstore -user Root \"%s\"\n", opts.ExportCA)
		return nil
	}

	// Start proxy server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}

	// Print startup info - format address for display
	addr := server.ListenAddr()
	// Handle IPv6 any address [::]:port -> localhost:port
	if len(addr) > 4 && addr[:4] == "[::]" {
		addr = "localhost" + addr[4:]
	} else if len(addr) > 0 && addr[0] == ':' {
		addr = "localhost" + addr
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Proxy server started on %s\n", addr)
	if opts.EnableHTTPS {
		fmt.Fprintf(cmd.OutOrStdout(), "HTTPS interception enabled\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  Export CA: currier proxy --export-ca ca.crt\n")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "HTTPS interception disabled (--https=false)\n")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nConfigure your applications to use this proxy:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  export http_proxy=http://%s\n", addr)
	fmt.Fprintf(cmd.OutOrStdout(), "  export https_proxy=http://%s\n", addr)
	fmt.Fprintf(cmd.OutOrStdout(), "\nOr use directly with curl:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  curl --proxy http://%s URL\n", addr)
	fmt.Fprintf(cmd.OutOrStdout(), "\nPress Ctrl+C to stop...\n")

	// Add a simple capture listener for console output
	if opts.Verbose {
		server.AddListener(proxy.CaptureListenerFunc(func(capture *proxy.CapturedRequest) {
			protocol := "HTTP"
			if capture.IsHTTPS {
				protocol = "HTTPS"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s %s -> %d (%dms)\n",
				protocol, capture.Method, capture.URL, capture.StatusCode, capture.Duration.Milliseconds())
		}))
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Fprintf(cmd.OutOrStdout(), "\nShutting down proxy server...\n")
	cancel()

	if err := server.Stop(); err != nil {
		return fmt.Errorf("error stopping proxy server: %w", err)
	}

	// Print capture stats
	stats := server.Stats()
	fmt.Fprintf(cmd.OutOrStdout(), "Captured %d requests\n", stats.TotalCount)

	return nil
}
