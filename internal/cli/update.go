package cli

import (
	"github.com/dl-alexandre/cli-tools/update"
	"github.com/dl-alexandre/cli-tools/version"
)

// AutoUpdateCheck performs a background update check (for use at startup)
// It returns immediately and doesn't block
func AutoUpdateCheck() {
	checker := update.New(update.Config{
		CurrentVersion: version.Version,
		BinaryName:     version.BinaryName,
		GitHubRepo:     "dl-alexandre/X-CLI",
		InstallCommand: "brew upgrade x-cli",
	})
	checker.AutoCheck()
}

// UpdateInfo is re-exported from cli-tools for backward compatibility
type UpdateInfo = update.Info
