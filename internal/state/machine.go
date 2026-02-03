// internal/state/machine.go
package state

import (
	"bytes"
	"errors"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/protocol"
)

// v1 template ID (hardcoded for first real compression step)
const v1TemplateID uint16 = 1

// connStats tracks per-connection metrics (encode side).
type connStats struct {
	rawIn   atomic.Uint64
	wireOut atomic.Uint64
	start   time.Time
}

func Run(cc *connctx.ConnectionContext) {
	defer cc.Cancel()

	if cc.Dictionary == nil {
		cc.Dictionary = NewDict()
	}

	stats := &connStats{start: time.Now()}

	// Reverse direction stays RAW passthrough (responses).
	doneBack := make(chan error, 1)
	go func() {
		_, err := io.Copy(cc.Source, cc.Destination)
		doneBack <- err
	}()

	var err error
	switch cc.Mode {
	case connctx.ModeEncode:
		err = runEncodeTemplateV1(cc, stats)
	case connctx.ModeDecode:
		err = runDecodeTemplateV1(cc)
	default:
		err = errors.New("unknown mode")
	}

	cc.Cancel()

	select {
	case backErr := <-doneBack:
		if backErr != nil && !errors.Is(backErr, io.EOF) {
			log.Printf("conn %d: reverse passthrough ended: %v", cc.ID, backErr)
		}
	default:
	}

	if cc.Mode == connctx.ModeEncode {
		printStats(cc, stats)
	}

	if err != nil && !errors.Is(err, io.EOF) {
		log.Printf("conn %d: forward path ended: %v", cc.ID, err)
	}
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

// ---------------- ENCODE ----------------

func runEncodeTemplateV1(cc *connctx.ConnectionContext, stats *connStats) error {
	log.Printf("conn %d: encode template v1 started (epoch=%d)", cc.ID, cc.Epoch)

	defined := false
	var constA []byte
	var constB []byte

	for {
		select {
		case <-cc.Ctx.Done():
			return cc.Ctx.Err()
		default:
		}

		line, err := cc.Source.ReadLine()
		if line != nil {
			stats.rawIn.Add(uint64(len(line)))
			payload := line

			if !defined {
				a, lane, b, ok := trySplitConstVarConst(payload)
				if ok {
					constA, constB = a, b

					defPayload, derr := protocol.BuildTemplateDefinePayload(
						v1TemplateID,
						constA,
						constB,
					)
					if derr != nil {
						return derr
					}
					if werr := writeFrameCount(
						cc,
						protocol.FrameTemplateDefine,
						defPayload,
						stats,
					); werr != nil {
						return werr
					}

					refPayload, rerr := protocol.BuildTemplateRefPayload(
						v1TemplateID,
						lane,
					)
					if rerr != nil {
						return rerr
					}
					if werr := writeFrameCount(
						cc,
						protocol.FrameTemplateRef,
						refPayload,
						stats,
					); werr != nil {
						return werr
					}

					defined = true
				} else {
					if werr := writeFrameCount(
						cc,
						protocol.FrameRawData,
						payload,
						stats,
					); werr != nil {
						return werr
					}
				}
			} else {
				lane, ok := tryExtractLane(payload, constA, constB)
				if ok {
					refPayload, rerr := protocol.BuildTemplateRefPayload(
						v1TemplateID,
						lane,
					)
					if rerr != nil {
						return rerr
					}
					if werr := writeFrameCount(
						cc,
						protocol.FrameTemplateRef,
						refPayload,
						stats,
					); werr != nil {
						return werr
					}
				} else {
					if werr := writeFrameCount(
						cc,
						protocol.FrameRawData,
						payload,
						stats,
					); werr != nil {
						return werr
					}
				}
			}
		}

		if err != nil {
			return err
		}
	}
}

func writeFrameCount(
	cc *connctx.ConnectionContext,
	t protocol.FrameType,
	payload []byte,
	stats *connStats,
) error {
	stats.wireOut.Add(uint64(protocol.HeaderSizeBytes + len(payload)))
	return protocol.WriteFrame(
		cc.Destination,
		t,
		protocol.FlagNone,
		payload,
	)
}

// ---------------- DECODE ----------------

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
			cc.Epoch++
			d.Templates = make(map[uint16]*Template)

		default:
			return errors.New("protocol violation: unsupported frame type")
		}
	}
}

// ---------------- Helpers ----------------

func trySplitConstVarConst(
	p []byte,
) (constA []byte, lane []byte, constB []byte, ok bool) {
	if len(p) < 3 || p[len(p)-1] != '\n' {
		return nil, nil, nil, false
	}

	idx := bytes.LastIndexByte(p, '=')
	if idx <= 0 || idx >= len(p)-2 {
		return nil, nil, nil, false
	}

	constA = append([]byte(nil), p[:idx+1]...)
	lane = append([]byte(nil), p[idx+1:len(p)-1]...)
	constB = []byte{'\n'}
	return constA, lane, constB, true
}

func tryExtractLane(
	p []byte,
	constA []byte,
	constB []byte,
) (lane []byte, ok bool) {
	if !bytes.HasPrefix(p, constA) || !bytes.HasSuffix(p, constB) {
		return nil, false
	}

	start := len(constA)
	end := len(p) - len(constB)
	if end < start {
		return nil, false
	}

	lane = append([]byte(nil), p[start:end]...)
	return lane, true
}
