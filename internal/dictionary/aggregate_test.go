package dictionary

import "testing"

func TestAggregate_InterPacketQualification(t *testing.T) {
	agg := NewAggregator(1024 * 1024)
	agg.WindowStart = 0
	agg.WindowEnd = 1 << 20

	for i := 0; i < 5; i++ {
		obs := []SpanObservation{
			{
				ConstA:         []byte("the quick "),
				ConstB:         []byte(" fox"),
				IntraHits:      1,
				SpanSize:       16,
				LastSeenOffset: uint64(100 + i),
			},
		}
		agg.ObservePacket(uint64(i+1), obs)
	}

	if len(agg.Dictionary.Entries) != 1 {
		t.Fatalf("expected 1 dictionary entry, got %d", len(agg.Dictionary.Entries))
	}
}

func TestAggregate_IntraPacketQualification(t *testing.T) {
	agg := NewAggregator(1024 * 1024)
	agg.WindowStart = 0
	agg.WindowEnd = 1 << 20

	obs := []SpanObservation{
		{
			ConstA:         []byte("abc"),
			ConstB:         []byte("d"),
			IntraHits:      5,
			SpanSize:       12,
			LastSeenOffset: 500,
		},
	}
	agg.ObservePacket(1, obs)

	if len(agg.Dictionary.Entries) != 1 {
		t.Fatalf("expected 1 dictionary entry, got %d", len(agg.Dictionary.Entries))
	}
}

func TestAggregate_NoCrossPacketHalf(t *testing.T) {
	agg := NewAggregator(1024 * 1024)
	agg.WindowStart = 0
	agg.WindowEnd = 1 << 20

	obs1 := []SpanObservation{
		{ConstA: []byte("hello"), ConstB: []byte("AAA"), IntraHits: 1, SpanSize: 10, LastSeenOffset: 10},
	}
	obs2 := []SpanObservation{
		{ConstA: []byte("AAA"), ConstB: []byte("world"), IntraHits: 1, SpanSize: 10, LastSeenOffset: 20},
	}

	agg.ObservePacket(1, obs1)
	agg.ObservePacket(2, obs2)

	if len(agg.Dictionary.Entries) != 0 {
		t.Fatalf("unexpected promotion from cross-packet halves")
	}
}
