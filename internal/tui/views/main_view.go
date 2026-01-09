package views

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/artpar/currier/internal/cookies"
	"github.com/artpar/currier/internal/core"
	"github.com/artpar/currier/internal/exporter"
	"github.com/artpar/currier/internal/history"
	"github.com/artpar/currier/internal/importer"
	"github.com/artpar/currier/internal/interfaces"
	"github.com/artpar/currier/internal/interpolate"
	httpclient "github.com/artpar/currier/internal/protocol/http"
	"github.com/artpar/currier/internal/protocol/websocket"
	"github.com/artpar/currier/internal/proxy"
	"github.com/artpar/currier/internal/runner"
	"github.com/artpar/currier/internal/script"
	"github.com/artpar/currier/internal/starred"
	"github.com/artpar/currier/internal/storage/filesystem"
	"github.com/artpar/currier/internal/tui"
	"github.com/artpar/currier/internal/tui/components"
)

// Pane represents which pane is focused.
type Pane int

const (
	PaneCollections Pane = iota
	PaneRequest
	PaneResponse
	PaneWebSocket
)

// ViewMode represents the current view mode.
type ViewMode int

const (
	ViewModeHTTP ViewMode = iota
	ViewModeWebSocket
)

// MainView is the main three-pane view.
type MainView struct {
	width        int
	height       int
	focusedPane  Pane
	viewMode     ViewMode
	tree         *components.CollectionTree
	request      *components.RequestPanel
	response     *components.ResponsePanel
	wsPanel      *components.WebSocketPanel
	wsClient     *websocket.Client
	showHelp     bool
	helpTab      int // Current help tab (0=Quick, 1=Navigation, 2=Collections, 3=Request, 4=Response, 5=Capture)
	helpScroll   int // Scroll position within help tab
	environment  *core.Environment
	interpolator *interpolate.Engine
	notification string    // Temporary notification message
	notifyUntil  time.Time // When to clear notification
	historyStore    history.Store             // Store for request history
	collectionStore *filesystem.CollectionStore // Store for collection persistence
	lastRequest     *core.RequestDefinition   // Last sent request for history

	// Environment switcher state
	environmentStore *filesystem.EnvironmentStore  // Store for environment persistence
	showEnvSwitcher  bool                          // Whether env switcher popup is visible
	envList          []filesystem.EnvironmentMeta  // Available environments
	envCursor        int                           // Current selection in env list

	// Environment editor state
	showEnvEditor     bool              // Whether env editor popup is visible
	editingEnv        *core.Environment // Environment being edited
	envVarKeys        []string          // Ordered list of variable keys for display
	envEditorCursor   int               // Current selection in variable list
	envEditorMode     int               // 0=browse, 1=add key, 2=add value, 3=edit key, 4=edit value
	envEditorKeyInput string            // Input buffer for key
	envEditorValInput string            // Input buffer for value
	envEditorOrigKey  string            // Original key when editing (for replacement)

	// Cookie jar for automatic cookie handling
	cookieJar *cookies.PersistentJar

	// Starred store for favorite requests
	starredStore starred.Store

	// Proxy and TLS configuration
	proxyURL           string
	tlsCertFile        string
	tlsKeyFile         string
	tlsCAFile          string
	tlsInsecureSkip    bool

	// Settings dialogs
	showProxyDialog    bool
	proxyInput         string
	showTLSDialog      bool
	tlsDialogField     int // 0=cert, 1=key, 2=ca, 3=insecure
	tlsCertInput       string
	tlsKeyInput        string
	tlsCAInput         string

	// Collection runner state
	showRunnerModal   bool
	runnerRunning     bool
	runnerProgress    int
	runnerTotal       int
	runnerSummary     *runner.RunSummary
	runnerCurrentReq  string
	runnerCancelFunc  context.CancelFunc

	// Capture proxy server
	captureProxy       *proxy.Server
	captureProxyCtx    context.Context
	captureProxyCancel context.CancelFunc

	// Startup options
	startCaptureOnInit bool // Start in capture mode with proxy
}

// clearNotificationMsg is sent to clear the notification.
type clearNotificationMsg struct{}

// environmentSwitchedMsg is sent when an environment is successfully switched.
type environmentSwitchedMsg struct {
	Environment *core.Environment
	Engine      *interpolate.Engine
}

// environmentLoadErrorMsg is sent when environment loading fails.
type environmentLoadErrorMsg struct {
	Error error
}

// runnerProgressMsg is sent to update runner progress.
type runnerProgressMsg struct {
	Current     int
	Total       int
	CurrentName string
}

// runnerCompleteMsg is sent when runner finishes.
type runnerCompleteMsg struct {
	Summary *runner.RunSummary
}

// NewMainView creates a new main view.
func NewMainView() *MainView {
	view := &MainView{
		tree:         components.NewCollectionTree(),
		request:      components.NewRequestPanel(),
		response:     components.NewResponsePanel(),
		wsPanel:      components.NewWebSocketPanel(),
		wsClient:     websocket.NewClient(nil),
		focusedPane:  PaneCollections,
		viewMode:     ViewModeHTTP,
		interpolator: interpolate.NewEngine(), // Default engine with builtins
	}
	view.tree.Focus()
	return view
}

// startCaptureModeMsg is sent to start capture mode after init.
type startCaptureModeMsg struct{}

// Init initializes the view.
func (v *MainView) Init() tea.Cmd {
	if v.startCaptureOnInit {
		return func() tea.Msg { return startCaptureModeMsg{} }
	}
	return nil
}

