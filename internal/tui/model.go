package tui

import (
	"context"
	"database/sql"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/esteban/rbchat/internal/db"
	"github.com/esteban/rbchat/internal/network"
)

func GetTeams() []string {
	result := make([]string, len(teams))
	copy(result, teams)
	return result
}

var teams = []string{
	"Animoto",
	"Delivra",
	"Duplex",
	"Leadpages",
	"Paved",
	"Shift",
	"Redbrick",
}

type IncomingNetworkMsg struct {
	Message network.Message
}

type SyncTimeoutMsg struct{}

type SendFailedMsg struct {
	Err error
}

// MentionTickMsg fires when an @mention banner's display time has elapsed and
// it should be cleared. Gen identifies the mention that scheduled it so a stale
// timer from a superseded mention doesn't clear a newer banner early.
type MentionTickMsg struct {
	Gen int
}

func WaitForNetworkMsg(ch <-chan network.IncomingMessage) tea.Cmd {
	return func() tea.Msg {
		incoming, ok := <-ch
		if !ok {
			return nil
		}
		return IncomingNetworkMsg{Message: incoming.Message}
	}
}

type Model struct {
	db                   *sql.DB
	listener             *network.Listener
	broadcaster          *network.Broadcaster
	username             string
	team                 string
	messages             []network.Message
	viewport             viewport.Model
	input                textinput.Model
	peerCount            int
	syncing              bool
	msgCh                chan network.IncomingMessage
	ctx                  context.Context
	cancel               context.CancelFunc
	err                  error
	quitting             bool
	lastSeen             map[string]time.Time
	seenIDs              map[string]struct{}
	ready                bool
	notificationsEnabled bool
	otherInstanceRunning bool
	showHelp             bool
	networkID            string
	version              string
	osIconMode           string
	mentionBy            string
	mentionGen           int
}

func NewModel(database *sql.DB, username, team string, listener *network.Listener, broadcaster *network.Broadcaster, msgCh chan network.IncomingMessage, ctx context.Context, cancel context.CancelFunc, notificationsEnabled bool, otherInstanceRunning bool, networkID, version string, osIconMode string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	messages := make([]network.Message, 0, 100)
	seenIDs := make(map[string]struct{})

	q := db.New(database)
	recent, err := q.GetRecentMessagesToday(ctx, db.GetRecentMessagesTodayParams{
		NetworkID: networkID,
		Limit:     500,
	})
	if err == nil {
		for i := len(recent) - 1; i >= 0; i-- {
			dbMsg := recent[i]
			if dbMsg.Signature == "" {
				continue
			}
			msgType := dbMsg.Type
			if msgType == "sync" {
				if dbMsg.Text == "sync_request" {
					continue
				}
				if dbMsg.Text == "joined the network" {
					msgType = "join"
				}
			}
			if msgType != "chat" && msgType != "join" {
				continue
			}
			msg := network.Message{
				Type:      msgType,
				Username:  dbMsg.Username,
				Team:      dbMsg.Team,
				Text:      dbMsg.Text,
				Timestamp: dbMsg.Timestamp,
				MessageID: dbMsg.MessageID,
				OS:        dbMsg.Os,
				Signature: dbMsg.Signature,
			}
			if !msg.Verify() {
				continue
			}
			seenIDs[msg.MessageID] = struct{}{}
			messages = append(messages, msg)
		}
	}

	return Model{
		db:                   database,
		username:             username,
		team:                 team,
		listener:             listener,
		broadcaster:          broadcaster,
		msgCh:                msgCh,
		ctx:                  ctx,
		cancel:               cancel,
		messages:             messages,
		seenIDs:              seenIDs,
		input:                ti,
		syncing:              true,
		lastSeen:             make(map[string]time.Time),
		notificationsEnabled: notificationsEnabled,
		otherInstanceRunning: otherInstanceRunning,
		networkID:            networkID,
		version:              version,
	}
}
