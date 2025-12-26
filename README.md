# Currier

A vim-modal TUI API client for developers and AI agents.

## Features

- **Vim-style keybindings** - Navigate and edit with familiar modal controls
- **Collections & Environments** - Organize requests with Postman-like collections
- **Variable interpolation** - Use `{{variable}}` syntax in URLs, headers, and bodies
- **Pre/Post-request scripts** - JavaScript-based scripting with assertions
- **Request history** - SQLite-backed history with search and replay
- **Import/Export** - Support for Postman, cURL, HAR, and OpenAPI formats
- **CLI mode** - Execute requests directly from the command line

## Screenshots

### Main Interface

```
╭────────────────────────────╮╭────────────────────────────────────────────────────────────────╮
│        Collections         ││                            Request                             │
│H→History                   ││ GET ▾   https://httpbin.org/get                        ↵ Send │
│→ 📁 Sample API             ││────────────────────────────────────────────────────────────────│
│    📁 Users                ││ URL   Headers   Query   Body   Auth   Pre-req   Tests         │
│       GET  Get Users       ││━━━━━                                                          │
│       POST Create User     ││URL: https://httpbin.org/get                                   │
│    📁 Posts                ││Method: GET                                                    │
│       GET  Get Posts       │╰────────────────────────────────────────────────────────────────╯
│                            │╭────────────────────────────────────────────────────────────────╮
│                            ││                           Response                             │
│                            ││ Body   Headers   Cookies   Timing   Console   Tests           │
│                            ││━━━━━━                                                         │
│                            ││{                                                              │
│                            ││  "args": {},                                                  │
│                            ││  "headers": {                                                 │
│                            ││    "Accept": "*/*",                                           │
│                            ││    "Host": "httpbin.org"                                      │
│                            ││  },                                                           │
│                            ││  "url": "https://httpbin.org/get"                             │
│                            ││}                                                              │
╰────────────────────────────╯╰────────────────────────────────────────────────────────────────╯
 j/k Navigate │ Enter Select │ / Search │ Tab Next pane │ n New │ ? Help │ q Quit
 NORMAL   Collections   No Environment                                    ? help  q quit
```

### History View

Press `H` to switch to request history:

```
╭──────────────────────────────╮╭──────────────────────────────────────────────────────────────╮
│          History             ││                           Request                            │
│C→Collections                 ││ GET ▾   https://httpbin.org/get                      ↵ Send │
│▶  POST httpbin.org/post  200 ││──────────────────────────────────────────────────────────────│
│   GET  httpbin.org/get   200 ││ URL   Headers   Query   Body   Auth   Pre-req   Tests       │
│   GET  api.example.com   200 ││━━━━━                                                        │
│   POST api.example.com   201 ││URL: https://httpbin.org/get                                 │
│   GET  httpbin.org/get   200 ││Method: GET                                                  │
│                              │╰──────────────────────────────────────────────────────────────╯
╰──────────────────────────────╯
```

### Key Features

- **Vim-style navigation**: Use `j/k` to move, `h/l` to collapse/expand, `gg/G` to jump
- **Tab switching**: Use `[` and `]` to switch between tabs within panels
- **Pane focus**: Use `1/2/3` or `Tab` to switch between Collections, Request, and Response
- **Search**: Press `/` to search collections or history
- **Selection indicators**: `→` for collections, `▶` for history items

## Installation

### Homebrew (macOS/Linux)

```bash
brew install artpar/tap/currier
```

### Go Install

```bash
go install github.com/artpar/currier/cmd/currier@latest
```

### macOS (Direct Download)

```bash
# Apple Silicon (M1/M2/M3)
curl -L https://github.com/artpar/currier/releases/latest/download/currier_darwin_arm64.tar.gz | tar xz
sudo mv currier /usr/local/bin/

# Intel Mac
curl -L https://github.com/artpar/currier/releases/latest/download/currier_darwin_amd64.tar.gz | tar xz
sudo mv currier /usr/local/bin/
```

