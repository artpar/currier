package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/artpar/currier/internal/tui/views"
)

// NewRootCommand creates the root command.
func NewRootCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "currier",
		Short:   "Currier - A TUI API client",
		Long:    "Currier is a vim-modal TUI API client for developers and AI agents.",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewSendCommand())

	return cmd
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

// runTUI starts the TUI application
func runTUI() error {
	model := tuiModel{
		view: views.NewMainView(),
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return err
	}
	return nil
}
