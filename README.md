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
- **curl import** - Run `currier curl <args>` to import any curl command into the TUI

## Demos

### Overview - Navigation & Layout
![Overview Demo](demos/demo-overview.gif?v=0.1.15)
*Stacked History/Collections • H/C to switch • Tab/1/2/3 for panes • ? for help*

### Creating & Sending Requests
![Request Demo](demos/demo-request.gif?v=0.1.15)
*n for new request • Type URL • Alt+Enter to send while editing*

### Request History
![History Demo](demos/demo-history.gif?v=0.1.15)
*H to focus history • j/k to navigate • Enter to load • Replay requests*

### Editing Requests
![Editing Demo](demos/demo-editing.gif?v=0.1.15)
*e to edit URL • [ ] to switch tabs • a to add headers • Tab between fields*

### Viewing Responses
![Response Demo](demos/demo-response.gif?v=0.1.15)
*j/k to scroll • G/gg for top/bottom • [ ] for tabs • y to copy*

### Search
![Search Demo](demos/demo-search.gif?v=0.1.15)
*/ to search • Filter history/collections • Enter to select*

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

### Import curl Commands

Import any curl command directly into the TUI - perfect for testing API examples from documentation:

```bash
# Simple GET request
currier curl https://httpbin.org/get

# POST with JSON body
currier curl -X POST https://httpbin.org/post \
  -H "Content-Type: application/json" \
  -d '{"name": "test", "value": 123}'

# With authentication
currier curl -u admin:secret https://api.example.com/protected

# Modern --json flag (sets Content-Type and Accept headers automatically)
currier curl --json '{"query": "search"}' https://api.example.com/search

# Copy curl from browser DevTools and run directly
currier curl -X POST 'https://api.example.com/endpoint' \
  -H 'Authorization: Bearer token123' \
  -H 'Content-Type: application/json' \
  --data-raw '{"key":"value"}'
```

**Supported curl options:**
- `-X, --request` - HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
- `-H, --header` - Custom headers
- `-d, --data` - Request body
- `--data-raw, --data-binary` - Raw request body
- `--json` - JSON body with automatic Content-Type
- `-u, --user` - Basic authentication
- `-A, --user-agent` - User-Agent header
- `-b, --cookie` - Cookie header
- `-e, --referer` - Referer header
- `-I, --head` - HEAD request
- `-L, --location` - Follow redirects (noted)
- `-k, --insecure` - Skip SSL verification (noted)
- `--compressed` - Accept compressed responses

## Keyboard Shortcuts

### Global
| Key | Action |
|-----|--------|
| `Tab` | Cycle between panes |
| `1/2/3` | Jump to pane |
| `n` | Create new request |
| `s` | Save request to collection |
| `w` | Toggle WebSocket mode |
| `?` | Show help |
| `q` | Quit |

### Collections Panel
| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `h/l` | Collapse/expand |
| `Enter` | Select request |
| `N` | Create new collection |
| `F` | Create new folder |
| `r` | Rename collection |
| `D` | Delete collection/folder |
| `d` | Delete request |
| `m` | Move request/folder |
| `y` | Duplicate/copy request |
| `R` | Rename request/folder |
| `/` | Search |
| `H` | Switch to History |

### Request Panel
| Key | Action |
|-----|--------|
| `e` | Edit URL |
| `m` | Cycle HTTP method |
| `[/]` | Switch tabs |
| `Enter` | Send request |
| `Alt+Enter` | Send (while editing) |

### Response Panel
| Key | Action |
|-----|--------|
| `j/k` | Scroll |
| `G/gg` | Top/bottom |
| `[/]` | Switch tabs |
| `y` | Copy response |

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
