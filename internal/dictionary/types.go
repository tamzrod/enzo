// internal/dictionary/types.go
package dictionary

// Candidate represents a learned pattern candidate.
// It may or may not yet be promoted to a dictionary entry.
type Candidate struct {
	// ID is assigned ONLY after qualification.
	// Zero means "not yet promoted".
	ID uint32

	// Pattern definition
	ConstA []byte
	ConstB []byte

	// Hit accounting
	// IntraPacketHits counts occurrences within a single packet
	IntraPacketHits uint16

	// InterPacketHits counts distinct packets where the full span appears
	InterPacketHits uint32

	// LastSeenOffset is the RAW window offset where this candidate
	// was last observed (used for aging / window relevance).
	LastSeenOffset uint64

	// SpanSize is the total byte length of CONST_A + VAR + CONST_B
	SpanSize uint32

	// Score = hits × effective_size
	// Precomputed for fast eviction comparison
	Score uint64
}

// DictionaryEntry is a promoted, ID-bearing pattern.
// Entries live inside the bounded dictionary memory.
type DictionaryEntry struct {
	ID uint32

	ConstA []byte
	ConstB []byte

	// Statistics
	TotalHits uint64
	SpanSize  uint32
	Score     uint64

	// Aging / eviction metadata
	LastSeenOffset uint64
}

// Dictionary is the bounded container of promoted entries.
// It does NOT own RAW memory and never blocks the stream.
type Dictionary struct {
	// Hard memory cap (bytes)
	MaxBytes uint32

	// Current usage
	UsedBytes uint32

	// Entries by ID (ID is never recycled within an epoch)
	Entries map[uint32]*DictionaryEntry

	// NextID monotonically increases until epoch reset
	NextID uint32
}
