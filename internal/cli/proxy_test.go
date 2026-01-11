package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProxyCommand(t *testing.T) {
	t.Run("creates proxy command", func(t *testing.T) {
		cmd := NewProxyCommand()
		assert.NotNil(t, cmd)
		assert.Equal(t, "proxy", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("has port flag", func(t *testing.T) {
		cmd := NewProxyCommand()
		flag := cmd.Flags().Lookup("port")
		require.NotNil(t, flag)
		assert.Equal(t, "p", flag.Shorthand)
		assert.Equal(t, ":0", flag.DefValue)
	})

	t.Run("has https flag", func(t *testing.T) {
		cmd := NewProxyCommand()
		flag := cmd.Flags().Lookup("https")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has export-ca flag", func(t *testing.T) {
		cmd := NewProxyCommand()
		flag := cmd.Flags().Lookup("export-ca")
		require.NotNil(t, flag)
	})

	t.Run("has verbose flag", func(t *testing.T) {
		cmd := NewProxyCommand()
		flag := cmd.Flags().Lookup("verbose")
		require.NotNil(t, flag)
		assert.Equal(t, "v", flag.Shorthand)
	})

	t.Run("has buffer flag", func(t *testing.T) {
		cmd := NewProxyCommand()
		flag := cmd.Flags().Lookup("buffer")
		require.NotNil(t, flag)
		assert.Equal(t, "1000", flag.DefValue)
	})

	t.Run("has exclude flag", func(t *testing.T) {
		cmd := NewProxyCommand()
		flag := cmd.Flags().Lookup("exclude")
		require.NotNil(t, flag)
	})

	t.Run("has include flag", func(t *testing.T) {
		cmd := NewProxyCommand()
		flag := cmd.Flags().Lookup("include")
		require.NotNil(t, flag)
	})
}

func TestProxyOptions(t *testing.T) {
	t.Run("creates with default values", func(t *testing.T) {
		opts := &ProxyOptions{}
		assert.Empty(t, opts.ListenAddr)
		assert.False(t, opts.EnableHTTPS)
		assert.Empty(t, opts.ExportCA)
		assert.False(t, opts.Verbose)
		assert.Equal(t, 0, opts.BufferSize)
		assert.Nil(t, opts.ExcludeHosts)
		assert.Nil(t, opts.IncludeHosts)
	})

	t.Run("can set all options", func(t *testing.T) {
		opts := &ProxyOptions{
			ListenAddr:   ":8080",
			EnableHTTPS:  true,
			ExportCA:     "/path/to/ca.crt",
			Verbose:      true,
			BufferSize:   2000,
			ExcludeHosts: []string{"*.example.com"},
			IncludeHosts: []string{"api.example.com"},
		}
		assert.Equal(t, ":8080", opts.ListenAddr)
		assert.True(t, opts.EnableHTTPS)
		assert.Equal(t, "/path/to/ca.crt", opts.ExportCA)
		assert.True(t, opts.Verbose)
		assert.Equal(t, 2000, opts.BufferSize)
		assert.Equal(t, []string{"*.example.com"}, opts.ExcludeHosts)
		assert.Equal(t, []string{"api.example.com"}, opts.IncludeHosts)
	})
}

func TestNewRootCommand_HasProxySubcommand(t *testing.T) {
	cmd := NewRootCommand("1.0.0")
	proxyCmd, _, err := cmd.Find([]string{"proxy"})
	require.NoError(t, err)
	assert.Contains(t, proxyCmd.Use, "proxy")
}

func TestNewRootCommand_HasMCPSubcommand(t *testing.T) {
	cmd := NewRootCommand("1.0.0")
	mcpCmd, _, err := cmd.Find([]string{"mcp"})
	require.NoError(t, err)
	assert.Contains(t, mcpCmd.Use, "mcp")
}

func TestNewRootCommand_HasRunSubcommand(t *testing.T) {
	cmd := NewRootCommand("1.0.0")
	runCmd, _, err := cmd.Find([]string{"run"})
	require.NoError(t, err)
	assert.Contains(t, runCmd.Use, "run")
}

func TestNewRootCommand_HasCaptureFlag(t *testing.T) {
	cmd := NewRootCommand("1.0.0")
	flag := cmd.Flags().Lookup("capture")
	require.NotNil(t, flag)
	assert.Equal(t, "c", flag.Shorthand)
}
