// internal/dictionary/evict_candidate.go
package dictionary

// EvictOne removes exactly one lowest-worth candidate
// from the evictable half of candidate stats.
// Returns true if an eviction occurred.
func EvictOne(
	stats map[PatternKey]*CandidateStats,
	currentEpoch uint64,
) bool {

	if len(stats) == 0 {
		return false
	}

	// Collect stats into slice
	all := make([]*CandidateStats, 0, len(stats))
	for _, s := range stats {
		all = append(all, s)
	}

	partition := PartitionStatsByAge(all, currentEpoch)

	if len(partition.Evictable) == 0 {
		return false
	}

	var (
		lowestKey   PatternKey
		lowestWorth uint64
		found       bool
	)

	// Find lowest worth among evictable candidates
	for key, s := range stats {
		for _, ev := range partition.Evictable {
			if s != ev {
				continue
			}

			w := Worth(s)

			if !found || w < lowestWorth {
				lowestWorth = w
				lowestKey = key
				found = true
			}
		}
	}

	if !found {
		return false
	}

	delete(stats, lowestKey)
	return true
}
