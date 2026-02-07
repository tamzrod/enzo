// internal/memory/dictionary.go
package memory

import "sort"

// Dictionary is a bounded, opportunistic cache of byte patterns.
// It must never be required for correctness and must never block RAW streaming.
type Dictionary struct {
	capBytes  int
	usedBytes int
	nextID    uint32
	nextSeq   uint64
	entries   map[uint32]*DictEntry
	byKey     map[string]uint32 // optional: caller-chosen stable key (e.g. hash) -> id
}

// DictEntry holds a single cached pattern (opaque bytes) and metadata needed for eviction.
type DictEntry struct {
	ID          uint32
	Key         string
	Bytes       []byte
	Size        int
	CreatedSeq  uint64
	LastUsedSeq uint64
}

// NewDictionary creates a dictionary with a hard cap in bytes.
// capBytes must be > 0.
func NewDictionary(capBytes int) *Dictionary {
	if capBytes <= 0 {
		panic("Dictionary capBytes must be > 0")
	}
	return &Dictionary{
		capBytes: capBytes,
		nextID:   1,
		nextSeq:  1,
		entries:  make(map[uint32]*DictEntry),
		byKey:    make(map[string]uint32),
	}
}

// Cap returns the hard cap (bytes).
func (d *Dictionary) Cap() int { return d.capBytes }

// Used returns current used bytes.
func (d *Dictionary) Used() int { return d.usedBytes }

// Count returns number of entries.
func (d *Dictionary) Count() int { return len(d.entries) }

// GetByID returns the entry, and marks it as used (LRU timestamp).
func (d *Dictionary) GetByID(id uint32) (*DictEntry, bool) {
	e, ok := d.entries[id]
	if !ok {
		return nil, false
	}
	d.touch(e)
	return e, true
}

// GetByKey returns the entry, and marks it as used (LRU timestamp).
func (d *Dictionary) GetByKey(key string) (*DictEntry, bool) {
	id, ok := d.byKey[key]
	if !ok {
		return nil, false
	}
	return d.GetByID(id)
}

// AddCandidate attempts to insert a new entry under the given key.
// If key already exists, it touches and returns the existing entry (no replacement in v1).
//
// Guarded eviction policy (LOCKED):
// - Evict by LRU, but ONLY from the older half of entries (by CreatedSeq).
// - Young half (newer 50%) is protected.
// - If eviction can't free enough space, reject the candidate WITH NO SIDE EFFECTS.
func (d *Dictionary) AddCandidate(key string, b []byte) (entry *DictEntry, inserted bool) {
	if len(b) == 0 {
		return nil, false
	}

	// If it already exists, do not replace bytes in v1 (boring + safe).
	if existing, ok := d.GetByKey(key); ok {
		return existing, false
	}

	need := len(b)
	if need > d.capBytes {
		// Candidate can never fit.
		return nil, false
	}

	if d.freeBytes() < need {
		if !d.evictToFit(need) {
			// Reject candidate; dictionary must remain unchanged if we can't fit.
			return nil, false
		}
	}

	id := d.allocID()
	seq := d.allocSeq()

	cpy := make([]byte, len(b))
	copy(cpy, b)

	e := &DictEntry{
		ID:          id,
		Key:         key,
		Bytes:       cpy,
		Size:        len(cpy),
		CreatedSeq:  seq,
		LastUsedSeq: seq,
	}

	d.entries[id] = e
	d.byKey[key] = id
	d.usedBytes += e.Size
	return e, true
}

// Delete removes an entry by ID (used internally for eviction).
func (d *Dictionary) Delete(id uint32) bool {
	e, ok := d.entries[id]
	if !ok {
		return false
	}
	delete(d.entries, id)
	delete(d.byKey, e.Key)
	d.usedBytes -= e.Size
	if d.usedBytes < 0 {
		d.usedBytes = 0
	}
	return true
}

// Clear drops all entries.
func (d *Dictionary) Clear() {
	d.entries = make(map[uint32]*DictEntry)
	d.byKey = make(map[string]uint32)
	d.usedBytes = 0
}

// ---- internal helpers ----

func (d *Dictionary) freeBytes() int {
	return d.capBytes - d.usedBytes
}

func (d *Dictionary) allocID() uint32 {
	id := d.nextID
	d.nextID++
	return id
}

func (d *Dictionary) allocSeq() uint64 {
	seq := d.nextSeq
	d.nextSeq++
	return seq
}

func (d *Dictionary) touch(e *DictEntry) {
	seq := d.allocSeq()
	e.LastUsedSeq = seq
}

// evictToFit tries to free at least "need" bytes using guarded LRU.
//
// IMPORTANT (LOCKED):
// - Only older 50% of entries (by CreatedSeq) are eligible.
// - Within that older half, evict least-recently-used (by LastUsedSeq).
// - If not enough space can be freed WITHOUT touching young half, reject with NO SIDE EFFECTS.
func (d *Dictionary) evictToFit(need int) bool {
	free := d.freeBytes()
	if free >= need {
		return true
	}
	if len(d.entries) == 0 {
		return false
	}

	// Build slice of entries for age-split.
	all := make([]*DictEntry, 0, len(d.entries))
	for _, e := range d.entries {
		all = append(all, e)
	}

	// Sort by CreatedSeq ascending (oldest first).
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedSeq < all[j].CreatedSeq
	})

	// Older half is eviction-eligible (older 50%).
	half := len(all) / 2
	if half == 0 {
		// Only 1 entry => it is "young half" by definition; cannot evict in v1 policy.
		return false
	}
	olderHalf := all[:half]

	// Sort eviction candidates by LRU within older half: least recently used first.
	sort.Slice(olderHalf, func(i, j int) bool {
		if olderHalf[i].LastUsedSeq == olderHalf[j].LastUsedSeq {
			return olderHalf[i].CreatedSeq < olderHalf[j].CreatedSeq
		}
		return olderHalf[i].LastUsedSeq < olderHalf[j].LastUsedSeq
	})

	// ---- PHASE 1: PLAN (NO MUTATION) ----
	freed := 0
	victims := make([]*DictEntry, 0, len(olderHalf))
	for _, v := range olderHalf {
		freed += v.Size
		victims = append(victims, v)
		if free+freed >= need {
			break
		}
	}

	// If we can't fit without touching young half, reject with no side effects.
	if free+freed < need {
		return false
	}

	// ---- PHASE 2: COMMIT (MUTATION) ----
	for _, v := range victims {
		d.Delete(v.ID)
		if d.freeBytes() >= need {
			return true
		}
	}

	return d.freeBytes() >= need
}
