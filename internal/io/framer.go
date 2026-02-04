// internal/io/framer.go
package io

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"
)

const (
	// MaxPayloadSize is a hard safety limit to prevent OOM.
	// This is an operational limit, not a protocol semantic.
	MaxPayloadSize = 16 * 1024 * 1024 // 16 MiB
)

// PayloadConn is a minimal wrapper over net.Conn that supports:
//
// - PeekByte() for stream classification (magic detection)
// - ReadPacket()/WritePacket() for explicit payload framing
//
// It does NOT:
// - split by delimiters
// - parse text
// - infer message boundaries
type PayloadConn struct {
	conn net.Conn
	r    *bufio.Reader
}

// NewPayloadConn wraps a net.Conn without altering semantics.
func NewPayloadConn(c net.Conn) *PayloadConn {
	return &PayloadConn{
		conn: c,
		r:    bufio.NewReader(c),
	}
}

// PeekByte returns the next byte without consuming it.
// Used only for stream classification.
func (p *PayloadConn) PeekByte() (byte, error) {
	b, err := p.r.Peek(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// ReadPacket reads exactly one payload packet from the stream,
// using an explicit uint32 BE length prefix.
//
// Producer responsibility: send [len][payload] atomically per dataset.
func (p *PayloadConn) ReadPacket() ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(p.r, lenBuf[:]); err != nil {
		return nil, err
	}

	n := binary.BigEndian.Uint32(lenBuf[:])
	if n == 0 {
		return []byte{}, nil
	}
	if n > MaxPayloadSize {
		return nil, errors.New("payloadconn: payload exceeds MaxPayloadSize")
	}

	payload := make([]byte, int(n))
	if _, err := io.ReadFull(p.r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// WritePacket writes exactly one payload packet to the stream,
// using an explicit uint32 BE length prefix.
func (p *PayloadConn) WritePacket(payload []byte) error {
	if len(payload) > MaxPayloadSize {
		return errors.New("payloadconn: payload exceeds MaxPayloadSize")
	}

	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(payload)))

	if _, err := p.conn.Write(lenBuf[:]); err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := p.conn.Write(payload)
	return err
}

// Read reads raw bytes from the connection.
// Reads flow through the buffered reader so any prior Peek is preserved.
func (p *PayloadConn) Read(b []byte) (int, error) {
	return p.r.Read(b)
}

// Write writes raw bytes to the connection.
func (p *PayloadConn) Write(b []byte) (int, error) {
	return p.conn.Write(b)
}

func (p *PayloadConn) Close() error {
	return p.conn.Close()
}

// ---- net.Conn forwarding ----

func (p *PayloadConn) LocalAddr() net.Addr {
	return p.conn.LocalAddr()
}

func (p *PayloadConn) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}

func (p *PayloadConn) SetDeadline(t time.Time) error {
	return p.conn.SetDeadline(t)
}

func (p *PayloadConn) SetReadDeadline(t time.Time) error {
	return p.conn.SetReadDeadline(t)
}

func (p *PayloadConn) SetWriteDeadline(t time.Time) error {
	return p.conn.SetWriteDeadline(t)
}
