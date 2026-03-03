# Post-Template Setup Guide

Congratulations on creating a project from the Go CLI Template! This guide covers the essential setup steps to get your CLI working with automated releases.

## Quick Verification

After running `./scripts/setup.sh`, verify everything works:

```bash
# 1. Build and test locally
make dev
make test

# 2. Test the CLI binary
./bin/abc --help
./bin/abc version

# 3. Generate shell completions
make completions
```

## GitHub Repository Setup

### 1. Create Your Repository

If you used the GitHub "Use this template" button, your repo is already created. Otherwise:

```bash
gh repo create abc --public --source=. --push
```

### 2. Enable Repository Settings

Go to **Settings** → **Actions** → **General** and ensure:

- ✅ **Workflow permissions**: Read and write permissions (for creating releases)
- ✅ **Allow GitHub Actions to create and approve pull requests** (optional, for Dependabot)

### 3. Secrets Configuration

**No additional secrets required** for basic releases! 

The workflow uses `secrets.GITHUB_TOKEN` which is automatically provided by GitHub Actions.

#### Optional: Homebrew Tap (Advanced)

To automatically publish to Homebrew when you release:

1. **Create the tap repository**:
   ```bash
   gh repo create homebrew-tap --public --description "Homebrew tap for abc"
   ```

2. **Create a Personal Access Token** (if needed for cross-repo pushes):
   - Go to **Settings** → **Developer settings** → **Personal access tokens** → **Tokens (classic)**
   - Generate new token with `repo` scope
   - Add to your repo: **Settings** → **Secrets and variables** → **Actions** → **New repository secret**
   - Name: `HOMEBREW_TAP_TOKEN`
   - Value: Your token

3. **Uncomment Homebrew in `.goreleaser.yml`**:
   ```yaml
   brews:
     - name: abc
       repository:
         owner: dl-alexandre
         name: homebrew-tap
         token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
   ```

## First Release

When you're ready to release:

```bash
# 1. Update CHANGELOG.md with your changes
# 2. Commit all changes
git add .
git commit -m "feat: initial release"

# 3. Tag the release
git tag -a v1.0.0 -m "Release v1.0.0"

# 4. Push (this triggers the release workflow)
git push origin main
git push origin v1.0.0
```

The GitHub Actions workflow will:
- Build binaries for Linux/macOS/Windows (amd64/arm64)
- Create a GitHub release with artifacts
- Generate checksums
- Build changelog from commit history
- Publish to Homebrew (if configured)

## Post-Release Verification

1. **Check the release**:
   - Go to **Releases** in your GitHub repo
   - Verify binaries are attached
   - Test the checksums: `sha256sum -c checksums.txt`

2. **Test installation**:
   ```bash
   # Direct download
   curl -L -o abc https://github.com/dl-alexandre/abc/releases/latest/download/abc-linux-amd64
   chmod +x abc
   ./abc version
   
   # Or via Homebrew (if configured)
   brew tap dl-alexandre/abc
   brew install abc
   ```

## pkg.go.dev Registration

To make your module discoverable:

1. Visit: `https://pkg.go.dev/github.com/dl-alexandre/abc`
2. Click **"Request indexing"**
3. This allows `go install github.com/dl-alexandre/abc/cmd/abc@latest` to work

## Dependabot

Your project already has Dependabot configured (`.github/dependabot.yml`). It will:
- Check for Go module updates weekly
- Check for GitHub Actions updates monthly
- Create PRs automatically

## Next Steps

1. **Customize the API client** (`internal/api/client.go`):
   - Replace the example endpoints with your actual API
   - Update request/response types

2. **Add your commands** (`internal/cli/cli.go`):
   - Add new command structs
   - Implement the `Run` methods

3. **Write tests** (`*_test.go` files):
   - Add unit tests for your commands
   - Add integration tests (with `//go:build integration` tag)

4. **Documentation**:
   - Update `README.md` with actual usage examples
   - Add examples to command help text
   - Document API authentication

5. **CI/CD**:
   - The template has cross-platform CI validation
   - Customize `.github/workflows/ci.yml` as needed
   - Uncomment Homebrew/Scoop in `.goreleaser.yml` when ready

## Troubleshooting

### Release workflow fails

**Problem**: "Permission denied" or "Resource not accessible"
**Solution**: Go to **Settings** → **Actions** → **General** → **Workflow permissions** and select "Read and write permissions"

### GoReleaser fails locally

```bash
# Dry-run release locally
goreleaser release --snapshot --clean --skip-publish
```

### Template still has placeholder values

**Problem**: Some files still contain `dl-alexandre` or `abc`
**Solution**: Manually replace remaining placeholders or run the setup script again

### Binary doesn't work on Windows

**Problem**: Path handling or TTY detection issues
**Solution**: The template already includes Windows-safe path handling in `config.go` and `cache.go`. If issues persist, check that you're using `filepath.Join()` instead of string concatenation.

## Support

- **Template issues**: https://github.com/dl-alexandre/go-cli-template/issues
- **Your project issues**: Create issues in your own repository
- **Kong docs**: https://github.com/alecthomas/kong
- **GoReleaser docs**: https://goreleaser.com