// Update handles messages.
func (v *MainView) Update(msg tea.Msg) (tui.Component, tea.Cmd) {
	// Handle help overlay first
	if v.showHelp {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return v.handleHelpKey(keyMsg)
		}
		return v, nil
	}

	// Handle environment switcher overlay
	if v.showEnvSwitcher {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return v.handleEnvSwitcherKey(keyMsg)
		}
		return v, nil
	}

	// Handle environment editor overlay
	if v.showEnvEditor {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return v.handleEnvEditorKey(keyMsg)
		}
		return v, nil
	}

	// Handle proxy settings dialog
	if v.showProxyDialog {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return v.handleProxyDialogKey(keyMsg)
		}
		return v, nil
	}

	// Handle TLS settings dialog
	if v.showTLSDialog {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return v.handleTLSDialogKey(keyMsg)
		}
		return v, nil
	}

	// Handle runner modal
	if v.showRunnerModal {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return v.handleRunnerModalKey(keyMsg)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.updatePaneSizes()
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case components.SelectionMsg:
		v.request.SetRequest(msg.Request)
		v.viewMode = ViewModeHTTP
		v.focusPane(PaneRequest)
		v.updatePaneSizes()
		return v, nil

	case components.SelectWebSocketMsg:
		v.wsPanel.SetDefinition(msg.WebSocket)
		v.viewMode = ViewModeWebSocket
		v.focusPane(PaneWebSocket)
		v.updatePaneSizes()
		return v, nil

	case components.SelectHistoryItemMsg:
		// Create a request from history entry
		name := msg.Entry.RequestName
		if name == "" {
			name = "History Request"
		}
		req := core.NewRequestDefinition(
			name,
			msg.Entry.RequestMethod,
			msg.Entry.RequestURL,
		)
		// Set body if present
		if msg.Entry.RequestBody != "" {
			req.SetBody(msg.Entry.RequestBody)
		}
		// Set headers if present
		for key, value := range msg.Entry.RequestHeaders {
			req.SetHeader(key, value)
		}
		v.request.SetRequest(req)
		v.focusPane(PaneRequest)
		return v, nil

	case components.DeleteRequestMsg:
		// Persist the modified collection
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}
		v.notification = "Request deleted"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.CreateCollectionMsg:
		// Persist the new collection
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}
		v.notification = fmt.Sprintf("Created '%s'", msg.Collection.Name())
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.DeleteCollectionMsg:
		// Delete collection from disk
		if v.collectionStore != nil && msg.CollectionID != "" {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Delete(ctx, msg.CollectionID)
			}()
		}
		v.notification = "Collection deleted"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.RenameCollectionMsg:
		// Persist renamed collection
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}
		v.notification = fmt.Sprintf("Renamed to '%s'", msg.Collection.Name())
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.MoveRequestMsg:
		// Persist both source and target collections
		if v.collectionStore != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if msg.SourceCollection != nil {
					_ = v.collectionStore.Save(ctx, msg.SourceCollection)
				}
				if msg.TargetCollection != nil {
					_ = v.collectionStore.Save(ctx, msg.TargetCollection)
				}
			}()
		}
		// Show folder name if moved to folder, otherwise collection name
		targetName := "collection"
		if msg.TargetFolder != nil {
			targetName = msg.TargetFolder.Name()
		} else if msg.TargetCollection != nil {
			targetName = msg.TargetCollection.Name()
		}
		v.notification = fmt.Sprintf("Moved to '%s'", targetName)
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.MoveFolderMsg:
		// Persist both source and target collections
		if v.collectionStore != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if msg.SourceCollection != nil {
					_ = v.collectionStore.Save(ctx, msg.SourceCollection)
				}
				if msg.TargetCollection != nil {
					_ = v.collectionStore.Save(ctx, msg.TargetCollection)
				}
			}()
		}
		// Show target name
		targetName := "collection"
		if msg.TargetFolder != nil {
			targetName = msg.TargetFolder.Name()
		} else if msg.TargetCollection != nil {
			targetName = msg.TargetCollection.Name()
		}
		v.notification = fmt.Sprintf("Moved folder to '%s'", targetName)
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.DuplicateRequestMsg:
		// Persist collection with duplicated request
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}
		v.notification = "Request duplicated"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.DuplicateFolderMsg:
		// Persist collection with duplicated folder
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}
		v.notification = "Folder duplicated"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.CopyAsCurlMsg:
		// Generate cURL command and copy to clipboard
		if msg.Request != nil {
			curlExporter := exporter.NewCurlExporter()
			curlExporter.Pretty = false // Single line for clipboard
			ctx := context.Background()
			curlBytes, err := curlExporter.ExportRequest(ctx, msg.Request)
			if err == nil {
				curlCmd := string(curlBytes)
				err = clipboard.WriteAll(curlCmd)
				if err != nil {
					v.notification = "✗ Copy failed"
				} else {
					v.notification = "✓ Copied as cURL"
				}
			} else {
				v.notification = "✗ Failed to generate cURL"
			}
			v.notifyUntil = time.Now().Add(2 * time.Second)
			return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return clearNotificationMsg{}
			})
		}

	case components.BulkCopyAsCurlMsg:
		// Generate cURL commands for multiple requests and copy to clipboard
		if len(msg.Requests) > 0 {
			curlExporter := exporter.NewCurlExporter()
			curlExporter.Pretty = true // Multi-line for readability
			ctx := context.Background()

			var curlCommands []string
			for _, req := range msg.Requests {
				curlBytes, err := curlExporter.ExportRequest(ctx, req)
				if err == nil {
					curlCommands = append(curlCommands, string(curlBytes))
				}
			}

			if len(curlCommands) > 0 {
				combined := strings.Join(curlCommands, "\n\n")
				err := clipboard.WriteAll(combined)
				if err != nil {
					v.notification = "✗ Copy failed"
				} else {
					v.notification = fmt.Sprintf("✓ Copied %d requests as cURL", len(curlCommands))
				}
			} else {
				v.notification = "✗ Failed to generate cURL"
			}
			v.notifyUntil = time.Now().Add(2 * time.Second)
			return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return clearNotificationMsg{}
			})
		}

	case components.BulkDeleteRequestsMsg:
		// Persist all modified collections
		if v.collectionStore != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				for _, coll := range msg.Collections {
					_ = v.collectionStore.Save(ctx, coll)
				}
			}()
		}
		v.notification = fmt.Sprintf("✓ Deleted %d items", len(msg.RequestIDs))
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.BulkDeleteFoldersMsg:
		// Persist all modified collections
		if v.collectionStore != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				for _, coll := range msg.Collections {
					_ = v.collectionStore.Save(ctx, coll)
				}
			}()
		}
		v.notification = fmt.Sprintf("✓ Deleted %d folders", len(msg.FolderIDs))
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.BulkMoveMsg:
		// Persist all affected collections (source and target)
		if v.collectionStore != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				// Save source collections
				for _, coll := range msg.SourceCollections {
					_ = v.collectionStore.Save(ctx, coll)
				}
				// Save target collection
				if msg.TargetCollection != nil {
					_ = v.collectionStore.Save(ctx, msg.TargetCollection)
				}
			}()
		}
		totalMoved := len(msg.Requests) + len(msg.Folders)
		v.notification = fmt.Sprintf("✓ Moved %d items", totalMoved)
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.ToggleStarMsg:
		// Persist starred status change
		if v.starredStore != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				if msg.Starred {
					_ = v.starredStore.Star(ctx, msg.RequestID)
				} else {
					_ = v.starredStore.Unstar(ctx, msg.RequestID)
				}
			}()
		}
		// Brief notification
		if msg.Starred {
			v.notification = "★ Starred"
		} else {
			v.notification = "☆ Unstarred"
		}
		v.notifyUntil = time.Now().Add(1 * time.Second)
		return v, tea.Tick(1*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.ExportCollectionMsg:
		// Export collection to Postman JSON
		if msg.Collection != nil {
			postmanExporter := exporter.NewPostmanExporter()
			ctx := context.Background()
			data, err := postmanExporter.Export(ctx, msg.Collection)
			if err != nil {
				v.notification = "✗ Export failed"
			} else {
				// Write to file in current directory
				filename := sanitizeFilename(msg.Collection.Name()) + ".postman_collection.json"
				err = writeFile(filename, data)
				if err != nil {
					v.notification = fmt.Sprintf("✗ Failed to write %s", filename)
				} else {
					v.notification = fmt.Sprintf("✓ Exported to %s", filename)
				}
			}
			v.notifyUntil = time.Now().Add(3 * time.Second)
			return v, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return clearNotificationMsg{}
			})
		}

	case components.ImportCollectionMsg:
		// Import collection from Postman JSON or OpenAPI spec (auto-detected)
		if msg.FilePath != "" {
			data, err := os.ReadFile(msg.FilePath)
			if err != nil {
				v.notification = fmt.Sprintf("✗ Failed to read file: %s", err.Error())
			} else {
				// Use registry with auto-detection to support multiple formats
				registry := importer.NewRegistry()
				registry.Register(importer.NewPostmanImporter())
				registry.Register(importer.NewOpenAPIImporter())
				ctx := context.Background()
				result, err := registry.DetectAndImport(ctx, data)
				if err != nil {
					v.notification = fmt.Sprintf("✗ Import failed: %s", err.Error())
				} else {
					// Add to collections and save
					collections := v.tree.Collections()
					collections = append(collections, result.Collection)
					v.tree.SetCollections(collections)
					if v.collectionStore != nil {
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
							defer cancel()
							_ = v.collectionStore.Save(ctx, result.Collection)
						}()
					}
					v.notification = fmt.Sprintf("✓ Imported %s (%s)", result.Collection.Name(), result.SourceFormat)
				}
			}
			v.notifyUntil = time.Now().Add(3 * time.Second)
			return v, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return clearNotificationMsg{}
			})
		}

	case components.ReorderRequestMsg:
		// Persist collection after reorder
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}

	case components.RenameRequestMsg:
		// Persist collection with renamed request
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}
		v.notification = fmt.Sprintf("Renamed to '%s'", msg.Request.Name())
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.CreateFolderMsg:
		// Persist collection with new folder
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}
		v.notification = "Folder created"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.RenameFolderMsg:
		// Persist collection with renamed folder
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}
		v.notification = fmt.Sprintf("Folder renamed to '%s'", msg.Folder.Name())
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.DeleteFolderMsg:
		// Persist collection with folder removed
		if v.collectionStore != nil && msg.Collection != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, msg.Collection)
			}()
		}
		v.notification = "Folder deleted"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.ToggleProxyMsg:
		return v.handleToggleProxy()

	case components.CaptureReceivedMsg:
		// Update the collection tree with new captures
		v.tree.AddCapture(msg.Capture)
		return v, nil

	case components.RefreshCapturesMsg:
		// Refresh captures from proxy server
		if v.captureProxy != nil && v.captureProxy.IsRunning() {
			v.tree.SetProxyServer(v.captureProxy)
			// Continue ticking while proxy is running
			return v, tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
				return components.RefreshCapturesMsg{}
			})
		}
		return v, nil

	case components.SelectCaptureItemMsg:
		// Display the captured request/response in the panels
		if msg.Capture != nil {
			v.displayCapture(msg.Capture)
		}
		return v, nil

	case components.ExportCaptureMsg:
		// Export captured request to a collection
		if msg.Capture != nil {
			return v.handleExportCapture(msg.Capture)
		}
		return v, nil

	case startCaptureModeMsg:
		// Start capture mode with proxy auto-started
		v.tree.SetViewMode(components.ViewCapture)
		v.startCaptureOnInit = false // Don't run again
		return v.handleToggleProxy() // Start the proxy

	case components.ProxyStartedMsg:
		addr := msg.Address
		// Handle IPv6 any address [::]:port -> localhost:port
		if len(addr) > 4 && addr[:4] == "[::]" {
			addr = "localhost" + addr[4:]
		} else if len(addr) > 0 && addr[0] == ':' {
			addr = "localhost" + addr
		}
		v.notification = "Proxy started! Use: curl --proxy http://" + addr + " URL"
		v.notifyUntil = time.Now().Add(5 * time.Second)
		return v, tea.Tick(5*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.ProxyStoppedMsg:
		v.notification = "Proxy stopped"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.ProxyErrorMsg:
		v.notification = "Proxy error: " + msg.Error.Error()
		v.notifyUntil = time.Now().Add(3 * time.Second)
		return v, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.SendRequestMsg:
		v.response.SetLoading(true)
		v.focusPane(PaneResponse)
		v.lastRequest = msg.Request // Save for history
		httpConfig := HTTPClientConfig{
			CookieJar:    v.cookieJar,
			ProxyURL:     v.proxyURL,
			CertFile:     v.tlsCertFile,
			KeyFile:      v.tlsKeyFile,
			CAFile:       v.tlsCAFile,
			InsecureSkip: v.tlsInsecureSkip,
		}
		return v, sendRequest(msg.Request, v.interpolator, httpConfig)

	case components.ResponseReceivedMsg:
		v.response.SetLoading(false)
		v.response.SetResponse(msg.Response)
		// Set test results (if any)
		if len(msg.TestResults) > 0 {
			v.response.SetTestResults(msg.TestResults)
		}
		// Set console messages (if any)
		if len(msg.Console) > 0 {
			v.response.SetConsoleMessages(msg.Console)
		}
		// Save to history
		if v.historyStore != nil && v.lastRequest != nil {
			go v.saveToHistory(v.lastRequest, msg.Response, nil)
		}
		return v, nil

	case components.RequestErrorMsg:
		v.response.SetLoading(false)
		v.response.SetError(msg.Error)
		// Save failed request to history too
		if v.historyStore != nil && v.lastRequest != nil {
			go v.saveToHistory(v.lastRequest, nil, msg.Error)
		}
		return v, nil

	case components.CopyMsg:
		return v.handleCopy(msg.Content)

	case components.FeedbackMsg:
		return v.handleFeedback(msg)

	case clearNotificationMsg:
		v.notification = ""
		return v, nil

	case environmentSwitchedMsg:
		v.environment = msg.Environment
		v.interpolator = msg.Engine
		v.notification = fmt.Sprintf("Switched to '%s'", msg.Environment.Name())
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case environmentLoadErrorMsg:
		v.notification = "Failed to switch environment"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case runnerProgressMsg:
		v.runnerProgress = msg.Current
		v.runnerTotal = msg.Total
		v.runnerCurrentReq = msg.CurrentName
		return v, nil

	case runnerCompleteMsg:
		v.runnerRunning = false
		v.runnerSummary = msg.Summary
		return v, nil

	// WebSocket messages
	case components.WSConnectCmd:
		return v, v.connectWebSocket(msg.Definition)

	case components.WSDisconnectCmd:
		return v, v.disconnectWebSocket()

	case components.WSReconnectCmd:
		def := v.wsPanel.Definition()
		if def != nil {
			return v, v.connectWebSocket(def)
		}
		return v, nil

	case components.WSSendMessageCmd:
		return v, v.sendWebSocketMessage(msg.Content)

	case components.WSConnectedMsg:
		v.wsPanel.SetConnectionID(msg.ConnectionID)
		v.wsPanel.SetConnectionState(interfaces.ConnectionStateConnected)
		v.notification = "✓ WebSocket connected"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.WSDisconnectedMsg:
		v.wsPanel.SetConnectionID("")
		v.wsPanel.SetConnectionState(interfaces.ConnectionStateDisconnected)
		if msg.Error != nil {
			v.notification = "✗ Disconnected: " + msg.Error.Error()
		} else {
			v.notification = "WebSocket disconnected"
		}
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case components.WSMessageReceivedMsg:
		v.wsPanel.AddMessage(msg.Message)
		return v, nil

	case components.WSMessageSentMsg:
		v.wsPanel.AddMessage(msg.Message)
		return v, nil

	case components.WSStateChangedMsg:
		v.wsPanel.SetConnectionState(msg.State)
		return v, nil

	case components.WSErrorMsg:
		v.notification = "✗ WS Error: " + msg.Error.Error()
		v.notifyUntil = time.Now().Add(3 * time.Second)
		return v, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return clearNotificationMsg{}
		})
	}

	// Forward messages to focused pane
	return v.forwardToFocusedPane(msg)
}

func (v *MainView) handleCopy(content string) (tui.Component, tea.Cmd) {
	err := clipboard.WriteAll(content)
	if err != nil {
		v.notification = "✗ Copy failed"
	} else {
		size := len(content)
		if size > 1024 {
			v.notification = fmt.Sprintf("✓ Copied %.1fKB", float64(size)/1024)
		} else {
			v.notification = fmt.Sprintf("✓ Copied %dB", size)
		}
	}
	v.notifyUntil = time.Now().Add(2 * time.Second)

	// Schedule clearing the notification
	return v, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}

func (v *MainView) handleFeedback(msg components.FeedbackMsg) (tui.Component, tea.Cmd) {
	if msg.IsError {
		v.notification = "✗ " + msg.Message
	} else {
		v.notification = "💡 " + msg.Message
	}
	v.notifyUntil = time.Now().Add(2 * time.Second)

	// Schedule clearing the notification
	return v, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}

func (v *MainView) handleSaveToCollection() (tui.Component, tea.Cmd) {
	// Get current request from request panel
	req := v.request.Request()
	if req == nil {
		v.notification = "No request to save"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearNotificationMsg{}
		})
	}

	// Get or create a collection
	collections := v.tree.Collections()
	var targetCollection *core.Collection

	if len(collections) == 0 {
		// Create a default collection
		targetCollection = core.NewCollection("My Collection")
		v.tree.SetCollections([]*core.Collection{targetCollection})
	} else {
		// Use the first collection
		targetCollection = collections[0]
	}

	// Clone the request to avoid modifying the original
	savedReq := core.NewRequestDefinition(req.Name(), req.Method(), req.URL())
	savedReq.SetBody(req.Body())
	for k, val := range req.Headers() {
		savedReq.SetHeader(k, val)
	}

	// Add to collection
	targetCollection.AddRequest(savedReq)
	v.tree.RebuildItems()

	// Persist the collection
	if v.collectionStore != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = v.collectionStore.Save(ctx, targetCollection)
		}()
	}

	v.notification = fmt.Sprintf("Saved to %s", targetCollection.Name())
	v.notifyUntil = time.Now().Add(2 * time.Second)
	return v, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}

