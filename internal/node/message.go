package node

import (
	"fmt"
	"net"
	"time"

	"github.com/t0sic/D7024E-Kademlia/internal/kademlia"
	kadnet "github.com/t0sic/D7024E-Kademlia/internal/net"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

func (n *Node) PingSync(addr *net.UDPAddr, timeout time.Duration) (util.ID, error) {
	req := kadnet.Message{
		Type: kadnet.MSG_PING,
		Args: []string{n.ID.String()},
	}
	resp, err := n.Server.SendAndWait(addr, req, timeout)
	if err != nil {
		return util.ID{}, err
	}
	if resp.Type != kadnet.MSG_PONG || len(resp.Args) < 1 {
		return util.ID{}, fmt.Errorf("unexpected PONG response: %s %v", resp.Type, resp.Args)
	}
	peerID, err := util.ParseHexID(resp.Args[0])
	if err != nil {
		return util.ID{}, fmt.Errorf("invalid peer ID in PONG: %w", err)
	}
	return peerID, nil
}

func (n *Node) FindNodesSync(addr *net.UDPAddr, fromID, target util.ID, timeout time.Duration) ([]kademlia.Contact, error) {
	req := kadnet.Message{
		Type: kadnet.MSG_FIND_NODE,
		Args: []string{fromID.String(), target.String()},
	}
	resp, err := n.Server.SendAndWait(addr, req, timeout)
	if err != nil {
		return nil, err
	}
	if resp.Type != kadnet.MSG_NODES || len(resp.Args) < 1 {
		return nil, fmt.Errorf("bad NODES response")
	}
	// resp.Args: [responderID, id@host:port, ...]
	contacts := make([]kademlia.Contact, 0, len(resp.Args)-1)
	tgt := target // distance relative to lookup target
	for _, tok := range resp.Args[1:] {
		if c, err := kademlia.DecodeContactTokenWithDistance(tok, &tgt); err == nil {
			contacts = append(contacts, *c)
		}
	}
	return contacts, nil
}
