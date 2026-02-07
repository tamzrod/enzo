// internal/state/stats.go
package state

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/tamzrod/enzo/internal/connctx"
)

// connStats is per-connection statistics.
//
// IMPORTANT (compatibility contract):
// Existing code already directly references stats.rawIn and stats.wireOut.
// Those fields MUST remain present to avoid touching encode_v1.go/http_helpers.go.
//
// Duplex-safe extension:
// We add forward/reverse counters, but we do NOT require all call sites
// to be updated yet. Aggregates remain authoritative until plumbing is wired.
type connStats struct {
	// ---- Backwards-compatible aggregate counters (do not remove) ----
	rawIn   atomic.Uint64 // "logical input" bytes for the active transform path
	wireOut atomic.Uint64 // bytes written to the wire by ENZO framing / passthrough

	// ---- Duplex extension (optional to use now; wire in later) ----
	fwdRawIn   atomic.Uint64
	fwdWireOut atomic.Uint64
	revRawIn   atomic.Uint64
	revWireOut atomic.Uint64

	start time.Time
}

func newConnStats() *connStats {
	return &connStats{start: time.Now()}
}

// The following helpers allow duplex-aware call sites (e.g., state/machine.go)
// to attribute stats by direction WITHOUT breaking existing code.
// If you don't call these yet, aggregates still work.

func (s *connStats) AddForwardRaw(n uint64) {
	s.rawIn.Add(n)
	s.fwdRawIn.Add(n)
}

func (s *connStats) AddForwardWire(n uint64) {
	s.wireOut.Add(n)
	s.fwdWireOut.Add(n)
}

func (s *connStats) AddReverseRaw(n uint64) {
	s.rawIn.Add(n)
	s.revRawIn.Add(n)
}

func (s *connStats) AddReverseWire(n uint64) {
	s.wireOut.Add(n)
	s.revWireOut.Add(n)
}

// printStats remains intentionally "boring":
// - It always prints aggregates (existing behavior expectation).
// - If per-path counters are being used, it also prints forward/reverse.
func printStats(cc *connctx.ConnectionContext, s *connStats) {
	raw := s.rawIn.Load()
	wire := s.wireOut.Load()
	dur := time.Since(s.start)

	var savings float64
	if raw > 0 {
		savings = 100.0 * (1.0 - float64(wire)/float64(raw))
	}

	fRaw := s.fwdRawIn.Load()
	fWire := s.fwdWireOut.Load()
	rRaw := s.revRawIn.Load()
	rWire := s.revWireOut.Load()

	// If duplex counters are unused, keep log output compatible-ish and minimal.
	if (fRaw|fWire|rRaw|rWire) == 0 {
		log.Printf(
			"conn %d stats: raw_in=%dB wire_out=%dB savings=%.2f%% duration=%s",
			cc.ID, raw, wire, savings, dur.Round(time.Millisecond),
		)
		return
	}

	log.Printf(
		"conn %d stats: raw_in=%dB wire_out=%dB savings=%.2f%% duration=%s | fwd raw=%dB wire=%dB | rev raw=%dB wire=%dB",
		cc.ID, raw, wire, savings, dur.Round(time.Millisecond),
		fRaw, fWire, rRaw, rWire,
	)
}
