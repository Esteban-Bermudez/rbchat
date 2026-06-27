package network_test

import (
	"encoding/json"
	"testing"

	"github.com/esteban/rbchat/internal/network"
)

func TestMessageJSONRoundTrip(t *testing.T) {
	original := network.Message{
		Type:      "chat",
		Username:  "esteban",
		Team:      "Redbrick",
		Text:      "Hello world",
		Timestamp: "2026-06-24T14:30:00Z",
		MessageID: "abc-123",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded network.Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Type != original.Type {
		t.Fatalf("expected type %s, got %s", original.Type, decoded.Type)
	}
	if decoded.Username != original.Username {
		t.Fatalf("expected username %s, got %s", original.Username, decoded.Username)
	}
	if decoded.Team != original.Team {
		t.Fatalf("expected team %s, got %s", original.Team, decoded.Team)
	}
	if decoded.Text != original.Text {
		t.Fatalf("expected text %s, got %s", original.Text, decoded.Text)
	}
	if decoded.Timestamp != original.Timestamp {
		t.Fatalf("expected timestamp %s, got %s", original.Timestamp, decoded.Timestamp)
	}
	if decoded.MessageID != original.MessageID {
		t.Fatalf("expected message_id %s, got %s", original.MessageID, decoded.MessageID)
	}
}

func TestSyncMessageJSON(t *testing.T) {
	msg := network.Message{
		Type:      "sync",
		Username:  "esteban",
		Team:      "Redbrick",
		Text:      "Hey, anyone here?",
		Timestamp: "2026-06-24T14:30:00Z",
		MessageID: "sync-abc",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded network.Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Type != "sync" {
		t.Fatalf("expected type sync, got %s", decoded.Type)
	}
}

func TestJoinMessageJSON(t *testing.T) {
	msg := network.Message{
		Type:      "join",
		Username:  "esteban",
		Team:      "Redbrick",
		Text:      "joined the network",
		Timestamp: "2026-06-24T14:30:00Z",
		MessageID: "join-abc",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded network.Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Type != "join" {
		t.Fatalf("expected type join, got %s", decoded.Type)
	}
}
