package tests

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	kadnet "github.com/t0sic/D7024E-Kademlia/internal/net"
	"github.com/t0sic/D7024E-Kademlia/internal/node"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

const NODE_COUNT = 1000
const DROP_PERCENTAGE = 60
const SEED = "King Solomon"
const BASE = 10000

func TestPackageLoss(t *testing.T) {
	makeMock := func(a string) kadnet.Network { return kadnet.NewMockUDP(a) }

	// Bootstrap on BASE
	bootstrap := node.CreateNode(node.NodeConfig{
		ID:        util.NewIDFromSeed(SEED),
		Addr:      fmt.Sprintf("127.0.0.1:%d", BASE),
		NewNet:    makeMock,
		Bootstrap: true,
	})
	defer bootstrap.Shutdown(context.Background())

	nodes := []*node.Node{}
	// start from BASE+1 to avoid clobbering bootstrap
	for i := 1; i <= NODE_COUNT; i++ {
		n := node.CreateNode(node.NodeConfig{
			ID:     util.NewIDFromSeed(SEED + fmt.Sprintf("%d", i)),
			Addr:   fmt.Sprintf("127.0.0.1:%d", BASE+i), // 10001..11000
			NewNet: makeMock,
			Peers:  []string{bootstrap.Addr},
		})
		nodes = append(nodes, n)
	}

	// Allow join/RT exchange to settle a bit for large N
	time.Sleep(300 * time.Millisecond)

	// Store
	hash, err := bootstrap.Put([]byte("Hello"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Drop some nodes (0% here so none will drop)
	for i := 0; i < len(nodes); i++ {
		if util.Chance(DROP_PERCENTAGE, SEED) {
			nodes[i].Shutdown(context.Background())
		}
	}

	// Lookup
	key, err := util.ParseHexID(hex.EncodeToString(hash[:]))
	if err != nil {
		t.Fatalf("Failed to parse hash: %v", err)
	}

	// If your IterativeFindValue needs context, use it; otherwise keep as-is.
	val, _, err := bootstrap.IterativeFindValue(key, 800*time.Millisecond)
	if err != nil {
		t.Fatalf("IterativeFindValue failed: %v", err)
	}
	if string(val) != "Hello" {
		t.Fatalf("Value mismatch: got %q want %q", string(val), "Hello")
	}
}
