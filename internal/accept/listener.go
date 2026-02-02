// internal/accept/listener.go
package accept

import (
	"context"
	"log"
	"net"
	"sync/atomic"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/protocol"
)

// Listener owns the TCP accept loop.
type Listener struct {
	ListenAddr string
	DestAddr   string

	nextID atomic.Uint64
}

// Run starts the blocking accept loop.
// It exits only on fatal listener error or context cancellation.
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

		src, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		go l.handleConn(ctx, src)
	}
}

func (l *Listener) handleConn(parent context.Context, src net.Conn) {
	defer src.Close()

	// Dial destination immediately
	dst, err := net.Dial("tcp", l.DestAddr)
	if err != nil {
		log.Printf("dial destination failed: %v", err)
		return
	}

	// Destination lifecycle is bound to context
	// Do not defer dst.Close() here

	// Peek first byte to classify stream
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
		Epoch:       0,
		Dictionary:  nil,
		Ctx:         ctx,
		Cancel:      cancel,
	}

	log.Printf("conn %d accepted (%s)", cc.ID, modeString(cc.Mode))

	// HANDOFF POINT
	// State machine will take ownership here.
	<-ctx.Done()

	dst.Close()
}

func classifyMode(c net.Conn) (connctx.Mode, error) {
	buf := make([]byte, 1)

	n, err := c.Read(buf)
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, net.ErrClosed
	}

	// NOTE:
	// This read consumes the byte.
	// A buffered wrapper will re-inject this byte in io/framer.go.
	if buf[0] == protocol.MagicByte {
