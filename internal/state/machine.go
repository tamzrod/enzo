// internal/state/machine.go
package state

import (
	"errors"
	"io"
	"log"
	"time"

	"github.com/tamzrod/enzo/internal/connctx"
)

func Run(cc *connctx.ConnectionContext) {
	defer cc.Cancel()

	if cc.Dictionary == nil {
		cc.Dictionary = NewDict()
	}

	stats := newConnStats()

	// ---- Periodic stats logger (every 5 seconds) ----
	statsTicker := time.NewTicker(5 * time.Second)
	defer statsTicker.Stop()

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
			}
		}
	}()

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
	<-done // wait for stats goroutine to stop

	select {
	case backErr := <-doneBack:
		if backErr != nil && !errors.Is(backErr, io.EOF) {
			log.Printf("conn %d: reverse passthrough ended: %v", cc.ID, backErr)
		}
	default:
	}

	// Optional final stats snapshot on exit
	if cc.Mode == connctx.ModeEncode {
		printStats(cc, stats)
	}

	if err != nil && !errors.Is(err, io.EOF) {
		log.Printf("conn %d: forward path ended: %v", cc.ID, err)
	}
}
