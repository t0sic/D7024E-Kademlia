package node

import (
	"fmt"
	"net"
	"time"

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
	NewNet    func(addr string) kadnet.Network
}

type Node struct {
	ID           util.ID
	Addr         string
	Server       kadnet.Network
	RoutingTable *kademlia.RoutingTable
	Config       NodeConfig
}

func CreateNode(config NodeConfig) *Node {

	fmt.Printf("Node ID: %s starting on %s\n", config.ID, config.Addr)
	if len(config.Peers) > 0 {
		fmt.Printf("Bootstrap peers: %v\n", config.Peers)
	}

	newNet := config.NewNet
	if newNet == nil {
		newNet = func(a string) kadnet.Network { return kadnet.CreateUDPServer(a) }
	}

	udpAddr, err := net.ResolveUDPAddr("udp", config.Addr)
	if err != nil {
		panic(fmt.Errorf("resolve local addr %q: %w", config.Addr, err))
	}
	var contact kademlia.Contact = kademlia.NewContact(&config.ID, udpAddr)

	node := &Node{
		ID:           config.ID,
		Addr:         config.Addr,
		Server:       newNet(config.Addr),
		RoutingTable: kademlia.CreateRoutingTable(contact),
		Config:       config,
	}

	node.Server.On(kadnet.MSG_PING, node.HandlePing)
	node.Server.On(kadnet.MSG_PONG, node.HandlePong)
	node.Server.On(kadnet.MSG_FIND_NODE, node.HandleFindNode)

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
			fmt.Printf("Resolve %s: %v\n", peer, err)
			continue
		}

		peerID, err := n.PingSync(addr, 800*time.Millisecond)
		if err != nil {
			fmt.Printf("PING -> %s failed: %v\n", peer, err)
			continue
		}
		fmt.Printf("Peer %s is alive (id=%s)\n", peer, peerID.String())

		contacts, ferr := n.FindNodesSync(addr, n.ID, n.ID, 800*time.Millisecond)
		if ferr != nil {
			fmt.Printf("FIND_NODE -> %s failed: %v\n", peer, ferr)
			continue
		}

		for _, c := range contacts {
			n.RoutingTable.AddContact(c)
		}

	}
}

// FIND_NODE <fromID> <targetID>
func (n *Node) HandleFindNode(from *net.UDPAddr, msg kadnet.Message) (*kadnet.Message, error) {
	fmt.Println("Received FIND_NODE from", from.String(), "with args:", msg.Args)
	if len(msg.Args) == 0 {
		return nil, fmt.Errorf("FIND_NODE message missing target ID argument")
	}

	fromID, err := util.ParseHexID(msg.Args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid node ID in FIND_NODE message: %w", err)
	}

	targetID, err := util.ParseHexID(msg.Args[1])
	if err != nil {
		return nil, fmt.Errorf("invalid target ID in FIND_NODE message: %w", err)
	}

	n.RoutingTable.AddContact(kademlia.NewContactWithDistance(&n.ID, from, &fromID))
	closest := n.RoutingTable.FindClosestContacts(&targetID, kademlia.K)

	// Build: NODES <myID> <id@host:port>...
	args := []string{n.ID.String()}
	for _, c := range closest {
		// skip the requester and self
		if c.ID != nil && (*c.ID == fromID || *c.ID == n.ID) {
			continue
		}
		args = append(args, kademlia.EncodeContactToken(c))
	}

	return &kadnet.Message{
		Type:  kadnet.MSG_NODES,
		RPCID: msg.RPCID,
		Args:  args,
	}, nil
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
	n.RoutingTable.AddContact(kademlia.NewContactWithDistance(&n.ID, from, &senderID))

	return nil, nil // No reply needed for PONG
}

func (n *Node) HandlePing(from *net.UDPAddr, msg kadnet.Message) (*kadnet.Message, error) {
	if len(msg.Args) == 0 {
		return nil, fmt.Errorf("PING missing sender ID")
	}
	_, err := util.ParseHexID(msg.Args[0])
	if err != nil {
		return nil, fmt.Errorf("PING bad sender ID: %w", err)
	}

	return &kadnet.Message{
		Type:  kadnet.MSG_PONG,
		RPCID: msg.RPCID,
		Args:  []string{n.ID.String()},
	}, nil

}
