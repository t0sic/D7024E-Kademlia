package net

import (
	"net"
	"sync"
	"time"
)

const DEFAULT_ADDRESS = ":6881"
const BUFFER_SIZE = 4096
const READ_TIMEOUT = 5 * time.Second
const WRITE_TIMEOUT = 5 * time.Second

type UDPServer struct {
	addr        *net.UDPAddr
	conn        *net.UDPConn
	mu		  	sync.RWMutex
}

// Initialize the server
func CreateUDPServer(addr string) *UDPServer {

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}

	return &UDPServer{
		addr		: udpAddr,
	}
}

// Start the server
func Start(s *UDPServer) {
	conn, err := net.ListenUDP("udp", s.addr)
	if err != nil {
		panic(err)
	}

	s.conn = conn
}