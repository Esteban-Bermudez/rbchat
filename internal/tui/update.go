package tui

import (
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/esteban/rbchat/internal/db"
	"github.com/esteban/rbchat/internal/network"
	"github.com/google/uuid"
)

const (
	maxMessages       = 10000
	syncTimeout       = 2 * time.Second
	peerWindow        = 60 * time.Second
	heartbeatInterval = 30 * time.Second
	replayWindow      = 60 * time.Second
	multicastAddr     = "239.255.0.1:9999"
	helpHeight        = 9
)

func (m Model) Init() tea.Cmd {
	m.broadcaster.Send(network.Message{
		Type:      "sync",
		Username:  m.username,
		Team:      m.team,
		Text:      "sync_request",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		MessageID: uuid.New().String(),
		NetworkID: m.networkID,
		OS:        runtime.GOOS,
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
			m.notificationMode = m.notificationMode.Next()
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
		case "esc":
			m.mentionBy = ""
			return m, nil
		case "?":
			if m.showHelp || m.input.Value() == "" {
				// Close user list if open to avoid stacking footer panels.
				if m.showUserList {
					m.showUserList = false
					if m.ready {
						m.viewport.Height += helpHeight
					}
				}
				m.showHelp = !m.showHelp
				if m.ready {
					if m.showHelp {
						m.viewport.Height -= helpHeight
					} else {
						m.viewport.Height += helpHeight
					}
				}
				return m, nil
			}
		case "ctrl+t":
			if m.syncing {
				return m, nil
			}
			m.showUserList = !m.showUserList
			if m.showUserList {
				m.viewport.SetContent(buildUserListContent(m.viewport.Width, m.lastSeen, m.osIconMode))
				m.viewport.GotoTop()
			} else {
				m.refreshViewport()
			}
			return m, nil
		case "enter":
			if m.syncing {
				return m, nil
			}
			text := m.input.Value()
			if strings.TrimSpace(text) == "" {
				return m, nil
			}
			m.input.SetValue("")
			m.mentionBy = ""

			chatMsg := network.Message{
				Type:      "chat",
				Username:  m.username,
				Team:      m.team,
				Text:      text,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				MessageID: uuid.New().String(),
				NetworkID: m.networkID,
				OS:        runtime.GOOS,
			}
			chatMsg.Sign()

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
				NetworkID: m.networkID,
				OS:        runtime.GOOS,
			})
		}
		return m, tea.Batch(
			WaitForNetworkMsg(m.msgCh),
			tea.Tick(heartbeatInterval, func(t time.Time) tea.Msg {
				return HeartbeatTickMsg{}
			}),
		)

	case HeartbeatTickMsg:
		m.broadcaster.Send(network.Message{
			Type:      "heartbeat",
			Username:  m.username,
			Team:      m.team,
			Text:      "heartbeat",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			MessageID: uuid.New().String(),
			NetworkID: m.networkID,
			OS:        runtime.GOOS,
		})
		return m, tea.Batch(
			WaitForNetworkMsg(m.msgCh),
			tea.Tick(heartbeatInterval, func(t time.Time) tea.Msg {
				return HeartbeatTickMsg{}
			}),
		)

	case SendFailedMsg:
		m.err = fmt.Errorf("Failed to send message: %v", msg.Err)
		return m, nil
	}

	return m, nil
}

func (m *Model) respondToSync() {
	q := db.New(m.db)
	messages, err := q.GetRecentMessagesForSync(m.ctx, db.GetRecentMessagesForSyncParams{
		NetworkID: m.networkID,
		Limit:     500,
	})
	if err != nil {
		return
	}
	for _, dbMsg := range messages {
		if dbMsg.Text == "sync_request" {
			continue
		}
		if dbMsg.Signature == "" {
			continue
		}
		msgType := dbMsg.Type
		if dbMsg.Type == "sync" && dbMsg.Text == "joined the network" {
			msgType = "join"
		}
		if msgType == "chat" || msgType == "join" {
			msg := network.Message{
				Type:      msgType,
				Username:  dbMsg.Username,
				Team:      dbMsg.Team,
				Text:      dbMsg.Text,
				Timestamp: dbMsg.Timestamp,
				MessageID: dbMsg.MessageID,
				OS:        dbMsg.Os,
				Signature: dbMsg.Signature,
				Replay:    true,
			}
			if !msg.Verify() {
				continue
			}
			m.broadcaster.Send(msg)
		}
	}
}

