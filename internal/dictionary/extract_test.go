package dictionary

import (
	"testing"
)

func TestExtractSpans_EmptyPacket(t *testing.T) {
	ctx := PacketContext{
		PacketBytes:  []byte{},
		PacketOffset: 0,
		PacketID:     1,
	}

	out := ExtractSpans(ctx)
	if len(out) != 0 {
		t.Fatalf("expected no spans, got %d", len(out))
	}
}

func TestExtractSpans_SingleOccurrence(t *testing.T) {
	ctx := PacketContext{
		PacketBytes:  []byte("the quick brown fox"),
		PacketOffset: 100,
		PacketID:     1,
	}

	out := ExtractSpans(ctx)
	if len(out) == 0 {
		t.Fatalf("expected spans, got none")
	}

	// Sanity: all spans must have at least 1 intra hit
	for _, s := range out {
		if s.IntraHits < 1 {
			t.Fatalf("span has invalid intra hits: %d", s.IntraHits)
		}
	}
}

func TestExtractSpans_IntraPacketRepetition(t *testing.T) {
	ctx := PacketContext{
		PacketBytes:  []byte("abcXdef abcYdef abcZdef"),
		PacketOffset: 0,
		PacketID:     1,
	}

	out := ExtractSpans(ctx)

	found := false
	for _, s := range out {
		// Look for CONST_A="abc", CONST_B="d"
		if string(s.ConstA) == "abc" && string(s.ConstB) == "d" {
			found = true
			if s.IntraHits < 3 {
				t.Fatalf("expected >=3 intra hits, got %d", s.IntraHits)
			}
		}
	}

	if !found {
		t.Fatalf("expected repeated span not found")
	}
}

func TestExtractSpans_OverlapAllowed(t *testing.T) {
	ctx := PacketContext{
		PacketBytes:  []byte("the quick brown fox"),
		PacketOffset: 50,
		PacketID:     1,
	}

	out := ExtractSpans(ctx)

	// We expect multiple overlapping spans
	if len(out) < 2 {
		t.Fatalf("expected overlapping spans, got %d", len(out))
	}
}

func TestExtractSpans_LastSeenOffset(t *testing.T) {
	ctx := PacketContext{
		PacketBytes:  []byte("aaaXbbb aaaYbbb"),
		PacketOffset: 1000,
		PacketID:     1,
	}

	out := ExtractSpans(ctx)

	for _, s := range out {
		if s.LastSeenOffset < ctx.PacketOffset {
			t.Fatalf("invalid LastSeenOffset: %d", s.LastSeenOffset)
		}
	}
}

func TestExtractSpans_PacketLocalOnly(t *testing.T) {
	ctx1 := PacketContext{
		PacketBytes:  []byte("helloAAA"),
		PacketOffset: 0,
		PacketID:     1,
	}

	ctx2 := PacketContext{
		PacketBytes:  []byte("AAAworld"),
		PacketOffset: 8,
		PacketID:     2,
	}

	out1 := ExtractSpans(ctx1)
	out2 := ExtractSpans(ctx2)

	// There must be no assumption of cross-packet stitching
	for _, s := range append(out1, out2...) {
		if string(s.ConstA) == "hello" && string(s.ConstB) == "world" {
			t.Fatalf("cross-packet span detected (violation)")
		}
	}
}
