// internal/state/encode_v1.go
package state

import (
	"errors"
	"io"
	"log"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/protocol"
)

// v1 template ID (reserved for future use once template define can be made 1:1)
// NOTE: Under the locked architecture (1 payload -> 1 frame),
// emitting TEMPLATE_DEFINE + TEMPLATE_REF for a single payload is forbidden.
// So v1 currently runs in RAW-only mode.
const v1TemplateID uint16 = 1

func runEncodeTemplateV1(cc *connctx.ConnectionContext, stats *connStats) error {
	log.Printf("conn %d: encode v1 started (epoch=%d)", cc.ID, cc.Epoch)

	// Template state is intentionally kept for future,
	// but MUST NOT emit multi-frame output per payload.
	_ = v1TemplateID

	for {
		select {
		case <-cc.Ctx.Done():
			return cc.Ctx.Err()
		default:
		}

		// One payload in (explicit length-framed by producer).
		payload, err := cc.Source.ReadPacket()
		if payload != nil {
			log.Printf("[ENCODE IN ] len=%d", len(payload))
			stats.rawIn.Add(uint64(len(payload)))

			// Under the locked architecture:
			// - exactly one input payload
			// - must produce exactly one ENZO frame
			//
			// Therefore: RAW_DATA only for now.
			if werr := writeFrameCount(
				cc,
				protocol.FrameRawData,
				payload,
				stats,
			); werr != nil {
				return werr
			}
		}

		if err != nil {
			// Normal connection close.
			if errors.Is(err, io.EOF) {
				return err
			}
			return err
		}
	}
}
