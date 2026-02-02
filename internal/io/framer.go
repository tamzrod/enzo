// internal/io/framer.go
package io

import (
	"bufio"
	"net"
)

// BufferedConn wraps a net.Conn and allows
// peeking and re-reading bytes safely.
type BufferedConn struct {
	Conn   net.Conn
	Reader *bufio.Reader
	Writer *bufio.Writer
}

// NewBufferedConn creates a buffered wrapper over net.Conn.
func NewBufferedConn(c net.Conn) *BufferedConn {
	return &BufferedConn{
		Conn:   c,
		Reader: bufio.NewReader(c),
		Writer: bufio.NewWriter(c),
	}
}

// PeekByte returns the next byte without consuming it.
func (b *BufferedConn) PeekByte() (byte, error) {
	buf, err := b.Reader.Peek(1)
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

// Read implements io.Reader using the buffered reader.
func (b *BufferedConn) Read(p []byte) (int, error) {
	return b.Reader.Read(p)
}

// Write implements io.Writer using the buffered writer.
func (b *BufferedConn) Write(p []byte) (int, error) {
	n, err := b.Writer.Write(p)
	if err != nil {
		return n, err
	}
	return n, b.Writer.Flush()
}

// Close closes the underlying connection.
func (b *BufferedConn) Close() error {
	return b.Conn.Close()
}
