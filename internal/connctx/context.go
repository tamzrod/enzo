// internal/connctx/context.go
package connctx

import (
	"context"
	"sync/atomic"

	enio "github.com/tamzrod/enzo/internal/io"
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

	// Lifecycle control
	Ctx    context.Context
	Cancel context.CancelFunc

	// Internal flags
	Closed atomic.Bool
}
