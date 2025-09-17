package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/spf13/cobra"
)

// in run.go
var cmdRun = &cobra.Command{
	Use:   "run",
	Short: "Start a Kademlia node",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1) Always start the node
		n := newNode() // starts UDP server, may bootstrap, etc.
		fmt.Printf("Node ID: %s starting on %s\n", n.ID, n.Addr)

		// Ensure graceful shutdown
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = n.Shutdown(ctx)
		}()

		// 2) Decide how to wait: REPL (interactive) or signals (headless)
		isInteractive := term.IsTerminal(int(os.Stdin.Fd()))
		if !isInteractive {
			// Headless under Docker/Compose: block until SIGINT/SIGTERM
			sigc := make(chan os.Signal, 1)
			signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
			<-sigc
			fmt.Println("Signal received; shutting down node…")
			return nil
		}

		// 3) REPL path (interactive)
		ctx := withNode(cmd.Context(), n)
		fmt.Println("Interactive mode. Type 'help' or 'exit'.")
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("kad> ")
			line, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					fmt.Println()
					break
				}
				return err
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			args := strings.Fields(line)
			replRoot := replRootCmd() // root without "run"
			replRoot.SetArgs(args)
			if err := replRoot.ExecuteContext(ctx); err != nil {
				if errors.Is(err, errREPLExit) {
					break
				}
				fmt.Fprintln(os.Stderr, "error:", err)
			}
		}
		fmt.Println("Exiting REPL and shutting down node…")
		return nil
	},
}
