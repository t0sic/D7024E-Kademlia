package net

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const BUFFER_SIZE 		= 4096
const READ_TIMEOUT 		= 5 * time.Second
const WRITE_TIMEOUT 	= 5 * time.Second

// MESSAGE TYPES
const MSG_PING      	= "PING"
const MSG_PONG      	= "PONG"

// PROTOCOL
type Message struct {
	Type string
	Args []string
}

func ParseMessage(s string) (Message, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Message{}, fmt.Errorf("empty message")
	}
	parts := strings.Fields(s)
	return Message{
		Type: parts[0],
		Args: parts[1:],
	}, nil
}

func (m Message) String() string {
	if len(m.Args) == 0 {
		return m.Type
	}
	return m.Type + " " + strings.Join(m.Args, " ")
}

// UDP SERVER

// Handler defined how we process incoming messages
type Handler func(from *net.UDPAddr, msg Message) (*Message, error)

type UDPServer struct {
	addr 		*net.UDPAddr
	conn 		*net.UDPConn

	closing 	bool
	mu   		sync.RWMutex
	wg 			sync.WaitGroup

	handlers 	map[string]Handler
}

// Close shuts down the UDP server connection.
func (s *UDPServer) Close() error {
	s.mu.Lock()
	s.closing = true
	s.mu.Unlock()

	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.wg.Wait()
	return nil
}

func (s *UDPServer) isClosing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closing
}

// Initialize the server
func CreateUDPServer(addr string) *UDPServer {
    udpAddr, err := net.ResolveUDPAddr("udp", addr)
    if err != nil {
        panic(err)
    }
    return &UDPServer{
        addr:     udpAddr,
        handlers: make(map[string]Handler),
    }
}

// RegisterHandler for a specific message type
func (s *UDPServer) dispatch(from *net.UDPAddr, msg Message) (*Message, error) {
	s.mu.RLock()
	h := s.handlers[msg.Type]
	s.mu.RUnlock()

	if h == nil {
		return nil, fmt.Errorf("no handler for message type: %s", msg.Type)
	}

	return h(from, msg)
}

// On registers a handler for a specific message type
func (s *UDPServer) On(msgType string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[strings.ToUpper(strings.TrimSpace(msgType))] = h
}

// writeTo sends a message to a specific peer
func (s *UDPServer) writeTo(peer *net.UDPAddr, payload string) error {
	_ = s.conn.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT))
	_, err := s.conn.WriteToUDP([]byte(payload), peer)
	return err
}

// Start the server
func (s *UDPServer) Start() error {
	conn, err := net.ListenUDP("udp", s.addr)
	if err != nil {
		panic(err)
	}
	// Update s.addr to the actual port assigned (important for :0)
	s.addr = conn.LocalAddr().(*net.UDPAddr)
	fmt.Println("Starting UDP server on", s.addr.String())

	defer conn.Close()

	s.conn = conn

	buf := make([]byte, BUFFER_SIZE)
	s.wg.Add(1)
	defer s.wg.Done()

	for {

		s.conn.SetReadDeadline(time.Now().Add(READ_TIMEOUT))
		n, peer, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if s.isClosing() {
					return nil
				}
				// just loop again (keep listening)
				continue
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			fmt.Println("UDP read error:", err)
			continue
		}

		raw := strings.TrimSpace(string(buf[:n]))
		msg, perr := ParseMessage(raw)
		if perr != nil {
			fmt.Println("UDP parse error:", perr)
			continue
		}

		reply, rerr := s.dispatch(peer, msg)
		if rerr != nil {
			fmt.Println("UDP dispatch error:", rerr)
			continue
		}

		if reply != nil {
			if err := s.writeTo(peer, reply.String()); err != nil {
				fmt.Println("UDP write error:", err)
			}
		}

	}
}

