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
}

func NewModel(database *sql.DB, username, team string, listener *network.Listener, broadcaster *network.Broadcaster, msgCh chan network.IncomingMessage, ctx context.Context, cancel context.CancelFunc, notificationsEnabled bool, otherInstanceRunning bool) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	messages := make([]network.Message, 0, 100)
	seenIDs := make(map[string]struct{})

	q := db.New(database)
	recent, err := q.GetRecentMessagesToday(ctx, 500)
	if err == nil {
		for i := len(recent) - 1; i >= 0; i-- {
			dbMsg := recent[i]
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
				Signature: dbMsg.Signature,
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
	}
}
