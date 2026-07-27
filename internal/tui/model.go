package tui

import (
	"context"
	"database/sql"
	"net"
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

// NotificationMode controls which incoming chat messages trigger a desktop
// notification. Cycling (via ctrl+n) moves NotifyAll -> NotifyMentions ->
// NotifyNone -> NotifyAll ...
type NotificationMode int

const (
	NotifyAll NotificationMode = iota
	NotifyMentions
	NotifyNone
)

// Next returns the next mode in the cycle.
func (n NotificationMode) Next() NotificationMode {
	switch n {
	case NotifyAll:
		return NotifyMentions
	case NotifyMentions:
		return NotifyNone
	default:
		return NotifyAll
	}
}

type IncomingNetworkMsg struct {
	Message network.Message
	From    *net.UDPAddr
}

type SyncTimeoutMsg struct{}

// HeartbeatTickMsg is sent periodically to broadcast a presence announcement.
type HeartbeatTickMsg struct{}

// SyncResponseMsg is sent after a random jitter delay to respond to a sync
// request from a specific source. The delay spreads out responses so peers
// don't all reply at the same millisecond.
type SyncResponseMsg struct {
	SourceKey string
}

type SendFailedMsg struct {
	Err error
}

func WaitForNetworkMsg(ch <-chan network.IncomingMessage) tea.Cmd {
	return func() tea.Msg {
		incoming, ok := <-ch
		if !ok {
			return nil
		}
		return IncomingNetworkMsg{Message: incoming.Message, From: incoming.From}
	}
}

type peerInfo struct {
	lastSeen time.Time
	team     string
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
	lastSeen             map[string]peerInfo
	seenIDs              map[string]struct{}
	ready                bool
	notificationMode     NotificationMode
	otherInstanceRunning bool
	showHelp             bool
	showUserList         bool
	networkID            string
	version              string
	osIconMode           string
	mentionBy            string
	syncLastResponse     map[string]time.Time
	updateAvailable      string
	signingDisabled      bool
}

func NewModel(database *sql.DB, username, team string, listener *network.Listener, broadcaster *network.Broadcaster, msgCh chan network.IncomingMessage, ctx context.Context, cancel context.CancelFunc, notificationsEnabled bool, otherInstanceRunning bool, networkID, version string, osIconMode string, signingDisabled bool) Model {
	notificationMode := NotifyAll
	if !notificationsEnabled {
		notificationMode = NotifyNone
	}
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
		lastSeen:             make(map[string]peerInfo),
		notificationMode:     notificationMode,
		otherInstanceRunning: otherInstanceRunning,
		networkID:            networkID,
		version:              version,
		syncLastResponse:     make(map[string]time.Time),
		signingDisabled:      signingDisabled,
	}
}
