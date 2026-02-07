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
	rawWindowSizeBytes = 2 * 1024 * 1024 // 2 MB (LOCKED v1)
	telemetryPeriod    = 10 * time.Second
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

	stats := newConnStats()

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
					printStats(cc, stats)
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
			}
		}
	}()

	// ---- Reverse path: RAW passthrough ----
	doneBack := make(chan error, 1)
	go func() {
		buf := make([]byte, 32*1024)
		for {
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
	<-done

	select {
	case backErr := <-doneBack:
		if backErr != nil && !errors.Is(backErr, io.EOF) {
			log.Printf("reverse ended: %v", backErr)
		}
	default:
	}

	if cc.Mode == connctx.ModeEncode {
		printStats(cc, stats)
	}

	if err != nil && !errors.Is(err, io.EOF) {
		log.Printf("forward ended: %v", err)
	}
}
