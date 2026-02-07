// internal/accept/listener.go
package accept

import (
	"errors"
	stdio "io"
	"log"
	"net"

	"github.com/tamzrod/enzo/internal/connctx"
	enio "github.com/tamzrod/enzo/internal/io"
	"github.com/tamzrod/enzo/internal/protocol"
)

// ListenAndServe accepts incoming TCP connections and classifies them.
func ListenAndServe(
	ln net.Listener,
	ctxFactory func(*enio.PayloadConn, connctx.Mode) *connctx.ConnectionContext,
	run func(*connctx.ConnectionContext),
) error {

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go handleConn(conn, ctxFactory, run)
	}
}

func handleConn(
	c net.Conn,
	ctxFactory func(*enio.PayloadConn, connctx.Mode) *connctx.ConnectionContext,
	run func(*connctx.ConnectionContext),
) {
	defer c.Close()

	pc := enio.NewPayloadConn(c)

	// Decide mode from the FIRST byte only.
	// - If MagicByte: ENZO-framed stream => decode
	// - Else: raw application stream => encode
	b, err := pc.PeekByte()
	if err != nil {
		if errors.Is(err, stdio.EOF) || errors.Is(err, net.ErrClosed) {
			return
		}
		log.Printf("stream classify failed: %v", err)
		return
	}

	mode := classifyFirstByte(b)

	cc := ctxFactory(pc, mode)
	if cc == nil {
		// Destination unavailable or context creation failed.
		// Per-connection failure only; listener must continue.
		return
	}

	run(cc)
}

func classifyFirstByte(b byte) connctx.Mode {
	if b == protocol.MagicByte {
		return connctx.ModeDecode
	}
	return connctx.ModeEncode
}
