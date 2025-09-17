package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var errREPLExit = fmt.Errorf("repl-exit")

var cmdExit = &cobra.Command{
	Use:     "exit",
	Aliases: []string{"quit"},
	Short:   "Stop the node (prints the node ID, then exits the REPL)",
	RunE: func(cmd *cobra.Command, args []string) error {
		n := getNode(cmd)
		if n == nil {
			fmt.Println("no running node in context; start with 'run'")
			return errREPLExit
		}
		fmt.Printf("Node ID: %s stopping on %s\n", n.ID, n.Addr)
		return errREPLExit
	},
}
