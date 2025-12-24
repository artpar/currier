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

## Installation

### From Source

```bash
git clone https://github.com/artpar/currier.git
cd currier
make build
```

### Pre-built Binaries

Download from the [Releases](https://github.com/artpar/currier/releases) page.

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