### Linux (Debian/Ubuntu)

```bash
curl -LO https://github.com/artpar/currier/releases/latest/download/currier_linux_amd64.deb
sudo dpkg -i currier_linux_amd64.deb
```

### Linux (RHEL/Fedora)

```bash
curl -LO https://github.com/artpar/currier/releases/latest/download/currier_linux_amd64.rpm
sudo rpm -i currier_linux_amd64.rpm
```

### Linux (Alpine)

```bash
curl -LO https://github.com/artpar/currier/releases/latest/download/currier_linux_amd64.apk
sudo apk add --allow-untrusted currier_linux_amd64.apk
```

### Linux (Arch)

```bash
curl -LO https://github.com/artpar/currier/releases/latest/download/currier_linux_amd64.pkg.tar.zst
sudo pacman -U currier_linux_amd64.pkg.tar.zst
```

### Windows (Scoop)

```powershell
scoop bucket add artpar https://github.com/artpar/scoop-bucket
scoop install currier
```

Or download from [Releases](https://github.com/artpar/currier/releases).

### From Source

```bash
git clone https://github.com/artpar/currier.git
cd currier
make build
```

### Direct Binary Download

Download pre-built binaries for your platform from the [Releases](https://github.com/artpar/currier/releases) page.

## Usage

### TUI Mode

```bash
currier
```

### CLI Mode

```bash
# Send a GET request
currier send https://api.example.com/users

# Send a POST request with JSON body
currier send -X POST -H "Content-Type: application/json" \
  -d '{"name": "John"}' https://api.example.com/users

# Use an environment
currier send -e production https://{{host}}/api/users
```

## Keyboard Shortcuts

| Mode   | Key       | Action              |
|--------|-----------|---------------------|
| Normal | `j/k`     | Navigate up/down    |
| Normal | `h/l`     | Collapse/expand     |
| Normal | `Enter`   | Select item         |
| Normal | `i`       | Enter insert mode   |
| Normal | `/`       | Search              |
| Normal | `:`       | Command mode        |
| Insert | `Esc`     | Return to normal    |

## Project Structure

```
currier/
├── cmd/currier/       # Application entry point
├── internal/
│   ├── app/           # Application orchestration
│   ├── cli/           # CLI commands
│   ├── core/          # Domain models (Request, Response, Collection)
│   ├── exporter/      # Export to cURL, Postman formats
│   ├── history/       # Request history storage
│   ├── importer/      # Import from Postman, cURL, HAR, OpenAPI
│   ├── interfaces/    # Interface definitions
│   ├── interpolate/   # Variable interpolation engine
│   ├── protocol/      # HTTP client implementation
│   ├── script/        # JavaScript scripting engine
│   ├── storage/       # Collection/environment persistence
│   └── tui/           # Terminal UI components
├── tests/             # Integration tests
└── testdata/          # Test fixtures
```

## Development

### Prerequisites

- Go 1.24+
- Make

### Building

```bash
make build          # Build for current platform
make build-all      # Build for all platforms
```

### Testing

```bash
make test           # Run all tests
make test-unit      # Run unit tests only
make test-integration # Run integration tests only
make coverage       # Generate coverage report
```

### Quality Checks

```bash
make fmt            # Format code
make vet            # Run go vet
make check          # Run all quality checks
```

## Import Formats

Currier can import from:

- **Postman** - Collection v2.1 format
- **cURL** - Command line import
- **HAR** - HTTP Archive format
- **OpenAPI** - OpenAPI 3.0 specification

## Export Formats

Export requests to:

- **cURL** - Generate cURL commands
- **Postman** - Collection v2.1 format

## Configuration

Currier stores data in:

- `~/.currier/collections/` - Collection files
- `~/.currier/environments/` - Environment files
- `~/.currier/history.db` - Request history

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
