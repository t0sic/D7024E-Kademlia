package net

import (
	"net"
	"time"
)

// Network interface abstracts the network layer for Kademlia nodes
type Network interface {
	On(msgType string, h Handler)
	Start() error
	Close() error
	Addr() *net.UDPAddr

	Send(to *net.UDPAddr, msg Message) error

	SendAndWait(to *net.UDPAddr, msg Message, timeout time.Duration) (Message, error)
}
