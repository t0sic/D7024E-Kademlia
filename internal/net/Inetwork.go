// internal/net/Inetwork.go
package net

import (
	"net"
	"time" // add this

	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

// Network is the only thing Node needs.
type Network interface {
	On(msgType string, h Handler)
	Start() error
	Close() error
	Addr() *net.UDPAddr

	// fire-and-forget (optional, used by JoinNetwork)
	Send(to *net.UDPAddr, msg Message) error
	Ping(to *net.UDPAddr, id util.ID) error

	// request/response used by PingSync/FindNodesSync
	SendAndWait(to *net.UDPAddr, msg Message, timeout time.Duration) (Message, error)
}
