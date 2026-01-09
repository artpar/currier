# Currier

A vim-modal TUI API client for developers and AI agents.

## Features

- **Vim-style keybindings** - Navigate and edit with familiar modal controls
- **Collections & Environments** - Organize requests with Postman-like collections
- **Environment Switcher** - Press `V` to switch between environments on the fly
- **Variable interpolation** - Use `{{variable}}` syntax in URLs, headers, and bodies
- **Automatic Cookie Management** - Captures Set-Cookie headers and persists cookies to SQLite
- **Pre/Post-request scripts** - JavaScript-based scripting with assertions
- **Request history** - SQLite-backed history with search and replay
- **Import/Export** - Support for Postman, cURL, HAR, and OpenAPI formats
- **CLI mode** - Execute requests directly from the command line
- **curl import** - Run `currier curl <args>` to import any curl command into the TUI
- **Collection Runner** - Batch execute all requests in a collection with test results
- **Form-data / File Upload** - Multipart form-data body type with file upload support
- **Proxy Support** - HTTP, HTTPS, and SOCKS5 proxy configuration
- **Client Certificates** - mTLS support with custom CA certificates
- **Traffic Capture** - HTTP proxy to capture and inspect traffic from any application
- **MCP Server** - AI assistant integration via Model Context Protocol (32 tools)

## Demos

### Overview - Three-Pane Layout
![Overview Demo](demos/demo-overview.gif?v=0.1.37)
*Navigate the three-pane interface: Collections/History on left, Request editor in center, Response viewer on right. Use H/C to switch views, Tab or 1/2/3 to jump between panes, ? for help overlay.*

### Creating & Sending Requests
![Request Demo](demos/demo-request.gif?v=0.1.37)
*Complete workflow: Create a GET request, send it, view the JSON response. Then create a POST request with JSON body, send it, and save to a collection.*

### Request History
![History Demo](demos/demo-history.gif?v=0.1.37)
*Browse your request history with vim-style navigation (j/k/G/gg). Select any past request to reload it, replay it to get a fresh response, and see the new entry added to history.*

### Editing Requests
![Editing Demo](demos/demo-editing.gif?v=0.1.37)
*Build a complete request: Add custom headers (X-Custom-Header), query parameters (?search=currier&limit=10), and a JSON body. Send to httpbin.org/anything which echoes everything back.*

### Viewing Responses
![Response Demo](demos/demo-response.gif?v=0.1.37)
*Explore response data across tabs: Body (scrollable JSON), Headers (server response headers), and Cookies. Copy response with 'y'. See cookies that were set by the server.*

### Search
![Search Demo](demos/demo-search.gif?v=0.1.37)
*Filter history or collections with '/'. Type a query to see matching results, navigate with j/k, press Enter to load the request. Escape clears the filter.*

### Environment Variables
![Environment Demo](demos/demo-environment.gif?v=0.1.37)
*Use {{variables}} in URLs. Press V to switch environments (dev/staging/prod). Send the same request to different hosts by changing environment. See the actual URL resolved in the response.*

### Cookie Management
![Cookie Demo](demos/demo-cookies.gif?v=0.1.37)
*Set a cookie via httpbin, then VIEW it in the Cookies tab. Make another request to verify cookies are automatically sent. Clear all cookies with Ctrl+K and confirm they're gone.*

### Proxy Settings
![Proxy Demo](demos/demo-proxy.gif?v=0.1.37)
*Configure an HTTP proxy with P. All requests route through the proxy. Clear the proxy URL to disable. Useful for debugging with mitmproxy, Charles, or Fiddler.*

### TLS/Certificate Settings
![TLS Demo](demos/demo-tls.gif?v=0.1.37)
*Configure mTLS: Set client certificate, private key, and CA certificate paths. Toggle "Skip TLS Verification" for self-signed certs. Settings persist across sessions.*

### Collection Runner
![Runner Demo](demos/demo-runner.gif?v=0.1.37)
*Run all requests in a collection with Ctrl+R. Watch progress as each request executes. View pass/fail results, response times, and test assertion outcomes.*

