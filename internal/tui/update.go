package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/esteban/rbchat/internal/db"
	"github.com/esteban/rbchat/internal/network"
	"github.com/google/uuid"
)

const (
	maxMessages   = 10000
	syncTimeout   = 2 * time.Second
	peerWindow    = 60 * time.Second
	multicastAddr = "224.0.0.1:9999"
)

func (m Model) Init() tea.Cmd {
	q := db.New(m.db)
	last50, err := q.GetRecentMessagesToday(m.ctx, 50)
	if err == nil {
		for i := len(last50) - 1; i >= 0; i-- {
			dbMsg := last50[i]
			msg := network.Message{
				Type:      dbMsg.Type,
				Username:  dbMsg.Username,
				Team:      dbMsg.Team,
				Text:      dbMsg.Text,
				Timestamp: dbMsg.Timestamp,
				MessageID: dbMsg.MessageID,
			}
			m.seenIDs[msg.MessageID] = struct{}{}
			m.messages = append(m.messages, msg)
		}
	}

	m.broadcaster.Send(network.Message{
		Type:      "sync",
		Username:  m.username,
		Team:      m.team,
		Text:      "sync_request",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		MessageID: uuid.New().String(),
	})

	return tea.Batch(
		WaitForNetworkMsg(m.msgCh),
		tea.Tick(syncTimeout, func(t time.Time) tea.Msg {
			return SyncTimeoutMsg{}
		}),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-5)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 5
		}
		m.input.Width = msg.Width - 2
		return m, nil

	case tea.KeyMsg:
		if m.quitting {
			return m, tea.Quit
		}
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			m.err = fmt.Errorf("Shutting down...")
			return m, tea.Quit
		case "ctrl+n":
			m.notificationsEnabled = !m.notificationsEnabled
			return m, nil
		case "enter":
			if m.syncing {
				return m, nil
			}
			text := m.input.Value()
			if text == "" {
				return m, nil
			}
			m.input.SetValue("")

			chatMsg := network.Message{
				Type:      "chat",
				Username:  m.username,
				Team:      m.team,
				Text:      text,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				MessageID: uuid.New().String(),
			}

			if err := m.broadcaster.Send(chatMsg); err != nil {
				return m, func() tea.Msg {
					return SendFailedMsg{Err: err}
				}
			}
			m.err = nil

			m.appendMessage(chatMsg)
			m.dbInsertMessage(chatMsg)
			m.refreshViewport()
			return m, WaitForNetworkMsg(m.msgCh)
		}

		if !m.syncing {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil

	case IncomingNetworkMsg:
		m.handleIncoming(msg.Message)
		if msg.Message.Type == "sync" && msg.Message.Text == "sync_request" {
			m.respondToSync()
		}
		return m, WaitForNetworkMsg(m.msgCh)

	case SyncTimeoutMsg:
		m.syncing = false
		m.refreshViewport()

		m.broadcaster.Send(network.Message{
			Type:      "join",
			Username:  m.username,
			Team:      m.team,
			Text:      "joined the network",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			MessageID: uuid.New().String(),
		})
		return m, WaitForNetworkMsg(m.msgCh)

	case SendFailedMsg:
		m.err = fmt.Errorf("Failed to send message: %v", msg.Err)
		return m, nil
	}

	return m, nil
}

func (m *Model) respondToSync() {
	q := db.New(m.db)
	messages, err := q.GetRecentMessagesForSync(m.ctx, 50)
	if err != nil {
		return
	}
	for _, dbMsg := range messages {
		if dbMsg.Type == "sync" {
			continue
		}
		m.broadcaster.Send(network.Message{
			Type:      "sync",
			Username:  dbMsg.Username,
			Team:      dbMsg.Team,
			Text:      dbMsg.Text,
			Timestamp: dbMsg.Timestamp,
			MessageID: dbMsg.MessageID,
		})
	}
}

func (m *Model) handleIncoming(msg network.Message) {
	m.lastSeen[msg.Username] = time.Now()
	m.peerCount = m.countActivePeers()

	if msg.Type == "sync" {
		m.appendMessage(msg)
		m.dbInsertMessage(msg)
		if !m.syncing {
			m.refreshViewport()
		}
		return
	}

	if msg.Type == "join" {
		if m.syncing {
			return
		}
		m.appendMessage(msg)
		m.dbInsertMessage(msg)
		m.refreshViewport()
		return
	}

	if msg.Type == "chat" {
		m.appendMessage(msg)
		m.dbInsertMessage(msg)
		if !m.syncing {
			m.refreshViewport()
		}
		if m.notificationsEnabled && msg.Username != m.username {
			notify(msg.Username, msg.Team, msg.Text)
		}
		return
	}
}

func (m *Model) appendMessage(msg network.Message) {
	if _, seen := m.seenIDs[msg.MessageID]; seen {
		return
	}
	m.seenIDs[msg.MessageID] = struct{}{}
	if len(m.messages) >= maxMessages {
		m.messages = m.messages[1:]
	}
	m.messages = append(m.messages, msg)
}

func (m *Model) dbInsertMessage(msg network.Message) {
	q := db.New(m.db)
	q.InsertMessage(m.ctx, db.InsertMessageParams{
		MessageID: msg.MessageID,
		Type:      msg.Type,
		Username:  msg.Username,
		Team:      msg.Team,
		Text:      msg.Text,
		Timestamp: msg.Timestamp,
	})
}

func (m *Model) refreshViewport() {
	var content string
	for _, msg := range m.messages {
		content += renderMessage(msg) + "\n"
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *Model) countActivePeers() int {
	now := time.Now()
	count := 0
	for _, t := range m.lastSeen {
		if now.Sub(t) < peerWindow {
			count++
		}
	}
	return count
}
