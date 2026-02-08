// internal/dictionary/aggregate.go
package dictionary

const (
	// Hard-coded for v1; YAML later
	MinIntraHits uint16 = 5
	MinInterHits uint32 = 5
	MinSpanSize  uint32 = 8 // conservative floor
)

// Aggregator owns candidate accumulation and qualification.
// It does NOT implement eviction itself; it delegates insertion decisions
// to Dictionary.TryInsert (which may evict or reject).
type Aggregator struct {
	Index      *CandidateIndex
	Dictionary *Dictionary

	// RAW window bounds (sliding)
	WindowStart uint64
	WindowEnd   uint64
}

// NewAggregator initializes aggregation state.
func NewAggregator(dictBytes uint32) *Aggregator {
	return &Aggregator{
		Index: &CandidateIndex{
			ByKey: make(map[PatternKey]*Candidate),
		},
		Dictionary: &Dictionary{
			MaxBytes:  dictBytes,
			UsedBytes: 0,
			Entries:   make(map[uint32]*DictionaryEntry),
			NextID:    1,
		},
	}
}

// ObservePacket ingests span observations from ONE packet.
// packetID is used to count inter-packet hits once per packet.
func (a *Aggregator) ObservePacket(packetID uint64, obs []SpanObservation) {
	_ = packetID // reserved for future de-dupe strategies; current de-dupe is key-local

	seenThisPacket := make(map[PatternKey]bool)

	for _, o := range obs {
		key := makePatternKey(o.ConstA, o.ConstB)

		c, ok := a.Index.ByKey[key]
		if !ok {
			c = &Candidate{
				ID:              0,
				ConstA:          append([]byte(nil), o.ConstA...),
				ConstB:          append([]byte(nil), o.ConstB...),
				IntraPacketHits: 0,
				InterPacketHits: 0,
				LastSeenOffset:  o.LastSeenOffset,
				SpanSize:        o.SpanSize,
				Score:           0,
			}
			a.Index.ByKey[key] = c
		}

		// Update intra-packet hits (accumulate)
		c.IntraPacketHits += o.IntraHits

		// Update inter-packet hits once per packet (per candidate)
		if !seenThisPacket[key] {
			c.InterPacketHits++
			seenThisPacket[key] = true
		}

		// Update last seen (monotonic)
		if o.LastSeenOffset > c.LastSeenOffset {
			c.LastSeenOffset = o.LastSeenOffset
		}

		// Update score (simple + fast)
		totalHits := uint64(c.IntraPacketHits) + uint64(c.InterPacketHits)
		c.Score = totalHits * uint64(c.SpanSize)

		// Try qualification (may promote)
		a.tryQualify(c)
	}
}

// tryQualify promotes a candidate to dictionary entry if it qualifies.
func (a *Aggregator) tryQualify(c *Candidate) {
	if c.ID != 0 {
		return // already promoted
	}

	// RAW window relevance check
	if c.LastSeenOffset < a.WindowStart || c.LastSeenOffset > a.WindowEnd {
		return
	}

	// Path A: cross-packet stability
	if c.InterPacketHits >= MinInterHits {
		a.promote(c)
		return
	}

	// Path B: strong single-packet evidence
	if c.IntraPacketHits >= MinIntraHits && c.SpanSize >= MinSpanSize {
		a.promote(c)
		return
	}
}

// promote attempts to insert a promoted entry into the bounded dictionary.
// The dictionary decides if it fits (evict / reject). Aggregator never evicts.
func (a *Aggregator) promote(c *Candidate) {
	// Allocate an ID ONLY if insertion succeeds (no gaps, no recycle).
	id := a.Dictionary.NextID

	entry := &DictionaryEntry{
		ID:             id,
		ConstA:         append([]byte(nil), c.ConstA...),
		ConstB:         append([]byte(nil), c.ConstB...),
		TotalHits:      uint64(c.IntraPacketHits) + uint64(c.InterPacketHits),
		SpanSize:       c.SpanSize,
		Score:          c.Score,
		LastSeenOffset: c.LastSeenOffset,
	}

	if !a.Dictionary.TryInsert(entry) {
		// Reject promotion silently; RAW continues.
		return
	}

	// Commit ID only on success
	a.Dictionary.NextID++
	c.ID = id
}

// makePatternKey hashes CONST_A and CONST_B into a stable key.
// Hash choice is opaque for now.
func makePatternKey(aBytes, bBytes []byte) PatternKey {
	var h uint64 = 1469598103934665603 // FNV-1a 64-bit offset
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
