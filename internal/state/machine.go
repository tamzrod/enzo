// internal/state/machine.go
package state

import (
	"log"

	"github.com/tamzrod/enzo/internal/connctx"
)

// Run is the entry point for per-connection state handling.
// It dispatches based on the connection mode and owns the lifecycle.
//
// This function must NOT return until the connection is finished.
func Run(cc *connctx.ConnectionContext) {
	defer cc.Cancel()

	switch cc.Mode {
	case connctx.ModeEncode:
		runEncode(cc)
	case connctx.ModeDecode:
		runDecode(cc)
	default:
		log.Printf("conn %d: unknown mode, closing", cc.ID)
	}
}

// runEncode handles the ENCODE path (raw -> compressed).
// Placeholder only — logic will be added later.
func runEncode(cc *connctx.ConnectionContext) {
	log.Printf("conn %d: encode mode started (epoch=%d)", cc.ID, cc.Epoch)

	// HANDOFF POINT:
	// Encode state machine will live here.
	// For now, block until cancelled.
	<-cc.Ctx.Done()

	log.Printf("conn %d: encode mode stopped", cc.ID)
}

// runDecode handles the DECODE path (compressed -> raw).
// Placeholder only — logic will be added later.
func runDecode(cc *connctx.ConnectionContext) {
	log.Printf("conn %d: decode mode started", cc.ID)

	// HANDOFF POINT:
	// Decode state machine will live here.
	<-cc.Ctx.Done()

	log.Printf("conn %d: decode mode stopped", cc.ID)
}
