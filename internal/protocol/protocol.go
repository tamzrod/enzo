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
	FrameEpochReset     FrameType = 0x01
	FrameTemplateDefine FrameType = 0x02
	FrameTemplateRef    FrameType = 0x03
	FrameRawData        FrameType = 0x04

	// FrameTemplateInline carries both template definition and lane in ONE frame.
	// This exists to preserve the locked invariant: 1 payload -> 1 frame.
	FrameTemplateInline FrameType = 0x05
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
	InitialEpoch uint32 = 0
)

// Safety bounds (protocol-level, not config)
const (
	MaxFramePayloadBytes = 16 * 1024 * 1024 // 16 MB hard cap
)

// Template encoding (v1)
//
// TEMPLATE_DEFINE payload:
//   u16 templateID
//   u8  segmentCount
//   repeating segments:
//     u8  segType
//     u16 segLen
//     segBytes (only if segType == CONST_BYTES)
//
// TEMPLATE_REF payload:
//   u16 templateID
//   u8  laneCount
//   repeating lanes:
//     u16 laneLen
//     laneBytes
//
// TEMPLATE_INLINE payload (v1):
//   u16 templateID
//   u16 constALen
//   constA bytes
//   u16 constBLen
//   constB bytes
//   u16 laneLen
//   lane bytes
type SegmentType uint8

const (
	SegConstBytes SegmentType = 0x01
	SegVarLane    SegmentType = 0x02
)
