package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/t0sic/D7024E-Kademlia/internal/node"
)

type ctxKey int

const nodeKey ctxKey = iota

func withNode(ctx context.Context, n *node.Node) context.Context {
	return context.WithValue(ctx, nodeKey, n)
}

func getNode(cmd *cobra.Command) *node.Node {
	if v := cmd.Context().Value(nodeKey); v != nil {
		if n, ok := v.(*node.Node); ok {
			return n
		}
	}
	return nil
}
