# abc

[![Go Report Card](https://goreportcard.com/badge/github.com/dl-alexandre/abc)](https://goreportcard.com/report/github.com/dl-alexandre/abc)
[![CI](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/actions/workflows/ci.yml/badge.svg)](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go%20version-%3E=1.24-61CFDD.svg)](https://golang.org/)
[![Latest Release](https://img.shields.io/github/v/release/dl-alexandre/Apple-Business-Connect-CLI)](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/releases)
[![Code of Conduct](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

CLI for Apple Business Connect API v3.0

Manage your business presence on Apple Maps, Wallet, and Siri with this command-line tool. Built with Go 1.24 using the production-grade CLI template.

## Features

- **Apple Business Connect API v3.0**: Full support for Locations, Showcases, and Insights
- **OAuth2 Authentication**: Secure client credentials flow
- **Modern CLI Framework**: Built with [Kong](https://github.com/alecthomas/kong) for declarative, struct-based commands
- **Multiple Output Formats**: Table, JSON, and markdown output
- **Flexible Configuration**: Via config files, environment variables, or flags
- **Caching Layer**: Built-in file-based caching with TTL
- **Cross-Platform**: Linux, macOS, and Windows (AMD64 and ARM64)
- **Shell Completions**: Bash, Zsh, Fish, and PowerShell
- **Release Automation**: GoReleaser for automated releases

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap dl-alexandre/abc
brew install abc
```

### Go Install

```bash
go install github.com/dl-alexandre/abc/cmd/abc@latest
```

### Binary Download

Download from the [releases page](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/releases).

## Quick Start

### 1. Set Up Credentials

Get your OAuth2 credentials from the [Apple Business Connect portal](https://businessconnect.apple.com):

```bash
# Option 1: Environment variables
export ABC_API_CLIENT_ID=your-client-id
export ABC_API_CLIENT_SECRET=your-client-secret

# Option 2: Config file
cat > ~/.config/abc/config.yaml << EOF
api:
  client_id: your-client-id
  client_secret: your-client-secret
EOF
```

### 2. Example Commands

```bash
# List all locations
abc locations list

# Get a specific location
abc locations get <location-id>

# Create a new location
abc locations create "My Store" \
  --street "123 Main St" \
  --city "San Francisco" \
  --region "CA" \
  --postal-code "94102" \
  --country "US" \
  --phone "+1-555-123-4567"

# List showcases for a location
abc showcases list <location-id>

# Create a showcase
abc showcases create <location-id> "Summer Sale" \
  --description "20% off everything" \
  --type OFFER

# Get insights for a location
abc insights get <location-id> --period MONTH
```

## Configuration

Configuration sources (in order of priority):

1. **Command-line flags**: `--client-id`, `--client-secret`, etc.
2. **Environment variables**: `ABC_API_CLIENT_ID`, `ABC_API_CLIENT_SECRET`, etc.
3. **Config file**: `~/.config/abc/config.yaml`

### Example Config File

```yaml
api:
  url: "https://api.businessconnect.apple.com/v3.0"
  timeout: 30
  client_id: "your-client-id"
  client_secret: "your-client-secret"

cache:
  enabled: true
  ttl: 60
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ABC_API_CLIENT_ID` | OAuth2 client ID | - |
| `ABC_API_CLIENT_SECRET` | OAuth2 client secret | - |
| `ABC_API_URL` | API base URL | `https://api.businessconnect.apple.com/v3.0` |
| `ABC_TIMEOUT` | Request timeout (seconds) | `30` |
| `ABC_FORMAT` | Default output format | `table` |

## Commands

### Locations

```bash
abc locations list [flags]          # List all locations
abc locations get <id> [flags]      # Get location by ID
abc locations create <name> [flags] # Create new location
abc locations update <id> [flags]   # Update location
abc locations delete <id> --confirm # Delete location
abc locations sync <file> [flags]   # Sync locations from CSV/JSON
```

#### Bulk Sync (CSV/JSON)

Import or update multiple locations at once:

```bash
# Preview changes without applying
abc locations sync locations.csv --dry-run

# Sync with confirmation
abc locations sync locations.csv

# Force sync without confirmation
abc locations sync locations.csv --confirm

# Adjust concurrency (default: 5 workers, 100ms rate limit)
abc locations sync locations.csv --workers 3 --rate-ms 200
```

**Rate Limiting**: Apple Business Connect API has rate limits. The sync command includes built-in protection:
- **Default**: 5 concurrent workers with 100ms between requests (max ~10 req/sec)
- **Conservative**: Use `--workers 3 --rate-ms 200` for large batches (500+ locations)
- **Aggressive**: Use `--workers 10 --rate-ms 50` for small batches (careful with rate limits)

### Showcases

```bash
abc showcases list <location-id> [flags]          # List showcases
abc showcases get <location-id> <showcase-id>     # Get showcase
abc showcases create <location-id> <title> [flags] # Create showcase
abc showcases update <location-id> <showcase-id> [flags]
abc showcases delete <location-id> <showcase-id> --confirm
```

### Authentication

Securely store credentials in your OS keyring:

```bash
# Store credentials (prompts for input)
abc auth login

# Or provide via environment/flags
ABC_API_CLIENT_ID=xxx ABC_API_CLIENT_SECRET=yyy abc auth login

# Check status
abc auth status

# Remove stored credentials
abc auth logout
```

Credential resolution order: CLI flags → Environment variables → OS keyring

### Insights

```bash
abc insights get <location-id> [flags]  # Get location insights
  --period=MONTH                        # DAY, WEEK, or MONTH
  --start-date=2024-01-01
  --end-date=2024-01-31
```

## Output Formats

```bash
# Default table format
abc locations list

# JSON output
abc locations list --format json

# Markdown output
abc locations get <id> --format markdown

# Override global default for a single command
abc locations list --output-format json
```

## Development

### Prerequisites

- Go 1.24 or later
- golangci-lint
- GoReleaser (optional, for releases)

### Building

```bash
# Build for current platform
go build -o abc ./cmd/abc

# Run tests
go test ./...

# Run linter
golangci-lint run

# Install locally
go install ./cmd/abc
```

### Make Targets

```bash
make build          # Build for current platform
make test           # Run tests
make lint           # Run linter
make format         # Format code
make release        # Build optimized release
make clean          # Clean build artifacts
```

## CI/CD Integration

### GitHub Actions

Automatically sync locations when changes are merged:

```yaml
# .github/workflows/abc-sync.yml
name: Sync Apple Business Connect

on:
  push:
    branches: [ main ]
    paths: [ 'locations.csv' ]

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install abc CLI
        run: |
          curl -sL https://github.com/dl-alexandre/Apple-Business-Connect-CLI/releases/latest/download/abc-linux-amd64 -o abc
          chmod +x abc && sudo mv abc /usr/local/bin/
      
      - name: Sync locations
        env:
          ABC_API_CLIENT_ID: ${{ secrets.ABC_API_CLIENT_ID }}
          ABC_API_CLIENT_SECRET: ${{ secrets.ABC_API_CLIENT_SECRET }}
        run: abc locations sync locations.csv --confirm --workers 3
```

See `.github/workflows/examples/` for complete workflows including:
- **Dry-run previews** on Pull Requests
- **Validation gates** before sync
- **Rate limiting** for large batches

### Environment Variables in CI

Store credentials as repository secrets:
- `ABC_API_CLIENT_ID` - Your OAuth2 client ID
- `ABC_API_CLIENT_SECRET` - Your OAuth2 client secret

## Shell Completions

### Bash

```bash
abc completion bash > /usr/local/etc/bash_completion.d/abc
```

### Zsh

```bash
abc completion zsh > "${fpath[1]}/_abc"
```

### Fish

```bash
abc completion fish > ~/.config/fish/completions/abc.fish
```

### PowerShell

```powershell
abc completion powershell | Out-String | Invoke-Expression
```

## Project Structure

```
.
├── cmd/abc/              # Entry point
├── internal/
│   ├── cli/              # CLI command definitions
│   ├── api/              # HTTP client with OAuth2
│   ├── config/           # Configuration management
│   ├── output/           # Output formatters
│   └── cache/            # Caching layer
├── .github/workflows/    # CI/CD
├── Makefile              # Build automation
├── .goreleaser.yml       # Release config
└── config.example.yaml   # Example config
```

## API Reference

This CLI implements the [Apple Business Connect API v3.0](https://businessconnect.apple.com/docs/data-specification/v3.0/introduction):

- **Locations**: Create, read, update, delete business locations
- **Showcases**: Manage promotional content (events, offers)
- **Insights**: View analytics and metrics

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests and linting (`make test && make lint`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

Read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE)

## Acknowledgments

- [Kong](https://github.com/alecthomas/kong) - CLI framework
- [resty](https://github.com/go-resty/resty) - HTTP client
- [Viper](https://github.com/spf13/viper) - Configuration
- [rodaine/table](https://github.com/rodaine/table) - Table formatting
- Original template by [go-cli-template](https://github.com/dl-alexandre/go-cli-template)
