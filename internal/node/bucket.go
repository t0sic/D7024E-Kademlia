package node

import "container/list"

// bucket definition
// contains a List
type bucket struct {
	list *list.List
}
