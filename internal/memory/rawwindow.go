// internal/memory/rawwindow.go
package memory

// RawWindow is a bounded, sliding FIFO observer of recently seen raw bytes.
// It is NOT a buffer for wire flow. Callers must always forward bytes first,
// then optionally Append to this window.
type RawWindow struct {
	buf  []byte
	head int // next write index
	size int // number of valid bytes currently stored (<= len(buf))
}

// NewRawWindow creates a fixed-size sliding window.
// sizeBytes must be > 0.
func NewRawWindow(sizeBytes int) *RawWindow {
	if sizeBytes <= 0 {
		panic("RawWindow sizeBytes must be > 0")
	}
	return &RawWindow{
		buf: make([]byte, sizeBytes),
	}
}

// Cap returns the fixed capacity in bytes.
func (w *RawWindow) Cap() int { return len(w.buf) }

// Len returns the current number of bytes stored (<= Cap()).
func (w *RawWindow) Len() int { return w.size }

// Append observes bytes by copying them into the ring.
// If len(p) exceeds capacity, only the last Cap() bytes are retained.
// Eviction is FIFO (oldest bytes are dropped).
func (w *RawWindow) Append(p []byte) {
	if len(p) == 0 {
		return
	}
	capacity := len(w.buf)
	if capacity == 0 {
		return
	}

	// If incoming chunk is larger than the window, keep only the tail.
	if len(p) >= capacity {
		p = p[len(p)-capacity:]
		// After this append, window becomes exactly full.
		w.copyIntoRing(p)
		w.size = capacity
		return
	}

	// Ensure size reflects sliding FIFO behavior.
	// New size is min(capacity, oldSize + len(p)).
	newSize := w.size + len(p)
	if newSize > capacity {
		newSize = capacity
	}
	w.copyIntoRing(p)
	w.size = newSize
}

func (w *RawWindow) copyIntoRing(p []byte) {
	capacity := len(w.buf)
	if capacity == 0 || len(p) == 0 {
		return
	}

	// Write p into ring at head, wrapping as needed.
	remaining := len(p)
	src := 0

	// First segment: from head to end.
	n := capacity - w.head
	if n > remaining {
		n = remaining
	}
	copy(w.buf[w.head:w.head+n], p[src:src+n])
	w.head = (w.head + n) % capacity
	remaining -= n
	src += n

	// Second segment: from start.
	if remaining > 0 {
		copy(w.buf[0:remaining], p[src:src+remaining])
		w.head = remaining % capacity
	}
}

// Snapshot returns a copy of the current window content ordered oldest→newest.
// This is for diagnostics/tests only; avoid calling in hot paths.
func (w *RawWindow) Snapshot() []byte {
	if w.size == 0 {
		return nil
	}
	capacity := len(w.buf)

	// Oldest byte index is (head - size) mod capacity.
	start := w.head - w.size
	if start < 0 {
		start += capacity
	}

	out := make([]byte, w.size)
	if start+w.size <= capacity {
		copy(out, w.buf[start:start+w.size])
		return out
	}

	// Wrap case.
	firstLen := capacity - start
	copy(out, w.buf[start:])
	copy(out[firstLen:], w.buf[:w.size-firstLen])
	return out
}
