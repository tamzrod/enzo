// internal/state/decode_v1.go
package state

import (
	"errors"
	"log"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/protocol"
)

func runDecodeTemplateV1(cc *connctx.ConnectionContext) error {
	log.Printf("conn %d: decode template v1 started", cc.ID)

	d, _ := cc.Dictionary.(*Dict)
	if d == nil {
		d = NewDict()
		cc.Dictionary = d
	}

	for {
		select {
		case <-cc.Ctx.Done():
			return cc.Ctx.Err()
		default:
		}

		h, payload, err := protocol.ReadFrame(cc.Source)
		if err != nil {
			return err
		}

		switch h.Type {
		case protocol.FrameRawData:
			if len(payload) > 0 {
				if _, werr := cc.Destination.Write(payload); werr != nil {
					return werr
				}
			}

		case protocol.FrameTemplateDefine:
			tid, a, b, perr := protocol.ParseTemplateDefinePayload(payload)
			if perr != nil {
				return perr
			}
			d.Templates[tid] = &Template{
				ID:     tid,
				ConstA: a,
				ConstB: b,
			}

		case protocol.FrameTemplateRef:
			tid, lane, perr := protocol.ParseTemplateRefPayload(payload)
			if perr != nil {
				return perr
			}
			t := d.Templates[tid]
			if t == nil {
				return errors.New("protocol violation: template ref before define")
			}

			out := make([]byte, 0, len(t.ConstA)+len(lane)+len(t.ConstB))
			out = append(out, t.ConstA...)
			out = append(out, lane...)
			out = append(out, t.ConstB...)

			if _, werr := cc.Destination.Write(out); werr != nil {
				return werr
			}

		case protocol.FrameEpochReset:
			epochID, perr := protocol.ParseEpochResetPayload(payload)
			if perr != nil {
				return perr
			}

			cc.Epoch = epochID
			d.Templates = make(map[uint16]*Template)

		default:
			return errors.New("protocol violation: unsupported frame type")
		}
	}
}