func (v *MainView) handleKeyMsg(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	// Check if we're in INSERT mode (editing text in any pane)
	// In INSERT mode, forward ALL keys to the focused pane except Ctrl+C
	isEditing := v.request.IsEditing() || v.tree.IsSearching() || v.tree.IsRenaming() || v.tree.IsMoving()

	// Ctrl+C always quits
	if msg.Type == tea.KeyCtrlC {
		return v, tea.Quit
	}

	// In INSERT mode, forward everything to the focused pane
	// Only Escape exits insert mode (handled by the pane itself)
	if isEditing {
		return v.forwardToFocusedPane(msg)
	}

	// NORMAL mode - handle global shortcuts
	switch msg.Type {
	case tea.KeyTab:
		// Tab always cycles panes
		v.cycleFocusForward()
		return v, nil

	case tea.KeyShiftTab:
		// Shift+Tab always cycles panes backward
		v.cycleFocusBackward()
		return v, nil

	case tea.KeyEsc:
		// Already in normal mode, nothing to do
		return v, nil

	case tea.KeyCtrlK:
		// Clear all cookies
		if v.cookieJar != nil {
			if err := v.cookieJar.Clear(); err != nil {
				v.notification = "Failed to clear cookies"
			} else {
				v.notification = "Cookies cleared"
			}
			v.notifyUntil = time.Now().Add(2 * time.Second)
			return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return clearNotificationMsg{}
			})
		}
		return v, nil

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "q":
			return v, tea.Quit
		case "?":
			v.showHelp = true
			return v, nil
		case "1":
			v.focusPane(PaneCollections)
			return v, nil
		case "2":
			v.focusPane(PaneRequest)
			return v, nil
		case "3":
			v.focusPane(PaneResponse)
			return v, nil
		case "n":
			// Create a scratch request (not added to collection until saved with 's')
			newReq := core.NewRequestDefinition("New Request", "GET", "")

			// Set it in the request panel
			v.request.SetRequest(newReq)
			v.focusPane(PaneRequest)
			// Auto-enter URL edit mode
			v.request.StartURLEdit()
			return v, nil
		case "w":
			// Toggle WebSocket mode or create new WebSocket
			if v.viewMode == ViewModeWebSocket {
				v.viewMode = ViewModeHTTP
				v.focusPane(PaneRequest)
			} else {
				// Create a new WebSocket definition if none exists
				if v.wsPanel.Definition() == nil {
					wsDef := core.NewWebSocketDefinition("New WebSocket", "wss://")
					v.wsPanel.SetDefinition(wsDef)
				}
				v.viewMode = ViewModeWebSocket
				v.focusPane(PaneWebSocket)
			}
			v.updatePaneSizes()
			return v, nil
		case "4":
			// Focus WebSocket panel
			if v.viewMode == ViewModeWebSocket {
				v.focusPane(PaneWebSocket)
			}
			return v, nil
		case "s":
			// Save current request to collection
			return v.handleSaveToCollection()
		case "V":
			// Open environment switcher (capital V for Variables)
			return v.openEnvSwitcher()
		case "P":
			// Open proxy settings dialog
			v.showProxyDialog = true
			v.proxyInput = v.proxyURL
			return v, nil
		}
	}

	// Handle Ctrl+T for TLS settings
	if msg.Type == tea.KeyCtrlT {
		v.showTLSDialog = true
		v.tlsDialogField = 0
		v.tlsCertInput = v.tlsCertFile
		v.tlsKeyInput = v.tlsKeyFile
		v.tlsCAInput = v.tlsCAFile
		return v, nil
	}

	// Handle Ctrl+R to run collection
	if msg.Type == tea.KeyCtrlR {
		return v.startCollectionRunner()
	}

	// Forward to focused pane for other keys
	return v.forwardToFocusedPane(msg)
}

func (v *MainView) forwardToFocusedPane(msg tea.Msg) (tui.Component, tea.Cmd) {
	var cmd tea.Cmd

	switch v.focusedPane {
	case PaneCollections:
		updated, c := v.tree.Update(msg)
		v.tree = updated.(*components.CollectionTree)
		cmd = c
	case PaneRequest:
		updated, c := v.request.Update(msg)
		v.request = updated.(*components.RequestPanel)
		cmd = c
	case PaneResponse:
		updated, c := v.response.Update(msg)
		v.response = updated.(*components.ResponsePanel)
		cmd = c
	case PaneWebSocket:
		updated, c := v.wsPanel.Update(msg)
		v.wsPanel = updated.(*components.WebSocketPanel)
		cmd = c
	}

	return v, cmd
}

func (v *MainView) cycleFocusForward() {
	if v.viewMode == ViewModeWebSocket {
		// In WebSocket mode: Collections -> WebSocket -> Collections
		if v.focusedPane == PaneCollections {
			v.focusPane(PaneWebSocket)
		} else {
			v.focusPane(PaneCollections)
		}
	} else {
		v.focusPane(Pane((int(v.focusedPane) + 1) % 3))
	}
}

func (v *MainView) cycleFocusBackward() {
	if v.viewMode == ViewModeWebSocket {
		// In WebSocket mode: Collections -> WebSocket -> Collections
		if v.focusedPane == PaneCollections {
			v.focusPane(PaneWebSocket)
		} else {
			v.focusPane(PaneCollections)
		}
	} else {
		v.focusPane(Pane((int(v.focusedPane) + 2) % 3))
	}
}

func (v *MainView) focusPane(pane Pane) {
	// Blur all
	v.tree.Blur()
	v.request.Blur()
	v.response.Blur()
	v.wsPanel.Blur()

	// Focus the target
	v.focusedPane = pane
	switch pane {
	case PaneCollections:
		v.tree.Focus()
	case PaneRequest:
		v.request.Focus()
	case PaneResponse:
		v.response.Focus()
	case PaneWebSocket:
		v.wsPanel.Focus()
	}
}

func (v *MainView) updatePaneSizes() {
	if v.width == 0 || v.height == 0 {
		return
	}

	// Postman-like layout:
	// [Sidebar 25%] | [Request/Response stacked vertically 75%]
	sidebarWidth := v.width * 25 / 100
	if sidebarWidth < 25 {
		sidebarWidth = 25
	}
	if sidebarWidth > 60 {
		sidebarWidth = 60
	}
	rightWidth := v.width - sidebarWidth

	// Reserve 2 lines for help bar + status bar
	totalHeight := v.height - 2
	if totalHeight < 2 {
		totalHeight = 2
	}

	// Set sidebar size
	v.tree.SetSize(sidebarWidth, totalHeight)

	if v.viewMode == ViewModeWebSocket {
		// WebSocket mode: single panel takes the full right side
		v.wsPanel.SetSize(rightWidth, totalHeight)
	} else {
		// HTTP mode: Split right side vertically: Request on top (45%), Response on bottom (55%)
		requestHeight := totalHeight * 45 / 100
		if requestHeight < 8 {
			requestHeight = 8
		}
		responseHeight := totalHeight - requestHeight

		v.request.SetSize(rightWidth, requestHeight)
		v.response.SetSize(rightWidth, responseHeight)
	}
}

// View renders the view.
func (v *MainView) View() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}

	// Render help overlay if showing
	if v.showHelp {
		return v.renderHelp()
	}

	// Render environment switcher overlay if showing
	if v.showEnvSwitcher {
		return v.renderEnvSwitcher()
	}

	// Render environment editor overlay if showing
	if v.showEnvEditor {
		return v.renderEnvEditor()
	}

	// Render proxy settings dialog if showing
	if v.showProxyDialog {
		return v.renderProxyDialog()
	}

	// Render TLS settings dialog if showing
	if v.showTLSDialog {
		return v.renderTLSDialog()
	}

	// Render runner modal if showing
	if v.showRunnerModal {
		return v.renderRunnerModal()
	}

	// Render sidebar (Collections)
	sidebar := v.tree.View()

	var rightStack string
	if v.viewMode == ViewModeWebSocket {
		// WebSocket mode: single WebSocket panel
		rightStack = v.wsPanel.View()
	} else {
		// HTTP mode: Request on top, Response on bottom
		requestPane := v.request.View()
		responsePane := v.response.View()
		rightStack = lipgloss.JoinVertical(lipgloss.Left, requestPane, responsePane)
	}

	// Join sidebar with right stack horizontally
	panes := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightStack)

	// Render help bar and status bar
	helpBar := v.renderHelpBar()
	statusBar := v.renderStatusBar()

	// Join panes, help bar, and status bar vertically
	return lipgloss.JoinVertical(lipgloss.Left, panes, helpBar, statusBar)
}

// renderHelpBar renders context-sensitive keyboard shortcuts.
func (v *MainView) renderHelpBar() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	sep := sepStyle.Render(" │ ")

	var hints []string

	// Context-sensitive hints based on focused pane and mode
	switch v.focusedPane {
	case PaneCollections:
		if v.tree.IsSearching() {
			hints = []string{
				keyStyle.Render("Enter") + descStyle.Render(" Apply"),
				keyStyle.Render("Esc") + descStyle.Render(" Cancel"),
				keyStyle.Render("Ctrl+U") + descStyle.Render(" Clear"),
			}
		} else if v.tree.ViewMode() == components.ViewHistory {
			hints = []string{
				keyStyle.Render("j/k") + descStyle.Render(" Navigate"),
				keyStyle.Render("Enter") + descStyle.Render(" Load"),
				keyStyle.Render("r") + descStyle.Render(" Refresh"),
				keyStyle.Render("C") + descStyle.Render(" Collections"),
			}
		} else {
			hints = []string{
				keyStyle.Render("j/k") + descStyle.Render(" Navigate"),
				keyStyle.Render("h/l") + descStyle.Render(" Collapse/Expand"),
				keyStyle.Render("d") + descStyle.Render(" Delete"),
				keyStyle.Render("H") + descStyle.Render(" History"),
			}
		}
	case PaneRequest:
		if v.request.IsEditing() {
			hints = []string{
				keyStyle.Render("Enter/Esc") + descStyle.Render(" Save"),
				keyStyle.Render("Alt+Enter") + descStyle.Render(" Send"),
				keyStyle.Render("Ctrl+U") + descStyle.Render(" Clear"),
			}
		} else {
			hints = []string{
				keyStyle.Render("e") + descStyle.Render(" Edit URL"),
				keyStyle.Render("m") + descStyle.Render(" Method"),
				keyStyle.Render("Enter/Alt+Enter") + descStyle.Render(" Send"),
				keyStyle.Render("[/]") + descStyle.Render(" Switch tab"),
			}
		}
	case PaneResponse:
		hints = []string{
			keyStyle.Render("j/k") + descStyle.Render(" Scroll"),
			keyStyle.Render("G/gg") + descStyle.Render(" Top/Bottom"),
			keyStyle.Render("y") + descStyle.Render(" Copy"),
			keyStyle.Render("[/]") + descStyle.Render(" Tab"),
		}
	case PaneWebSocket:
		if v.wsPanel.IsInputMode() {
			hints = []string{
				keyStyle.Render("Enter") + descStyle.Render(" Send"),
				keyStyle.Render("Esc") + descStyle.Render(" Cancel"),
				keyStyle.Render("Ctrl+U") + descStyle.Render(" Clear"),
			}
		} else {
			hints = []string{
				keyStyle.Render("i/Enter") + descStyle.Render(" Type"),
				keyStyle.Render("c") + descStyle.Render(" Connect"),
				keyStyle.Render("d") + descStyle.Render(" Disconnect"),
				keyStyle.Render("[/]") + descStyle.Render(" Tab"),
			}
		}
	}

	// Always add global hints
	hints = append(hints,
		keyStyle.Render("n")+descStyle.Render(" New"),
		keyStyle.Render("s")+descStyle.Render(" Save"),
		keyStyle.Render("w")+descStyle.Render(" WS"),
		keyStyle.Render("1/2/3")+descStyle.Render(" Pane"),
		keyStyle.Render("?")+descStyle.Render(" Help"),
		keyStyle.Render("q")+descStyle.Render(" Quit"),
	)

	content := strings.Join(hints, sep)

	// Help bar style
	barStyle := lipgloss.NewStyle().
		Width(v.width).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	return barStyle.Render(content)
}

