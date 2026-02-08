// internal/dictionary/candidate_stats.go
package dictionary

// CandidateStats holds Phase-1 / Phase-2 learning metadata.
// It does NOT represent a dictionary entry.
// It has no semantic or wire meaning.
type CandidateStats struct {
	// Number of packets where the full pattern appeared
	Hits uint32

	// Total span size in bytes
	Size uint32

	// Epoch (packet id) when first observed
	FirstSeenEpoch uint64

	// Epoch (packet id) when last observed
	LastSeenEpoch uint64
}
