package cli

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/importer"
	"github.com/artpar/currier/internal/tui/views"
)

// NewRootCommand creates the root command.
func NewRootCommand(version string) *cobra.Command {
	var importFiles []string

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

			return runTUI(collections)
		},
	}

	// Add flags
	cmd.Flags().StringArrayVarP(&importFiles, "import", "i", nil,
		"Import collection file(s). Supports Postman, OpenAPI, cURL, HAR formats")

	// Add subcommands
	cmd.AddCommand(NewSendCommand())

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

// runTUI starts the TUI application with optional collections.
func runTUI(collections []*core.Collection) error {
	view := views.NewMainView()

	// Load collections if provided
	if len(collections) > 0 {
		view.SetCollections(collections)
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
