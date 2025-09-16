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

		// print result save it in a variable if needed
		contacts := n.IterativeFindNode(n.ID, 800*time.Millisecond)

		// print contacts
		for _, c := range contacts {
			fmt.Printf("Found contact: %s\n", c.String())
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

	n.AddContact(kademlia.NewContactWithDistance(&n.ID, from, &fromID))
	shortlist := n.RoutingTable.FindClosestContacts(&targetID, kademlia.K)

	// Build: NODES <myID> <id@host:port>...
	args := []string{n.ID.String()}
	for _, c := range shortlist {
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
	n.AddContact(kademlia.NewContactWithDistance(&n.ID, from, &senderID))

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

func (n *Node) AddContact(c kademlia.Contact) {
	contact := n.RoutingTable.AddContact(c)
	if contact != nil {
		_, err := n.PingSync(&c.Address, 800*time.Millisecond)
		if err != nil {
			n.RoutingTable.RemoveContact(*contact)
			n.RoutingTable.AddContact(c)
			fmt.Printf("PING -> %s failed: %v\n", c.Address.String(), err)
		}

	}
}

// IterativeFindNode runs the Kademlia iterative FIND_NODE lookup.
// Returns up to kademlia.K closest contacts to the target.
func (n *Node) IterativeFindNode(target util.ID, timeout time.Duration) []kademlia.Contact {
	// Start shortlist from routing table
	shortlist := n.RoutingTable.FindClosestContacts(&target, kademlia.K)

	// Track queried nodes
	queried := make(map[string]bool)

	progress := true
	for progress {
		progress = false

		// Select up to ALPHA closest unqueried contacts
		batch := make([]kademlia.Contact, 0, kademlia.ALPHA)
		for _, c := range shortlist {
			if len(batch) >= kademlia.ALPHA {
				break
			}
			if !queried[c.ID.String()] {
				batch = append(batch, c)
			}
		}

		// If no new nodes to query, we’re done
		if len(batch) == 0 {
			break
		}

		// Query them in parallel
		type result struct {
			from     kademlia.Contact
			contacts []kademlia.Contact
			err      error
		}
		results := make(chan result, len(batch))
		for _, c := range batch {
			go func(c kademlia.Contact) {
				queried[c.ID.String()] = true
				contacts, err := n.FindNodesSync(&c.Address, n.ID, target, timeout)
				if err != nil {
					results <- result{from: c, contacts: nil, err: err}
					return
				}
				results <- result{from: c, contacts: contacts, err: nil}
			}(c)
		}

		// Collect results
		for i := 0; i < len(batch); i++ {
			res := <-results
			if res.err != nil {
				continue
			}
			for _, c := range res.contacts {
				c.CalcDistance(&target)
				n.AddContact(c) // maintain table
				// Add if not seen
				if !contains(shortlist, c) {
					shortlist = append(shortlist, c)
					progress = true
				}
			}
		}

		// Sort shortlist by distance to target
		cand := kademlia.ContactCandidates{}
		cand.Append(shortlist)
		cand.Sort()
		shortlist = cand.GetContacts(kademlia.K)
	}

	return shortlist
}

// contains checks if a Contact with same ID exists in slice
func contains(list []kademlia.Contact, c kademlia.Contact) bool {
	for _, x := range list {
		if x.ID.Equals(c.ID) {
			return true
		}
	}
	return false
}
