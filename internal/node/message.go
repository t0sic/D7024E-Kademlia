package node

import (
	"encoding/hex"
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

func (n *Node) SendGetSync(to kademlia.Contact, keyHex string, timeout time.Duration) ([]byte, bool, error) {
	req := kadnet.Message{
		Type: kadnet.MSG_GET,
		Args: []string{
			n.ID.String(),
			keyHex,
		},
	}
	resp, err := n.Server.SendAndWait(&to.Address, req, timeout)
	if err != nil {
		return nil, false, err
	}

	switch resp.Type {
	case kadnet.MSG_VALUE:
		if len(resp.Args) < 3 {
			return nil, false, fmt.Errorf("VALUE malformed response")
		}
		valHex := resp.Args[2]
		b, err := hex.DecodeString(valHex)
		if err != nil {
			return nil, false, fmt.Errorf("VALUE not hex: %w", err)
		}
		return b, true, nil
	case kadnet.MSG_NOT_FOUND:
		return nil, false, nil
	default:
		return nil, false, fmt.Errorf("unexpected response type %q", resp.Type)
	}
}

func (n *Node) SendStoreSync(to kademlia.Contact, keyHex string, value []byte, timeout time.Duration) error {
	valHex := hex.EncodeToString(value)
	msg := kadnet.Message{
		Type: kadnet.MSG_STORE,
		Args: []string{
			n.ID.String(),
			keyHex,
			valHex,
		},
	}

	_, err := n.Server.SendAndWait(&to.Address, msg, timeout)
	return err
}