// renderStatusBar renders the bottom status bar with environment info.
func (v *MainView) renderStatusBar() string {
	// Build status items
	var items []string

	// Mode indicator
	modeStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)
	isEditing := v.request.IsEditing() || v.tree.IsSearching() || v.tree.IsRenaming() || v.tree.IsMoving()
	if isEditing {
		modeStyle = modeStyle.
			Background(lipgloss.Color("214")).
			Foreground(lipgloss.Color("0"))
		items = append(items, modeStyle.Render("INSERT"))
	} else {
		modeStyle = modeStyle.
			Background(lipgloss.Color("34")).
			Foreground(lipgloss.Color("255"))
		items = append(items, modeStyle.Render("NORMAL"))
	}

	// Show pending 'g' indicator for gg sequence
	gPending := (v.focusedPane == PaneCollections && v.tree.GPressed()) ||
		(v.focusedPane == PaneResponse && v.response.GPressed())
	if gPending {
		pendingStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("214")).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1)
		items = append(items, pendingStyle.Render("g-"))
	}

	// View mode indicator
	if v.viewMode == ViewModeWebSocket {
		wsStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("33")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1)
		items = append(items, wsStyle.Render("WS"))
	}

	// Focused pane indicator
	paneStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)
	paneName := "Collections"
	switch v.focusedPane {
	case PaneRequest:
		paneName = "Request"
	case PaneResponse:
		paneName = "Response"
	case PaneWebSocket:
		paneName = "WebSocket"
	}
	items = append(items, paneStyle.Render(paneName))

	// Environment indicator
	if v.environment != nil {
		envStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("229")).
			Padding(0, 1).
			Bold(true)
		items = append(items, envStyle.Render("ENV: "+v.environment.Name()))
	} else {
		envStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("250")).
			Padding(0, 1)
		items = append(items, envStyle.Render("No Environment"))
	}

	// Add variable count if environment exists
	if v.environment != nil {
		vars := v.environment.Variables()
		secrets := v.environment.SecretNames()
		countStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)
		items = append(items, countStyle.Render(fmt.Sprintf("%d vars, %d secrets", len(vars), len(secrets))))
	}

	// Add notification if present
	if v.notification != "" {
		notifyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("34")).
			Bold(true).
			Padding(0, 1)
		if strings.HasPrefix(v.notification, "✗") {
			notifyStyle = notifyStyle.Foreground(lipgloss.Color("160"))
		}
		items = append(items, notifyStyle.Render(v.notification))
	}

	// Help hint on the right
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Padding(0, 1)
	helpHint := helpStyle.Render("? help  q quit")

	// Calculate spacing
	leftContent := strings.Join(items, " ")
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(helpHint)
	spacerWidth := v.width - leftWidth - rightWidth - 2
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := strings.Repeat(" ", spacerWidth)

	// Status bar style
	barStyle := lipgloss.NewStyle().
		Width(v.width).
		Background(lipgloss.Color("236"))

	return barStyle.Render(leftContent + spacer + helpHint)
}

// handleHelpKey handles keyboard input in the help screen.
func (v *MainView) handleHelpKey(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		v.showHelp = false
		v.helpTab = 0
		v.helpScroll = 0
		return v, nil
	case tea.KeyTab, tea.KeyRight:
		v.helpTab = (v.helpTab + 1) % 6
		v.helpScroll = 0
		return v, nil
	case tea.KeyShiftTab, tea.KeyLeft:
		v.helpTab = (v.helpTab + 5) % 6
		v.helpScroll = 0
		return v, nil
	}

	switch string(msg.Runes) {
	case "?", "q":
		v.showHelp = false
		v.helpTab = 0
		v.helpScroll = 0
		return v, nil
	case "j":
		v.helpScroll++
		return v, nil
	case "k":
		if v.helpScroll > 0 {
			v.helpScroll--
		}
		return v, nil
	case "1":
		v.helpTab = 0
		v.helpScroll = 0
	case "2":
		v.helpTab = 1
		v.helpScroll = 0
	case "3":
		v.helpTab = 2
		v.helpScroll = 0
	case "4":
		v.helpTab = 3
		v.helpScroll = 0
	case "5":
		v.helpTab = 4
		v.helpScroll = 0
	case "6":
		v.helpTab = 5
		v.helpScroll = 0
	case "g":
		v.helpScroll = 0
	case "G":
		v.helpScroll = 999 // Will be clamped in render
	}
	return v, nil
}

// getHelpTabs returns the help tab names.
func (v *MainView) getHelpTabs() []string {
	return []string{"Quick Start", "Navigation", "Collections", "Request", "Response", "Capture"}
}

// getHelpContent returns the content for the current help tab.
func (v *MainView) getHelpContent() []string {
	switch v.helpTab {
	case 0: // Quick Start
		return []string{
			"QUICK START",
			"",
			"Currier is a vim-style API client. Here's how to get started:",
			"",
			"1. MAKE A REQUEST",
			"   n          Create new request",
			"   e          Edit URL (type your endpoint)",
			"   m          Change HTTP method",
			"   Enter      Send the request",
			"",
			"2. NAVIGATE",
			"   1/2/3      Switch panes (Collections/Request/Response)",
			"   j/k        Move up/down in any list",
			"   Tab        Cycle between panes",
			"",
			"3. SAVE YOUR WORK",
			"   s          Save request to collection",
			"   N          Create new collection",
			"",
			"4. VIEW HISTORY",
			"   H          Toggle History view",
			"   Enter      Load a past request",
			"",
			"5. CAPTURE TRAFFIC",
			"   C          Switch to Capture mode (press twice)",
			"   p          Start proxy server",
			"            Then: curl --proxy http://localhost:PORT url",
			"",
			"Press Tab/Arrow keys to see more help sections.",
		}
	case 1: // Navigation
		return []string{
			"NAVIGATION",
			"",
			"SWITCHING PANES",
			"   1          Collections/History/Capture pane",
			"   2          Request pane",
			"   3          Response pane",
			"   Tab        Cycle forward through panes",
			"   Shift+Tab  Cycle backward",
			"",
			"MOVING WITHIN PANES",
			"   j          Move down / Scroll down",
			"   k          Move up / Scroll up",
			"   gg         Jump to top",
			"   G          Jump to bottom",
			"   Ctrl+D     Page down",
			"   Ctrl+U     Page up",
			"",
			"SWITCHING TABS (Request/Response)",
			"   [          Previous tab",
			"   ]          Next tab",
			"",
			"GLOBAL",
			"   ?          Show/hide this help",
			"   q          Quit Currier",
			"   Ctrl+C     Quit Currier",
		}
	case 2: // Collections
		return []string{
			"COLLECTIONS PANE",
			"",
			"BROWSING",
			"   j/k        Navigate items",
			"   h          Collapse folder/collection",
			"   l          Expand folder/collection",
			"   Enter      Load selected request",
			"   /          Search collections",
			"   Esc        Clear search",
			"",
			"CREATING",
			"   n          New request",
			"   N          New collection",
			"   F          New folder in collection",
			"",
			"ORGANIZING",
			"   s          Save current request to collection",
			"   m          Move request/folder",
			"   K          Move item up",
			"   J          Move item down",
			"   y          Duplicate request/folder",
			"",
			"EDITING",
			"   R          Rename item",
			"   d          Delete request",
			"   D          Delete collection/folder",
			"",
			"IMPORT/EXPORT",
			"   I          Import (Postman/OpenAPI/cURL/HAR)",
			"   E          Export collection",
			"   c          Copy request as cURL command",
			"",
			"VIEW MODES",
			"   H          Switch to History view",
			"   C          Switch to Capture view",
		}
	case 3: // Request
		return []string{
			"REQUEST PANE",
			"",
			"TABS: URL | Query | Headers | Auth | Body",
			"   [          Previous tab",
			"   ]          Next tab",
			"",
			"EDITING",
			"   e          Edit current field (URL/header/body)",
			"   Enter      Confirm edit / Send request",
			"   Esc        Cancel edit",
			"   Alt+Enter  Send request (even while editing)",
			"",
			"HTTP METHOD",
			"   m          Next method (GET→POST→PUT...)",
			"   M          Previous method",
			"",
			"HEADERS & QUERY PARAMS",
			"   a          Add new header/param",
			"   d          Delete selected header/param",
			"   j/k        Navigate headers/params",
			"   Tab        Switch between key and value",
			"",
			"BODY",
			"   e          Edit body content",
			"   Body types: JSON, Form, Raw, File",
			"",
			"SENDING",
			"   Enter      Send request",
			"   Alt+Enter  Send request (works everywhere)",
		}
	case 4: // Response
		return []string{
			"RESPONSE PANE",
			"",
			"TABS: Body | Headers | Cookies | Timing | Console",
			"   [          Previous tab",
			"   ]          Next tab",
			"",
			"SCROLLING",
			"   j          Scroll down",
			"   k          Scroll up",
			"   gg         Scroll to top",
			"   G          Scroll to bottom",
			"   Ctrl+D     Page down",
			"   Ctrl+U     Page up",
			"",
			"COPY",
			"   y          Copy response body to clipboard",
			"",
			"TIMING TAB",
			"   Shows DNS, Connect, TLS, TTFB, Transfer times",
			"",
			"CONSOLE TAB",
			"   Shows test results from pm.test() scripts",
		}
	case 5: // Capture
		return []string{
			"CAPTURE MODE - Traffic Proxy",
			"",
			"Capture intercepts HTTP traffic from any app.",
			"",
			"HOW TO USE",
			"   1. Press C (twice) to enter Capture mode",
			"   2. Press p to start the proxy",
			"   3. Note the port number shown",
			"   4. Route your traffic through the proxy:",
			"",
			"      curl --proxy http://localhost:PORT url",
			"",
			"      Or set environment variables:",
			"      export http_proxy=http://localhost:PORT",
			"      export https_proxy=http://localhost:PORT",
			"",
			"   5. Captured requests appear in real-time",
			"   6. Press Enter on any capture to inspect it",
			"",
			"KEYBOARD SHORTCUTS",
			"   C          Enter Capture mode",
			"   p          Start/Stop proxy",
			"   j/k        Navigate captures",
			"   Enter      Load capture into request pane",
			"   m          Filter by method (GET/POST/...)",
			"   x          Clear filter",
			"   X          Clear all captures",
			"   H          Return to History view",
			"",
			"TIP: Use --capture flag to start in capture mode:",
			"     currier --capture",
		}
	}
	return []string{}
}

func (v *MainView) renderHelp() string {
	// Calculate dimensions
	boxWidth := v.width - 4
	if boxWidth > 70 {
		boxWidth = 70
	}
	boxHeight := v.height - 4
	if boxHeight > 30 {
		boxHeight = 30
	}
	contentHeight := boxHeight - 5 // Account for tabs and footer

	// Styles
	tabStyle := lipgloss.NewStyle().
		Padding(0, 1)
	activeTabStyle := tabStyle.
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("229")).
		Bold(true)
	inactiveTabStyle := tabStyle.
		Foreground(lipgloss.Color("243"))
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229"))
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Align(lipgloss.Center)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(boxWidth)

	// Build tab bar
	tabs := v.getHelpTabs()
	var tabBar string
	for i, tab := range tabs {
		style := inactiveTabStyle
		if i == v.helpTab {
			style = activeTabStyle
		}
		// Shorten tab names for narrow screens
		displayTab := tab
		if boxWidth < 60 {
			switch i {
			case 0:
				displayTab = "Quick"
			case 1:
				displayTab = "Nav"
			case 2:
				displayTab = "Coll"
			case 3:
				displayTab = "Req"
			case 4:
				displayTab = "Resp"
			case 5:
				displayTab = "Cap"
			}
		}
		tabBar += style.Render(displayTab)
	}

	// Get content and handle scrolling
	content := v.getHelpContent()
	totalLines := len(content)

	// Clamp scroll position
	maxScroll := totalLines - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if v.helpScroll > maxScroll {
		v.helpScroll = maxScroll
	}

	// Get visible slice
	startLine := v.helpScroll
	endLine := startLine + contentHeight
	if endLine > totalLines {
		endLine = totalLines
	}
	visibleContent := content[startLine:endLine]

	// Format content lines
	var contentLines []string
	for _, line := range visibleContent {
		// Highlight keyboard shortcuts (lines with letters followed by spaces)
		if len(line) > 3 && line[0] == ' ' && line[1] == ' ' && line[2] == ' ' {
			// Find the key part (up to first double space or description)
			parts := strings.SplitN(strings.TrimLeft(line, " "), "  ", 2)
			if len(parts) == 2 {
				key := strings.TrimLeft(line, " ")[:len(parts[0])]
				desc := parts[1]
				formattedLine := "   " + keyStyle.Render(key) + "  " + contentStyle.Render(desc)
				contentLines = append(contentLines, formattedLine)
				continue
			}
		}
		// Check if it's a title (all caps or starts with number)
		if len(line) > 0 && (line == strings.ToUpper(line) || (line[0] >= '1' && line[0] <= '9')) {
			contentLines = append(contentLines, titleStyle.Render(line))
		} else {
			contentLines = append(contentLines, contentStyle.Render(line))
		}
	}

	// Pad content to fill height
	for len(contentLines) < contentHeight {
		contentLines = append(contentLines, "")
	}

	// Scroll indicator
	scrollIndicator := ""
	if totalLines > contentHeight {
		scrollIndicator = fmt.Sprintf(" (%d/%d)", v.helpScroll+1, maxScroll+1)
	}

	// Build footer
	footer := footerStyle.Render("Tab/←→ sections │ j/k scroll │ 1-6 jump │ q close" + scrollIndicator)

	// Assemble the help box
	helpBox := lipgloss.JoinVertical(
		lipgloss.Left,
		tabBar,
		"",
		strings.Join(contentLines, "\n"),
		"",
		footer,
	)

	// Center the box on screen
	centeredStyle := lipgloss.NewStyle().
		Width(v.width).
		Height(v.height).
		Align(lipgloss.Center, lipgloss.Center)

	return centeredStyle.Render(boxStyle.Render(helpBox))
}

