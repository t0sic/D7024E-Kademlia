package node

import (
	"fmt"
	"net"

	"github.com/t0sic/D7024E-Kademlia/internal/kademlia"
	kadnet "github.com/t0sic/D7024E-Kademlia/internal/net"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

// TODO: implement node structure and methods

type NodeConfig struct {
	ID        util.ID
	Addr      string
	Bootstrap bool
	Peers     []string
}

type Node struct {
	ID           util.ID
	Addr         string
	Server       *kadnet.UDPServer
	RoutingTable *kademlia.RoutingTable
}

func CreateNode(config NodeConfig) *Node {

	fmt.Printf("Node ID: %s starting on %s\n", config.ID, config.Addr)
	if len(config.Peers) > 0 {
		fmt.Printf("Bootstrap peers: %v\n", config.Peers)
	}

	var contact kademlia.Contact = kademlia.NewContact(&config.ID, &net.UDPAddr{IP: net.ParseIP(config.Addr)})

	node := &Node{
		ID:           config.ID,
		Addr:         config.Addr,
		Server:       kadnet.CreateUDPServer(config.Addr),
		RoutingTable: kademlia.CreateRoutingTable(contact),
	}

	node.Server.On("PING", node.HandlePing)

	defer node.Server.Start()

	return node
}

func (n *Node) JoinNetwork(peers []string) {
	for _, peer := range peers {
		err := n.Server.Ping(&net.UDPAddr{IP: net.ParseIP(peer)})
		if err != nil {
			fmt.Printf("Error pinging peer %s: %v\n", peer, err)
		} else {
			var contact kademlia.Contact = kademlia.NewContactWithDistance(&n.ID, &net.UDPAddr{IP: net.ParseIP(peer)}, &n.ID)
			n.RoutingTable.AddContact(contact)
		}
	}
}

func (n *Node) HandlePing(from *net.UDPAddr, msg kadnet.Message) (*kadnet.Message, error) {
	// Example: reply with PONG and our node ID as an arg
	n.RoutingTable.AddContact(kademlia.NewContactWithDistance(&n.ID, from, &n.ID))

	return &kadnet.Message{
		Type: kadnet.MSG_PONG,
		Args: []string{n.ID.String()},
	}, nil
}
