// internal/dictionary/aggregate_test.go
package dictionary

import "testing"

func TestAggregate_CrossPacketHits(t *testing.T) {
	agg := NewAggregator(1024)

	obs := []SpanObservation{
		{
			ConstA:   []byte("the quick "),
			ConstB:   []byte(" fox"),
			SpanSize: 16,
		},
	}

	agg.ObservePacket(1, obs)
	agg.ObservePacket(2, obs)

	if len(agg.Stats) != 1 {
		t.Fatalf("expected 1 candidate stat, got %d", len(agg.Stats))
	}

	for _, s := range agg.Stats {
		if s.Hits != 2 {
			t.Fatalf("expected Hits=2, got %d", s.Hits)
		}
	}
}
