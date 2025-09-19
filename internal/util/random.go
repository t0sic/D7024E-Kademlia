package util

import (
	"hash/fnv"
	"math/rand"
	"sync"
)

var (
	chanceMu  sync.Mutex
	chanceRng = map[string]*rand.Rand{}
)

// Chance returns true with the given percentage probability (0–100).
// For example: Chance(25) ~25% chance of true.
func Chance(percentage int, seed string) bool {
	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}

	chanceMu.Lock()
	r := chanceRng[seed]
	if r == nil {
		h := fnv.New64a()
		_, _ = h.Write([]byte(seed))
		r = rand.New(rand.NewSource(int64(h.Sum64())))
		chanceRng[seed] = r
	}
	// Use the RNG and advance its state
	n := r.Intn(100)
	chanceMu.Unlock()

	return n < percentage
}
