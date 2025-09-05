package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/t0sic/D7024E-Kademlia/internal/node"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

func main() {
	addr := flag.String("addr", ":6881", "address to listen on")
	idHex := flag.String("id", "", "node ID in hex (optional)")
	bootstrap := flag.Bool("bootstrap", false, "whether to bootstrap to the network (optional)")
	peerCSV := flag.String("peers", "", "comma-separated list of bootstrap peers (optional)")
	idSeed := flag.String("id-seed", "", "seed for node ID generation (optional)")

	flag.Parse()

	// Determine node ID
	var id util.ID
	var err error

	switch {
		case *idHex != "":
			id, err = util.ParseHexID(*idHex)
			if err != nil {
				log.Fatalf("invalid node ID: %v", err)
			}
		case *idSeed != "":
			id = util.NewIDFromSeed(*idSeed)
		default:
			id = util.NewRandomID()
	}

	// Parse Peers
	var peers []string
	if s := strings.TrimSpace(*peerCSV); s != "" {
		for _, p := range strings.Split(s, ",") {
			if p = strings.TrimSpace(p); p != "" {
				peers = append(peers, p)
			}
		}
	}

	var config = node.NodeConfig{
		ID:        id,
		Addr:      *addr,
		Bootstrap: *bootstrap,
		Peers:     peers,
	}

	node.CreateNode(config)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	fmt.Println("Shutting down...")

}
