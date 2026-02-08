// internal/dictionary/partition_test.go
package dictionary

import "testing"

func TestPartitionStatsByAge(t *testing.T) {
	stats := []*CandidateStats{
		{LastSeenEpoch: 100},
		{LastSeenEpoch: 200},
		{LastSeenEpoch: 300},
		{LastSeenEpoch: 400},
	}

	currentEpoch := uint64(500)

	res := PartitionStatsByAge(stats, currentEpoch)

	if len(res.Protected) != 2 {
		t.Fatalf("expected 2 protected, got %d", len(res.Protected))
	}

	if len(res.Evictable) != 2 {
		t.Fatalf("expected 2 evictable, got %d", len(res.Evictable))
	}

	if res.Protected[0].LastSeenEpoch != 400 {
		t.Fatalf("expected newest in protected zone")
	}

	if res.Evictable[len(res.Evictable)-1].LastSeenEpoch != 100 {
		t.Fatalf("expected oldest in evictable zone")
	}
}
