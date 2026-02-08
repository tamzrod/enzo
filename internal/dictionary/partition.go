// internal/dictionary/partition.go
package dictionary

import "sort"

// PartitionResultStats represents a logical split of candidate stats.
type PartitionResultStats struct {
	Protected []*CandidateStats
	Evictable []*CandidateStats
}

// PartitionStatsByAge splits stats into protected and evictable halves
// based purely on LastSeenEpoch (newest = protected).
func PartitionStatsByAge(
	stats []*CandidateStats,
	currentEpoch uint64,
) PartitionResultStats {

	if len(stats) == 0 {
		return PartitionResultStats{}
	}

	sort.Slice(stats, func(i, j int) bool {
		ageI := currentEpoch - stats[i].LastSeenEpoch
		ageJ := currentEpoch - stats[j].LastSeenEpoch
		return ageI < ageJ
	})

	half := len(stats) / 2

	return PartitionResultStats{
		Protected: stats[:half],
		Evictable: stats[half:],
	}
}
