package kademlia

import (
	"sync"

	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

const K = 20
const ALPHA = 3

// RoutingTable definition
// keeps a refrence contact of me and an array of buckets
type RoutingTable struct {
	mu      sync.RWMutex
	me      Contact
	buckets [util.IDBytes * 8]*Bucket
}

// createRoutingTable returns a new instance of a RoutingTable
func CreateRoutingTable(me Contact) *RoutingTable {
	routingTable := &RoutingTable{}
	for i := 0; i < util.IDBytes*8; i++ {
		routingTable.buckets[i] = newBucket()
	}
	routingTable.me = me
	return routingTable
}

// AddContact add a new contact to the correct Bucket
func (routingTable *RoutingTable) AddContact(contact Contact) *Contact {
	routingTable.mu.Lock()
	defer routingTable.mu.Unlock()
	bucketIndex := routingTable.getBucketIndex(contact.ID)
	bucket := routingTable.buckets[bucketIndex]

	if !bucket.isFull() {
		bucket.AddContact(contact)
		return nil
	}
	return bucket.GetLeastRecentlySeen()
}

// RemoveContact removes a contact from the correct Bucket
func (routingTable *RoutingTable) RemoveContact(contact Contact) {
	routingTable.mu.Lock()
	defer routingTable.mu.Unlock()
	bucketIndex := routingTable.getBucketIndex(contact.ID)
	bucket := routingTable.buckets[bucketIndex]

	bucket.RemoveContact(contact)
}

// FindClosestContacts finds the count closest Contacts to the target in the RoutingTable
func (routingTable *RoutingTable) FindClosestContacts(target *util.ID, count int) []Contact {
	var candidates ContactCandidates
	bucketIndex := routingTable.getBucketIndex(target)
	bucket := routingTable.buckets[bucketIndex]

	routingTable.mu.RLock()
	candidates.Append(bucket.GetContactAndCalcDistance(target))

	for i := 1; (bucketIndex-i >= 0 || bucketIndex+i < util.IDBytes*8) && candidates.Len() < count; i++ {
		if bucketIndex-i >= 0 {
			bucket = routingTable.buckets[bucketIndex-i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
		if bucketIndex+i < util.IDBytes*8 {
			bucket = routingTable.buckets[bucketIndex+i]
			candidates.Append(bucket.GetContactAndCalcDistance(target))
		}
	}
	routingTable.mu.RUnlock()

	candidates.Sort()

	if count > candidates.Len() {
		count = candidates.Len()
	}

	return candidates.GetContacts(count)
}

// getBucketIndex get the correct Bucket index for the KademliaID
func (routingTable *RoutingTable) getBucketIndex(id *util.ID) int {
	distance := id.CalcDistance(routingTable.me.ID)
	for i := 0; i < util.IDBytes; i++ {
		for j := 0; j < 8; j++ {
			if (distance[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}

	return util.IDBytes*8 - 1
}
