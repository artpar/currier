package interfaces

import (
	"context"
)

// Plugin represents a loadable extension.
type Plugin interface {
	// Metadata returns plugin metadata.
	Metadata() PluginMetadata

	// Activate initializes the plugin.
	Activate(ctx PluginContext) error

	// Deactivate cleans up the plugin.
	Deactivate() error
}

// PluginMetadata contains plugin information.
type PluginMetadata struct {
	Name        string
	Version     string
	Description string
	Author      string
	License     string
	Homepage    string
	Requires    string // Minimum Currier version
	Hooks       []string
	Settings    []PluginSetting
	Enabled     bool
	Path        string
}

// PluginSetting describes a plugin configuration option.
type PluginSetting struct {
	Name        string
	Type        string // "string", "number", "boolean", "select"
	Description string
	Default     any
	Required    bool
	Enum        []string // For "select" type
}

// PluginContext provides services to plugins.
type PluginContext interface {
	// Log logs a message.
	Log(msg string)

	// LogError logs an error.
	LogError(msg string, err error)

	// Settings returns plugin settings.
	Settings() map[string]any

	// GetSetting retrieves a specific setting.
	GetSetting(key string) (any, bool)

	// RegisterHook registers a hook handler.
	RegisterHook(hook string, handler HookHandler) error

	// RegisterCommand registers a custom command.
	RegisterCommand(name string, cmd PluginCommand) error

	// RegisterAuthProvider registers a custom auth type.
	RegisterAuthProvider(provider AuthProvider) error

	// RegisterFormatter registers a custom body formatter.
	RegisterFormatter(formatter BodyFormatter) error

	// RegisterImporter registers a custom importer.
	RegisterImporter(importer Importer) error

	// RegisterExporter registers a custom exporter.
	RegisterExporter(exporter Exporter) error
}

// HookHandler is a function that handles a hook event.
type HookHandler func(ctx context.Context, data any) (any, error)

// PluginCommand represents a custom command.
type PluginCommand struct {
	Name        string
	Description string
	Execute     func(args []string, ctx PluginCommandContext) error
}

// PluginCommandContext provides context for command execution.
type PluginCommandContext interface {
	// Output writes output to the user.
	Output(msg string)

	// Error writes an error message.
	Error(msg string)

	// Prompt asks for user input.
	Prompt(msg string) (string, error)

	// Confirm asks for confirmation.
	Confirm(msg string) (bool, error)
}

// AuthProvider defines a custom authentication type.
type AuthProvider struct {
	Name        string
	DisplayName string
	Fields      []AuthField
	Apply       func(req Request, params map[string]string) error
}

// AuthField describes an authentication field.
type AuthField struct {
	Name        string
	Type        string // "string", "password", "select"
	Label       string
	Required    bool
	Default     string
	Placeholder string
}

// BodyFormatter provides custom body formatting.
type BodyFormatter struct {
	Name         string
	DisplayName  string
	ContentTypes []string
	Format       func(body []byte) (string, error)
	Parse        func(formatted string) ([]byte, error)
}

// PluginManager manages plugin lifecycle.
type PluginManager interface {
	// Install installs a plugin from a source.
	Install(ctx context.Context, source string) error

	// Uninstall removes a plugin.
	Uninstall(ctx context.Context, name string) error

	// Enable enables a plugin.
	Enable(ctx context.Context, name string) error

	// Disable disables a plugin.
	Disable(ctx context.Context, name string) error

	// List returns all installed plugins.
	List() []PluginMetadata

	// Get retrieves a plugin by name.
	Get(name string) (Plugin, error)

	// Reload reloads a plugin.
	Reload(ctx context.Context, name string) error

	// ExecuteHook executes all handlers for a hook.
	ExecuteHook(ctx context.Context, hook string, data any) (any, error)

	// GetAuthProviders returns all registered auth providers.
	GetAuthProviders() []AuthProvider

	// GetFormatters returns all registered formatters.
	GetFormatters() []BodyFormatter

	// GetCommands returns all registered commands.
	GetCommands() []PluginCommand
}

// Hook names
const (
	HookPreRequest   = "pre_request"
	HookPostResponse = "post_response"
	HookFormatBody   = "format_body"
	HookAuthProvider = "auth_provider"
	HookImporter     = "importer"
	HookExporter     = "exporter"
	HookCommand      = "command"
	HookKeybinding   = "keybinding"
)
