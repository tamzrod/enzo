// internal/protocol/protocol.go
package protocol

// This file defines ENZO wire-protocol constants.
// These values are ARCHITECTURAL INVARIANTS.
// They must not be configurable.

// Protocol versioning
const (
	ProtocolVersion uint8 = 0x01
)

// MagicByte identifies an ENZO-compressed stream.
// Presence of this byte as the first byte indicates DECODE mode.
const (
	MagicByte uint8 = 0xEC
)

// FrameType defines the semantic meaning of a frame.
type FrameType uint8

const (
	FrameEpochReset FrameType = 0x01
	FrameTemplateDefine FrameType = 0x02
	FrameTemplateRef    FrameType = 0x03
	FrameRawData        FrameType = 0x04
)

// Header layout (fixed, v1)
//
// +--------+--------+--------+--------+
// | Magic  | Ver    | Type   | Flags  |
// +--------+--------+--------+--------+
// |        Length (uint32 BE)          |
// +-----------------------------------+
const (
	HeaderSizeBytes = 8
)

// Flags are reserved for future use.
// Must be zero in v1.
const (
	FlagNone uint8 = 0x00
)

// Epoch rules
const (
	// Epoch starts at zero and increments monotonically.
	InitialEpoch uint32 = 0
)

// Dictionary / template ID sizing (wire-level)
const (
	TemplateIDSizeBytes = 2 // uint16
)

// Safety bounds (protocol-level, not config)
const (
	MaxFramePayloadBytes = 16 * 1024 * 1024 // 16 MB hard cap
)
