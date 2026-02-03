// internal/accept/listener.go
package accept

import (
	stdio "io"
	"errors"
	"log"
	"net"

	"github.com/tamzrod/enzo/internal/connctx"
	enio "github.com/tamzrod/enzo/internal/io"
)

// ListenAndServe accepts incoming TCP connections and classifies them.
func ListenAndServe(
	ln net.Listener,
	ctxFactory func(*enio.BufferedConn, connctx.Mode) *connctx.ConnectionContext,
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
	ctxFactory func(*enio.BufferedConn, connctx.Mode) *connctx.ConnectionContext,
	run func(*connctx.ConnectionContext),
) {
	defer c.Close()

	bc := enio.NewBufferedConn(c)

	b, err := bc.PeekByte()
	if err != nil {
		if errors.Is(err, stdio.EOF) || errors.Is(err, net.ErrClosed) {
			return // normal empty connection
		}
		log.Printf("stream classify failed: %v", err)
		return
	}

	mode := classify(b)
	cc := ctxFactory(bc, mode)
	run(cc)
}

func classify(b byte) connctx.Mode {
	// ENZO-framed streams start with protocol magic (handled later)
	// Default to encode for now
	return connctx.ModeEncode
}
