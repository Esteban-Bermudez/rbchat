//go:build darwin

package tui

import (
	"os"
	"os/exec"
	"strings"

	"github.com/gen2brain/beeep"
)

// macOS attributes notifications posted by a CLI process to the "responsible"
// GUI app via the __CFBundleIdentifier environment variable. Inside Apple
// Terminal that resolves to com.apple.Terminal, which usually has no
// notification permission, so banners are silently dropped. (This is why
// notifications work in Warp/iTerm but not Terminal.app: those terminals are
// granted permission, Terminal.app is not.) Clearing the variable at startup
// lets the notifier post under its own permitted bundle identity, so
// notifications work consistently in every terminal.
func init() {
	os.Unsetenv("__CFBundleIdentifier")

	// terminal-notifier is normally installed via Homebrew. Make sure its
	// location is searchable even when rbchat is launched with a minimal PATH,
	// otherwise we silently fall back to the (blocked) osascript path.
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

// sendNotification posts a desktop notification. It prefers terminal-notifier,
// which posts under its own bundle identity and shows reliably regardless of
// the host terminal; it falls back to beeep (osascript) when terminal-notifier
// is not installed. "-sound default" asks macOS to play the default alert
// sound; whether it actually plays (and whether a banner is shown) still
// depends on the notification style granted to terminal-notifier in
// System Settings → Notifications.
func sendNotification(title, text string) {
	if path, err := exec.LookPath("terminal-notifier"); err == nil {
		if err := exec.Command(path, "-title", title, "-message", text, "-sound", "default").Run(); err == nil {
			return
		}
	}
	beeep.Notify(title, text, "")
}
