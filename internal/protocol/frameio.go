// internal/protocol/frameio.go
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

var (
	be = binary.BigEndian
)

// Header represents the fixed ENZO v1 frame header (8 bytes).
type Header struct {
	Magic   uint8
	Version uint8
	Type    FrameType
	Flags   uint8
	Length  uint32
}

// WriteFrame writes one complete ENZO frame: header + payload.
//
// Invariants enforced:
// - MagicByte and ProtocolVersion are fixed
// - payload length must match Length
// - payload length must be within MaxFramePayloadBytes
func WriteFrame(w io.Writer, t FrameType, flags uint8, payload []byte) error {
	if len(payload) > MaxFramePayloadBytes {
		return fmt.Errorf("protocol: payload too large: %d > %d", len(payload), MaxFramePayloadBytes)
	}

	h := Header{
		Magic:   MagicByte,
		Version: ProtocolVersion,
		Type:    t,
		Flags:   flags,
		Length:  uint32(len(payload)),
	}

	if err := writeHeader(w, &h); err != nil {
		return err
	}

	if len(payload) == 0 {
		return nil
	}

	_, err := w.Write(payload)
	return err
}

// ReadFrame reads one complete ENZO frame: header + payload.
//
// Strict rules:
// - Magic must match MagicByte
// - Version must match ProtocolVersion
// - Length must be <= MaxFramePayloadBytes
func ReadFrame(r io.Reader) (Header, []byte, error) {
	h, err := readHeader(r)
	if err != nil {
		return Header{}, nil, err
	}

	if h.Magic != MagicByte {
		return Header{}, nil, fmt.Errorf("protocol: bad magic: 0x%02X", h.Magic)
	}
	if h.Version != ProtocolVersion {
		return Header{}, nil, fmt.Errorf("protocol: bad version: 0x%02X", h.Version)
	}
	if int(h.Length) > MaxFramePayloadBytes {
		return Header{}, nil, fmt.Errorf("protocol: payload too large: %d > %d", h.Length, MaxFramePayloadBytes)
	}

	if h.Length == 0 {
		return h, nil, nil
	}

	payload := make([]byte, int(h.Length))
	if _, err := io.ReadFull(r, payload); err != nil {
		return Header{}, nil, err
	}
	return h, payload, nil
}

func writeHeader(w io.Writer, h *Header) error {
	// Header is always 8 bytes.
	var buf [HeaderSizeBytes]byte
	buf[0] = h.Magic
	buf[1] = h.Version
	buf[2] = byte(h.Type)
	buf[3] = h.Flags
	be.PutUint32(buf[4:8], h.Length)

	_, err := w.Write(buf[:])
	return err
}

func readHeader(r io.Reader) (Header, error) {
	var buf [HeaderSizeBytes]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Header{}, err
	}

	return Header{
		Magic:   buf[0],
		Version: buf[1],
		Type:    FrameType(buf[2]),
		Flags:   buf[3],
		Length:  be.Uint32(buf[4:8]),
	}, nil
}
