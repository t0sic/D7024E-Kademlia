package net

import (
	"net"

	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

func (s *UDPServer) Ping(addr *net.UDPAddr, id util.ID) error {
	msg := Message{
		Type: MSG_PING,
		Args: []string{id.String()},
	}
	return s.writeTo(addr, msg.String())
}
