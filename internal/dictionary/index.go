// internal/dictionary/index.go
package dictionary

// PatternKey uniquely identifies a candidate pattern.
// It is derived from CONST_A and CONST_B only.
type PatternKey struct {
	Hash uint64
}

// CandidateIndex provides fast lookup for candidates
// discovered during packet scanning.
type CandidateIndex struct {
	// Map from pattern key to candidate
	ByKey map[PatternKey]*Candidate
}
