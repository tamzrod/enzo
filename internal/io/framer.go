// internal/io/framer.go
package io

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"time"
)

// BufferedConn wraps net.Conn and adds buffering + record awareness
// while still fully implementing net.Conn.
type BufferedConn struct {
	net.Conn
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

// ReadLine reads a full line ending in '\n'.
func (b *BufferedConn) ReadLine() ([]byte, error) {
	line, err := b.Reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return line, nil
}

// Read implements io.Reader.
func (b *BufferedConn) Read(p []byte) (int, error) {
	return b.Reader.Read(p)
}

// Write implements io.Writer and flushes immediately.
func (b *BufferedConn) Write(p []byte) (int, error) {
	n, err := b.Writer.Write(p)
	if err != nil {
		return n, err
	}
	return n, b.Writer.Flush()
}

// WriteLine writes a full logical record.
func (b *BufferedConn) WriteLine(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	if !bytes.HasSuffix(p, []byte{'\n'}) {
		return errors.New("framer: line does not end with newline")
	}
	_, err := b.Write(p)
	return err
}

// ---- net.Conn interface forwarding ----

func (b *BufferedConn) LocalAddr() net.Addr {
	return b.Conn.LocalAddr()
}

func (b *BufferedConn) RemoteAddr() net.Addr {
	return b.Conn.RemoteAddr()
}

func (b *BufferedConn) SetDeadline(t time.Time) error {
	return b.Conn.SetDeadline(t)
}

func (b *BufferedConn) SetReadDeadline(t time.Time) error {
	return b.Conn.SetReadDeadline(t)
}

func (b *BufferedConn) SetWriteDeadline(t time.Time) error {
	return b.Conn.SetWriteDeadline(t)
}
