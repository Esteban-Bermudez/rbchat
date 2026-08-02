//go:build darwin

package tui

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gen2brain/beeep"
)

// alerterTimeout auto-closes the notification after this many seconds so the
// alerter process exits instead of lingering until the user clicks the alert.
const alerterTimeout = 10

func init() {
	// macOS attributes a notification posted by a CLI to the "responsible" GUI
	// app via the __CFBundleIdentifier env var. Inside Apple Terminal that
	// resolves to com.apple.Terminal, which usually has no notification
	// permission, so the beeep/osascript fallback is silently dropped (this is
	// why notifications work in Warp/iTerm but not Terminal.app). Clearing it
	// lets the fallback post under its own permitted bundle identity.
	os.Unsetenv("__CFBundleIdentifier")

	// alerter (and terminal-notifier, used by beeep) are normally installed via
	// Homebrew. Make sure their location is searchable even when rbchat is
	// launched with a minimal PATH.
	ensurePATH("/opt/homebrew/bin", "/usr/local/bin")
}

// ensurePATH appends the given directories to PATH if they are not already
// present.
func ensurePATH(dirs ...string) {
	sep := string(os.PathListSeparator)
	entries := strings.Split(os.Getenv("PATH"), sep)
	present := make(map[string]bool, len(entries))
	for _, e := range entries {
		present[e] = true
	}
	for _, d := range dirs {
		if !present[d] {
			entries = append(entries, d)
		}
	}
	os.Setenv("PATH", strings.Join(entries, sep))
}

// sendNotification posts a desktop notification. It prefers alerter, an actively
// maintained alternative to the (unmaintained) terminal-notifier that beeep
// falls back on; alerter posts through the modern UserNotifications API so
// banners show reliably regardless of the host terminal. When alerter is not
// installed it falls back to beeep (terminal-notifier -> osascript).
//
// alerter stays alive until the alert is dismissed or times out, so it is
// started in the background (and reaped in a goroutine) to avoid stalling the
// TUI update loop.
func sendNotification(title, text string) {
	if path, err := exec.LookPath("alerter"); err == nil {
		cmd := exec.Command(path,
			"--title", title,
			"--message", text,
			"--sound", "default",
			"--timeout", strconv.Itoa(alerterTimeout),
		)
		if err := cmd.Start(); err == nil {
			go cmd.Wait()
			return
		}
	}
	beeep.Notify(title, text, "")
}
