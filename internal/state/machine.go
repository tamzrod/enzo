// internal/state/machine.go
package state

import (
	"errors"
	"io"
	"log"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/protocol"
)

// Run is the entry point for per-connection state handling.
// It dispatches based on the connection mode and owns the lifecycle.
//
// This function must NOT return until the connection is finished.
func Run(cc *connctx.ConnectionContext) {
	defer cc.Cancel()

	// Downstream passthrough (Destination -> Source) always stays RAW.
	// This preserves application responses (e.g., HTTP 204/200) without ENZO framing.
	// ENZO v0 compresses only the forward direction (Source -> Destination).
	doneBack := make(chan error, 1)
	go func() {
		_, err := io.Copy(cc.Source, cc.Destination)
		doneBack <- err
	}()

	var err error
	switch cc.Mode {
	case connctx.ModeEncode:
		err = runEncodeV0RawOnly(cc)
	case connctx.ModeDecode:
		err = runDecodeV0RawOnly(cc)
	default:
		err = errors.New("unknown mode")
	}

	// Stop the reverse copy and unwind.
	cc.Cancel()

	// Drain reverse copy result (non-blocking best effort).
	select {
	case backErr := <-doneBack:
		if backErr != nil && !errors.Is(backErr, io.EOF) {
			log.Printf("conn %d: reverse passthrough ended: %v", cc.ID, backErr)
		}
	default:
	}

	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, contextCanceledLike(err)) {
		log.Printf("conn %d: forward path ended: %v", cc.ID, err)
	}
}

// runEncodeV0RawOnly reads raw bytes from Source and sends ENZO frames to Destination.
// Frame type used: RAW_DATA only.
func runEncodeV0RawOnly(cc *connctx.ConnectionContext) error {
	log.Printf("conn %d: encode v0 started (epoch=%d)", cc.ID, cc.Epoch)

	// Keep chunks comfortably below MaxFramePayloadBytes.
	// This is v0 framing, not compression.
	buf := make([]byte, 32*1024)

	for {
		// Exit on cancellation
		select {
		case <-cc.Ctx.Done():
			log.Printf("conn %d: encode v0 cancelled", cc.ID)
			return cc.Ctx.Err()
		default:
		}

		n, err := cc.Source.Read(buf)
		if n > 0 {
			payload := buf[:n]
			if werr := protocol.WriteFrame(cc.Destination, protocol.FrameRawData, protocol.FlagNone, payload); werr != nil {
				return werr
			}
		}

		if err != nil {
			// io.EOF is a normal shutdown path.
			return err
		}
	}
}

// runDecodeV0RawOnly reads ENZO frames from Source and writes raw bytes to Destination.
// Supported frames in v0: RAW_DATA, EPOCH_RESET (ignored, dictionary-free v0).
func runDecodeV0RawOnly(cc *connctx.ConnectionContext) error {
	log.Printf("conn %d: decode v0 started", cc.ID)

	for {
		// Exit on cancellation
		select {
		case <-cc.Ctx.Done():
			log.Printf("conn %d: decode v0 cancelled", cc.ID)
			return cc.Ctx.Err()
		default:
		}

		h, payload, err := protocol.ReadFrame(cc.Source)
		if err != nil {
			return err
		}

		switch h.Type {
		case protocol.FrameRawData:
			if len(payload) == 0 {
				continue
			}
			if _, werr := cc.Destination.Write(payload); werr != nil {
				return werr
			}

		case protocol.FrameEpochReset:
			// v0 does not maintain dictionary state.
			// Epoch reset is accepted but has no operational effect yet.
			cc.Epoch++ // optional visibility; epoch handling becomes real in v1 templates
			continue

		default:
			// In v0 we are strict: unknown frame types are violations.
			return errors.New("protocol violation: unsupported frame type in v0")
		}
	}
}

// contextCanceledLike normalizes context cancellation without importing context here.
// We treat common cancellation text/EOF as non-fatal noise in logs.
func contextCanceledLike(err error) error {
	// Avoid importing context just for comparisons.
	// If needed later, we can tighten this.
	return err
}
