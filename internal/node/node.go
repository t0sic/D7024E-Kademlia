package node

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/t0sic/D7024E-Kademlia/internal/kademlia"
	kadnet "github.com/t0sic/D7024E-Kademlia/internal/net"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

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
	// simple local storage for PUT/STORE operations (in-memory)
	store   map[string][]byte
	storeMu sync.RWMutex
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
	node.Server.On(kadnet.MSG_STORE, node.HandleStore)
	node.Server.On(kadnet.MSG_GET, node.HandleGet)

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

		// Add bootstrap peer explicitly to routing table
		n.AddContact(kademlia.NewContactWithDistance(&n.ID, addr, &peerID))

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

// handle store
func (n *Node) HandleStore(from *net.UDPAddr, msg kadnet.Message) (*kadnet.Message, error) {
	if len(msg.Args) < 3 {
		return nil, fmt.Errorf("STORE missing args: want <fromID> <keyHex> <valueHex>")
	}

	fromID, err := util.ParseHexID(msg.Args[0])
	if err != nil {
		return nil, fmt.Errorf("STORE bad fromID: %w", err)
	}

	_ = fromID

	keyHex := msg.Args[1]
	valHex := msg.Args[2]

	value, err := hex.DecodeString(valHex)
	if err != nil {
		return nil, fmt.Errorf("STORE value not hex: %w", err)
	}

	n.storeMu.Lock()
	if n.store == nil {
		n.store = make(map[string][]byte)
	}
	buf := make([]byte, len(value))
	copy(buf, value)
	n.store[keyHex] = buf
	n.storeMu.Unlock()

	// Ack
	return &kadnet.Message{
		Type:  kadnet.MSG_STORED,
		RPCID: msg.RPCID,
		Args:  []string{n.ID.String(), keyHex},
	}, nil
}

func (n *Node) HandleGet(from *net.UDPAddr, msg kadnet.Message) (*kadnet.Message, error) {
	fmt.Println("Received GET from", from.String(), "with args:", msg.Args)
	if len(msg.Args) < 2 {
		return nil, fmt.Errorf("GET missing args: want <fromID> <keyHex>")
	}
	keyHex := msg.Args[1]

	n.storeMu.RLock()
	val, ok := n.store[keyHex]
	n.storeMu.RUnlock()

	if !ok {
		return &kadnet.Message{
			Type:  kadnet.MSG_NOT_FOUND,
			RPCID: msg.RPCID,
			Args:  []string{n.ID.String(), keyHex},
		}, nil
	}

	return &kadnet.Message{
		Type:  kadnet.MSG_VALUE,
		RPCID: msg.RPCID,
		Args:  []string{n.ID.String(), keyHex, hex.EncodeToString(val)},
	}, nil
}

func (n *Node) AddContact(c kademlia.Contact) {
	evictCandidate := n.RoutingTable.AddContact(c)
	if evictCandidate != nil {
		_, err := n.PingSync(&evictCandidate.Address, 800*time.Millisecond)
		if err != nil {
			n.RoutingTable.RemoveContact(*evictCandidate)
			n.RoutingTable.AddContact(c)
			fmt.Printf("PING -> %s failed: %v\n", evictCandidate.Address.String(), err)
		}
	}
}

// Put stores the provided data locally and returns the SHA-1 hash bytes for the stored value.
func (n *Node) Put(data []byte) ([]byte, error) {
	fmt.Println("Recieved PUT with data length:", len(data))
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot store empty data")
	}

	// 1) key = sha1(data) as hex
	h := sha1.Sum(data)
	keyHex := hex.EncodeToString(h[:])

	// 2) lookup k-closest to key
	keyID, err := util.ParseHexID(keyHex)
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}
	timeout := 800 * time.Millisecond
	closest := n.IterativeFindNode(keyID, timeout)

	// 3) send STORE to each
	for _, c := range closest {
		_ = n.SendStoreSync(c, keyHex, data, timeout)
	}

	// 4) also store locally
	n.storeMu.Lock()
	if n.store == nil {
		n.store = make(map[string][]byte)
	}
	b := make([]byte, len(data))
	copy(b, data)
	n.store[keyHex] = b
	n.storeMu.Unlock()

	// 5) return same as before so CLI prints hex
	return h[:], nil
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
				queried[c.ID.String()] = true
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

// IterativeFindValue returnerar (value, fromContact, error).
// fromContact == nil betyder att värdet hittades lokalt.
func (n *Node) IterativeFindValue(keyID util.ID, perNodeTimeout time.Duration) ([]byte, *kademlia.Contact, error) {
	keyHex := keyID.String()

	// 1) Hämta k-närmsta via din befintliga iterative FIND_NODE
	closest := n.IterativeFindNode(keyID, perNodeTimeout)
	if len(closest) == 0 {
		return nil, nil, fmt.Errorf("no closest contacts for %s", keyHex)
	}

	// 2) Fråga i ALPHA-vågor parallellt. Bryt på första VALUE.
	for i := 0; i < len(closest); i += kademlia.ALPHA {
		end := i + kademlia.ALPHA
		if end > len(closest) {
			end = len(closest)
		}
		batch := closest[i:end]

		type res struct {
			val  []byte
			from kademlia.Contact
			ok   bool
			err  error
		}
		resCh := make(chan res, len(batch))
		var wg sync.WaitGroup

		for _, c := range batch {
			wg.Go(func() {
				val, found, err := n.SendGetSync(c, keyHex, perNodeTimeout)
				if err != nil {
					resCh <- res{nil, c, false, err}
					return
				}
				if found {
					resCh <- res{val, c, true, nil}
					return
				}
				resCh <- res{nil, c, false, nil}
			})
		}

		go func() { wg.Wait(); close(resCh) }()

		for r := range resCh {
			if r.ok && r.err == nil {
				return r.val, &r.from, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("value %s not found", keyHex)
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

func (n *Node) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		_ = n.Server.Close() // Close waits for the listener goroutine to finish
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
