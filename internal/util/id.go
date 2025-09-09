package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const IDBytes = 20 // 160-bit

type ID [IDBytes]byte

// NewRandomID returns a cryptographically random 160-bit ID.
func NewRandomID() ID {
	var id ID
	if _, err := rand.Read(id[:]); err != nil {
		panic(fmt.Errorf("rand.Read: %w", err))
	}
	return id
}

// NewIDFromSeed deterministically derives a 160-bit ID from a seed string.
func NewIDFromSeed(seed string) ID {
	sum := sha256.Sum256([]byte(seed)) // 32 bytes
	var id ID
	copy(id[:], sum[:IDBytes]) // take first 20 bytes
	return id
}

// ParseHexID parses a 40-char hex string into an ID.
func ParseHexID(h string) (ID, error) {
	var id ID
	b, err := hex.DecodeString(h)
	if err != nil {
		return id, fmt.Errorf("decode hex: %w", err)
	}
	if len(b) != IDBytes {
		return id, fmt.Errorf("wrong length: got %d, want %d", len(b), IDBytes)
	}
	copy(id[:], b)
	return id, nil
}

// Hex returns the lowercase hex encoding of the ID.
func (id ID) Hex() string { return hex.EncodeToString(id[:]) }

// String implements fmt.Stringer (prints hex).
func (id ID) String() string { return id.Hex() }

// Less returns true if id < otherKademliaID (bitwise)
func (id ID) Less(otherKademliaID *ID) bool {
	for i := 0; i < IDBytes; i++ {
		if id[i] != otherKademliaID[i] {
			return id[i] < otherKademliaID[i]
		}
	}
	return false
}

// Equals returns true if id == otherKademliaID (bitwise)
func (id ID) Equals(otherKademliaID *ID) bool {
	for i := 0; i < IDBytes; i++ {
		if id[i] != otherKademliaID[i] {
			return false
		}
	}
	return true
}

// CalcDistance returns a new instance of a ID that is built
// through a bitwise XOR operation betweeen id and target
func (id ID) CalcDistance(target *ID) *ID {
	result := ID{}
	for i := 0; i < IDBytes; i++ {
		result[i] = id[i] ^ target[i]
	}
	return &result
}
