package network

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

var obfuscatedSecret []byte

func SetSecret(s string) {
	obfuscatedSecret = xorBytes([]byte(s))
}

func signingSecret() []byte {
	return xorBytes(obfuscatedSecret)
}

func SigningEnabled() bool {
	return len(obfuscatedSecret) > 0
}

type Message struct {
	Type      string `json:"type"`
	Username  string `json:"username"`
	Team      string `json:"team"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	MessageID string `json:"message_id"`
	NetworkID string `json:"network_id,omitempty"`
	Replay    bool   `json:"replay,omitempty"`
	Signature string `json:"signature,omitempty"`
}

func (m *Message) Sign() {
	s := signingSecret()
	if len(s) == 0 {
		return
	}
	h := hmac.New(sha256.New, s)
	h.Write([]byte(m.MessageID))
	h.Write([]byte(m.Type))
	h.Write([]byte(m.Username))
	h.Write([]byte(m.Team))
	h.Write([]byte(m.Text))
	h.Write([]byte(m.Timestamp))
	m.Signature = hex.EncodeToString(h.Sum(nil))
}

func (m *Message) Verify() bool {
	s := signingSecret()
	if len(s) == 0 {
		return true
	}
	h := hmac.New(sha256.New, s)
	h.Write([]byte(m.MessageID))
	h.Write([]byte(m.Type))
	h.Write([]byte(m.Username))
	h.Write([]byte(m.Team))
	h.Write([]byte(m.Text))
	h.Write([]byte(m.Timestamp))
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(m.Signature), []byte(expected))
}
