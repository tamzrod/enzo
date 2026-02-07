// internal/memory/dictionary_test.go
package memory

import "testing"

func TestDictionaryRejectsTooLargeCandidate(t *testing.T) {
	d := NewDictionary(8)
	if _, ok := d.AddCandidate("k1", []byte("012345678")); ok {
		t.Fatalf("expected reject for oversize candidate")
	}
}

func TestDictionaryDoesNotEvictYoungHalf(t *testing.T) {
	// cap 10
	d := NewDictionary(10)

	// Insert 4 entries of size 2 => used 8, free 2.
	// Age order: e1,e2,e3,e4. Older half = e1,e2. Young half = e3,e4 (protected).
	e1, _ := d.AddCandidate("e1", []byte("aa"))
	e2, _ := d.AddCandidate("e2", []byte("bb"))
	e3, _ := d.AddCandidate("e3", []byte("cc"))
	e4, _ := d.AddCandidate("e4", []byte("dd"))

	// Touch e1 recently so that LRU in older half becomes e2 first.
	if _, ok := d.GetByID(e1.ID); !ok {
		t.Fatalf("expected e1 present")
	}

	// Need 4 bytes; free is 2 => must evict 2 bytes from older half (should evict e2).
	if _, ok := d.AddCandidate("new", []byte("EEEE")); !ok {
		t.Fatalf("expected insert after eviction")
	}

	// e2 should be evicted (older half + LRU), e3/e4 should remain.
	if _, ok := d.GetByID(e2.ID); ok {
		t.Fatalf("expected e2 evicted")
	}
	if _, ok := d.GetByID(e3.ID); !ok {
		t.Fatalf("expected e3 protected")
	}
	if _, ok := d.GetByID(e4.ID); !ok {
		t.Fatalf("expected e4 protected")
	}
}

func TestDictionaryRejectsWhenEvictionCannotFreeEnough(t *testing.T) {
	// cap 6
	d := NewDictionary(6)

	// Insert 2 entries of size 3 => used 6, free 0.
	// len=2 => half=1 => older half has only the oldest entry (evictable),
	// young half has newest (protected).
	e1, _ := d.AddCandidate("e1", []byte("AAA"))
	_, _ = d.AddCandidate("e2", []byte("BBB"))

	// Candidate needs 4 bytes; even evicting e1 frees only 3 => still not enough => reject.
	if _, ok := d.AddCandidate("new", []byte("CCCC")); ok {
		t.Fatalf("expected reject when cannot free enough")
	}

	// Ensure dictionary still consistent.
	if _, ok := d.GetByID(e1.ID); !ok {
		t.Fatalf("expected e1 still present or at least not corrupted")
	}
}
