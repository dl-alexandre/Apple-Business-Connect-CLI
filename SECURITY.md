# Security Policy

## Supported Versions

We release security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| 0.1.x   | :x:                |
| < 0.1.0 | :x:                |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in the Apple Business Connect CLI, please report it to us as soon as possible.

### How to Report

**Please DO NOT** create a public GitHub issue for security vulnerabilities.

Instead, please report security issues privately by:

1. **Email**: [Contact the maintainers directly - add your security contact email here]
2. **GitHub Security Advisory**: Use [GitHub Security Advisories](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/security/advisories) to report privately

### What to Include

When reporting a vulnerability, please include:

- **Description**: Clear description of the vulnerability
- **Impact**: What could an attacker accomplish?
- **Steps to Reproduce**: Detailed steps to reproduce the issue
- **Affected Versions**: Which versions are affected
- **Environment**: OS, Go version, CLI version
- **Proof of Concept**: If possible, include code or commands that demonstrate the vulnerability
- **Mitigation**: Any suggested fixes or workarounds

### Response Timeline

We aim to respond to security reports within:

- **24 hours**: Acknowledgment of receipt
- **72 hours**: Initial assessment and response plan
- **7 days**: Fix implemented (for critical issues)
- **14 days**: Fix released and advisory published

## Security Best Practices for Users

### Credential Management

1. **Never commit credentials to version control**
   - Use environment variables or OS keyring
   - Never hardcode API keys in scripts

2. **Use the OS keyring**
   ```bash
   abc auth login  # Stores securely in macOS Keychain, Windows Credential Manager, or Linux secret service
   ```

3. **Rotate credentials regularly**
   - Update your Apple Business Connect OAuth2 credentials periodically
   - Revoke old credentials after rotation

4. **Use minimal permissions**
   - Create API credentials with only the permissions needed
   - Use separate credentials for different environments (dev/staging/prod)

### Configuration Security

1. **Secure your config files**
   ```bash
   chmod 600 ~/.config/abc/config.yaml
   ```

2. **Use environment variables in CI/CD**
   - Store `ABC_API_CLIENT_ID` and `ABC_API_CLIENT_SECRET` as repository secrets
   - Never echo or log these values

3. **Audit your setup**
   ```bash
   abc doctor  # Check for security misconfigurations
   ```

### Network Security

1. **Use HTTPS only**
   - The CLI defaults to HTTPS for all API calls
   - Never override to use HTTP in production

2. **Verify SSL/TLS**
   - The CLI validates certificates by default
   - Only disable verification (`--insecure`) for testing in controlled environments

3. **Be cautious with proxies**
   - If using a proxy, ensure it's trusted
   - Monitor for MITM attacks in corporate environments

### Data Handling

1. **Protect exported data**
   - Insights and location data may contain sensitive business information
   - Secure CSV/JSON export files with appropriate permissions

2. **Audit access logs**
   - Monitor who has access to the CLI and when it's used
   - Review Apple Business Connect portal audit logs regularly

## Security Features in the CLI

### Built-in Protections

- **OAuth2 Client Credentials Flow**: Secure authentication without password exposure
- **Token Encryption**: Credentials stored in OS keyring are encrypted
- **Request Timeouts**: Prevents hanging connections (default: 30s)
- **Rate Limiting**: Built-in protection against API throttling
- **Audit Logging**: Verbose mode shows requests without exposing credentials

### Security-Related Commands

```bash
# Check security configuration
abc doctor

# Securely store credentials
abc auth login

# Remove stored credentials
abc auth logout

# Verify authentication status
abc auth status
```

## Known Security Considerations

### Limitations

1. **Config file permissions**: The CLI cannot enforce file permissions across all platforms. Users must manually ensure `~/.config/abc/config.yaml` is readable only by the owner.

2. **Shell history**: Commands with flags may appear in shell history. Use environment variables or the interactive `auth login` command to avoid this.

3. **Memory dumps**: While credentials are encrypted at rest, they exist in memory during execution. Secure your workstation against memory analysis attacks.

### Dependencies

We regularly audit our dependencies for security vulnerabilities:

- Run `go list -m all` to see all dependencies
- Check [Go Vulnerability Database](https://pkg.go.dev/vuln/) for known issues
- Automated Dependabot alerts are enabled

## Security Updates

Security updates are released as patch versions (e.g., 0.2.1) and include:

- Detailed changelog entry (without vulnerability details until after disclosure period)
- GitHub Security Advisory with CVE (if applicable)
- Update notification in the CLI

To stay updated:

```bash
# Check for updates
abc check-update

# Enable update notifications (future feature)
# abc config set check_updates true
```

## Acknowledgments

We thank the following security researchers and users who have responsibly disclosed vulnerabilities:

- *(No disclosed vulnerabilities yet - this is a new project)*

## License

This security policy is provided under the same license as the project (MIT).
