// internal/state/encode_helpers.go
package state

import (
	"errors"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/protocol"
)

// writeFrameCount is the ONLY allowed outbound write path for encoder output.
//
// HARD INVARIANT (LOCKED):
// - If ENZO touches the data, it MUST emit an ENZO frame.
// - All encoder output MUST go through this function.
// - Bypassing this function is a hard violation.
//
// This function enforces:
// - explicit framing
// - magic byte presence
// - correct wire accounting
func writeFrameCount(
	cc *connctx.ConnectionContext,
	t protocol.FrameType,
	payload []byte,
	stats *connStats,
) error {

	if cc == nil || cc.Destination == nil {
		return errors.New("encode: nil connection context or destination")
	}

	// Defensive check: encoder must never emit unknown frame types
	switch t {
	case protocol.FrameRawData,
		protocol.FrameTemplateDefine,
		protocol.FrameTemplateRef,
		protocol.FrameTemplateInline,
		protocol.FrameEpochReset:
		// allowed
	default:
		return errors.New("encode: invalid frame type")
	}

	// Wire accounting: header cost is always paid (bounded worst-case loss)
	stats.wireOut.Add(uint64(protocol.HeaderSizeBytes + len(payload)))

	// SINGLE write path to the wire.
	return protocol.WriteFrame(
		cc.Destination,
		t,
		protocol.FlagNone,
		payload,
	)
}
