// internal/dictionary/evict.go
package dictionary

import "sort"

// EnsureCapacity tries to make room for `needBytes`.
// Returns true if capacity is available (after eviction if needed).
func (d *Dictionary) EnsureCapacity(needBytes uint32) bool {
	if d.UsedBytes+needBytes <= d.MaxBytes {
		return true
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

	if len(items) == 0 {
		return false
	}

	// Sort by LastSeenOffset (oldest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].entry.LastSeenOffset < items[j].entry.LastSeenOffset
	})

	// Older half is eviction-eligible
	cut := len(items) / 2
	evictable := items[:cut]
	if len(evictable) == 0 {
		return false
	}

	// Among evictable, sort by Score (lowest first)
	sort.Slice(evictable, func(i, j int) bool {
		return evictable[i].entry.Score < evictable[j].entry.Score
	})

	// Evict until enough space or exhausted
	for _, it := range evictable {
		// Estimate bytes freed by this entry
		freed := uint32(len(it.entry.ConstA) + len(it.entry.ConstB))
		delete(d.Entries, it.id)
		if d.UsedBytes >= freed {
			d.UsedBytes -= freed
		} else {
			d.UsedBytes = 0
		}

		if d.UsedBytes+needBytes <= d.MaxBytes {
			return true
		}
	}

	return false
}
