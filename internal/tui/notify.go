package tui

import "github.com/gen2brain/beeep"

func notify(username, team, text string) {
	title := username
	if team != "" {
		title += " (" + team + ")"
	}
	beeep.Notify("rbchat - "+title, text, "")
}