// renderEnvSwitcher renders the environment switcher popup.
func (v *MainView) renderEnvSwitcher() string {
	// Box dimensions - wider to accommodate variable preview
	leftWidth := 35
	rightWidth := 35
	boxWidth := leftWidth + rightWidth + 3 // +3 for separator
	if boxWidth > v.width-4 {
		boxWidth = v.width - 4
		leftWidth = boxWidth / 2
		rightWidth = boxWidth - leftWidth - 3
	}

	// Header style
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Width(boxWidth - 4).
		Align(lipgloss.Center).
		Padding(0, 1)

	// Build left panel (environment list)
	var leftLines []string
	for i, env := range v.envList {
		prefix := "  "
		if i == v.envCursor {
			prefix = "→ "
		}

		// Active indicator
		active := ""
		if env.IsActive {
			active = " ●"
		}

		name := env.Name
		// Truncate if needed
		maxNameLen := leftWidth - len(prefix) - len(active) - 2
		if maxNameLen > 3 && len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		line := prefix + name + active

		// Style selected item
		style := lipgloss.NewStyle()
		if i == v.envCursor {
			style = style.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("229"))
		} else if env.IsActive {
			style = style.
				Foreground(lipgloss.Color("34")).
				Bold(true)
		}

		// Pad line to width
		padding := leftWidth - len(line)
		if padding > 0 {
			line += strings.Repeat(" ", padding)
		}

		leftLines = append(leftLines, style.Render(line))
	}

	// Build right panel (variable preview for selected env)
	var rightLines []string
	varHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Underline(true)
	rightLines = append(rightLines, varHeaderStyle.Render("Variables:"))

	if v.envCursor >= 0 && v.envCursor < len(v.envList) {
		selectedEnv := v.envList[v.envCursor]
		varStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))

		if len(selectedEnv.VarNames) == 0 {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
			rightLines = append(rightLines, dimStyle.Render("(no variables)"))
		} else {
			// Show up to 8 variables
			maxVars := 8
			for i, varName := range selectedEnv.VarNames {
				if i >= maxVars {
					moreStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
					rightLines = append(rightLines, moreStyle.Render(fmt.Sprintf("  +%d more...", len(selectedEnv.VarNames)-maxVars)))
					break
				}
				// Truncate long variable names
				displayName := varName
				if len(displayName) > rightWidth-4 {
					displayName = displayName[:rightWidth-7] + "..."
				}
				rightLines = append(rightLines, varStyle.Render("  {{"+displayName+"}}"))
			}
		}
	}

	// Ensure both panels have same height
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, strings.Repeat(" ", leftWidth))
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	// Combine panels with separator
	var contentLines []string
	contentLines = append(contentLines, headerStyle.Render("Select Environment"))
	contentLines = append(contentLines, "") // Empty line

	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	for i := 0; i < maxLines; i++ {
		left := leftLines[i]
		right := rightLines[i]
		// Pad right to width
		rightPadding := rightWidth - len(right)
		if rightPadding > 0 {
			right += strings.Repeat(" ", rightPadding)
		}
		contentLines = append(contentLines, left+separatorStyle.Render(" │ ")+right)
	}

	contentLines = append(contentLines, "") // Empty line

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Width(boxWidth - 4).
		Align(lipgloss.Center)
	contentLines = append(contentLines, footerStyle.Render("j/k: navigate  Enter: select  e: edit  Esc: cancel"))

	content := strings.Join(contentLines, "\n")

	// Box style with border
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	box := boxStyle.Render(content)

	// Center the box on screen
	containerStyle := lipgloss.NewStyle().
		Width(v.width).
		Height(v.height).
		Align(lipgloss.Center, lipgloss.Center)

	return containerStyle.Render(box)
}

// renderProxyDialog renders the proxy settings dialog.
func (v *MainView) renderProxyDialog() string {
	boxWidth := 60
	if boxWidth > v.width-4 {
		boxWidth = v.width - 4
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Width(boxWidth - 4).
		Align(lipgloss.Center).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("236")).
		Width(boxWidth - 8).
		Padding(0, 1)

	var lines []string
	lines = append(lines, headerStyle.Render("Proxy Settings"))
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Proxy URL (http://, https://, socks5://):"))
	lines = append(lines, inputStyle.Render(v.proxyInput+"█"))
	lines = append(lines, "")

	// Current status
	if v.proxyURL != "" {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
		lines = append(lines, statusStyle.Render("Current: "+v.proxyURL))
	} else {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		lines = append(lines, statusStyle.Render("No proxy configured"))
	}
	lines = append(lines, "")

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Width(boxWidth - 4).
		Align(lipgloss.Center)
	lines = append(lines, footerStyle.Render("Enter: save  Esc: cancel"))

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	box := boxStyle.Render(content)

	containerStyle := lipgloss.NewStyle().
		Width(v.width).
		Height(v.height).
		Align(lipgloss.Center, lipgloss.Center)

	return containerStyle.Render(box)
}

// renderTLSDialog renders the TLS/certificate settings dialog.
func (v *MainView) renderTLSDialog() string {
	boxWidth := 60
	if boxWidth > v.width-4 {
		boxWidth = v.width - 4
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Width(boxWidth - 4).
		Align(lipgloss.Center).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	selectedLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Width(boxWidth - 8).
		Padding(0, 1)

	selectedInputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Width(boxWidth - 8).
		Padding(0, 1)

	var lines []string
	lines = append(lines, headerStyle.Render("TLS / Certificate Settings"))
	lines = append(lines, "")

	// Client Certificate
	if v.tlsDialogField == 0 {
		lines = append(lines, selectedLabelStyle.Render("→ Client Certificate (.pem):"))
		lines = append(lines, selectedInputStyle.Render(v.tlsCertInput+"█"))
	} else {
		lines = append(lines, labelStyle.Render("  Client Certificate (.pem):"))
		lines = append(lines, inputStyle.Render(v.tlsCertInput))
	}

	// Client Key
	if v.tlsDialogField == 1 {
		lines = append(lines, selectedLabelStyle.Render("→ Client Key (.pem):"))
		lines = append(lines, selectedInputStyle.Render(v.tlsKeyInput+"█"))
	} else {
		lines = append(lines, labelStyle.Render("  Client Key (.pem):"))
		lines = append(lines, inputStyle.Render(v.tlsKeyInput))
	}

	// CA Certificate
	if v.tlsDialogField == 2 {
		lines = append(lines, selectedLabelStyle.Render("→ CA Certificate (.pem):"))
		lines = append(lines, selectedInputStyle.Render(v.tlsCAInput+"█"))
	} else {
		lines = append(lines, labelStyle.Render("  CA Certificate (.pem):"))
		lines = append(lines, inputStyle.Render(v.tlsCAInput))
	}

	// Insecure Skip Verify
	checkBox := "[ ]"
	if v.tlsInsecureSkip {
		checkBox = "[x]"
	}
	if v.tlsDialogField == 3 {
		lines = append(lines, selectedLabelStyle.Render("→ "+checkBox+" Skip certificate verification"))
	} else {
		lines = append(lines, labelStyle.Render("  "+checkBox+" Skip certificate verification"))
	}

	lines = append(lines, "")

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Width(boxWidth - 4).
		Align(lipgloss.Center)
	lines = append(lines, footerStyle.Render("Tab: next  Space: toggle  Enter: save  Esc: cancel"))

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	box := boxStyle.Render(content)

	containerStyle := lipgloss.NewStyle().
		Width(v.width).
		Height(v.height).
		Align(lipgloss.Center, lipgloss.Center)

	return containerStyle.Render(box)
}

// Name returns the view name.
func (v *MainView) Name() string {
	return "Main"
}

// Title returns the view title.
func (v *MainView) Title() string {
	return "Currier"
}

// Focused returns true if focused.
func (v *MainView) Focused() bool {
	return true // MainView is always focused
}

// Focus sets focus.
func (v *MainView) Focus() {}

// Blur removes focus.
func (v *MainView) Blur() {}

// SetSize sets dimensions.
func (v *MainView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.updatePaneSizes()
}

// Width returns the width.
func (v *MainView) Width() int {
	return v.width
}

// Height returns the height.
func (v *MainView) Height() int {
	return v.height
}

// FocusedPane returns the currently focused pane.
func (v *MainView) FocusedPane() Pane {
	return v.focusedPane
}

// FocusPane focuses a specific pane.
func (v *MainView) FocusPane(pane Pane) {
	v.focusPane(pane)
}

// CollectionTree returns the collection tree component.
func (v *MainView) CollectionTree() *components.CollectionTree {
	return v.tree
}

// RequestPanel returns the request panel component.
func (v *MainView) RequestPanel() *components.RequestPanel {
	return v.request
}

// ResponsePanel returns the response panel component.
func (v *MainView) ResponsePanel() *components.ResponsePanel {
	return v.response
}

// SetCollections sets the collections to display and auto-selects the first request.
func (v *MainView) SetCollections(collections []*core.Collection) {
	v.tree.SetCollections(collections)

	// Auto-select the first request from the first collection
	if len(collections) > 0 {
		for _, col := range collections {
			if req := col.FirstRequest(); req != nil {
				v.request.SetRequest(req)
				break
			}
		}
	}
}

// SetEnvironment sets the environment and interpolation engine.
func (v *MainView) SetEnvironment(env *core.Environment, engine *interpolate.Engine) {
	v.environment = env
	v.interpolator = engine
}

// SetHistoryStore sets the history store for browsing request history.
func (v *MainView) SetHistoryStore(store history.Store) {
	v.historyStore = store
	v.tree.SetHistoryStore(store)
}

// SetCollectionStore sets the collection store for persistence and loads existing collections.
func (v *MainView) SetCollectionStore(store *filesystem.CollectionStore) {
	v.collectionStore = store

	// Load existing collections from storage
	if store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		metas, err := store.List(ctx)
		if err == nil {
			var collections []*core.Collection
			for _, meta := range metas {
				coll, err := store.Get(ctx, meta.ID)
				if err == nil {
					collections = append(collections, coll)
				}
			}
			if len(collections) > 0 {
				v.tree.SetCollections(collections)
			}
		}
	}
}

// SetEnvironmentStore sets the environment store for switching environments.
func (v *MainView) SetEnvironmentStore(store *filesystem.EnvironmentStore) {
	v.environmentStore = store
}

// SetCookieJar sets the cookie jar for automatic cookie handling.
func (v *MainView) SetCookieJar(jar *cookies.PersistentJar) {
	v.cookieJar = jar
}

// SetStarredStore sets the starred store for favorite requests.
func (v *MainView) SetStarredStore(store starred.Store) {
	v.starredStore = store
	v.tree.SetStarredStore(store)
}

// EnableCaptureMode enables capture mode and auto-starts the proxy on first update.
func (v *MainView) EnableCaptureMode() {
	v.startCaptureOnInit = true
}

// openEnvSwitcher opens the environment switcher popup.
func (v *MainView) openEnvSwitcher() (tui.Component, tea.Cmd) {
	if v.environmentStore == nil {
		v.notification = "No environment store available"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})
	}

	// Load environments list
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	envList, err := v.environmentStore.List(ctx)
	if err != nil {
		v.notification = "Failed to load environments"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})
	}

	if len(envList) == 0 {
		v.notification = "No environments available"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})
	}

	v.envList = envList
	v.showEnvSwitcher = true

	// Set cursor to current active environment if exists
	v.envCursor = 0
	for i, env := range envList {
		if env.IsActive {
			v.envCursor = i
			break
		}
	}

	return v, nil
}

