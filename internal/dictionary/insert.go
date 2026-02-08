// internal/dictionary/insert.go
package dictionary

import "sort"

// TryInsert attempts to insert a dictionary entry.
// It handles capacity checks and eviction internally.
// Returns true if the entry was inserted, false if rejected.
//
// IMPORTANT:
// - Does NOT recycle IDs
// - Does NOT block
// - Safe to call even when dictionary is full
func (d *Dictionary) TryInsert(entry *DictionaryEntry) bool {
	need := uint32(len(entry.ConstA) + len(entry.ConstB))

	// Fast path: enough space
	if d.UsedBytes+need <= d.MaxBytes {
		d.insert(entry, need)
		return true
	}

	// Attempt eviction
	if !d.evictFor(need) {
		return false
	}

	d.insert(entry, need)
	return true
}

// insert performs the actual insertion (no checks).
func (d *Dictionary) insert(entry *DictionaryEntry, need uint32) {
	d.Entries[entry.ID] = entry
	d.UsedBytes += need
}

// evictFor tries to free enough space for `need` bytes.
// Eviction rules:
// - Only older half eligible
// - Lowest score evicted first
func (d *Dictionary) evictFor(need uint32) bool {
	if len(d.Entries) == 0 {
		return false
	}

	// Collect entries
	type item struct {
		id    uint32
		entry *DictionaryEntry
	}
	items := make([]item, 0, len(d.Entries))
	for id, e := range d.Entries {
		items = append(items, item{id: id, entry: e})
	}

	// Sort by age (oldest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].entry.LastSeenOffset < items[j].entry.LastSeenOffset
	})

	// Older half only
	cut := len(items) / 2
	if cut == 0 {
		return false
	}
	evictable := items[:cut]

	// Sort evictable by score (lowest first)
	sort.Slice(evictable, func(i, j int) bool {
		return evictable[i].entry.Score < evictable[j].entry.Score
	})

	// Evict until enough space
	for _, it := range evictable {
		freed := uint32(len(it.entry.ConstA) + len(it.entry.ConstB))

		delete(d.Entries, it.id)
		if d.UsedBytes >= freed {
			d.UsedBytes -= freed
		} else {
			d.UsedBytes = 0
		}

		if d.UsedBytes+need <= d.MaxBytes {
			return true
		}
	}

	return false
}
