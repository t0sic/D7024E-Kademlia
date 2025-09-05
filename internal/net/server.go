package net

import (
	"fmt"
	"net"
	"strings"
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
func StartUDPServer(s *UDPServer) {
	fmt.Println("Starting UDP server on", s.addr.String())

	conn, err := net.ListenUDP("udp", s.addr)
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	s.conn = conn

	buf := make([]byte, BUFFER_SIZE)

	for {

		n, peer, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("UDP read error:", err)
			continue
		}

		msg := strings.TrimSpace(string(buf[:n]))
		fmt.Printf("UDP message from %s: %s\n", peer.String(), msg)

		var resp string
		if strings.EqualFold(msg, "PING") {
			resp = "PONG"
		} else {
			resp = msg
		}

		_, err = conn.WriteToUDP([]byte(resp), peer)
		if err != nil {
			fmt.Println("write error:", err)
			continue
		}

	}
}