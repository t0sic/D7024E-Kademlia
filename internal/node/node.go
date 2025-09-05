package node

import (
	"fmt"

	"github.com/t0sic/D7024E-Kademlia/internal/net"
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
	Server *net.UDPServer
}

func CreateNode(config NodeConfig) *Node {

	fmt.Printf("Node ID: %s starting on %s\n", config.ID, config.Addr)
	if len(config.Peers) > 0 {
		fmt.Printf("Bootstrap peers: %v\n", config.Peers)
	}

	server := net.CreateUDPServer(config.Addr)
	defer net.StartUDPServer(server)

	return &Node{
		ID:   config.ID,
		Addr: config.Addr,
		Server: server,
	}
}
