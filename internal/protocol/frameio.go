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
	// v1 flags are reserved and must be 0x00.
	if flags != 0 {
		return fmt.Errorf("protocol: invalid flags for v1: 0x%02X", flags)
	}

	if len(payload) > MaxFramePayloadBytes {
		return fmt.Errorf("protocol: payload too large: %d > %d", len(payload), MaxFramePayloadBytes)
	}

	// Frame-type structural invariants (v1).
	if err := validatePayloadShapeV1(t, payload); err != nil {
		return err
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
// - Flags must be 0x00 in v1
// - Length must be <= MaxFramePayloadBytes
// - Frame-type structural invariants must hold (v1)
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
	// v1 flags are reserved and must be 0x00.
	if h.Flags != 0 {
		return Header{}, nil, fmt.Errorf("protocol: invalid flags for v1: 0x%02X", h.Flags)
	}
	if int(h.Length) > MaxFramePayloadBytes {
		return Header{}, nil, fmt.Errorf("protocol: payload too large: %d > %d", h.Length, MaxFramePayloadBytes)
	}

	if h.Length == 0 {
		// Validate shape even when empty payload (some types may require bytes).
		if err := validatePayloadShapeV1(h.Type, nil); err != nil {
			return Header{}, nil, err
		}
		return h, nil, nil
	}

	payload := make([]byte, int(h.Length))
	if _, err := io.ReadFull(r, payload); err != nil {
		return Header{}, nil, err
	}

	// Frame-type structural invariants (v1).
	if err := validatePayloadShapeV1(h.Type, payload); err != nil {
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

// validatePayloadShapeV1 enforces frame-type payload shape constraints for v1.
//
// NOTE:
// This is *structural* validation, not semantic interpretation.
// It exists to prevent receivers from "guessing" meaning when the payload
// does not match the protocol's required layout.
func validatePayloadShapeV1(t FrameType, payload []byte) error {
	switch t {
	case FrameEpochReset:
		// v1 requires 4-byte epoch ID (uint32 BE).
		if len(payload) != 4 {
			return fmt.Errorf("protocol: invalid epoch reset payload length: %d (expected 4)", len(payload))
		}
		return nil

	case FrameRawData:
		// RAW_DATA may be empty or any length (already bounded by MaxFramePayloadBytes).
		return nil

	case FrameTemplateDefine:
		// Validated by the template parser.
		return nil

	case FrameTemplateRef:
		// Validated by the ref parser.
		return nil

	case FrameTemplateInline:
		// Validated by the inline parser.
		return nil

	default:
		// v1 strict mode: unknown frame type rejected.
		return fmt.Errorf("protocol: unknown frame type: 0x%02X", byte(t))
	}
}

// ParseEpochResetPayload parses the v1 EPOCH_RESET payload (uint32 BE epoch ID).
func ParseEpochResetPayload(payload []byte) (uint32, error) {
	if len(payload) != 4 {
		return 0, fmt.Errorf("protocol: invalid epoch reset payload length: %d (expected 4)", len(payload))
	}
	return be.Uint32(payload), nil
}

// BuildEpochResetPayload builds the v1 EPOCH_RESET payload (uint32 BE epoch ID).
func BuildEpochResetPayload(epochID uint32) []byte {
	var b [4]byte
	be.PutUint32(b[:], epochID)
	return b[:]
}
