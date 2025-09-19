package tests

import (
	"context"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/t0sic/D7024E-Kademlia/internal/kademlia"
	kadnet "github.com/t0sic/D7024E-Kademlia/internal/net"
	"github.com/t0sic/D7024E-Kademlia/internal/node"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

// TestStoreAndRetrieve stores a value on node A and verifies node B can get it.
func TestStoreAndRetrieve(t *testing.T) {
	makeMock := func(a string) kadnet.Network { return kadnet.NewMockUDP(a) }
	a := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:22001", NewNet: makeMock, Bootstrap: true,
	})
	defer a.Server.Close()
	b := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:22002", NewNet: makeMock, Peers: []string{a.Addr},
	})
	defer b.Server.Close()
	time.Sleep(100 * time.Millisecond)

	key := util.NewRandomID()
	value := []byte("Hello, StoreAndRetrieve")
	msg := kadnet.Message{
		Type: kadnet.MSG_STORE,
		Args: []string{a.ID.String(), key.String(), hex.EncodeToString(value)},
	}
	if _, err := a.HandleStore(nil, msg); err != nil {
		t.Fatalf("HandleStore failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	val, _, err := b.IterativeFindValue(context.Background(), key, 800*time.Millisecond)
	if err != nil || val == nil {
		t.Fatalf("B failed to find value: err=%v val=%v", err, val)
	}
	if string(val) != string(value) {
		t.Fatalf("value mismatch: got %s want %s", string(val), string(value))
	}
}

// TestRemoveNodeWithStoredValue stores a value on node A, removes A from B's routing table
// (and shuts down A), then asserts B can no longer find the value via IterativeFindValue.
func TestRemoveNodeWithStoredValue(t *testing.T) {
	makeMock := func(a string) kadnet.Network { return kadnet.NewMockUDP(a) }
	a := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:21001", NewNet: makeMock, Bootstrap: true,
	})
	defer a.Server.Close()
	b := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:21002", NewNet: makeMock, Peers: []string{a.Addr},
	})
	defer b.Server.Close()

	// store a value on A via HandleStore (simulate incoming RPC)
	key := util.NewRandomID()
	value := []byte("Goodbye, Node A")
	msg := kadnet.Message{
		Type: kadnet.MSG_STORE,
		Args: []string{a.ID.String(), key.String(), hex.EncodeToString(value)},
	}
	if _, err := a.HandleStore(nil, msg); err != nil {
		t.Fatalf("HandleStore failed: %v", err)
	}

	// ensure B can find the value initially
	time.Sleep(50 * time.Millisecond)
	if val, _, err := b.IterativeFindValue(context.Background(), key, 800*time.Millisecond); err != nil || val == nil {
		t.Fatalf("Expected B to find value before removal, err=%v val=%v", err, val)
	}

	// Remove A: close its server and remove contact from B's routing table
	_ = a.Server.Close()

	// Construct a contact with A's ID to remove from B's table
	addr := &net.UDPAddr{IP: nil, Port: 0}
	c := kademlia.NewContact(&a.ID, addr)
	b.RoutingTable.RemoveContact(c)

	time.Sleep(50 * time.Millisecond)
	if _, _, err := b.IterativeFindValue(context.Background(), key, 200*time.Millisecond); err == nil {
		t.Fatalf("Expected B to NOT find value after A removal, but lookup succeeded")
	}
}
