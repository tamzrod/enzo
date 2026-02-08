// internal/dictionary/worth.go
package dictionary

// Worth computes the relative value of a candidate.
// Phase 3 (read-only):
//   - No side effects
//   - No age component
//   - No normalization
//
// Baseline formula (LOCKED FOR NOW):
//   Worth = Hits * Size
func Worth(stats *CandidateStats) uint64 {
	if stats == nil {
		return 0
	}

	return uint64(stats.Hits) * uint64(stats.Size)
}
