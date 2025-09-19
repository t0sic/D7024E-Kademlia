package kademlia

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/t0sic/D7024E-Kademlia/internal/util"
)

// Contact definition
// stores the util.ID, the ip address and the distance
type Contact struct {
	ID       *util.ID
	Address  net.UDPAddr
	Distance *util.ID
}

// NewContact returns a new instance of a Contact
func NewContact(id *util.ID, address *net.UDPAddr) Contact {
	return Contact{id, *address, nil}
}

func NewContactWithDistance(ref *util.ID, address *net.UDPAddr, from *util.ID) Contact {
	distance := from.CalcDistance(ref)
	return Contact{
		ID:       from,
		Address:  *address,
		Distance: distance,
	}
}

// CalcDistance calculates the distance to the target and
// fills the contacts distance field
func (contact *Contact) CalcDistance(target *util.ID) {
	contact.Distance = contact.ID.CalcDistance(target)
}

// Less returns true if contact.distance < otherContact.distance
func (contact *Contact) Less(otherContact *Contact) bool {
	return contact.Distance.Less(otherContact.Distance)
}

// String returns a simple string representation of a Contact
func (contact *Contact) String() string {
	return fmt.Sprintf(`contact("%s", "%s")`, contact.ID.String(), contact.Address.String())
}

// ContactCandidates definition
// stores an array of Contacts
type ContactCandidates struct {
	contacts []Contact
}

// Append an array of Contacts to the ContactCandidates
func (candidates *ContactCandidates) Append(contacts []Contact) {
	candidates.contacts = append(candidates.contacts, contacts...)
}

// GetContacts returns the first count number of Contacts
func (candidates *ContactCandidates) GetContacts(count int) []Contact {
	if count > len(candidates.contacts) {
		count = len(candidates.contacts)
	}
	return candidates.contacts[:count]
}

// Sort the Contacts in ContactCandidates
func (candidates *ContactCandidates) Sort() {
	sort.Sort(candidates)
}

// Len returns the length of the ContactCandidates
func (candidates *ContactCandidates) Len() int {
	return len(candidates.contacts)
}

// Swap the position of the Contacts at i and j
// WARNING does not check if either i or j is within range
func (candidates *ContactCandidates) Swap(i, j int) {
	candidates.contacts[i], candidates.contacts[j] = candidates.contacts[j], candidates.contacts[i]
}

// Less returns true if the Contact at index i is smaller than
// the Contact at index j
func (candidates *ContactCandidates) Less(i, j int) bool {
	return candidates.contacts[i].Less(&candidates.contacts[j])
}

func EncodeContactToken(c Contact) string {
	idHex := c.ID.String()
	// Prefer UDPAddr.String() — it handles IPv6 brackets automatically.
	addr := c.Address.String()
	return idHex + "@" + addr
}

func DecodeContactToken(tok string) (*Contact, error) {
	parts := strings.SplitN(tok, "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("bad contact token %q (want <hexID>@<host:port>)", tok)
	}
	pid, err := util.ParseHexID(parts[0])
	if err != nil {
		return nil, fmt.Errorf("bad id in token %q: %w", tok, err)
	}
	addr, err := net.ResolveUDPAddr("udp", parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad addr in token %q: %w", tok, err)
	}
	c := NewContact(&pid, addr)
	return &c, nil
}

func DecodeContactTokenWithDistance(tok string, ref *util.ID) (*Contact, error) {
	c, err := DecodeContactToken(tok)
	if err != nil {
		return nil, err
	}
	c.CalcDistance(ref)
	return c, nil
}

func EncodeContactsForArgs(contacts []Contact) []string {
	out := make([]string, 0, len(contacts))
	for _, c := range contacts {
		out = append(out, EncodeContactToken(c))
	}
	return out
}

func DecodeContactsFromArgs(args []string, ref *util.ID) []Contact {
	out := make([]Contact, 0, len(args))
	for _, tok := range args {
		if c, err := DecodeContactTokenWithDistance(tok, ref); err == nil {
			out = append(out, *c)
		}
	}
	return out
}
