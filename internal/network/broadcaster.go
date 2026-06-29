package network

import (
	"encoding/json"
	"net"
)

type Broadcaster struct {
	conn *net.UDPConn
}

func NewBroadcaster(addr string) (*Broadcaster, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}
	return &Broadcaster{conn: conn}, nil
}

func (b *Broadcaster) Send(msg Message) error {
	msg.Sign()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = b.conn.Write(data)
	return err
}

func (b *Broadcaster) Close() error {
	return b.conn.Close()
}
