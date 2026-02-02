// cmd/enzo/main.go
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamzrod/enzo/internal/accept"
)

func main() {
	// Minimal runtime configuration
	listenAddr := flag.String("listen", "0.0.0.0:9000", "listen address")
	destAddr := flag.String("dest", "127.0.0.1:8086", "destination address")
	flag.Parse()

	log.Printf("enzo starting")
	log.Printf("listen=%s dest=%s", *listenAddr, *destAddr)

	// Root lifecycle context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// OS signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("signal received: %v", sig)
		cancel()
	}()

	// Listener
	l := &accept.Listener{
		ListenAddr: *listenAddr,
		DestAddr:   *destAddr,
	}

	// Run listener (blocking)
	if err := l.Run(ctx); err != nil {
		// Context cancellation is a normal exit path
		if err == context.Canceled {
			log.Printf("enzo shutting down")
		} else {
			log.Printf("listener error: %v", err)
		}
	}

	// Small grace period for goroutines to unwind
	time.Sleep(100 * time.Millisecond)
	log.Printf("enzo exited")
}
