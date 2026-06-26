package tui

import (
	"context"
	"database/sql"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/esteban/rbchat/internal/network"
)

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
	db          *sql.DB
	listener    *network.Listener
	broadcaster *network.Broadcaster
	username    string
	team        string
	messages    []network.Message
	viewport    viewport.Model
	input       textinput.Model
	peerCount   int
	syncing     bool
	msgCh       chan network.IncomingMessage
	ctx         context.Context
	cancel      context.CancelFunc
	err         error
	quitting    bool
	lastSeen    map[string]time.Time
	ready       bool
}

func NewModel(db *sql.DB, username, team string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	return Model{
		db:       db,
		username: username,
		team:     team,
		messages: make([]network.Message, 0, 100),
		input:    ti,
		syncing:  true,
		lastSeen: make(map[string]time.Time),
	}
}
