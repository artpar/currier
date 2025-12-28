package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/history/sqlite"
	"github.com/artpar/currier/internal/importer"
	"github.com/artpar/currier/internal/interpolate"
	"github.com/artpar/currier/internal/storage/filesystem"
	"github.com/artpar/currier/internal/tui/views"
)

// NewRootCommand creates the root command.
func NewRootCommand(version string) *cobra.Command {
	var importFiles []string
	var envFiles []string

	cmd := &cobra.Command{
		Use:     "currier",
		Short:   "Currier - A TUI API client",
		Long:    "Currier is a vim-modal TUI API client for developers and AI agents.",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load collections from import files
			collections, err := loadCollections(importFiles)
			if err != nil {
				return fmt.Errorf("failed to load collections: %w", err)
			}

			// Load environment from files
			var env *core.Environment
			if len(envFiles) > 0 {
				env, err = core.LoadMultipleEnvironments(envFiles)
				if err != nil {
					return fmt.Errorf("failed to load environment: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Loaded environment: %s\n", env.Name())
			}

			// Merge collection variables into environment
			if len(collections) > 0 {
				env = core.MergeCollectionVariables(env, collections)
			}

			return runTUI(collections, env)
		},
	}

	// Add flags
	cmd.Flags().StringArrayVarP(&importFiles, "import", "i", nil,
		"Import collection file(s). Supports Postman, OpenAPI, cURL, HAR formats")
	cmd.Flags().StringArrayVarP(&envFiles, "env", "e", nil,
		"Environment file(s) for variable substitution")

	// Add subcommands
	cmd.AddCommand(NewSendCommand())
	cmd.AddCommand(NewCurlCommand())

	return cmd
}

// loadCollections loads collections from the given file paths.
func loadCollections(paths []string) ([]*core.Collection, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	registry := newImporterRegistry()
	collections := make([]*core.Collection, 0, len(paths))

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", path, err)
		}

		result, err := registry.DetectAndImport(context.Background(), content)
		if err != nil {
			return nil, fmt.Errorf("failed to import %s: %w", path, err)
		}

		collections = append(collections, result.Collection)
		fmt.Fprintf(os.Stderr, "Loaded collection: %s (format: %s)\n",
			result.Collection.Name(), result.SourceFormat)
	}

	return collections, nil
}

// newImporterRegistry creates a registry with all available importers.
func newImporterRegistry() *importer.Registry {
	registry := importer.NewRegistry()
	registry.Register(importer.NewPostmanImporter())
	registry.Register(importer.NewOpenAPIImporter())
	registry.Register(importer.NewCurlImporter())
	registry.Register(importer.NewHARImporter())
	return registry
}

// tuiModel wraps the MainView for bubbletea
type tuiModel struct {
	view *views.MainView
}

func (m tuiModel) Init() tea.Cmd {
	return m.view.Init()
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.view.Update(msg)
	m.view = updated.(*views.MainView)
	return m, cmd
}

func (m tuiModel) View() string {
	return m.view.View()
}

// runTUI starts the TUI application with optional collections and environment.
func runTUI(collections []*core.Collection, env *core.Environment) error {
	view := views.NewMainView()

	// Initialize history store
	historyStore, err := initHistoryStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not initialize history: %v\n", err)
	} else {
		view.SetHistoryStore(historyStore)
	}

	// Initialize collection store for persistence
	collectionStore, err := initCollectionStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not initialize collection store: %v\n", err)
	} else {
		view.SetCollectionStore(collectionStore)
	}

	// Load collections if provided
	if len(collections) > 0 {
		view.SetCollections(collections)
	}

	// Set up interpolation engine with environment
	if env != nil {
		engine := interpolate.NewEngine()
		engine.SetVariables(env.ExportAll())
		view.SetEnvironment(env, engine)
	}

	model := tuiModel{
		view: view,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}

// initHistoryStore creates and initializes the SQLite history store.
func initHistoryStore() (*sqlite.Store, error) {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not determine config directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	// Create currier directory
	currierDir := filepath.Join(configDir, "currier")
	if err := os.MkdirAll(currierDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create config directory: %w", err)
	}

	// Create history store
	dbPath := filepath.Join(currierDir, "history.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open history database: %w", err)
	}

	return store, nil
}

// initCollectionStore creates and initializes the filesystem collection store.
func initCollectionStore() (*filesystem.CollectionStore, error) {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not determine config directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	// Create collections directory
	collectionsDir := filepath.Join(configDir, "currier", "collections")
	store, err := filesystem.NewCollectionStore(collectionsDir)
	if err != nil {
		return nil, fmt.Errorf("could not create collection store: %w", err)
	}

	return store, nil
}
