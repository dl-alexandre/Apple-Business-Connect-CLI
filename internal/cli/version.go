package cli

import "github.com/dl-alexandre/cli-tools/version"

// Build-time variables (re-exported from cli-tools/version for compatibility).
var (
	Version    = version.Version
	BinaryName = version.BinaryName
	GitHubRepo = "dl-alexandre/Apple-Business-Connect-CLI"
	GitCommit  = version.GitCommit
	BuildTime  = version.BuildTime
)

func init() {
	version.BinaryName = "abc"
	BinaryName = version.BinaryName
}