### Traffic Capture (HTTP & HTTPS)
![Capture Demo](demos/demo-capture.gif?v=0.1.49)
*Capture HTTP and HTTPS traffic from any application. Start with `currier --capture`, route traffic through the proxy with `-k` flag for HTTPS. Captured HTTPS requests show a 🔒 lock icon. Filter by method with m, inspect any request with Enter.*

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

### Linux (Debian/Ubuntu - APT Repository)

```bash
# Add the APT repository
echo "deb [trusted=yes] https://raw.githubusercontent.com/artpar/apt-repo/main stable main" | sudo tee /etc/apt/sources.list.d/artpar.list
sudo apt update
sudo apt install currier
```

Or download directly:

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
currier send GET https://api.example.com/users

# Send a POST request with JSON body
currier send POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John"}'

# Use an environment
currier send GET https://{{host}}/api/users -e production.json

# Use a proxy
currier send GET https://api.example.com/users --proxy http://localhost:8080

# Use client certificates (mTLS)
currier send GET https://api.example.com/secure \
  --cert client.pem --key client-key.pem

# Skip TLS verification (insecure)
currier send GET https://self-signed.example.com -k
```

### MCP Server (AI Assistant Integration)

Currier includes an MCP (Model Context Protocol) server that enables AI assistants like Claude to use Currier for API testing and development.

```bash
# Start MCP server
currier mcp
```

#### Configure with Claude Code

Add to your Claude Code MCP settings (`~/.claude.json` or project `.mcp.json`):

```json
{
  "mcpServers": {
    "currier": {
      "command": "currier",
      "args": ["mcp"]
    }
  }
}
```

#### Available MCP Tools (32 tools)

| Category | Tools |
|----------|-------|
| **Requests** | `send_request`, `send_curl` |
| **Collections** | `list_collections`, `get_collection`, `create_collection`, `delete_collection`, `rename_collection` |
| **Requests CRUD** | `get_request`, `save_request`, `update_request`, `delete_request` |
| **Folders** | `create_folder`, `delete_folder` |
| **Environments** | `list_environments`, `get_environment`, `create_environment`, `delete_environment`, `set_environment_variable`, `delete_environment_variable` |
| **History** | `get_history`, `search_history` |
| **Cookies** | `list_cookies`, `clear_cookies` |
| **Import/Export** | `import_collection`, `export_collection`, `export_as_curl` |
| **Runner** | `run_collection` |
| **WebSocket** | `websocket_connect`, `websocket_disconnect`, `websocket_send`, `websocket_list_connections`, `websocket_get_messages` |

#### Example AI Workflows

```
User: "Test the /users endpoint"
Claude: [calls send_request with GET /users]

User: "Run the payment API tests"
Claude: [calls run_collection with "Payment API"]

User: "Import this OpenAPI spec and test all endpoints"
Claude: [calls import_collection, then run_collection]

User: "Create a new collection for the auth endpoints"
Claude: [calls create_collection, then save_request for each endpoint]

User: "Connect to the WebSocket at wss://example.com/ws and send a ping"
Claude: [calls websocket_connect, then websocket_send with message]

User: "Show me the messages from the WebSocket connection"
Claude: [calls websocket_get_messages to retrieve buffered messages]
```

#### MCP Resources

| Resource | Description |
|----------|-------------|
| `collections://list` | List of all collections |
| `history://recent` | Recent request history |

### Collection Runner

Run all requests in a collection sequentially:

```bash
# Run a collection
currier run my-collection.json

# With environment
currier run my-collection.json -e production.json

# Verbose output (shows each request)
currier run my-collection.json -v

# JSON output for CI/CD
currier run my-collection.json --json
```

Output example:
```
Running collection: My API
✓ GET Get Users (234ms) - 3/3 tests
✓ POST Create User (156ms) - 2/2 tests
✗ GET Get User 999 (89ms) - 1/2 tests
  ✗ Status should be 404
    Expected 404 to be 500

Summary:
  Requests: 2/3 passed
  Tests: 6/7 passed
  Total time: 479ms
```

