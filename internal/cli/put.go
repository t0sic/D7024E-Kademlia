package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var cmdPut = &cobra.Command{
	Use:   "put <data|@/path/to/file|->",
	Short: "Store bytes and print their SHA-1 content hash",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		n := getNode(cmd)
		if n == nil {
			return fmt.Errorf("no running node in context; start with 'run'")
		}

		// read payload from arg (literal, @file, or - for stdin)
		payload, err := readSingleArg(args[0])
		if err != nil {
			return err
		}

		hash, err := n.Put(payload)
		if err != nil {
			return err
		}

		// Print ONLY the hex hash
		fmt.Printf("%x\n", hash)
		return nil
	},
}

// readSingleArg supports:
//
//	put hello
//	put @/path/to/file
//	put -   (read from stdin)
func readSingleArg(arg string) ([]byte, error) {
	switch {
	case arg == "-":
		return io.ReadAll(bufio.NewReader(os.Stdin))
	case strings.HasPrefix(arg, "@"):
		path := strings.TrimPrefix(arg, "@")
		return os.ReadFile(path)
	default:
		return []byte(arg), nil
	}
}