// isFreshMessage reports whether the message's timestamp is within the
// replay window. Stale messages are likely replays. Sync replays (which
// carry Replay: true) are exempt — they include their original timestamp.
func isFreshMessage(msg network.Message) bool {
	t, err := time.Parse(time.RFC3339, msg.Timestamp)
	if err != nil {
		return false
	}
	return time.Since(t) < replayWindow
}

// handleIncoming processes an inbound network message, updating the message
// list, peer state, and mention banner as appropriate.
func (m *Model) handleIncoming(msg network.Message) {
	if m.networkID != "" && msg.NetworkID != "" && msg.NetworkID != m.networkID {
		return
	}

	// Reject stale non-replay messages — they're likely replays of previously
	// captured traffic. The timestamp is inside the signed payload so an
	// attacker cannot modify it without invalidating the signature.
	if !msg.Replay && !isFreshMessage(msg) {
		return
	}

	// Heartbeat messages are ephemeral presence signals — never stored or
	// displayed. Just update the lastSeen timestamp and recalculate the peer
	// count.
	if msg.Type == "heartbeat" {
		if !m.syncing {
			m.lastSeen[msg.Username] = peerInfo{
				lastSeen: time.Now(),
				team:     msg.Team,
			}
			m.peerCount = m.countActivePeers()
		}
		return
	}

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
				m.lastSeen[msg.Username] = peerInfo{
					lastSeen: time.Now(),
					team:     msg.Team,
				}
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
				m.lastSeen[msg.Username] = peerInfo{
					lastSeen: time.Now(),
					team:     msg.Team,
				}
				m.peerCount = m.countActivePeers()
				if msg.Username != m.username {
					isMention := mentionsUser(msg.Text, m.username)
					switch m.notificationMode {
					case NotifyAll:
						notify(msg.Username, msg.Team, msg.Text)
					case NotifyMentions:
						if isMention {
							notify(msg.Username, msg.Team, msg.Text)
						}
					}
					if isMention {
						m.mentionBy = msg.Username
					}
				}
			}
		}
		return
	}
}

// mentionRegex builds a case-insensitive regex that matches @username tokens.
// Uses \B (non-word-boundary) before @ so "email@user" doesn't match, and
// \b after username so "@u" doesn't match inside "@username".
func mentionRegex(username string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\B@` + regexp.QuoteMeta(username) + `\b`)
}

// mentionsUser reports whether text mentions username via an "@username" token.
func mentionsUser(text, username string) bool {
	if username == "" {
		return false
	}
	return mentionRegex(username).MatchString(text)
}

func (m *Model) appendMessage(msg network.Message) {
	if !msg.Verify() {
		return
	}
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
	if !msg.Verify() {
		return
	}
	q := db.New(m.db)
	if err := q.InsertMessage(m.ctx, db.InsertMessageParams{
		MessageID: msg.MessageID,
		Type:      msg.Type,
		Username:  msg.Username,
		Team:      msg.Team,
		Text:      msg.Text,
		Timestamp: msg.Timestamp,
		Os:        msg.OS,
		Signature: msg.Signature,
		NetworkID: m.networkID,
	}); err != nil {
		m.err = fmt.Errorf("db write: %v", err)
	}
}

func (m *Model) refreshViewport() {
	if m.showUserList {
		return
	}
	var content string
	var lastDate string
	for _, msg := range m.messages {
		rendered := renderMessage(msg, m.viewport.Width, m.osIconMode, m.username)
		if rendered == "" {
			continue
		}
		date := messageDate(msg.Timestamp)
		if date != "" && date != lastDate && lastDate != "" {
			dateStr := fmt.Sprintf("     %s     ", date)
			dateLen := lipgloss.Width(dateStr)
			dashes := m.viewport.Width - dateLen
			if dashes > 0 {
				leftDashes := dashes / 2
				rightDashes := dashes - leftDashes
				content += "\n" + dividerStyle.Render(strings.Repeat("─", leftDashes)+dateStr+strings.Repeat("─", rightDashes)) + "\n"
			} else {
				content += "\n" + dividerStyle.Render(dateStr) + "\n"
			}
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
	for _, info := range m.lastSeen {
		if now.Sub(info.lastSeen) < peerWindow {
			count++
		}
	}
	return count
}
