---
name: 🐛 Bug Report
about: Report a bug to help us improve the CLI
title: "[BUG] "
labels: ["bug", "triage"]
assignees: []
---

## 🐛 Bug Description

<!-- A clear and concise description of what the bug is -->

## To Reproduce

Steps to reproduce the behavior:

1. Run command: `abc ...`
2. See error

## Expected Behavior

<!-- A clear and concise description of what you expected to happen -->

## Actual Output

<!-- Include any error messages, stack traces, or output -->

```bash
# Paste your command output here
```

## Environment

- **OS**: <!-- e.g., macOS 14.0, Ubuntu 22.04, Windows 11 -->
- **CLI Version**: <!-- Run `abc --version` -->
- **Go Version**: <!-- Run `go version` if building from source -->
- **Installation Method**: <!-- Homebrew, go install, binary download, source build -->

## Configuration

<!-- If relevant, include your config (remove sensitive info) -->

```yaml
# ~/.config/abc/config.yaml
api:
  url: "https://api.businessconnect.apple.com/v3.0"
  # DO NOT include client_id or client_secret
```

## Additional Context

<!-- Add any other context about the problem here -->

- Does this happen consistently or intermittently?
- Have you tried with `--verbose` or `--debug` flags?
- Are you behind a proxy or VPN?

## Checklist

- [ ] I've searched existing issues for similar bugs
- [ ] I've tried the latest version of the CLI
- [ ] I've included all relevant environment information
- [ ] I've removed sensitive credentials from any output
