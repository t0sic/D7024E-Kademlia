// tests/pingpong_mock_test.go
package tests

import (
	"net"
	"testing"
	"time"

	kadnet "github.com/t0sic/D7024E-Kademlia/internal/net"
	"github.com/t0sic/D7024E-Kademlia/internal/node"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

func TestPingPongWithMock(t *testing.T) {
	makeMock := func(a string) kadnet.Network { return kadnet.NewMockUDP(a) }

	a := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:10001", NewNet: makeMock,
	})
	t.Logf("Created node a: ID=%s Addr=%s", a.ID.String(), a.Addr)
	b := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:10002", NewNet: makeMock,
	})
	t.Logf("Created node b: ID=%s Addr=%s", b.ID.String(), b.Addr)

	defer a.Server.Close()
	defer b.Server.Close()

	peer, _ := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	t.Logf("Resolved peer address: %s", peer.String())
	id, err := b.PingSync(peer, 500*time.Millisecond)
	t.Logf("PingSync returned id=%s, err=%v", id.String(), err)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if id == (util.ID{}) {
		t.Fatalf("empty id from pong")
	}
	t.Logf("PingPong test successful: received pong with id=%s", id.String())
}
