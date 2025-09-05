package net

import (
	"net"
	"testing"
	"time"
)

// wait until the server has bound a real socket
func waitStarted(t *testing.T, s *UDPServer) *net.UDPAddr {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s != nil && s.conn != nil {
			if la, ok := s.conn.LocalAddr().(*net.UDPAddr); ok && la.Port != 0 {
				return la
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not start in time")
	return nil
}

func TestPingPong(t *testing.T) {
	// 1) create on ephemeral port; prefer loopback explicitly
	s := CreateUDPServer("127.0.0.1:0")

	// 2) start the blocking loop in a goroutine
	go StartUDPServer(s)

	// 3) wait for it to bind and expose the actual addr
	addr := waitStarted(t, s)

	// 4) dial and talk UDP
	c, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("DialUDP failed: %v", err)
	}
	defer c.Close()

	_ = c.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if _, err := c.Write([]byte("PING")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	buf := make([]byte, 64)
	_ = c.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, _, err := c.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if got := string(buf[:n]); got != "PONG" {
		t.Fatalf("expected PONG, got %q", got)
	}

	// 5) stop the server so the goroutine exits
	_ = s.conn.Close()
}

func TestTwoNodesPingPong(t *testing.T) {
	// Start Node A
	nodeA := CreateUDPServer("127.0.0.1:0")
	go StartUDPServer(nodeA)
	addrA := waitStarted(t, nodeA)
	defer nodeA.Close()

	// Start Node B
	nodeB := CreateUDPServer("127.0.0.1:0")
	go StartUDPServer(nodeB)
	addrB := waitStarted(t, nodeB)
	defer nodeB.Close()

	// Node A dials Node B and sends PING
	conn, err := net.DialUDP("udp", nil, addrB)
	if err != nil {
		t.Fatalf("DialUDP failed: %v", err)
	}
	defer conn.Close()

	_ = conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if _, err := conn.Write([]byte("PING")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	buf := make([]byte, 64)
	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if got := string(buf[:n]); got != "PONG" {
		t.Fatalf("expected PONG from nodeB, got %q", got)
	}

	t.Logf("Node A (%s) got %q from Node B (%s)", addrA, "PONG", addrB)
}