### Traffic Capture (HTTP & HTTPS)

Capture HTTP and HTTPS traffic from any application:

```bash
# Start Currier directly in capture mode
currier --capture
```

Or manually: Press `C` twice to enter Capture mode, then `p` to start the proxy.

#### Capturing Traffic

```bash
# HTTP traffic - works directly
curl --proxy http://localhost:PORT http://httpbin.org/get

# HTTPS traffic - use -k to skip certificate verification
curl --proxy http://localhost:PORT -k https://httpbin.org/get

# Set as environment variable for all requests
export http_proxy=http://localhost:PORT
export https_proxy=http://localhost:PORT
curl -k https://api.example.com/users
```

**Note:** HTTPS requests show a 🔒 lock icon in the capture list. The `-k` flag tells curl to accept Currier's auto-generated CA certificate.

#### Capture Mode Shortcuts

| Key | Action |
|-----|--------|
| `p` | Start/Stop proxy |
| `j/k` | Navigate captures |
| `Enter` | Load capture into request panel |
| `m` | Cycle method filter (ALL → GET → POST → ...) |
| `x` | Clear method filter |
| `X` | Clear all captures |
| `H` | Switch to History mode |

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
| `H/C` | Switch to History/cycle Collections/Capture |
| `n` | Create new request |
| `s` | Save request to collection |
| `w` | Toggle WebSocket mode |
| `V` | Switch environment |
| `P` | Proxy settings |
| `Ctrl+T` | TLS/certificate settings |
| `Ctrl+R` | Run collection |
| `Ctrl+K` | Clear all cookies |
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
| `y` | Duplicate request/folder |
| `c` | Copy request as cURL |
| `E` | Export collection to Postman |
| `I` | Import collection (Postman/OpenAPI) |
| `K/J` | Move request up/down |
| `R` | Rename request/folder |
| `/` | Search |
| `H` | Switch to History |

### Capture Panel
| Key | Action |
|-----|--------|
| `p` | Start/Stop proxy |
| `j/k` | Navigate captures |
| `Enter` | Load capture into request |
| `m` | Cycle method filter |
| `x` | Clear method filter |
| `X` | Clear all captures |
| `/` | Search captures |
| `H` | Switch to History |

### Request Panel
| Key | Action |
|-----|--------|
| `e` | Edit URL / Edit field |
| `m` | Cycle HTTP method |
| `[/]` | Switch tabs |
| `Enter` | Send request |
| `Alt+Enter` | Send (while editing) |
| `t` | Cycle body type (Raw/JSON/Form) |
| `a` | Add header/query/form field |
| `f` | Add file field (form-data) |
| `d` | Delete field |
| `T` | Toggle field type (text/file) |

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
│   ├── cli/           # CLI commands (send, run, curl, mcp)
│   ├── cookies/       # Cookie jar with SQLite persistence
│   ├── core/          # Domain models (Request, Response, Collection)
│   ├── exporter/      # Export to cURL, Postman formats
│   ├── history/       # Request history storage
│   ├── importer/      # Import from Postman, cURL, HAR, OpenAPI
│   ├── interfaces/    # Interface definitions
│   ├── interpolate/   # Variable interpolation engine
│   ├── mcp/           # MCP server for AI assistant integration
│   ├── protocol/      # HTTP client (proxy, TLS, cookies)
│   ├── proxy/         # HTTP proxy server for traffic capture
│   ├── runner/        # Collection runner for batch execution
│   ├── script/        # JavaScript scripting engine
│   ├── storage/       # Collection/environment persistence
│   └── tui/           # Terminal UI components
├── e2e/               # End-to-end tests (Docker-based)
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

| Platform | Base Directory |
|----------|----------------|
| macOS | `~/Library/Application Support/currier/` |
| Linux | `~/.config/currier/` |
| Windows | `%AppData%\currier\` |

Within this directory:
- `collections/` - Collection files
- `environments/` - Environment files
- `history.db` - Request history
- `cookies.db` - Persistent cookie storage

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
