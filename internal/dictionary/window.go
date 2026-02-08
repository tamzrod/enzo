// internal/dictionary/window.go
package dictionary

// WindowRef links an observation or entry to the RAW window
// without owning or modifying RAW memory.
type WindowRef struct {
	// FirstSeenOffset is the RAW offset where this pattern
	// was first observed.
	FirstSeenOffset uint64

	// LastSeenOffset is the most recent RAW offset where
	// this pattern was observed.
	LastSeenOffset uint64
}

// AgeClass represents the relative age of a dictionary entry.
// Used only for eviction eligibility.
type AgeClass uint8

const (
	AgeYoung AgeClass = iota
	AgeOld
)

// AgingMeta holds metadata used to determine eviction eligibility.
type AgingMeta struct {
	// AgeClass indicates whether this entry belongs to
	// the protected (young) or evictable (old) half.
	AgeClass AgeClass

	// LastAccessOffset mirrors WindowRef.LastSeenOffset
	// and is used for LRU-style ordering within an age class.
	LastAccessOffset uint64
}