// handleEnvSwitcherKey handles keyboard input when the environment switcher is open.
func (v *MainView) handleEnvSwitcherKey(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		v.showEnvSwitcher = false
		return v, nil

	case tea.KeyEnter:
		return v.selectEnvironment()

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			if v.envCursor < len(v.envList)-1 {
				v.envCursor++
			}
		case "k":
			if v.envCursor > 0 {
				v.envCursor--
			}
		case "q":
			v.showEnvSwitcher = false
		case "e":
			// Open environment editor for selected environment
			return v.openEnvEditor()
		}
	}

	return v, nil
}

// selectEnvironment selects the currently highlighted environment.
func (v *MainView) selectEnvironment() (tui.Component, tea.Cmd) {
	if v.envCursor < 0 || v.envCursor >= len(v.envList) {
		v.showEnvSwitcher = false
		return v, nil
	}

	selectedMeta := v.envList[v.envCursor]
	v.showEnvSwitcher = false

	// Load the full environment asynchronously
	store := v.environmentStore
	return v, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Set this environment as active
		if err := store.SetActive(ctx, selectedMeta.ID); err != nil {
			return environmentLoadErrorMsg{Error: err}
		}

		// Load the full environment
		env, err := store.Get(ctx, selectedMeta.ID)
		if err != nil {
			return environmentLoadErrorMsg{Error: err}
		}

		// Create new interpolation engine
		engine := interpolate.NewEngine()
		engine.SetVariables(env.ExportAll())

		return environmentSwitchedMsg{
			Environment: env,
			Engine:      engine,
		}
	}
}

// openEnvEditor opens the environment editor for the selected environment.
func (v *MainView) openEnvEditor() (tui.Component, tea.Cmd) {
	if len(v.envList) == 0 || v.envCursor >= len(v.envList) {
		return v, nil
	}

	selectedMeta := v.envList[v.envCursor]
	v.showEnvSwitcher = false

	// Load the environment to edit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	env, err := v.environmentStore.Get(ctx, selectedMeta.ID)
	if err != nil {
		v.notification = "Failed to load environment"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, nil
	}

	// Initialize editor state
	v.editingEnv = env
	v.envVarKeys = make([]string, 0, len(env.Variables()))
	for k := range env.Variables() {
		v.envVarKeys = append(v.envVarKeys, k)
	}
	// Sort keys for consistent display
	sort.Strings(v.envVarKeys)

	v.envEditorCursor = 0
	v.envEditorMode = 0
	v.envEditorKeyInput = ""
	v.envEditorValInput = ""
	v.showEnvEditor = true

	return v, nil
}

// handleEnvEditorKey handles keyboard input when the environment editor is open.
func (v *MainView) handleEnvEditorKey(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	// Handle input modes (adding/editing)
	if v.envEditorMode != 0 {
		return v.handleEnvEditorInput(msg)
	}

	// Browse mode
	switch msg.Type {
	case tea.KeyEsc:
		return v.saveAndCloseEnvEditor()

	case tea.KeyUp, tea.KeyCtrlP:
		if v.envEditorCursor > 0 {
			v.envEditorCursor--
		}

	case tea.KeyDown, tea.KeyCtrlN:
		if v.envEditorCursor < len(v.envVarKeys)-1 {
			v.envEditorCursor++
		}

	case tea.KeyEnter:
		// Edit selected variable
		if len(v.envVarKeys) > 0 && v.envEditorCursor < len(v.envVarKeys) {
			key := v.envVarKeys[v.envEditorCursor]
			v.envEditorOrigKey = key
			v.envEditorKeyInput = key
			v.envEditorValInput = v.editingEnv.GetVariable(key)
			v.envEditorMode = 3 // Edit key
		}

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			if v.envEditorCursor < len(v.envVarKeys)-1 {
				v.envEditorCursor++
			}
		case "k":
			if v.envEditorCursor > 0 {
				v.envEditorCursor--
			}
		case "a":
			// Add new variable
			v.envEditorKeyInput = ""
			v.envEditorValInput = ""
			v.envEditorMode = 1 // Add key
		case "e":
			// Edit selected variable
			if len(v.envVarKeys) > 0 && v.envEditorCursor < len(v.envVarKeys) {
				key := v.envVarKeys[v.envEditorCursor]
				v.envEditorOrigKey = key
				v.envEditorKeyInput = key
				v.envEditorValInput = v.editingEnv.GetVariable(key)
				v.envEditorMode = 3 // Edit key
			}
		case "d":
			// Delete selected variable
			if len(v.envVarKeys) > 0 && v.envEditorCursor < len(v.envVarKeys) {
				key := v.envVarKeys[v.envEditorCursor]
				v.editingEnv.DeleteVariable(key)
				v.envVarKeys = append(v.envVarKeys[:v.envEditorCursor], v.envVarKeys[v.envEditorCursor+1:]...)
				if v.envEditorCursor >= len(v.envVarKeys) && v.envEditorCursor > 0 {
					v.envEditorCursor--
				}
			}
		case "g":
			v.envEditorCursor = 0
		case "G":
			if len(v.envVarKeys) > 0 {
				v.envEditorCursor = len(v.envVarKeys) - 1
			}
		}
	}

	return v, nil
}

// handleEnvEditorInput handles text input when adding/editing variables.
func (v *MainView) handleEnvEditorInput(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Cancel input
		v.envEditorMode = 0
		v.envEditorKeyInput = ""
		v.envEditorValInput = ""
		return v, nil

	case tea.KeyTab:
		// Switch between key and value fields
		switch v.envEditorMode {
		case 1: // Add key -> Add value
			if v.envEditorKeyInput != "" {
				v.envEditorMode = 2
			}
		case 2: // Add value -> Add key
			v.envEditorMode = 1
		case 3: // Edit key -> Edit value
			v.envEditorMode = 4
		case 4: // Edit value -> Edit key
			v.envEditorMode = 3
		}
		return v, nil

	case tea.KeyEnter:
		// Save the variable
		key := strings.TrimSpace(v.envEditorKeyInput)
		if key == "" {
			v.envEditorMode = 0
			return v, nil
		}

		value := v.envEditorValInput

		if v.envEditorMode == 1 || v.envEditorMode == 2 {
			// Adding new variable
			v.editingEnv.SetVariable(key, value)
			// Add to keys list if not exists
			found := false
			for _, k := range v.envVarKeys {
				if k == key {
					found = true
					break
				}
			}
			if !found {
				v.envVarKeys = append(v.envVarKeys, key)
				sort.Strings(v.envVarKeys)
				// Update cursor to the new key
				for i, k := range v.envVarKeys {
					if k == key {
						v.envEditorCursor = i
						break
					}
				}
			}
		} else {
			// Editing existing variable
			if v.envEditorOrigKey != key {
				// Key changed - delete old, add new
				v.editingEnv.DeleteVariable(v.envEditorOrigKey)
				// Update keys list
				for i, k := range v.envVarKeys {
					if k == v.envEditorOrigKey {
						v.envVarKeys[i] = key
						break
					}
				}
				sort.Strings(v.envVarKeys)
			}
			v.editingEnv.SetVariable(key, value)
		}

		v.envEditorMode = 0
		v.envEditorKeyInput = ""
		v.envEditorValInput = ""
		return v, nil

	case tea.KeyBackspace:
		// Delete character from current field
		if v.envEditorMode == 1 || v.envEditorMode == 3 {
			if len(v.envEditorKeyInput) > 0 {
				v.envEditorKeyInput = v.envEditorKeyInput[:len(v.envEditorKeyInput)-1]
			}
		} else {
			if len(v.envEditorValInput) > 0 {
				v.envEditorValInput = v.envEditorValInput[:len(v.envEditorValInput)-1]
			}
		}
		return v, nil

	case tea.KeyRunes:
		// Add character to current field
		if v.envEditorMode == 1 || v.envEditorMode == 3 {
			v.envEditorKeyInput += string(msg.Runes)
		} else {
			v.envEditorValInput += string(msg.Runes)
		}
		return v, nil

	case tea.KeySpace:
		// Add space to current field
		if v.envEditorMode == 1 || v.envEditorMode == 3 {
			v.envEditorKeyInput += " "
		} else {
			v.envEditorValInput += " "
		}
		return v, nil
	}

	return v, nil
}

// saveAndCloseEnvEditor saves the environment and closes the editor.
func (v *MainView) saveAndCloseEnvEditor() (tui.Component, tea.Cmd) {
	v.showEnvEditor = false

	if v.editingEnv == nil || v.environmentStore == nil {
		return v, nil
	}

	// Save the environment
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := v.environmentStore.Save(ctx, v.editingEnv); err != nil {
		v.notification = "Failed to save environment"
		v.notifyUntil = time.Now().Add(2 * time.Second)
	} else {
		v.notification = "Environment saved: " + v.editingEnv.Name()
		v.notifyUntil = time.Now().Add(2 * time.Second)

		// Update the current environment if it's the one being edited
		if v.environment != nil && v.environment.ID() == v.editingEnv.ID() {
			v.environment = v.editingEnv
			if v.interpolator != nil {
				v.interpolator.SetVariables(v.editingEnv.ExportAll())
			}
		}
	}

	v.editingEnv = nil
	return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}

// renderEnvEditor renders the environment editor popup.
func (v *MainView) renderEnvEditor() string {
	if v.editingEnv == nil {
		return ""
	}

	// Box dimensions
	boxWidth := 60
	if boxWidth > v.width-4 {
		boxWidth = v.width - 4
	}
	boxHeight := 20
	if boxHeight > v.height-4 {
		boxHeight = v.height - 4
	}

	// Styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Width(boxWidth - 4).
		Align(lipgloss.Center).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243"))

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	activeInputStyle := inputStyle.
		Background(lipgloss.Color("62"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("229"))

	// Build content lines
	var lines []string
	lines = append(lines, headerStyle.Render("Edit Environment: "+v.editingEnv.Name()))
	lines = append(lines, "")

	// Show input fields if in add/edit mode
	if v.envEditorMode != 0 {
		keyLabel := "Key:"
		valLabel := "Value:"
		if v.envEditorMode == 1 || v.envEditorMode == 2 {
			lines = append(lines, labelStyle.Render("  Add New Variable"))
		} else {
			lines = append(lines, labelStyle.Render("  Edit Variable"))
		}
		lines = append(lines, "")

		// Key input
		keyStyle := inputStyle
		if v.envEditorMode == 1 || v.envEditorMode == 3 {
			keyStyle = activeInputStyle
		}
		keyDisplay := v.envEditorKeyInput
		if v.envEditorMode == 1 || v.envEditorMode == 3 {
			keyDisplay += "█"
		}
		keyField := fmt.Sprintf("  %s %s", keyLabel, keyStyle.Render(keyDisplay))
		lines = append(lines, keyField)

		// Value input
		valStyle := inputStyle
		if v.envEditorMode == 2 || v.envEditorMode == 4 {
			valStyle = activeInputStyle
		}
		valDisplay := v.envEditorValInput
		if v.envEditorMode == 2 || v.envEditorMode == 4 {
			valDisplay += "█"
		}
		valField := fmt.Sprintf("  %s %s", valLabel, valStyle.Render(valDisplay))
		lines = append(lines, valField)

		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("  Tab: switch field  Enter: save  Esc: cancel"))
	} else {
		// Show variables list
		if len(v.envVarKeys) == 0 {
			lines = append(lines, labelStyle.Render("  No variables defined"))
			lines = append(lines, "")
		} else {
			// Calculate visible range
			maxVisible := boxHeight - 8
			startIdx := 0
			if v.envEditorCursor >= maxVisible {
				startIdx = v.envEditorCursor - maxVisible + 1
			}
			endIdx := startIdx + maxVisible
			if endIdx > len(v.envVarKeys) {
				endIdx = len(v.envVarKeys)
			}

			for i := startIdx; i < endIdx; i++ {
				key := v.envVarKeys[i]
				value := v.editingEnv.GetVariable(key)

				// Truncate value if too long
				maxValLen := boxWidth - len(key) - 12
				if maxValLen > 3 && len(value) > maxValLen {
					value = value[:maxValLen-3] + "..."
				}

				prefix := "  "
				if i == v.envEditorCursor {
					prefix = "→ "
				}

				line := fmt.Sprintf("%s%s = %s", prefix, key, value)

				// Style selected line
				style := lipgloss.NewStyle()
				if i == v.envEditorCursor {
					style = selectedStyle
					// Pad to width
					padding := boxWidth - 4 - len(line)
					if padding > 0 {
						line += strings.Repeat(" ", padding)
					}
				}

				lines = append(lines, style.Render(line))
			}
		}

		lines = append(lines, "")

		// Footer with help
		footerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Width(boxWidth - 4).
			Align(lipgloss.Center)
		lines = append(lines, footerStyle.Render("a: add  e/Enter: edit  d: delete  Esc: save & close"))
	}

	// Create bordered box
	content := strings.Join(lines, "\n")
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(content)

	// Center the box
	centered := lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)

	return centered
}

