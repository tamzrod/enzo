// internal/memory/rawwindow_test.go
package memory

import "testing"

func TestRawWindowAppendAndEvictFIFO(t *testing.T) {
	w := NewRawWindow(8)

	w.Append([]byte("abcd"))
	if got := string(w.Snapshot()); got != "abcd" {
		t.Fatalf("expected abcd, got %q", got)
	}

	w.Append([]byte("ef"))
	if got := string(w.Snapshot()); got != "abcdef" {
		t.Fatalf("expected abcdef, got %q", got)
	}

	// Overflow by 1 (6 + 3 - 8 = 1). FIFO drops exactly the oldest 1 byte.
	w.Append([]byte("XYZ"))
	if got := string(w.Snapshot()); got != "bcdefXYZ" {
		t.Fatalf("expected bcdefXYZ, got %q", got)
	}

	// Append larger than capacity => keep tail.
	w.Append([]byte("0123456789"))
	if got := string(w.Snapshot()); got != "23456789" {
		t.Fatalf("expected 23456789, got %q", got)
	}
}
