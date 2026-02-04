// cmd/enzo/main.go
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"sync/atomic"

	"github.com/tamzrod/enzo/internal/accept"
	"github.com/tamzrod/enzo/internal/connctx"
	enio "github.com/tamzrod/enzo/internal/io"
	"github.com/tamzrod/enzo/internal/state"
)

var nextConnID atomic.Uint64

func main() {
	var listenAddr string
	var destAddr string

	flag.StringVar(&listenAddr, "listen", "", "listen address (ip:port)")
	flag.StringVar(&destAddr, "dest", "", "destination address (ip:port)")
	flag.Parse()

	if listenAddr == "" || destAddr == "" {
		log.Fatal("listen and dest must be specified")
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	log.Printf("enzo listening on %s -> %s", listenAddr, destAddr)

	ctxFactory := func(src *enio.PayloadConn, mode connctx.Mode) *connctx.ConnectionContext {
		dstConn, err := net.Dial("tcp", destAddr)
		if err != nil {
			log.Printf("destination dial failed: %v", err)
			src.Close()
			return nil
		}

		dst := enio.NewPayloadConn(dstConn)

		ctx, cancel := context.WithCancel(context.Background())

		return &connctx.ConnectionContext{
			ID:          nextConnID.Add(1),
			Mode:        mode,
			Source:      src,
			Destination: dst,
			Epoch:       0,
			Dictionary:  nil,
			Ctx:         ctx,
			Cancel:      cancel,
		}
	}

	if err := accept.ListenAndServe(ln, ctxFactory, state.Run); err != nil {
		log.Fatalf("accept loop failed: %v", err)
	}
}
