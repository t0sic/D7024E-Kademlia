package cli

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/t0sic/D7024E-Kademlia/internal/node"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

var (
	flagAddr      string
	flagIDHex     string
	flagIDSeed    string
	flagBootstrap bool
	flagPeersCSV  string

	rootCmd = &cobra.Command{
		Use:   "kad",
		Short: "Kademlia node CLI",
		Long:  "Run a Kademlia node and perform basic RPCs (PING, FIND_NODE).",
	}
)

var cmdTest = &cobra.Command{
	Use:   "test",
	Short: "Test",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Test command executed")
	},
}

func init() {
	// Global flags shared by current/future subcommands
	rootCmd.PersistentFlags().StringVar(&flagAddr, "addr", ":6882", "address to listen on (e.g. :6882 or 127.0.0.1:0)")
	rootCmd.PersistentFlags().StringVar(&flagIDHex, "id", "", "node ID in hex (optional)")
	rootCmd.PersistentFlags().StringVar(&flagIDSeed, "id-seed", "", "seed for ID generation (optional)")
	rootCmd.PersistentFlags().BoolVar(&flagBootstrap, "bootstrap", false, "bootstrap to provided peers (optional)")
	rootCmd.PersistentFlags().StringVar(&flagPeersCSV, "peers", "", "comma-separated list of bootstrap peers (optional)")

	rootCmd.AddCommand(cmdRun)
}

func replRootCmd() *cobra.Command {
	r := &cobra.Command{Use: "kad-repl"}
	r.AddCommand(cmdExit)
	r.AddCommand(cmdTest)
	return r
}

func buildID() util.ID {
	switch {
	case strings.TrimSpace(flagIDHex) != "":
		id, err := util.ParseHexID(flagIDHex)
		if err != nil {
			log.Fatalf("invalid --id: %v", err)
		}
		return id
	case strings.TrimSpace(flagIDSeed) != "":
		return util.NewIDFromSeed(flagIDSeed)
	default:
		return util.NewRandomID()
	}
}

func parsePeers() []string {
	var peers []string
	if s := strings.TrimSpace(flagPeersCSV); s != "" {
		for _, p := range strings.Split(s, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				peers = append(peers, p)
			}
		}
	}
	return peers
}

func newNode() *node.Node {
	cfg := node.NodeConfig{
		ID:        buildID(),
		Addr:      flagAddr,
		Bootstrap: flagBootstrap,
		Peers:     parsePeers(),
	}
	return node.CreateNode(cfg)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