// handleProxyDialogKey handles keyboard input for the proxy settings dialog.
func (v *MainView) handleProxyDialogKey(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		v.showProxyDialog = false
		return v, nil

	case tea.KeyEnter:
		// Save proxy settings
		v.proxyURL = v.proxyInput
		v.showProxyDialog = false
		if v.proxyURL != "" {
			v.notification = "Proxy set: " + v.proxyURL
		} else {
			v.notification = "Proxy disabled"
		}
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case tea.KeyBackspace:
		if len(v.proxyInput) > 0 {
			v.proxyInput = v.proxyInput[:len(v.proxyInput)-1]
		}

	case tea.KeyRunes:
		v.proxyInput += string(msg.Runes)
	}

	return v, nil
}

// handleTLSDialogKey handles keyboard input for the TLS settings dialog.
func (v *MainView) handleTLSDialogKey(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		v.showTLSDialog = false
		return v, nil

	case tea.KeyEnter:
		// Save TLS settings
		v.tlsCertFile = v.tlsCertInput
		v.tlsKeyFile = v.tlsKeyInput
		v.tlsCAFile = v.tlsCAInput
		v.showTLSDialog = false
		if v.tlsCertFile != "" || v.tlsCAFile != "" || v.tlsInsecureSkip {
			v.notification = "TLS settings saved"
		} else {
			v.notification = "TLS settings cleared"
		}
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})

	case tea.KeyTab, tea.KeyDown:
		// Move to next field
		v.tlsDialogField = (v.tlsDialogField + 1) % 4

	case tea.KeyShiftTab, tea.KeyUp:
		// Move to previous field
		v.tlsDialogField = (v.tlsDialogField + 3) % 4

	case tea.KeySpace:
		// Toggle insecure skip (only for field 3)
		if v.tlsDialogField == 3 {
			v.tlsInsecureSkip = !v.tlsInsecureSkip
		}

	case tea.KeyBackspace:
		switch v.tlsDialogField {
		case 0:
			if len(v.tlsCertInput) > 0 {
				v.tlsCertInput = v.tlsCertInput[:len(v.tlsCertInput)-1]
			}
		case 1:
			if len(v.tlsKeyInput) > 0 {
				v.tlsKeyInput = v.tlsKeyInput[:len(v.tlsKeyInput)-1]
			}
		case 2:
			if len(v.tlsCAInput) > 0 {
				v.tlsCAInput = v.tlsCAInput[:len(v.tlsCAInput)-1]
			}
		}

	case tea.KeyRunes:
		if v.tlsDialogField != 3 { // Don't add text to toggle field
			switch v.tlsDialogField {
			case 0:
				v.tlsCertInput += string(msg.Runes)
			case 1:
				v.tlsKeyInput += string(msg.Runes)
			case 2:
				v.tlsCAInput += string(msg.Runes)
			}
		}
	}

	return v, nil
}

// startCollectionRunner starts running the current collection.
func (v *MainView) startCollectionRunner() (tui.Component, tea.Cmd) {
	// Get the current collection
	coll := v.tree.GetSelectedCollection()
	if coll == nil {
		v.notification = "No collection selected"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})
	}

	// Initialize runner state
	v.showRunnerModal = true
	v.runnerRunning = true
	v.runnerProgress = 0
	v.runnerTotal = 0
	v.runnerSummary = nil
	v.runnerCurrentReq = ""

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	v.runnerCancelFunc = cancel

	// Start runner in background
	return v, func() tea.Msg {
		// Build runner options
		opts := []runner.Option{}

		if v.environment != nil {
			opts = append(opts, runner.WithEnvironment(v.environment))
		}

		// Create HTTP client with proxy/TLS settings
		clientOpts := []httpclient.Option{
			httpclient.WithTimeout(30 * time.Second),
		}
		if v.cookieJar != nil {
			clientOpts = append(clientOpts, httpclient.WithCookieJar(v.cookieJar))
		}
		if v.proxyURL != "" {
			clientOpts = append(clientOpts, httpclient.WithProxy(v.proxyURL))
		}
		if v.tlsCertFile != "" && v.tlsKeyFile != "" {
			clientOpts = append(clientOpts, httpclient.WithClientCert(v.tlsCertFile, v.tlsKeyFile))
		}
		if v.tlsCAFile != "" {
			clientOpts = append(clientOpts, httpclient.WithCACert(v.tlsCAFile))
		}
		if v.tlsInsecureSkip {
			clientOpts = append(clientOpts, httpclient.WithInsecureSkipVerify())
		}
		httpClient := httpclient.NewClient(clientOpts...)
		opts = append(opts, runner.WithHTTPClient(httpClient))

		// Create runner
		r := runner.NewRunner(coll, opts...)

		// Run the collection
		summary := r.Run(ctx)

		return runnerCompleteMsg{Summary: summary}
	}
}

// handleRunnerModalKey handles keyboard input for the runner modal.
func (v *MainView) handleRunnerModalKey(msg tea.KeyMsg) (tui.Component, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if v.runnerRunning && v.runnerCancelFunc != nil {
			// Cancel the runner
			v.runnerCancelFunc()
			v.runnerRunning = false
			v.notification = "Runner cancelled"
			v.notifyUntil = time.Now().Add(2 * time.Second)
		}
		v.showRunnerModal = false
		return v, nil

	case tea.KeyEnter:
		// Close modal if runner is complete
		if !v.runnerRunning {
			v.showRunnerModal = false
		}
		return v, nil
	}

	return v, nil
}

// renderRunnerModal renders the collection runner modal.
func (v *MainView) renderRunnerModal() string {
	boxWidth := 70
	if boxWidth > v.width-4 {
		boxWidth = v.width - 4
	}

	boxHeight := 20
	if boxHeight > v.height-4 {
		boxHeight = v.height - 4
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Width(boxWidth - 4).
		Align(lipgloss.Center)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	passedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	failedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

	var lines []string

	if v.runnerRunning {
		lines = append(lines, headerStyle.Render("Running Collection"))
		lines = append(lines, "")

		if v.runnerTotal > 0 {
			progress := fmt.Sprintf("Progress: %d / %d", v.runnerProgress, v.runnerTotal)
			lines = append(lines, labelStyle.Render(progress))

			// Progress bar
			barWidth := boxWidth - 10
			filled := int(float64(barWidth) * float64(v.runnerProgress) / float64(v.runnerTotal))
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			lines = append(lines, labelStyle.Render("  "+bar))
		} else {
			lines = append(lines, labelStyle.Render("Starting..."))
		}

		if v.runnerCurrentReq != "" {
			lines = append(lines, "")
			lines = append(lines, labelStyle.Render("Current: "+v.runnerCurrentReq))
		}

		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("Press Esc to cancel"))
	} else if v.runnerSummary != nil {
		s := v.runnerSummary
		lines = append(lines, headerStyle.Render("Collection Run Complete"))
		lines = append(lines, "")

		// Summary stats
		lines = append(lines, labelStyle.Render(fmt.Sprintf("Collection: %s", s.CollectionName)))
		lines = append(lines, labelStyle.Render(fmt.Sprintf("Duration: %s", s.TotalDuration.Round(time.Millisecond))))
		lines = append(lines, "")

		// Request stats
		requestLine := fmt.Sprintf("Requests: %d/%d passed", s.Passed, s.TotalRequests)
		if s.Failed > 0 {
			lines = append(lines, failedStyle.Render(requestLine))
		} else {
			lines = append(lines, passedStyle.Render(requestLine))
		}

		// Test stats
		if s.TotalTests > 0 {
			testLine := fmt.Sprintf("Tests: %d/%d passed", s.TestsPassed, s.TotalTests)
			if s.TestsFailed > 0 {
				lines = append(lines, failedStyle.Render(testLine))
			} else {
				lines = append(lines, passedStyle.Render(testLine))
			}
		}

		// Show failed requests
		if s.Failed > 0 {
			lines = append(lines, "")
			lines = append(lines, failedStyle.Render("Failed Requests:"))
			for _, r := range s.Results {
				if r.Error != nil {
					errLine := fmt.Sprintf("  ✗ %s %s", r.Method, r.RequestName)
					lines = append(lines, failedStyle.Render(errLine))
					errDetail := fmt.Sprintf("    %s", r.Error.Error())
					if len(errDetail) > boxWidth-6 {
						errDetail = errDetail[:boxWidth-9] + "..."
					}
					lines = append(lines, hintStyle.Render(errDetail))
				}
			}
		}

		// Show failed tests
		failedTests := 0
		for _, r := range s.Results {
			for _, t := range r.TestResults {
				if !t.Passed {
					failedTests++
				}
			}
		}
		if failedTests > 0 && len(lines) < boxHeight-4 {
			lines = append(lines, "")
			lines = append(lines, failedStyle.Render("Failed Tests:"))
			for _, r := range s.Results {
				for _, t := range r.TestResults {
					if !t.Passed && len(lines) < boxHeight-2 {
						testLine := fmt.Sprintf("  ✗ [%s] %s", r.RequestName, t.Name)
						if len(testLine) > boxWidth-4 {
							testLine = testLine[:boxWidth-7] + "..."
						}
						lines = append(lines, failedStyle.Render(testLine))
					}
				}
			}
		}

		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("Press Enter or Esc to close"))
	}

	// Build the box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(strings.Join(lines, "\n"))

	// Center the box
	containerStyle := lipgloss.NewStyle().
		Width(v.width).
		Height(v.height).
		Align(lipgloss.Center, lipgloss.Center)

	return containerStyle.Render(box)
}

