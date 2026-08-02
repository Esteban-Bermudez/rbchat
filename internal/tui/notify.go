package tui

// notify shows a desktop notification for an incoming chat message.
func notify(username, team, text string) {
	title := username
	if team != "" {
		title += " (" + team + ")"
	}
	sendNotification("rbchat - "+title, text)
}
