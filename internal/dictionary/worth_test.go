// internal/dictionary/worth_test.go
package dictionary

import "testing"

func TestWorth_IncreasesWithHits(t *testing.T) {
	low := &CandidateStats{
		Hits: 1,
		Size: 10,
	}

	high := &CandidateStats{
		Hits: 5,
		Size: 10,
	}

	if Worth(high) <= Worth(low) {
		t.Fatalf("expected higher worth with more hits")
	}
}

func TestWorth_IncreasesWithSize(t *testing.T) {
	low := &CandidateStats{
		Hits: 5,
		Size: 4,
	}

	high := &CandidateStats{
		Hits: 5,
		Size: 20,
	}

	if Worth(high) <= Worth(low) {
		t.Fatalf("expected higher worth with larger size")
	}
}

func TestWorth_ZeroCases(t *testing.T) {
	zeroHits := &CandidateStats{
		Hits: 0,
		Size: 100,
	}

	zeroSize := &CandidateStats{
		Hits: 10,
		Size: 0,
	}

	if Worth(zeroHits) != 0 {
		t.Fatalf("expected zero worth for zero hits")
	}

	if Worth(zeroSize) != 0 {
		t.Fatalf("expected zero worth for zero size")
	}

	if Worth(nil) != 0 {
		t.Fatalf("expected zero worth for nil stats")
	}
}