// saveToHistory saves a request/response pair to history.
func (v *MainView) saveToHistory(req *core.RequestDefinition, resp *core.Response, err error) {
	if v.historyStore == nil || req == nil {
		return
	}

	entry := history.Entry{
		RequestMethod:  req.Method(),
		RequestURL:     req.FullURL(),
		RequestName:    req.Name(),
		RequestBody:    req.Body(),
		RequestHeaders: req.Headers(),
		Timestamp:      time.Now(),
	}

	if resp != nil {
		entry.ResponseStatus = resp.Status().Code()
		entry.ResponseStatusText = resp.Status().Text()
		entry.ResponseBody = resp.Body().String()
		entry.ResponseTime = resp.Timing().Total.Milliseconds()
		entry.ResponseSize = resp.Body().Size()
		entry.ResponseHeaders = make(map[string]string)
		for _, key := range resp.Headers().Keys() {
			entry.ResponseHeaders[key] = resp.Headers().Get(key)
		}
	}

	if err != nil {
		entry.ResponseStatusText = "Error: " + err.Error()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, addErr := v.historyStore.Add(ctx, entry); addErr != nil {
		// Log error but don't crash - history is optional
		// Could add notification here if desired
	}
}

// Environment returns the current environment.
func (v *MainView) Environment() *core.Environment {
	return v.environment
}

// Interpolator returns the interpolation engine.
func (v *MainView) Interpolator() *interpolate.Engine {
	return v.interpolator
}

// ShowingHelp returns true if help is showing.
func (v *MainView) ShowingHelp() bool {
	return v.showHelp
}

// ShowHelp shows the help overlay.
func (v *MainView) ShowHelp() {
	v.showHelp = true
}

// HideHelp hides the help overlay.
func (v *MainView) HideHelp() {
	v.showHelp = false
}

// --- State accessors for E2E testing ---

// Notification returns the current notification message.
func (v *MainView) Notification() string {
	return v.notification
}

// HTTPClientConfig holds configuration for the HTTP client.
type HTTPClientConfig struct {
	CookieJar       *cookies.PersistentJar
	ProxyURL        string
	CertFile        string
	KeyFile         string
	CAFile          string
	InsecureSkip    bool
}

// sendRequest creates a tea.Cmd that sends an HTTP request asynchronously.
func sendRequest(reqDef *core.RequestDefinition, engine *interpolate.Engine, config HTTPClientConfig) tea.Cmd {
	return func() tea.Msg {
		// Early validation of URL
		url := reqDef.FullURL()
		if url == "" {
			return components.RequestErrorMsg{Error: fmt.Errorf("URL is empty. Press 'e' to edit the URL")}
		}

		// Basic URL validation
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return components.RequestErrorMsg{Error: fmt.Errorf("URL must start with http:// or https://")}
		}

		// Create script scope for pre-request and test scripts
		scope := script.NewScopeWithAssertions()
		var consoleMessages []components.ConsoleMessage

		// Set up console handler to capture console output
		scope.Engine().SetConsoleHandler(func(level, message string) {
			consoleMessages = append(consoleMessages, components.ConsoleMessage{
				Level:   level,
				Message: message,
			})
		})

		// Set up request context for scripts
		scope.SetRequestMethod(reqDef.Method())
		scope.SetRequestURL(reqDef.FullURL())
		scope.SetRequestHeaders(reqDef.Headers())
		scope.SetRequestBody(reqDef.Body())

		// Execute pre-request script (if any)
		preScript := reqDef.PreScript()
		if preScript != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := scope.Execute(ctx, preScript)
			cancel()
			if err != nil {
				return components.RequestErrorMsg{Error: fmt.Errorf("pre-request script error: %w", err)}
			}
		}

		// Convert RequestDefinition to Request (with or without interpolation)
		var req *core.Request
		var err error

		if engine != nil {
			req, err = reqDef.ToRequestWithEnv(engine)
		} else {
			req, err = reqDef.ToRequest()
		}

		if err != nil {
			return components.RequestErrorMsg{Error: err}
		}

		// Create HTTP client with timeout and configured options
		clientOpts := []httpclient.Option{
			httpclient.WithTimeout(30 * time.Second),
		}
		if config.CookieJar != nil {
			clientOpts = append(clientOpts, httpclient.WithCookieJar(config.CookieJar))
		}
		if config.ProxyURL != "" {
			clientOpts = append(clientOpts, httpclient.WithProxy(config.ProxyURL))
		}
		if config.CertFile != "" && config.KeyFile != "" {
			clientOpts = append(clientOpts, httpclient.WithClientCert(config.CertFile, config.KeyFile))
		}
		if config.CAFile != "" {
			clientOpts = append(clientOpts, httpclient.WithCACert(config.CAFile))
		}
		if config.InsecureSkip {
			clientOpts = append(clientOpts, httpclient.WithInsecureSkipVerify())
		}
		client := httpclient.NewClient(clientOpts...)

		// Send the request
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Send(ctx, req)
		if err != nil {
			return components.RequestErrorMsg{Error: err}
		}

		// Execute test script (if any)
		var testResults []script.TestResult
		testScript := reqDef.PostScript()
		if testScript != "" {
			// Convert headers to map[string]string for script context
			headersMap := make(map[string]string)
			for _, key := range resp.Headers().Keys() {
				headersMap[key] = resp.Headers().Get(key)
			}

			// Set up response context for test scripts
			scope.SetResponseStatus(resp.Status().Code())
			scope.SetResponseHeaders(headersMap)
			scope.SetResponseBody(resp.Body().String())
			scope.SetResponseTime(resp.Timing().Total.Milliseconds())

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := scope.Execute(ctx, testScript)
			cancel()
			if err != nil {
				// Add script error to console but don't fail the request
				consoleMessages = append(consoleMessages, components.ConsoleMessage{
					Level:   "error",
					Message: fmt.Sprintf("Test script error: %v", err),
				})
			}

			// Get test results
			testResults = scope.GetTestResults()
		}

		return components.ResponseReceivedMsg{
			Response:    resp,
			TestResults: testResults,
			Console:     consoleMessages,
		}
	}
}

// connectWebSocket creates a tea.Cmd that connects to a WebSocket endpoint.
func (v *MainView) connectWebSocket(def *core.WebSocketDefinition) tea.Cmd {
	return func() tea.Msg {
		if def == nil || def.Endpoint == "" {
			return components.WSErrorMsg{Error: fmt.Errorf("no WebSocket endpoint defined")}
		}

		// Validate endpoint
		if !strings.HasPrefix(def.Endpoint, "ws://") && !strings.HasPrefix(def.Endpoint, "wss://") {
			return components.WSErrorMsg{Error: fmt.Errorf("WebSocket URL must start with ws:// or wss://")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Build connection options
		opts := interfaces.ConnectionOptions{
			Headers: def.Headers,
			Timeout: 30 * time.Second,
		}

		// Connect
		conn, err := v.wsClient.Connect(ctx, def.Endpoint, opts)
		if err != nil {
			return components.WSDisconnectedMsg{Error: err}
		}

		// Get the actual WebSocket connection to set up callbacks
		wsConn, err := v.wsClient.GetWebSocketConnection(conn.ID())
		if err == nil {
			// Set up message callback
			wsConn.OnMessage(func(msg *websocket.Message) {
				// Convert to core.WebSocketMessage
				wsMsg := &core.WebSocketMessage{
					ID:           msg.ID,
					ConnectionID: msg.ConnectionID,
					Content:      string(msg.Data),
					Direction:    msg.Direction.String(),
					Timestamp:    msg.Timestamp,
					Type:         msg.Type.String(),
				}
				// Note: In a real app, we'd need a way to send this back to the Update loop
				// For now, messages are handled via the connection's Receive method
				_ = wsMsg
			})

			// Set up state change callback
			wsConn.OnStateChange(func(state interfaces.ConnectionState) {
				// State changes need to be propagated to the UI
				_ = state
			})

			// Set up error callback
			wsConn.OnError(func(err error) {
				_ = err
			})
		}

		return components.WSConnectedMsg{ConnectionID: conn.ID()}
	}
}

// disconnectWebSocket creates a tea.Cmd that disconnects the current WebSocket.
func (v *MainView) disconnectWebSocket() tea.Cmd {
	return func() tea.Msg {
		connID := v.wsPanel.ConnectionID()
		if connID == "" {
			return components.WSErrorMsg{Error: fmt.Errorf("no active WebSocket connection")}
		}

		err := v.wsClient.Disconnect(connID)
		if err != nil {
			return components.WSErrorMsg{Error: err}
		}

		return components.WSDisconnectedMsg{ConnectionID: connID}
	}
}

// sendWebSocketMessage creates a tea.Cmd that sends a message on the current WebSocket.
func (v *MainView) sendWebSocketMessage(content string) tea.Cmd {
	return func() tea.Msg {
		connID := v.wsPanel.ConnectionID()
		if connID == "" {
			return components.WSErrorMsg{Error: fmt.Errorf("no active WebSocket connection")}
		}

		conn, err := v.wsClient.GetConnection(connID)
		if err != nil {
			return components.WSErrorMsg{Error: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = conn.Send(ctx, []byte(content))
		if err != nil {
			return components.WSErrorMsg{Error: err}
		}

		// Create sent message
		msg := core.NewWebSocketMessage(connID, content, "sent")
		return components.WSMessageSentMsg{Message: msg}
	}
}

// WebSocketPanel returns the WebSocket panel component.
func (v *MainView) WebSocketPanel() *components.WebSocketPanel {
	return v.wsPanel
}

// ViewMode returns the current view mode.
func (v *MainView) ViewMode() ViewMode {
	return v.viewMode
}

// SetViewMode sets the view mode.
func (v *MainView) SetViewMode(mode ViewMode) {
	v.viewMode = mode
	v.updatePaneSizes()
}

// SetWebSocketDefinition sets the WebSocket definition to display.
func (v *MainView) SetWebSocketDefinition(def *core.WebSocketDefinition) {
	v.wsPanel.SetDefinition(def)
	v.viewMode = ViewModeWebSocket
	v.focusPane(PaneWebSocket)
	v.updatePaneSizes()
}

// sanitizeFilename removes or replaces characters that are invalid in filenames.
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscores
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	sanitized := re.ReplaceAllString(name, "_")
	// Trim spaces and dots from the end
	sanitized = strings.TrimRight(sanitized, " .")
	if sanitized == "" {
		sanitized = "collection"
	}
	return sanitized
}

// writeFile writes data to a file in the current directory.
func writeFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}

// handleToggleProxy starts or stops the capture proxy server.
func (v *MainView) handleToggleProxy() (tui.Component, tea.Cmd) {
	if v.captureProxy != nil && v.captureProxy.IsRunning() {
		// Stop the proxy
		if v.captureProxyCancel != nil {
			v.captureProxyCancel()
		}
		v.captureProxy.Stop()
		v.tree.SetProxyRunning(false)
		return v, func() tea.Msg { return components.ProxyStoppedMsg{} }
	}

	// Start the proxy (use :0 to let OS assign a free port)
	var err error
	v.captureProxy, err = proxy.NewServer(
		proxy.WithListenAddr(":0"),
		proxy.WithHTTPS(false), // Start without HTTPS by default for simplicity
		proxy.WithBufferSize(1000),
	)
	if err != nil {
		return v, func() tea.Msg { return components.ProxyErrorMsg{Error: err} }
	}

	// Set up capture listener to receive real-time updates
	v.captureProxy.AddListener(proxy.CaptureListenerFunc(func(capture *proxy.CapturedRequest) {
		// This will be called from a goroutine, but bubbletea handles this
		// The capture will be added to the tree via the CaptureReceivedMsg
	}))

	// Create context for the proxy
	v.captureProxyCtx, v.captureProxyCancel = context.WithCancel(context.Background())

	if err := v.captureProxy.Start(v.captureProxyCtx); err != nil {
		return v, func() tea.Msg { return components.ProxyErrorMsg{Error: err} }
	}

	// Set the proxy server on the tree so it can access captures
	v.tree.SetProxyServer(v.captureProxy)
	v.tree.SetProxyRunning(true)

	addr := v.captureProxy.ListenAddr()
	// Return both the started message and start the refresh tick
	return v, tea.Batch(
		func() tea.Msg { return components.ProxyStartedMsg{Address: addr} },
		tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
			return components.RefreshCapturesMsg{}
		}),
	)
}

// displayCapture shows a captured request/response in the panels.
func (v *MainView) displayCapture(capture *proxy.CapturedRequest) {
	// Convert capture to a request definition for display
	reqDef := core.NewRequestDefinition(
		fmt.Sprintf("%s %s", capture.Method, capture.Path),
		capture.Method,
		capture.URL,
	)

	// Set headers
	for name, values := range capture.RequestHeaders {
		for _, val := range values {
			reqDef.SetHeader(name, val)
		}
	}

	// Set body if present
	if len(capture.RequestBody) > 0 {
		reqDef.SetBody(string(capture.RequestBody))
	}

	// Display in request panel
	v.request.SetRequest(reqDef)

	// Create a response to display
	headers := core.NewHeaders()
	for name, values := range capture.ResponseHeaders {
		for _, val := range values {
			headers.Set(name, val)
		}
	}

	body := core.NewRawBody(capture.ResponseBody, headers.Get("Content-Type"))

	timing := interfaces.TimingInfo{
		Total: capture.Duration,
	}

	resp := core.NewResponse(capture.ID, "http", core.NewStatus(capture.StatusCode, capture.StatusText)).
		WithHeaders(headers).
		WithBody(body).
		WithTiming(timing)

	// Display in response panel
	v.response.SetResponse(resp)
	v.focusPane(PaneResponse)
}

// handleExportCapture exports a captured request to a new collection.
func (v *MainView) handleExportCapture(capture *proxy.CapturedRequest) (tui.Component, tea.Cmd) {
	// Create a new request definition from the capture
	reqDef := core.NewRequestDefinition(
		fmt.Sprintf("%s %s", capture.Method, capture.Path),
		capture.Method,
		capture.URL,
	)

	// Set headers
	for name, values := range capture.RequestHeaders {
		for _, val := range values {
			reqDef.SetHeader(name, val)
		}
	}

	// Set body if present
	if len(capture.RequestBody) > 0 {
		reqDef.SetBody(string(capture.RequestBody))
	}

	// Add to "Captured Requests" collection (create if doesn't exist)
	collection := v.tree.GetOrCreateCollection("Captured Requests")
	if collection != nil {
		collection.AddRequest(reqDef)

		// Persist if collection store is available
		if v.collectionStore != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = v.collectionStore.Save(ctx, collection)
			}()
		}

		v.notification = "Request exported to 'Captured Requests'"
		v.notifyUntil = time.Now().Add(2 * time.Second)
		return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{}
		})
	}

	v.notification = "Failed to export request"
	v.notifyUntil = time.Now().Add(2 * time.Second)
	return v, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}
