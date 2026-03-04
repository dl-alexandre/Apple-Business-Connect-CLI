# Support

Need help with the Apple Business Connect CLI? Here's how to get support:

## 📚 Documentation

Start with our comprehensive documentation:

- **[README.md](README.md)** - Overview, installation, and basic usage
- **[QUICKSTART.md](QUICKSTART.md)** - Step-by-step getting started guide
- **[SETUP_GUIDE.md](SETUP_GUIDE.md)** - Detailed configuration options
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - How to contribute to the project

## 🐛 Bug Reports

Found a bug? Please [open an issue](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/issues/new?template=bug_report.md) with:

- Clear description of the problem
- Steps to reproduce
- Your environment (OS, CLI version, Go version)
- Any error messages

## ✨ Feature Requests

Have an idea? [Request a feature](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/issues/new?template=feature_request.md) and tell us:

- What problem you're trying to solve
- How you envision it working
- Any examples or references

## 💬 Discussions

For questions, ideas, or general conversation:

- **[GitHub Discussions](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/discussions)** - Q&A, show and tell, general chat

## 🔒 Security Issues

**DO NOT** open public issues for security vulnerabilities.

Instead:
- See [SECURITY.md](SECURITY.md) for reporting procedures
- Use [GitHub Security Advisories](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/security/advisories) for private reporting

## 🛠️ Troubleshooting

### Common Issues

#### Authentication Problems
```bash
# Verify your setup
abc doctor

# Check auth status
abc auth status

# Re-authenticate if needed
abc auth logout && abc auth login
```

#### API Connection Issues
```bash
# Check connectivity
abc doctor

# Enable verbose output for debugging
abc <command> --verbose
```

#### Rate Limiting
The CLI includes built-in rate limiting, but if you're hitting limits:
- Use `--workers` and `--rate-ms` flags with bulk operations
- Check your Apple Business Connect API quotas

### Getting More Help

1. **Check existing issues**: Search [closed issues](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/issues?q=is%3Aissue+is%3Aclosed) for similar problems
2. **Run diagnostics**: Use `abc doctor` to check your setup
3. **Enable debug mode**: Use `--debug` flag for detailed output (be careful with credentials!)

## ⏰ Response Times

We aim to respond to:

| Type | Response Time |
|------|--------------|
| Security issues | 24 hours |
| Critical bugs | 48 hours |
| General issues | 5-7 days |
| Feature requests | 1-2 weeks |
| Discussions | When possible |

## 👥 Community

Join the community:

- ⭐ **Star the repo** if you find it useful
- 🍴 **Fork it** to contribute
- 🐦 **Share** your use cases

## 🏢 Enterprise Support

For enterprise users requiring:
- Priority support
- Custom features
- Training and onboarding
- SLA guarantees

Please contact: [Add your enterprise contact info here]

## 📝 Feedback

We love feedback! Tell us:
- What's working well?
- What's confusing?
- What features would make your life easier?

Open a [discussion](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/discussions) or [issue](https://github.com/dl-alexandre/Apple-Business-Connect-CLI/issues) anytime.

---

**Thank you for using Apple Business Connect CLI! 🎉**
