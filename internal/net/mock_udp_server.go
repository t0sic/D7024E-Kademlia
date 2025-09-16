package net

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

var (
	mockRegMu sync.RWMutex
	mockReg   = map[string]*MockUDP{}
)

type MockUDP struct {
	addr     *net.UDPAddr
	handlers map[string]Handler
}

func NewMockUDP(addr string) *MockUDP {
	udp, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}
	m := &MockUDP{addr: udp, handlers: map[string]Handler{}}

	mockRegMu.Lock()
	mockReg[udp.String()] = m
	mockRegMu.Unlock()

	return m
}

func (m *MockUDP) Addr() *net.UDPAddr { return m.addr }
func (m *MockUDP) Start() error       { return nil }
func (m *MockUDP) Close() error {
	mockRegMu.Lock()
	delete(mockReg, m.addr.String())
	mockRegMu.Unlock()
	return nil
}

func (m *MockUDP) On(typ string, h Handler) {
	m.handlers[strings.ToUpper(strings.TrimSpace(typ))] = h
}

func (m *MockUDP) Send(to *net.UDPAddr, msg Message) error {
	mockRegMu.RLock()
	dst := mockReg[to.String()]
	mockRegMu.RUnlock()
	if dst == nil {
		return fmt.Errorf("no mock peer at %s", to)
	}

	h := dst.handlers[msg.Type]
	if h == nil {
		return nil
	}
	reply, err := h(m.addr, msg)
	if err != nil || reply == nil {
		return err
	}

	if rh := m.handlers[reply.Type]; rh != nil {
		_, _ = rh(dst.addr, *reply)
	}
	return nil
}

func (m *MockUDP) SendAndWait(to *net.UDPAddr, msg Message, timeout time.Duration) (Message, error) {
	// mimic UDP SendAndWait: set an RPCID, call handler, return reply with same RPCID
	if msg.RPCID == "" {
		msg.RPCID = util.NewRandomID().Hex()
	}

	mockRegMu.RLock()
	dst := mockReg[to.String()]
	mockRegMu.RUnlock()
	if dst == nil {
		return Message{}, fmt.Errorf("no mock peer at %s", to)
	}

	h := dst.handlers[msg.Type]
	if h == nil {
		return Message{}, fmt.Errorf("no handler for %s at %s", msg.Type, to)
	}

	reply, err := h(m.addr, msg)
	if err != nil {
		return Message{}, err
	}
	if reply == nil {
		return Message{}, fmt.Errorf("no reply from %s", to)
	}
	// echo RPCID like the real protocol expects
	if reply.RPCID == "" {
		reply.RPCID = msg.RPCID
	}
	return *reply, nil
}
