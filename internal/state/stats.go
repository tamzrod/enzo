// internal/state/stats.go
package state

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/tamzrod/enzo/internal/connctx"
)

type connStats struct {
	rawIn   atomic.Uint64
	wireOut atomic.Uint64
	start   time.Time
}

func newConnStats() *connStats {
	return &connStats{start: time.Now()}
}

func printStats(cc *connctx.ConnectionContext, s *connStats) {
	raw := s.rawIn.Load()
	wire := s.wireOut.Load()
	dur := time.Since(s.start)

	var savings float64
	if raw > 0 {
		savings = 100.0 * (1.0 - float64(wire)/float64(raw))
	}

	log.Printf(
		"conn %d stats: raw_in=%dB wire_out=%dB savings=%.2f%% duration=%s",
		cc.ID, raw, wire, savings, dur.Round(time.Millisecond),
	)
}
