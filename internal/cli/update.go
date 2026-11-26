package cli

import (
	"github.com/dl-alexandre/cli-tools/update"
	"github.com/dl-alexandre/cli-tools/version"
)

// UpdateCheckCmd wraps the shared cli-tools update checker.
type UpdateCheckCmd struct {
	Force bool `help:"Force check, bypassing cache" flag:"force"`
}

// UpdateInfo is re-exported for compatibility with callers of the CLI package.
type UpdateInfo = update.Info

func updateChecker() *update.Checker {
	return update.New(update.Config{
		CurrentVersion: version.Version,
		BinaryName:     version.BinaryName,
		GitHubRepo:     GitHubRepo,
		InstallCommand: "brew upgrade abc",
	})
}

// Run executes the update check.
func (c *UpdateCheckCmd) Run(globals *Globals) error {
	info, err := updateChecker().Check(c.Force)
	if err != nil {
		return err
	}
	return update.DisplayUpdate(info, version.BinaryName, globals.Format)
}

// AutoUpdateCheck starts the shared non-blocking update check.
func AutoUpdateCheck(_ interface{}) {
	updateChecker().AutoCheck()
}
