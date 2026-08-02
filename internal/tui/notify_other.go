//go:build !darwin

package tui

import "github.com/gen2brain/beeep"

func sendNotification(title, text string) {
	beeep.Notify(title, text, "")
}
