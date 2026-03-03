# Go CLI Template Summary

## Overview

A comprehensive Go CLI template using Kong, designed based on patterns from 9 existing CLI projects in the workspace. This template provides a production-ready starting point with working examples, comprehensive tooling, and easy project initialization.

## Key Features

### CLI Framework
- **Kong** - Struct-based, declarative CLI definition (dominant in workspace: 6/9 projects)
- Global flags with environment variable support
- Automatic help generation
- Shell completion support (bash, zsh, fish, powershell)

### Included Components

1. **Example Commands** (working implementations)
   - `list` - Paginated listing with limit/offset
   - `get` - Retrieve single resource by ID
   - `search` - Search with query parameter
   - `version` - Version information
   - `completion` - Shell completion generation

2. **HTTP Client** (`internal/api/`)
   - Resty-based with timeout support
   - Error types (APIError, NotFoundError, ValidationError)
   - Context support for cancellation
   - Generic Item/Metadata types

3. **Configuration** (`internal/config/`)
   - Viper-based with YAML support
   - Environment variable binding
   - Flag-based overrides
   - Platform-specific config directories

4. **Output Formatting** (`internal/output/`)
   - Table output (rodaine/table)
   - JSON output (standard library)
   - Markdown output
   - TTY-aware color support
   - Printer interface for consistent formatting

5. **Caching** (`internal/cache/`)
   - File-based cache with SHA256 keys
   - TTL support with automatic cleanup
   - Stats and cleanup methods

6. **Build System** (Makefile)
   - Cross-compilation (Linux/macOS/Windows, amd64/arm64)
   - Test with coverage and race detection
   - Linting with golangci-lint
   - Format with gofmt/goimports
   - Release builds with ldflags
   - Git hooks installation

7. **Release Automation** (.goreleaser.yml)
   - Multi-platform builds
   - Checksum generation
   - Changelog generation
   - Homebrew tap support (commented, ready to enable)
   - Scoop bucket support (commented, ready to enable)
   - Docker support (commented)

8. **CI/CD** (.github/workflows/)
   - Lint workflow with golangci-lint
   - Test matrix (Go 1.23, 1.24)
   - Build verification
   - Automated releases on tag push

9. **Quality Tools**
   - Pre-commit hooks (format, vet, test)
   - golangci-lint configuration
   - Comprehensive test coverage
   - Race detection

## Project Structure

```
go-cli-template/
├── cmd/abc/              # Entry point (renamed by setup)
│   └── main.go
├── internal/
│   ├── api/                      # HTTP client
│   │   ├── client.go
│   │   └── client_test.go
│   ├── cli/                      # CLI commands
│   │   ├── cli.go
│   │   └── cli_test.go
│   ├── config/                   # Configuration
│   │   ├── config.go
│   │   └── config_test.go
│   ├── output/                   # Formatters
│   │   └── formatter.go
│   └── cache/                    # Caching
│       └── cache.go
├── .github/
│   └── workflows/                # CI/CD
│       ├── ci.yml
│       └── release.yml
├── .githooks/
│   └── pre-commit               # Git hooks
├── scripts/
│   └── setup.sh                 # Project setup script
├── .golangci.yml               # Linter config
├── .goreleaser.yml              # Release config
├── Makefile                     # Build automation
├── config.example.yaml          # Example config
├── go.mod                       # Go module
├── README.md                    # Documentation
├── CONTRIBUTING.md              # Contribution guide
├── CHANGELOG.md               # Version history
├── LICENSE                    # MIT License
└── .gitignore                 # Git ignore patterns
```

## Template Placeholders

Files use these placeholders (replaced by `scripts/setup.sh`):

- `dl-alexandre` - GitHub/GitLab username or organization
- `abc` - Application name (lowercase, single word)
- `Apple-Business-Connect-CLI` - Repository name
- `CLI for Apple Business Connect API v3.0` - Project description

## Usage

### Option 1: GitHub "Use this template" Button (Recommended)

1. Go to https://github.com/dl-alexandre/go-cli-template
2. Click the green **"Use this template"** button
3. Create your new repository
4. Clone your new repo and run `./scripts/setup.sh`

### Option 2: Clone and Setup

If you clone the template directly, the setup script will detect the template's git history and offer to reset it:

```bash
git clone https://github.com/dl-alexandre/go-cli-template.git my-new-cli
cd my-new-cli
./scripts/setup.sh
# Setup will ask: "Remove template git history now? (recommended) (Y/n):"
# Select Y to run: rm -rf .git && git init
```

### After Setup

```bash
# Download dependencies
go mod download

# Install git hooks
make install-hooks

# Build and test
make build && make test
```

2. Run setup script:
   ```bash
   make setup
   # or
   ./scripts/setup.sh
   ```

3. Follow prompts to customize:
   - GitHub/GitLab username
   - Application name
   - Repository name
   - Description

4. Initialize and build:
   ```bash
   git init
   git add .
   git commit -m "Initial commit from template"
   git remote add origin https://github.com/USER/REPO.git
   go mod download
   make install-hooks
   make build
   ```

### Development Workflow

```bash
# Run all checks
make check

# Build and test
make build
make test

# Format and lint
make format
make lint

# Cross-compile
make build-all

# Dry-run release
make release-dry
```

### Customization Checklist

After setup, customize these for your API:

1. [ ] Update `internal/api/client.go` with your API endpoints
2. [ ] Modify `internal/cli/cli.go` with your commands
3. [ ] Update `internal/output/formatter.go` for your data types
4. [ ] Add commands to the CLI struct in `cli.go`
5. [ ] Uncomment Homebrew section in `.goreleaser.yml`
6. [ ] Uncomment Scoop section in `.goreleaser.yml` (optional)
7. [ ] Update `.github/workflows/*.yml` with your repo info
8. [ ] Add more tests in `*_test.go` files
9. [ ] Update README.md with actual usage examples
10. [ ] Create GitHub repository and push

## Technology Stack

| Component | Library | Purpose |
|-----------|---------|---------|
| CLI Framework | Kong v1.8.1 | Command-line parsing |
| HTTP Client | resty/v2 v2.15.0 | HTTP requests |
| Configuration | Viper v1.19.0 | Config management |
| Table Output | rodaine/table v1.3.0 | Table formatting |
| TTY Detection | go-isatty v0.0.20 | Terminal detection |
| Go Version | 1.24.0 | Minimum Go version |

## Comparison: Minimal vs Comprehensive

This is a **comprehensive** template with:

✅ Working example commands (list, get, search)  
✅ Complete HTTP client implementation  
✅ Caching layer  
✅ Output formatters  
✅ Configuration system  
✅ Tests for all packages  
✅ Setup script  
✅ CI/CD workflows  
✅ Release automation  
✅ Documentation templates  

**Minimal template** would include only:
- Basic Kong CLI structure
- Empty command stubs
- Makefile with basic targets
- No examples, caching, or output formatters

## Files Summary

| Category | Count | Files |
|----------|-------|-------|
| Source Code | 7 | main.go, cli.go, client.go, config.go, formatter.go, cache.go, go.mod |
| Tests | 3 | cli_test.go, client_test.go, config_test.go |
| Config | 4 | .golangci.yml, .goreleaser.yml, config.example.yaml, .gitignore |
| CI/CD | 2 | ci.yml, release.yml |
| Build | 2 | Makefile, setup.sh |
| Docs | 4 | README.md, CONTRIBUTING.md, CHANGELOG.md, LICENSE |
| Hooks | 1 | pre-commit |

**Total: 23 files**

## Next Steps

1. **Use the template** - Copy to a new project and run setup
2. **Customize the API client** - Replace example endpoints with your API
3. **Add your commands** - Extend the CLI struct with your functionality
4. **Set up CI/CD** - Configure GitHub Actions for your repo
5. **Configure releases** - Enable Homebrew/Scoop in .goreleaser.yml

## Based On

Patterns from 9 Go CLI projects in the workspace:
- grokipedia-cli (Kong, caching, multiple formats)
- UniFi-Site-Manager-CLI (GoReleaser, Homebrew, Scoop)
- Local-UniFi-CLI (Viper config)
- Google-Play-Developer-CLI (exit codes, testing)
- MyMarketNews-CLI (query patterns)
- Google-Drive-CLI (OAuth, protobuf)
- cimis-cli (standard lib)
- Apple-Map-Server-CLI (custom Command struct)
- App-StoreKit-CLI (Cobra - different pattern)

## License

MIT - See LICENSE file
