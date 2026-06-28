package network

type Message struct {
	Type      string `json:"type"`
	Username  string `json:"username"`
	Team      string `json:"team"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	MessageID string `json:"message_id"`
	Replay    bool   `json:"replay,omitempty"`
}
