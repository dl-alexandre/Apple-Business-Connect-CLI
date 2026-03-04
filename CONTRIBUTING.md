# Contributing to abc

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to this project.

## Quick Links

- 📚 [Code of Conduct](CODE_OF_CONDUCT.md) - Our community standards
- 🔒 [Security Policy](SECURITY.md) - Reporting security vulnerabilities
- 📝 [Pull Request Template](.github/pull_request_template.md) - PR guidelines
- 🐛 [Issue Templates](.github/ISSUE_TEMPLATE/) - Bug reports & feature requests
- 📖 [Setup Guide](SETUP_GUIDE.md) - Detailed setup instructions

## Getting Started

### Prerequisites

- Go 1.24 or later
- Make
- golangci-lint (for code quality)
- GoReleaser (optional, for testing releases)

### Setting Up Your Environment

1. **Fork the repository** on GitHub

2. **Clone your fork:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/Apple-Business-Connect-CLI.git
   cd Apple-Business-Connect-CLI
   ```

3. **Install dependencies:**
   ```bash
   make deps
   ```

4. **Install git hooks:**
   ```bash
   make install-hooks
   ```

## Development Workflow

### Making Changes

1. **Create a new branch** for your feature or fix:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

2. **Make your changes** following our [Code Guidelines](#code-guidelines)

3. **Ensure all checks pass:**
   ```bash
   make check  # Runs format, vet, lint, and test
   ```

4. **Commit your changes** following [Conventional Commits](https://www.conventionalcommits.org/):
   ```bash
   git commit -m "feat: add new command for X"
   # or
   git commit -m "fix: resolve issue with Y"
   ```

5. **Push to your fork:**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Open a Pull Request** using our [PR template](.github/pull_request_template.md)

### Commit Message Format

We follow the Conventional Commits specification:

| Type | Description |
|------|-------------|
| `feat:` | New features |
| `fix:` | Bug fixes |
| `docs:` | Documentation changes |
| `style:` | Code style changes (formatting) |
| `refactor:` | Code refactoring |
| `perf:` | Performance improvements |
| `test:` | Adding or correcting tests |
| `chore:` | Changes to build process, dependencies |

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run integration tests (requires API access)
make test-integration

# Run linter
golangci-lint run

# Format code
make format
```

### Pre-commit Checks

The pre-commit hook automatically runs:
- `go fmt` - Format check
- `go vet` - Static analysis
- `go test -short` - Quick tests

If any check fails, the commit is blocked. Fix the issues and try again.

## Project Structure

```
.
├── cmd/abc/              # Entry point
├── internal/
│   ├── cli/              # CLI command definitions (Kong structs)
│   ├── api/              # HTTP client and API types
│   ├── config/           # Configuration management (Viper)
│   ├── output/           # Output formatters (table, json, markdown)
│   ├── auth/             # OAuth2 authentication
│   ├── cache/            # Caching layer
│   ├── queue/            # Offline operation queue
│   ├── showcase/         # Showcase management
│   └── validate/         # Input validation
├── .github/
│   ├── workflows/        # CI/CD workflows
│   ├── ISSUE_TEMPLATE/   # Issue templates
│   └── pull_request_template.md
├── scripts/              # Build and setup scripts
├── Makefile              # Build automation
└── README.md             # Documentation
```

## Code Guidelines

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go) and standard conventions
- Use meaningful variable names (e.g., `locationID` not `id`)
- Add comments for exported functions and types
- Keep functions focused and under 50 lines when possible
- Handle errors explicitly - never ignore errors
- Use context for cancellation and timeouts

### Adding New Commands

1. **Define the command struct** in `internal/cli/cli.go`:
   ```go
   type NewCmd struct {
       Arg1 string `arg:"" help:"Description"`
       Flag bool   `help:"Description"`
   }
   ```

2. **Implement the Run method:**
   ```go
   func (c *NewCmd) Run(globals *Globals) error {
       ctx := context.Background()
       // Implementation
       return nil
   }
   ```

3. **Register in the CLI struct:**
   ```go
   type CLI struct {
       // ...
       New NewCmd `cmd:"" help:"Description"`
   }
   ```

4. **Add tests** in appropriate `*_test.go` files

5. **Update documentation** in README.md and help text

### API Client Guidelines

- Use the `resty` client for HTTP requests
- Define clear request/response types in `internal/api/types.go`
- Handle errors with appropriate error types
- Support context for cancellation
- Add retries for transient failures
- Log at appropriate levels (avoid logging credentials)

### Output Format Guidelines

- Support table, JSON, and markdown formats
- Use the `output.Printer` for consistent formatting
- Respect the `--format` global flag
- Handle empty results gracefully with helpful messages
- Use color only when appropriate (respect `--no-color`)

## Testing

### Unit Tests

- Place tests in `*_test.go` files
- Use table-driven tests where appropriate
- Mock external dependencies using interfaces
- Test error cases, not just happy paths
- Aim for >80% coverage on new code

### Integration Tests

- Use the `integration` build tag: `//go:build integration`
- Test against a real API when possible
- Clean up resources after tests
- Mark expensive tests with `t.Skip()` for short runs

### Example Test

```go
func TestListCmd(t *testing.T) {
    cmd := &ListCmd{
        Limit: 10,
    }
    
    globals := &Globals{
        Format: "json",
    }
    
    // Mock client and test
    err := cmd.Run(globals)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}
```

## Documentation

- **README.md**: Update for user-facing changes
- **Help text**: Add examples to command help via struct tags
- **CHANGELOG.md**: Document changes in [Unreleased] section
- **Code comments**: Document exported functions and complex logic

## Release Process

1. Update CHANGELOG.md with version and date
2. Create a new tag:
   ```bash
   git tag -a v0.2.0 -m "Release version 0.2.0"
   git push origin v0.2.0
   ```
3. GitHub Actions automatically builds and releases via GoReleaser

## Community

### Communication Channels

- 🐛 **Bugs**: [Open an issue](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/issues/new?template=bug_report.md)
- ✨ **Features**: [Request a feature](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/issues/new?template=feature_request.md)
- 💬 **Questions**: [GitHub Discussions](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/discussions)
- 🔒 **Security**: See [SECURITY.md](SECURITY.md) for private reporting

### First Time Contributors

Welcome! Check out issues labeled:
- `good first issue` - Easy tasks to get started
- `help wanted` - Tasks where we need community help
- `documentation` - Docs improvements

We have a [welcome workflow](.github/workflows/welcome.yml) that will greet you on your first PR!

## Recognition

Contributors will be:
- Listed in release notes
- Mentioned in the README (for significant contributions)
- Added to our contributors graph

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to the maintainers.

## Questions?

- Check [SETUP_GUIDE.md](SETUP_GUIDE.md) for detailed setup
- Review [QUICKSTART.md](QUICKSTART.md) for usage examples
- Join [GitHub Discussions](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/discussions) for Q&A

---

**Thank you for contributing! 🎉**

Your contributions make this project better for everyone.
