package network

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

var secret []byte

func SetSecret(s string) {
	if s != "" {
		secret = []byte(s)
	}
}

type Message struct {
	Type      string `json:"type"`
	Username  string `json:"username"`
	Team      string `json:"team"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	MessageID string `json:"message_id"`
	Replay    bool   `json:"replay,omitempty"`
	Signature string `json:"signature,omitempty"`
}

func (m *Message) Sign() {
	if len(secret) == 0 {
		return
	}
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(m.MessageID))
	h.Write([]byte(m.Type))
	h.Write([]byte(m.Username))
	h.Write([]byte(m.Team))
	h.Write([]byte(m.Text))
	h.Write([]byte(m.Timestamp))
	m.Signature = hex.EncodeToString(h.Sum(nil))
}

func (m *Message) Verify() bool {
	if len(secret) == 0 {
		return true
	}
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(m.MessageID))
	h.Write([]byte(m.Type))
	h.Write([]byte(m.Username))
	h.Write([]byte(m.Team))
	h.Write([]byte(m.Text))
	h.Write([]byte(m.Timestamp))
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(m.Signature), []byte(expected))
}
