package tui_test

import (
	"testing"

	"github.com/esteban/rbchat/internal/network"
	"github.com/esteban/rbchat/internal/tui"
)

func TestTeamsNonEmpty(t *testing.T) {
	teams := tui.GetTeams()
	if len(teams) == 0 {
		t.Fatal("expected at least one team")
	}
}

func TestRenderMessageChat(t *testing.T) {
	msg := network.Message{
		Type:      "chat",
		Username:  "esteban",
		Team:      "Redbrick",
		Text:      "hello",
		Timestamp: "2026-06-24T14:30:00Z",
		MessageID: "abc",
	}
	rendered := tui.RenderMessage(msg)
	if rendered == "" {
		t.Fatal("expected non-empty render")
	}
}

func TestRenderMessageSync(t *testing.T) {
	msg := network.Message{
		Type:      "sync",
		Username:  "esteban",
		Team:      "Redbrick",
		Text:      "hello",
		Timestamp: "2026-06-24T14:30:00Z",
		MessageID: "abc",
	}
	rendered := tui.RenderMessage(msg)
	if rendered != "" {
		t.Fatal("expected empty render for sync messages")
	}
}

func TestRenderMessageJoin(t *testing.T) {
	msg := network.Message{
		Type:      "join",
		Username:  "esteban",
		Team:      "Redbrick",
		Text:      "joined the network",
		Timestamp: "2026-06-24T14:30:00Z",
		MessageID: "abc",
	}
	rendered := tui.RenderMessage(msg)
	if rendered == "" {
		t.Fatal("expected non-empty render for join messages")
	}
}
