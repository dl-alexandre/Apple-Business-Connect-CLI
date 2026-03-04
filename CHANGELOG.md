# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Visual Intelligence (Phase 11)**
  - `abc insights heatmap` - Generate engagement heatmap visualization (HTML and ASCII)
  - `abc insights compare` - A/B testing between locations with statistical analysis
  - Apple MapKit JS 5.7+ integration with Hybrid/Satellite toggle views
  
- **Doctor Command Enhancements**
  - Queue Sentinel - Detects when offline queue exceeds 100 items (API resilience indicator)
  - Showcase expiration alerts - Flags showcases expiring within 48 hours
  - Fixed nil pointer panic when API is unavailable

- **Final Boss Features**
  - `abc webhooks listen` - Local server for real-time Apple webhook callbacks
    - Slack and Discord notification support
    - Webhook signature verification
    - Simulated event types: showcase.approved, showcase.rejected, location.verified, location.denied
  
  - `abc audit` - Content quality checker
    - Photo dimension validation (480px minimum)
    - Phone format validation
    - CTA redirect detection (bit.ly, t.co, tinyurl, etc.)
    - Showcase description length validation
    - Opening hours completeness checks
    - `--strict` mode for additional warnings
  
  - `abc shell` - Interactive REPL mode
    - Command-line navigation for non-developers
    - Built-in help system
    - Quick access to locations, status, and insights

### Changed
- Enhanced HTML heatmap with modern CSS and responsive design
- Improved doctor command error handling

### Governance
- Added comprehensive issue templates (bug report, feature request)
- Added pull request template
- Added CODE_OF_CONDUCT.md based on Contributor Covenant 2.1
- Added SECURITY.md with vulnerability reporting process
- Added stale issue/PR automation workflow
- Added first-time contributor welcome workflow
- Enhanced README with CI badges and project statistics

## [0.2.0] - 2024-03-04

### Added
- Initial project structure with Kong CLI framework
- Example commands: list, get, search
- Multiple output formats (table, json, markdown)
- File-based caching with TTL
- Configuration via files, environment variables, and flags
- Shell completion support (bash, zsh, fish, powershell)
- Makefile with standard targets
- GitHub Actions CI/CD workflows
- GoReleaser configuration for releases
- golangci-lint configuration

## [1.0.0] - YYYY-MM-DD

### Added
- Initial release
- Basic API client with resty
- Configuration management with Viper
- Output formatting with rodaine/table
- Git hooks for pre-commit checks
- Comprehensive documentation

[Unreleased]: https://github.com/dl-alexandre/Apple-Business-Connect-CLI/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/dl-alexandre/Apple-Business-Connect-CLI/releases/tag/v0.2.0
[1.0.0]: https://github.com/dl-alexandre/Apple-Business-Connect-CLI/releases/tag/v1.0.0
