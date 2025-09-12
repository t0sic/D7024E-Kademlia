package net

import (
	"net"
	"time"

	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

// Network interface abstracts the network layer for Kademlia nodes
type Network interface {
	On(msgType string, h Handler)
	Start() error
	Close() error
	Addr() *net.UDPAddr

	Send(to *net.UDPAddr, msg Message) error
	Ping(to *net.UDPAddr, id util.ID) error

	SendAndWait(to *net.UDPAddr, msg Message, timeout time.Duration) (Message, error)
}
