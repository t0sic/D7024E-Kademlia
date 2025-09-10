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
	Config 	 	 NodeConfig
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
		Config:	   	  config,
	}
	
	node.Server.On("PING", node.HandlePing)
	node.Server.On("PONG", node.HandlePong)
	
	go func() {
		if err := node.Server.Start(); err != nil {
			fmt.Println("UDP server stopped:", err)
		}
	}()
	node.JoinNetwork()

	return node
}

func (n *Node) JoinNetwork() {
	fmt.Println("Joining network...")
	if len(n.Config.Peers) == 0 {
		return
	}

	for _, peer := range n.Config.Peers {
		addr, err := net.ResolveUDPAddr("udp", peer)
		if err != nil {
			fmt.Printf("Error resolving address %s: %v\n", peer, err)
			continue
		}
		// Send PING with our ID so the peer can add us too
		if err := n.Server.Ping(addr, n.ID); err != nil {
			fmt.Printf("Error pinging %s: %v\n", peer, err)
			continue
		}
	}
}


func (n *Node) HandlePong(from *net.UDPAddr, msg kadnet.Message) (*kadnet.Message, error) {
	if len(msg.Args) == 0 {
		return nil, fmt.Errorf("PONG message missing node ID argument")
	}
	fmt.Printf("Received PONG from %s with node ID %s\n", from.String(), msg.Args[0])

	// Add the contact to the routing table
	senderID, err := util.ParseHexID(msg.Args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid node ID in PONG message: %w", err)
	}
	n.RoutingTable.AddContact(kademlia.NewContactWithDistance(&senderID, from, &n.ID))

	return nil, nil // No reply needed for PONG
}

func (n *Node) HandlePing(from *net.UDPAddr, msg kadnet.Message) (*kadnet.Message, error) {
	n.RoutingTable.AddContact(kademlia.NewContactWithDistance(&n.ID, from, &n.ID))

	return &kadnet.Message{
		Type: kadnet.MSG_PONG,
		Args: []string{n.ID.String()},
	}, nil
}
