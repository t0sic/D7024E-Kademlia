package net

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

const BUFFER_SIZE = 4096
const READ_TIMEOUT = 5 * time.Second
const WRITE_TIMEOUT = 5 * time.Second

// MESSAGE TYPES
const MSG_PING = "PING"
const MSG_PONG = "PONG"
const MSG_NODES = "NODES"
const MSG_FIND_NODE = "FIND_NODE"

const MSG_STORE = "STORE"
const MSG_STORED = "STORED"

const MSG_GET = "GET"
const MSG_VALUE = "VALUE"
const MSG_NOT_FOUND = "NOT_FOUND"

// WAITER PROTOCOL
type waiter struct {
	ch chan Message
}

// PROTOCOL
type Message struct {
	Type  string
	RPCID string
	Args  []string
}

func ParseMessage(s string) (Message, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Message{}, fmt.Errorf("empty message")
	}
	parts := strings.Fields(s)
	msg := Message{Type: parts[0]}
	rest := parts[1:]
	if len(rest) > 0 && strings.HasPrefix(rest[0], "#") {
		msg.RPCID = strings.TrimPrefix(rest[0], "#")
		rest = rest[1:]
	}
	msg.Args = rest
	return msg, nil
}

func (m Message) String() string {
	b := strings.Builder{}
	b.WriteString(m.Type)
	if m.RPCID != "" {
		b.WriteByte(' ')
		b.WriteByte('#')
		b.WriteString(m.RPCID)
	}
	if len(m.Args) > 0 {
		b.WriteByte(' ')
		b.WriteString(strings.Join(m.Args, " "))
	}
	return b.String()
}

// UDP SERVER

// Handler defined how we process incoming messages
type Handler func(from *net.UDPAddr, msg Message) (*Message, error)

type UDPServer struct {
	addr *net.UDPAddr
	conn *net.UDPConn

	closing  bool
	mu       sync.RWMutex
	wg       sync.WaitGroup
	handlers map[string]Handler

	waiters map[string]*waiter
	wmu     sync.Mutex
}

// expose bound address (handy for :0)
func (s *UDPServer) Addr() *net.UDPAddr { return s.addr }

// simple fire-and-forget wrapper
func (s *UDPServer) Send(to *net.UDPAddr, msg Message) error {
	return s.WriteTo(to, msg.String())
}

// Close shuts down the UDP server connection.
func (s *UDPServer) Close() error {
	s.mu.Lock()
	s.closing = true
	s.mu.Unlock()

	// Unblock any goroutines waiting on RPC responses
	s.CancelAllWaiters()

	if s.conn != nil {
		_ = s.conn.Close() // triggers read deadline + loop exit
	}

	// Wait until Start() returns and wg.Done() executes
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
		waiters:  make(map[string]*waiter),
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

// WriteTo sends a message to a specific peer
func (s *UDPServer) WriteTo(peer *net.UDPAddr, payload string) error {
	_ = s.conn.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT))
	_, err := s.conn.WriteToUDP([]byte(payload), peer)
	return err
}

// addWaiter adds a waiter for a specific RPC ID and returns its channel
func (s *UDPServer) AddWaiter(rpcID string) <-chan Message {
	ch := make(chan Message, 1)
	s.wmu.Lock()
	s.waiters[rpcID] = &waiter{ch: ch}
	s.wmu.Unlock()
	return ch
}

func (s *UDPServer) Wait(rpcID string, timeout time.Duration) (Message, error) {
	s.wmu.Lock()
	w, ok := s.waiters[rpcID]
	s.wmu.Unlock()
	if !ok {
		return Message{}, fmt.Errorf("no waiter for rpcID=%s", rpcID)
	}

	select {
	case msg, ok := <-w.ch:
		if !ok {
			return Message{}, fmt.Errorf("waiter closed (server shutting down?)")
		}
		return msg, nil
	case <-time.After(timeout):
		s.CancelWaiter(rpcID)
		return Message{}, fmt.Errorf("timeout waiting for rpcID=%s", rpcID)
	}
}

// CancelWaiter removes a waiter and closes its channel
func (s *UDPServer) CancelWaiter(rpcID string) {
	s.wmu.Lock()
	if w, ok := s.waiters[rpcID]; ok {
		delete(s.waiters, rpcID)
		close(w.ch)
	}
	s.wmu.Unlock()
}

// Delivers a message to the corresponding waiter if exists
func (s *UDPServer) deliverToWaiter(msg Message) bool {
	if msg.RPCID == "" {
		return false
	}
	s.wmu.Lock()
	w, ok := s.waiters[msg.RPCID]
	if ok {
		delete(s.waiters, msg.RPCID)
	}
	s.wmu.Unlock()
	if ok {
		w.ch <- msg
		return true
	}
	return false
}

// Ensures msg.RPCID is set, registers waiter, sends, then waits.
func (s *UDPServer) SendAndWait(peer *net.UDPAddr, msg Message, timeout time.Duration) (Message, error) {
	if msg.RPCID == "" {
		msg.RPCID = util.NewRandomID().Hex()
	}
	ch := s.AddWaiter(msg.RPCID)
	_ = ch

	if err := s.WriteTo(peer, msg.String()); err != nil {
		s.CancelWaiter(msg.RPCID)
		return Message{}, err
	}
	return s.Wait(msg.RPCID, timeout)
}

// Start the server
func (s *UDPServer) Start() error {
	conn, err := net.ListenUDP("udp", s.addr)
	if err != nil {
		panic(err)
	}

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

		if s.deliverToWaiter(msg) {
			continue
		}

		reply, rerr := s.dispatch(peer, msg)
		if rerr != nil {
			fmt.Println("UDP dispatch error:", rerr)
			continue
		}

		if reply != nil {
			if err := s.WriteTo(peer, reply.String()); err != nil {
				fmt.Println("UDP write error:", err)
			}
		}

	}
}

func (s *UDPServer) CancelAllWaiters() {
	s.wmu.Lock()
	for id, w := range s.waiters {
		delete(s.waiters, id)
		close(w.ch)
	}
	s.wmu.Unlock()
}
