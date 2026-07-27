package tui

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// VersionCheckResultMsg is delivered after checking the latest release tag
// against the current binary version.
type VersionCheckResultMsg struct {
	LatestVersion string
	Available     bool
}

// CheckForUpdate returns a tea.Cmd that fetches the latest release from
// GitHub and compares it with the current binary version. It skips the
// check for dev builds (version == "" or "dev").
func CheckForUpdate(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		if currentVersion == "" || currentVersion == "dev" {
			return VersionCheckResultMsg{Available: false}
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("https://api.github.com/repos/Esteban-Bermudez/rbchat/releases/latest")
		if err != nil {
			return VersionCheckResultMsg{Available: false}
		}
		defer resp.Body.Close()

		var release struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return VersionCheckResultMsg{Available: false}
		}

		latest := strings.TrimPrefix(release.TagName, "v")
		current := strings.TrimPrefix(currentVersion, "v")

		return VersionCheckResultMsg{
			LatestVersion: release.TagName,
			Available:     latest != current,
		}
	}
}
