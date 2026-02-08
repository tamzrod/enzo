// internal/state/machine.go
package state

import (
	"errors"
	"io"
	"log"
	"time"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/memory"
)

const (
	rawWindowSizeBytes      = 2 * 1024 * 1024 // 2 MB (LOCKED v1)
	telemetryPeriod         = 10 * time.Second
	reverseFastPathBytes    = 256 // reverse TTFB fast-path threshold (LOCKED v1)
)

func Run(cc *connctx.ConnectionContext) {
	defer cc.Cancel()

	if cc.Dictionary == nil {
		cc.Dictionary = NewDict()
	}

	// ---- RAW WINDOW (OBSERVER ONLY) ----
	if cc.RawWindowFwd == nil {
		cc.RawWindowFwd = memory.NewRawWindow(rawWindowSizeBytes)
	}
	if cc.RawWindowRev == nil {
		cc.RawWindowRev = memory.NewRawWindow(rawWindowSizeBytes)
	}

	statsFwd := newConnStats()
	statsRev := newConnStats()

	statsTicker := time.NewTicker(5 * time.Second)
	defer statsTicker.Stop()

	telemetryTicker := time.NewTicker(telemetryPeriod)
	defer telemetryTicker.Stop()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-cc.Ctx.Done():
				return

			case <-statsTicker.C:
				if cc.Mode == connctx.ModeEncode {
					printStats(cc, statsFwd)
				}

			case <-telemetryTicker.C:
				if cc.RawWindowFwd != nil {
					log.Printf(
						"observer fwd: %d / %d bytes (%.1f%%)",
						cc.RawWindowFwd.Len(),
						cc.RawWindowFwd.Cap(),
						100.0*float64(cc.RawWindowFwd.Len())/float64(cc.RawWindowFwd.Cap()),
					)
				}
				if cc.RawWindowRev != nil {
					log.Printf(
						"observer rev: %d / %d bytes (%.1f%%)",
						cc.RawWindowRev.Len(),
						cc.RawWindowRev.Cap(),
						100.0*float64(cc.RawWindowRev.Len())/float64(cc.RawWindowRev.Cap()),
					)
				}
			}
		}
	}()

	// ---- Reverse path: FAST-PATH then ENZO ----
	doneBack := make(chan error, 1)
	go func() {
		// Fast-path passthrough until threshold reached
		var seen uint32
		buf := make([]byte, 32*1024)

		for seen < reverseFastPathBytes {
			n, err := cc.Destination.Read(buf)
			if n > 0 {
				out := buf[:n]
				for len(out) > 0 {
					wn, werr := cc.Source.Write(out)
					if werr != nil {
						doneBack <- werr
						return
					}
					if cc.RawWindowRev != nil && wn > 0 {
						cc.RawWindowRev.Append(out[:wn])
					}
					out = out[wn:]
				}
				seen += uint32(n)
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					doneBack <- nil
					return
				}
				doneBack <- err
				return
			}
		}

		// Threshold crossed: switch to ENZO reverse pipeline
		rc := *cc
		rc.Source = cc.Destination
		rc.Destination = cc.Source

		// For reverse ENZO lane, treat reverse window as its forward observer
		rc.RawWindowFwd = cc.RawWindowRev
		rc.RawWindowRev = cc.RawWindowFwd

		var err error
		switch cc.Mode {
		case connctx.ModeEncode:
			rc.Mode = connctx.ModeDecode
			err = runDecodeTemplateV1(&rc)
		case connctx.ModeDecode:
			rc.Mode = connctx.ModeEncode
			err = runEncodeTemplateV1(&rc, statsRev)
		default:
			err = errors.New("unknown mode")
		}

		doneBack <- err
	}()

	// ---- Forward lane ----
	var err error
	switch cc.Mode {
	case connctx.ModeEncode:
		err = runEncodeTemplateV1(cc, statsFwd)
	case connctx.ModeDecode:
		err = runDecodeTemplateV1(cc)
	default:
		err = errors.New("unknown mode")
	}

	cc.Cancel()
	<-done

	var backErr error
	select {
	case backErr = <-doneBack:
	default:
	}

	if backErr != nil && !errors.Is(backErr, io.EOF) {
		log.Printf("reverse ended: %v", backErr)
	}

	if cc.Mode == connctx.ModeEncode {
		printStats(cc, statsFwd)
	}
	if cc.Mode == connctx.ModeDecode {
		rc := *cc
		rc.Mode = connctx.ModeEncode
		printStats(&rc, statsRev)
	}

	if err != nil && !errors.Is(err, io.EOF) {
		log.Printf("forward ended: %v", err)
	}
}
