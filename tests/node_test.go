package tests

import (
	"fmt"
	"testing"

	kadnet "github.com/t0sic/D7024E-Kademlia/internal/net"
	"github.com/t0sic/D7024E-Kademlia/internal/node"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

func TestJoinNetwork(t *testing.T) {
	makeMock := func(a string) kadnet.Network { return kadnet.NewMockUDP(a) }

	a := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:10001", NewNet: makeMock, Bootstrap: true,
	})
	t.Logf("Created node a: ID=%s Addr=%s", a.ID.String(), a.Addr)
	b := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:10002", NewNet: makeMock, Peers: []string{a.Addr},
	})
	t.Logf("Created node b: ID=%s Addr=%s", b.ID.String(), b.Addr)
	c := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:10003", NewNet: makeMock, Peers: []string{a.Addr},
	})
	t.Logf("Created node c: ID=%s Addr=%s", c.ID.String(), c.Addr)

	defer a.Server.Close()
	defer b.Server.Close()
	defer c.Server.Close()

	// Loop through all the nodes in the a nodes routing table and assert that b and c id are there
	foundB := false
	foundC := false
	contacts := a.RoutingTable.FindClosestContacts(&a.ID, 20)
	for _, c := range contacts {
		if c.ID.Equals(&b.ID) {
			foundB = true
		}
		if c.ID.Equals(c.ID) {
			foundC = true
		}
	}
	if !foundB {
		t.Fatalf("node a should have b in its routing table")
	}
	if !foundC {
		t.Fatalf("node a should have c in its routing table")
	}
}

func Test50NodesJoinNetwork(t *testing.T) {
	makeMock := func(a string) kadnet.Network { return kadnet.NewMockUDP(a) }

	a := node.CreateNode(node.NodeConfig{
		ID: util.NewRandomID(), Addr: "127.0.0.1:10001", NewNet: makeMock, Bootstrap: true,
	})
	t.Logf("Created node a: ID=%s Addr=%s", a.ID.String(), a.Addr)
	defer a.Server.Close()

	nodes := []*node.Node{a}
	for i := 2; i <= 50; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", 10000+i)
		n := node.CreateNode(node.NodeConfig{
			ID:     util.NewRandomID(),
			Addr:   addr,
			NewNet: makeMock,
			Peers:  []string{a.Addr},
		})
		t.Logf("Created node %d: ID=%s Addr=%s", i, n.ID.String(), n.Addr)
		nodes = append(nodes, n)
	}
	defer func() {
		for _, n := range nodes {
			n.Server.Close()
		}
	}()
	// Loop through all the nodes and assert that each nodes has k cloest contacts in its routing table
	for i, n := range nodes {
		contacts := n.RoutingTable.FindClosestContacts(&n.ID, 20)
		if len(contacts) < 20 && len(nodes) > 20 {
			t.Fatalf("node %d should have at least 20 contacts in its routing table, got %d", i+1, len(contacts))
		}
		if len(contacts) != len(nodes)-1 && len(nodes) <= 20 {
			t.Fatalf("node %d should have %d contacts in its routing table, got %d", i+1, len(nodes)-1, len(contacts))
		}
	}
}
