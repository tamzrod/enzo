// internal/dictionary/aggregate.go
package dictionary

// Aggregator observes RAW spans and accumulates candidate statistics.
// External shape is CONTRACT-LOCKED.
type Aggregator struct {
	// Existing contract fields (DO NOT REMOVE)
	WindowStart uint64
	WindowEnd   uint64
	Dictionary  *Dictionary // may be nil in Phase 1/2

	// Phase-1 / Phase-2 learning stats (INTERNAL)
	Stats map[PatternKey]*CandidateStats
}

// NewAggregator preserves the original constructor contract.
// Dictionary ownership is NOT assumed here.
func NewAggregator(windowSize uint64) *Aggregator {
	return &Aggregator{
		WindowStart: 0,
		WindowEnd:   windowSize,
		Dictionary:  nil,
		Stats:       make(map[PatternKey]*CandidateStats),
	}
}

// ObservePacket ingests observations from ONE packet.
// packetID acts as the epoch.
func (a *Aggregator) ObservePacket(
	packetID uint64,
	obs []SpanObservation,
) {
	seenThisPacket := make(map[PatternKey]bool)

	for _, o := range obs {
		key := makePatternKey(o.ConstA, o.ConstB)

		s, exists := a.Stats[key]
		if !exists {
			s = &CandidateStats{
				Hits:           0,
				Size:           o.SpanSize,
				FirstSeenEpoch: packetID,
				LastSeenEpoch:  packetID,
			}
			a.Stats[key] = s
		}

		// Count once per packet
		if !seenThisPacket[key] {
			s.Hits++
			s.LastSeenEpoch = packetID
			seenThisPacket[key] = true
		}
	}
}

// makePatternKey hashes CONST_A and CONST_B into a stable signature.
func makePatternKey(aBytes, bBytes []byte) PatternKey {
	var h uint64 = 1469598103934665603 // FNV-1a offset
	for _, b := range aBytes {
		h ^= uint64(b)
		h *= 1099511628211
	}
	for _, b := range bBytes {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return PatternKey{Hash: h}
}
