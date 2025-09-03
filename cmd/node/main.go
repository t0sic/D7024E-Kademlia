package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const ID_BYTES = 20 // 160-bit

type ID [ID_BYTES]byte

func newRandomID() ID {
	var id ID
	_, err := rand.Read(id[:])
	if err != nil {
		log.Fatalf("failed to generate random ID: %v", err)
	}
	return id
}

func (id ID) String() string { return hex.EncodeToString(id[:]) }

func main() {
	id := newRandomID()
	fmt.Printf("Node ID: %s starting...\n", id)

	// Block until stopped
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done

	fmt.Println("Shutting down...")
}
