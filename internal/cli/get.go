package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

var cmdGet = &cobra.Command{
	Use:   "get <sha1-hex>",
	Short: "Fetch bytes by SHA-1 content hash and print source node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		n := getNode(cmd)
		if n == nil {
			return fmt.Errorf("no running node in context; start with 'run'")
		}

		keyHex := args[0]
		keyID, err := util.ParseHexID(keyHex)
		if err != nil {
			return fmt.Errorf("invalid hash: %w", err)
		}

		val, from, err := n.IterativeFindValue(keyID, 800*time.Millisecond)
		if err != nil {
			return err
		}

		fmt.Printf("hash: %s\n", keyHex)
		if from == nil {
			fmt.Printf("node: local (%s)\n", n.ID.String())
		} else {
			fmt.Printf("node: %s\n", from.String())
		}
		fmt.Println("--- content ---")
		os.Stdout.Write(val)
		if len(val) == 0 || val[len(val)-1] != '\n' {
			fmt.Println()
		}
		return nil
	},
}
