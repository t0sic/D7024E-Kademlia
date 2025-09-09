package net

import (
	"net"
)

func (s *UDPServer) Ping(addr *net.UDPAddr) error {
	msg := Message{
		Type: MSG_PING,
		Args: []string{addr.String()},
	}
	return s.writeTo(addr, msg.String())
}
