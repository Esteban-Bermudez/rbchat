package tui

import (
	"fmt"
	"sort"
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
	helpHeight    = 9
)

func (m Model) Init() tea.Cmd {
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
		vpHeight := msg.Height - 5
		if m.showHelp {
			vpHeight -= helpHeight
		}
		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
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
		case "ctrl+u":
			m.viewport.HalfViewUp()
			return m, nil
		case "ctrl+d":
			m.viewport.HalfViewDown()
			return m, nil
		case "pgup":
			m.viewport.PageUp()
			return m, nil
		case "pgdown":
			m.viewport.PageDown()
			return m, nil
		case "?":
			m.showHelp = !m.showHelp
			if m.ready {
				if m.showHelp {
					m.viewport.Height -= helpHeight
				} else {
					m.viewport.Height += helpHeight
				}
			}
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
			m.viewport.GotoBottom()
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
		sort.Slice(m.messages, func(i, j int) bool {
			return m.messages[i].Timestamp < m.messages[j].Timestamp
		})
		m.refreshViewport()

		if !m.otherInstanceRunning {
			m.broadcaster.Send(network.Message{
				Type:      "join",
				Username:  m.username,
				Team:      m.team,
				Text:      "joined the network",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				MessageID: uuid.New().String(),
			})
		}
		return m, WaitForNetworkMsg(m.msgCh)

	case SendFailedMsg:
		m.err = fmt.Errorf("Failed to send message: %v", msg.Err)
		return m, nil
	}

	return m, nil
}

func (m *Model) respondToSync() {
	q := db.New(m.db)
	messages, err := q.GetRecentMessagesForSync(m.ctx, 500)
	if err != nil {
		return
	}
	for _, dbMsg := range messages {
		if dbMsg.Text == "sync_request" {
			continue
		}
		msgType := dbMsg.Type
		if dbMsg.Type == "sync" && dbMsg.Text == "joined the network" {
			msgType = "join"
		}
		if msgType == "chat" || msgType == "join" {
			m.broadcaster.Send(network.Message{
				Type:      msgType,
				Username:  dbMsg.Username,
				Team:      dbMsg.Team,
				Text:      dbMsg.Text,
				Timestamp: dbMsg.Timestamp,
				MessageID: dbMsg.MessageID,
				Replay:    true,
			})
		}
	}
}

func (m *Model) handleIncoming(msg network.Message) {
	if msg.Type == "sync" {
		if msg.Text == "joined the network" {
			msg.Type = "join"
		} else {
			m.dbInsertMessage(msg)
			return
		}
	}

	if msg.Type == "join" {
		m.appendMessage(msg)
		m.dbInsertMessage(msg)
		if !m.syncing {
			m.refreshViewport()
			if !msg.Replay {
				m.lastSeen[msg.Username] = time.Now()
				m.peerCount = m.countActivePeers()
			}
		}
		return
	}

	if msg.Type == "chat" {
		m.appendMessage(msg)
		m.dbInsertMessage(msg)
		if !m.syncing {
			sort.Slice(m.messages, func(i, j int) bool {
				return m.messages[i].Timestamp < m.messages[j].Timestamp
			})
			m.refreshViewport()
			if !msg.Replay {
				m.lastSeen[msg.Username] = time.Now()
				m.peerCount = m.countActivePeers()
				if m.notificationsEnabled && msg.Username != m.username {
					notify(msg.Username, msg.Team, msg.Text)
				}
			}
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
	if err := q.InsertMessage(m.ctx, db.InsertMessageParams{
		MessageID: msg.MessageID,
		Type:      msg.Type,
		Username:  msg.Username,
		Team:      msg.Team,
		Text:      msg.Text,
		Timestamp: msg.Timestamp,
	}); err != nil {
		m.err = fmt.Errorf("db write: %v", err)
	}
}

func (m *Model) refreshViewport() {
	var content string
	var lastDate string
	for _, msg := range m.messages {
		rendered := renderMessage(msg, m.viewport.Width)
		if rendered == "" {
			continue
		}
		date := messageDate(msg.Timestamp)
		if date != "" && date != lastDate && lastDate != "" {
			content += dividerStyle.Render(fmt.Sprintf("── %s ──", date)) + "\n"
		}
		if date != "" {
			lastDate = date
		}
		content += rendered + "\n"
	}
	atBottom := m.viewport.AtBottom()
	m.viewport.SetContent(content)
	if atBottom {
		m.viewport.GotoBottom()
	}
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
