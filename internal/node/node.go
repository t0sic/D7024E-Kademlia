package node

import (
	"fmt"
	"net"

	kadnet "github.com/t0sic/D7024E-Kademlia/internal/net"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

// TODO: implement node structure and methods

type NodeConfig struct {
	ID       util.ID
	Addr     string
	Bootstrap bool
	Peers    []string
}

type Node struct {
	ID   util.ID
	Addr string
	Server *kadnet.UDPServer
}

func CreateNode(config NodeConfig) *Node {

	fmt.Printf("Node ID: %s starting on %s\n", config.ID, config.Addr)
	if len(config.Peers) > 0 {
		fmt.Printf("Bootstrap peers: %v\n", config.Peers)
	}
	node := &Node{
		ID:   config.ID,
		Addr: config.Addr,
		Server: kadnet.CreateUDPServer(config.Addr),
	}

	node.Server.On("PING", node.HandlePing)

	defer node.Server.Start()

	return node
}

func (n *Node) HandlePing(from *net.UDPAddr, msg kadnet.Message) (*kadnet.Message, error) {
	// Example: reply with PONG and our node ID as an arg
	return &kadnet.Message{
		Type: kadnet.MSG_PONG,
		Args: []string{n.ID.String()},
	}, nil
}
