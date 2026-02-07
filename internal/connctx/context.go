// internal/connctx/context.go
package connctx

import (
	"context"
	"sync/atomic"

	enio "github.com/tamzrod/enzo/internal/io"
	"github.com/tamzrod/enzo/internal/memory"
)

// Mode defines how this connection is processed.
// Determined once via magic-byte inspection.
type Mode uint8

const (
	ModeEncode Mode = iota
	ModeDecode
)

// ConnectionContext owns all state for exactly ONE TCP connection.
// It is never shared across connections.
type ConnectionContext struct {
	// Immutable after creation
	ID   uint64
	Mode Mode

	// Network endpoints (opaque byte transport; no record semantics)
	Source      *enio.PayloadConn
	Destination *enio.PayloadConn

	// Epoch state (monotonically increasing)
	Epoch uint32

	// Dictionary / agreement state (opaque for now)
	Dictionary any

	// ---- Observer-only memory (LOCKED v1) ----
	// RawWindowFwd observes the forward-direction "truth stream":
	// - In ENCODE: normalized HTTP header + dechunked body (what backend will see).
	// - In DECODE: reconstructed HTTP request bytes flushed to Destination.
	// It must NEVER influence wire flow.
	RawWindowFwd *memory.RawWindow

	// RawWindowRev observes reverse-direction raw bytes (responses).
	// It must NEVER influence wire flow.
	RawWindowRev *memory.RawWindow

	// Lifecycle control
	Ctx    context.Context
	Cancel context.CancelFunc

	// Internal flags
	Closed atomic.Bool
}
