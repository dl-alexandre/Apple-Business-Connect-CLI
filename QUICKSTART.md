# Quick Start Guide

Get up and running with Apple Business Connect CLI in 5 minutes.

## Prerequisites

- macOS, Linux, or Windows
- Apple Business Connect API credentials (Client ID + Secret)
- Go 1.24+ (if building from source)

## Installation

### Option 1: Download Binary (Recommended)

```bash
# macOS (Intel)
curl -sL https://github.com/dl-alexandre/Apple-Business-Connect-CLI/releases/latest/download/abc-darwin-amd64 -o abc
chmod +x abc && sudo mv abc /usr/local/bin/

# macOS (Apple Silicon)
curl -sL https://github.com/dl-alexandre/Apple-Business-Connect-CLI/releases/latest/download/abc-darwin-arm64 -o abc
chmod +x abc && sudo mv abc /usr/local/bin/

# Linux
curl -sL https://github.com/dl-alexandre/Apple-Business-Connect-CLI/releases/latest/download/abc-linux-amd64 -o abc
chmod +x abc && sudo mv abc /usr/local/bin/
```

### Option 2: Build from Source

```bash
git clone https://github.com/dl-alexandre/Apple-Business-Connect-CLI.git
cd Apple-Business-Connect-CLI
make build
sudo mv bin/abc /usr/local/bin/
```

### Option 3: Go Install

```bash
go install github.com/dl-alexandre/abc/cmd/abc@latest
```

## First-Time Setup

Run the interactive setup wizard:

```bash
abc setup
```

Or manually configure credentials:

```bash
# Store credentials securely in OS Keyring
abc auth login
# Enter Client ID: your-client-id
# Enter Client Secret: your-client-secret

# Verify authentication
abc auth status
```

## 5-Minute Tutorial

### 1. Check Your Account Status (30 seconds)

```bash
abc status
```

Output:
```
📊 Apple Business Connect Status Dashboard
════════════════════════════════════════════════════════════

📍 LOCATIONS
  Total:     48
  ✅ Verified: 45
  ⏳ Pending:  2
  ❌ Rejected: 1

  ⚠️  1 location(s) need attention

🎭 SHOWCASES
  ℹ️  Run 'abc showcases list <location-id>' for showcase status

📧 BRANDED MAIL
  ℹ️  Run 'abc mail check <domain>' for domain verification status

════════════════════════════════════════════════════════════
Last updated: 2026-03-03 15:30:22
```

### 2. Validate Your Branded Mail Setup (1 minute)

Before submitting to Apple, verify your DNS is ready:

```bash
abc mail check example.com
```

This checks:
- ✅ DMARC policy (`p=quarantine` or `p=reject` required)
- ✅ DKIM alignment (mandatory for Apple)
- ✅ SPF configuration (recommended)

### 3. Validate Your BIMI Logo (1 minute)

Ensure your SVG logo is BIMI-compliant:

```bash
abc bimi validate logo.svg
```

Validates:
- ✅ SVG Tiny-PS profile compliance
- ✅ Square aspect ratio (1:1)
- ✅ No scripts or external references
- ✅ Proper dimensions (32x32 minimum recommended)

### 4. Preview Location Changes (1 minute)

Create a CSV file with your locations:

```csv
partner_id,name,street,city,region,postal_code,country,phone
SF001,San Francisco Store,123 Market St,San Francisco,CA,94105,US,+1-415-555-0100
LA001,Los Angeles Store,456 Sunset Blvd,Los Angeles,CA,90028,US,+1-213-555-0200
```

Dry-run to preview changes:

```bash
abc locations sync locations.csv --dry-run
```

Output:
```
Found 2 location(s) in file

🔍 Running pre-flight validation...
✅ All 2 records passed validation

Found 48 existing location(s) in Apple Business Connect

[NEW] 2 location(s) to be created:
  + San Francisco Store (San Francisco, CA)
  + Los Angeles Store (Los Angeles, CA)

(Dry-run mode: no changes were made)
```

### 5. Apply Changes Safely (1 minute)

With blast radius protection:

```bash
abc locations sync locations.csv --max-deletes 0 --confirm
```

This prevents accidental deletions and applies the changes.

## Common Workflows

### Bulk Location Management

```bash
# Export current locations
abc locations list --format json > current-locations.json

# Edit the file, then sync with dry-run
abc locations sync updated-locations.json --dry-run

# Apply with conservative rate limiting for large batches
abc locations sync updated-locations.json --workers 3 --rate-ms 200 --confirm
```

### Dynamic Showcases (Marketing)

Create `promo.yaml`:

```yaml
showcases:
  - name: spring_sale
    type: OFFER
    title: "Spring Sale at {{.City}}!"
    description: "Visit our {{.City}} location for 20% off!"
    start_date: "2024-04-01"
    end_date: "2024-04-30"
    action_link:
      title: "Shop Now"
      url: "https://example.com/sale?store={{.PartnerID}}"
```

Sync to all locations:

```bash
abc showcases sync promo.yaml --data locations.csv --dry-run
abc showcases sync promo.yaml --data locations.csv --confirm
```

### CI/CD Integration

Add to your GitHub Actions workflow:

```yaml
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

## Troubleshooting

### Check CLI Health

```bash
abc doctor
```

Verifies:
- API connectivity
- Authentication status
- Configuration
- Cache settings
- OS Keyring access

### Debug Issues

```bash
# Verbose output
abc locations list --verbose

# Debug mode (shows API requests)
abc locations list --debug

# JSON output for scripting
abc locations list --format json | jq '.locations[] | {name, id}'
```

### Common Errors

**"No credentials found in keyring"**
```bash
abc auth login
```

**"Cannot connect to API"**
- Check your network connection
- Verify API credentials at https://businessconnect.apple.com
- Ensure firewall allows HTTPS to api.businessconnect.apple.com

**"Validation failed"**
- Check CSV format (required headers: name, street, city, region, postal_code, country)
- Validate coordinates are within valid ranges
- Ensure phone numbers are in E.164 format (+1-415-555-0100)

## Next Steps

1. **Explore Examples**: Check `examples/` directory for CSV, JSON, and YAML templates
2. **Read Full Docs**: See [README.md](README.md) for complete command reference
3. **GitHub Actions**: Copy workflows from `.github/workflows/examples/`
4. **Contribute**: Submit issues and PRs at https://github.com/dl-alexandre/Apple-Business-Connect-CLI

## Command Reference Cheat Sheet

```bash
# Authentication
abc auth login                    # Store credentials
abc auth logout                   # Remove credentials
abc auth status                   # Check auth status

# Locations
abc locations list                # List all locations
abc locations get <id>            # Get location details
abc locations sync file.csv       # Bulk import/update
abc locations create "Name" ...   # Create single location

# Showcases (Promotions)
abc showcases list <location-id>  # List showcases
abc showcases sync template.yaml  # Bulk create from template

# Branded Mail & BIMI
abc mail check domain.com         # Validate DNS (DMARC/DKIM)
abc bimi validate logo.svg        # Validate SVG logo
abc bimi check domain.com         # Check BIMI DNS records

# Monitoring
abc status                        # Account dashboard
abc doctor                        # Health check
abc insights get <location-id>    # View analytics

# Help
abc --help                        # Show all commands
abc <command> --help              # Show command help
```

---

**Ready to automate your Apple Business Connect workflow?** 
Start with `abc setup` and you'll be managing locations like a pro in minutes!
