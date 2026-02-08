package dictionary

import "testing"

// helper
func entry(id uint32, score uint64, last uint64, a, b string) *DictionaryEntry {
	return &DictionaryEntry{
		ID:             id,
		ConstA:         []byte(a),
		ConstB:         []byte(b),
		Score:          score,
		LastSeenOffset: last,
	}
}

// Young half must NEVER be evicted even if score is terrible
func TestEviction_YoungHalfProtected(t *testing.T) {
	d := &Dictionary{
		MaxBytes: 8,
		UsedBytes: 8,
		Entries: map[uint32]*DictionaryEntry{
			1: entry(1, 1,  1,  "a", "b"), // old
			2: entry(2, 2,  2,  "c", "d"), // old
			3: entry(3, 0, 10, "e", "f"), // young (very low score)
			4: entry(4, 0, 11, "g", "h"), // young
		},
	}

	ok := d.EnsureCapacity(2)
	if !ok {
		t.Fatalf("expected eviction to succeed")
	}

	if _, ok := d.Entries[3]; !ok {
		t.Fatalf("young entry was evicted (violation)")
	}
	if _, ok := d.Entries[4]; !ok {
		t.Fatalf("young entry was evicted (violation)")
	}
}

// When scores tie, eviction must still succeed deterministically
func TestEviction_ScoreTie(t *testing.T) {
	d := &Dictionary{
		MaxBytes: 6,
		UsedBytes: 6,
		Entries: map[uint32]*DictionaryEntry{
			1: entry(1, 10, 1, "a", "b"),
			2: entry(2, 10, 2, "c", "d"),
			3: entry(3, 10, 3, "e", "f"),
		},
	}

	ok := d.EnsureCapacity(2)
	if !ok {
		t.Fatalf("expected eviction despite score tie")
	}

	if len(d.Entries) != 2 {
		t.Fatalf("expected exactly one eviction, got %d entries", len(d.Entries))
	}
}

// Exact-fit eviction must succeed (no off-by-one rejection)
func TestEviction_ExactFit(t *testing.T) {
	d := &Dictionary{
		MaxBytes: 6,
		UsedBytes: 6,
		Entries: map[uint32]*DictionaryEntry{
			1: entry(1, 1, 1, "a", "b"), // 2 bytes
			2: entry(2, 2, 2, "c", "d"), // 2 bytes
			3: entry(3, 3, 3, "e", "f"), // 2 bytes
		},
	}

	ok := d.EnsureCapacity(2)
	if !ok {
		t.Fatalf("expected exact-fit eviction to succeed")
	}

	if d.UsedBytes != 4 {
		t.Fatalf("expected used bytes to be 4, got %d", d.UsedBytes)
	}
}

// If only young entries exist, eviction must fail
func TestEviction_RejectWhenOnlyYoung(t *testing.T) {
	d := &Dictionary{
		MaxBytes: 4,
		UsedBytes: 4,
		Entries: map[uint32]*DictionaryEntry{
			1: entry(1, 100, 10, "a", "b"),
		},
	}

	ok := d.EnsureCapacity(2)
	if ok {
		t.Fatalf("expected rejection when no evictable entries exist")
	}
}
