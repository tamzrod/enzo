// internal/accept/listener.go
package accept

import (
	"context"
	"log"
	"net"
	"sync/atomic"

	"github.com/tamzrod/enzo/internal/connctx"
	enio "github.com/tamzrod/enzo/internal/io"
	"github.com/tamzrod/enzo/internal/protocol"
	"github.com/tamzrod/enzo/internal/state"
)

// Listener owns the TCP accept loop.
type Listener struct {
	ListenAddr string
	DestAddr   string

	nextID atomic.Uint64
}

// Run starts the blocking accept loop.
func (l *Listener) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", l.ListenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Printf("enzo listening on %s -> %s", l.ListenAddr, l.DestAddr)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		srcConn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		go l.handleConn(ctx, srcConn)
	}
}

func (l *Listener) handleConn(parent context.Context, rawSrc net.Conn) {
	defer rawSrc.Close()

	rawDst, err := net.Dial("tcp", l.DestAddr)
	if err != nil {
		log.Printf("dial destination failed: %v", err)
		return
	}

	src := enio.NewBufferedConn(rawSrc)
	dst := enio.NewBufferedConn(rawDst)

	mode, err := classifyMode(src)
	if err != nil {
		log.Printf("stream classify failed: %v", err)
		dst.Close()
		return
	}

	ctx, cancel := context.WithCancel(parent)

	cc := &connctx.ConnectionContext{
		ID:          l.nextID.Add(1),
		Mode:        mode,
		Source:      src,
		Destination: dst,
		Epoch:       protocol.InitialEpoch,
		Dictionary:  nil,
		Ctx:         ctx,
		Cancel:      cancel,
	}

	log.Printf("conn %d accepted (%s)", cc.ID, modeString(cc.Mode))

	// HANDOFF: state machine owns lifecycle from here
	state.Run(cc)

	dst.Close()
}

func classifyMode(c *enio.BufferedConn) (connctx.Mode, error) {
	b, err := c.PeekByte()
	if err != nil {
		return 0, err
	}

	if b == protocol.MagicByte {
		return connctx.ModeDecode, nil
	}
	return connctx.ModeEncode, nil
}

func modeString(m connctx.Mode) string {
	if m == connctx.ModeDecode {
		return "decode"
	}
	return "encode"
}
