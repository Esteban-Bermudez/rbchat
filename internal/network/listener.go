package network

import (
	"context"
	"encoding/json"
	"net"
)

type Listener struct {
	conn *net.UDPConn
}

func NewListener(addr string) (*Listener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenMulticastUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}
	return &Listener{conn: conn}, nil
}

type IncomingMessage struct {
	Message Message
	From    *net.UDPAddr
}

func (l *Listener) Listen(ctx context.Context, msgCh chan<- IncomingMessage) {
	buf := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, src, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		var msg Message
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}
		if !msg.Verify() {
			continue
		}
		select {
		case msgCh <- IncomingMessage{Message: msg, From: src}:
		case <-ctx.Done():
			return
		}
	}
}

func (l *Listener) Close() error {
	return l.conn.Close()
}
