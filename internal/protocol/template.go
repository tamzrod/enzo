// internal/protocol/template.go
package protocol

import (
	"fmt"
)

// BuildTemplateDefinePayload builds the TEMPLATE_DEFINE payload for a simple
// [CONST][VAR][CONST] template.
func BuildTemplateDefinePayload(templateID uint16, constA []byte, constB []byte) ([]byte, error) {
	total := 2 + 1 +
		(1 + 2 + len(constA)) +
		(1 + 2) +
		(1 + 2 + len(constB))

	if total > MaxFramePayloadBytes {
		return nil, fmt.Errorf("protocol: define payload too large: %d", total)
	}

	out := make([]byte, total)
	i := 0

	be.PutUint16(out[i:i+2], templateID)
	i += 2

	out[i] = 3 // segmentCount
	i++

	// seg0 CONST
	out[i] = byte(SegConstBytes)
	i++
	be.PutUint16(out[i:i+2], uint16(len(constA)))
	i += 2
	copy(out[i:], constA)
	i += len(constA)

	// seg1 VAR
	out[i] = byte(SegVarLane)
	i++
	be.PutUint16(out[i:i+2], 0)
	i += 2

	// seg2 CONST
	out[i] = byte(SegConstBytes)
	i++
	be.PutUint16(out[i:i+2], uint16(len(constB)))
	i += 2
	copy(out[i:], constB)
	i += len(constB)

	return out, nil
}

// ParseTemplateDefinePayload parses the v1 define payload for [CONST][VAR][CONST] only.
func ParseTemplateDefinePayload(p []byte) (templateID uint16, constA []byte, constB []byte, err error) {
	if len(p) < 3 {
		return 0, nil, nil, fmt.Errorf("protocol: define payload too short")
	}
	i := 0

	templateID = be.Uint16(p[i : i+2])
	i += 2

	segCount := int(p[i])
	i++
	if segCount != 3 {
		return 0, nil, nil, fmt.Errorf("protocol: unsupported segment count: %d", segCount)
	}

	// seg0 CONST
	if SegmentType(p[i]) != SegConstBytes {
		return 0, nil, nil, fmt.Errorf("protocol: seg0 not CONST")
	}
	i++
	aLen := int(be.Uint16(p[i : i+2]))
	i += 2
	if aLen < 0 || i+aLen > len(p) {
		return 0, nil, nil, fmt.Errorf("protocol: bad constA length")
	}
	constA = append([]byte(nil), p[i:i+aLen]...)
	i += aLen

	// seg1 VAR
	if i >= len(p) || SegmentType(p[i]) != SegVarLane {
		return 0, nil, nil, fmt.Errorf("protocol: seg1 not VAR")
	}
	i++
	if i+2 > len(p) {
		return 0, nil, nil, fmt.Errorf("protocol: bad var segment")
	}
	i += 2 // len(0)

	// seg2 CONST
	if i >= len(p) || SegmentType(p[i]) != SegConstBytes {
		return 0, nil, nil, fmt.Errorf("protocol: seg2 not CONST")
	}
	i++
	if i+2 > len(p) {
		return 0, nil, nil, fmt.Errorf("protocol: bad constB length")
	}
	bLen := int(be.Uint16(p[i : i+2]))
	i += 2
	if bLen < 0 || i+bLen > len(p) {
		return 0, nil, nil, fmt.Errorf("protocol: bad constB length")
	}
	constB = append([]byte(nil), p[i:i+bLen]...)

	return templateID, constA, constB, nil
}

// BuildTemplateRefPayload builds a TEMPLATE_REF payload for a single lane.
func BuildTemplateRefPayload(templateID uint16, lane []byte) ([]byte, error) {
	total := 2 + 1 + 2 + len(lane)
	if total > MaxFramePayloadBytes {
		return nil, fmt.Errorf("protocol: ref payload too large: %d", total)
	}

	out := make([]byte, total)
	i := 0

	be.PutUint16(out[i:i+2], templateID)
	i += 2

	out[i] = 1 // laneCount
	i++

	be.PutUint16(out[i:i+2], uint16(len(lane)))
	i += 2

	copy(out[i:], lane)
	return out, nil
}

// ParseTemplateRefPayload parses the v1 ref payload for a single lane.
func ParseTemplateRefPayload(p []byte) (templateID uint16, lane []byte, err error) {
	if len(p) < 5 {
		return 0, nil, fmt.Errorf("protocol: ref payload too short")
	}
	i := 0

	templateID = be.Uint16(p[i : i+2])
	i += 2

	if p[i] != 1 {
		return 0, nil, fmt.Errorf("protocol: unsupported lane count")
	}
	i++

	lLen := int(be.Uint16(p[i : i+2]))
	i += 2

	if lLen < 0 || i+lLen > len(p) {
		return 0, nil, fmt.Errorf("protocol: bad lane length")
	}
	lane = append([]byte(nil), p[i:i+lLen]...)
	return templateID, lane, nil
}

// BuildTemplateInlinePayload builds TEMPLATE_INLINE for [CONST][VAR][CONST] + lane in ONE frame.
//
// Layout:
//   u16 templateID
//   u16 constALen
//   constA bytes
//   u16 constBLen
//   constB bytes
//   u16 laneLen
//   lane bytes
func BuildTemplateInlinePayload(templateID uint16, constA []byte, constB []byte, lane []byte) ([]byte, error) {
	total := 2 + 2 + len(constA) + 2 + len(constB) + 2 + len(lane)
	if total > MaxFramePayloadBytes {
		return nil, fmt.Errorf("protocol: inline payload too large: %d", total)
	}

	out := make([]byte, total)
	i := 0

	be.PutUint16(out[i:i+2], templateID)
	i += 2

	be.PutUint16(out[i:i+2], uint16(len(constA)))
	i += 2
	copy(out[i:], constA)
	i += len(constA)

	be.PutUint16(out[i:i+2], uint16(len(constB)))
	i += 2
	copy(out[i:], constB)
	i += len(constB)

	be.PutUint16(out[i:i+2], uint16(len(lane)))
	i += 2
	copy(out[i:], lane)

	return out, nil
}

// ParseTemplateInlinePayload parses TEMPLATE_INLINE payload.
func ParseTemplateInlinePayload(p []byte) (templateID uint16, constA []byte, constB []byte, lane []byte, err error) {
	if len(p) < 2+2+2+2 {
		return 0, nil, nil, nil, fmt.Errorf("protocol: inline payload too short")
	}
	i := 0

	templateID = be.Uint16(p[i : i+2])
	i += 2

	aLen := int(be.Uint16(p[i : i+2]))
	i += 2
	if aLen < 0 || i+aLen > len(p) {
		return 0, nil, nil, nil, fmt.Errorf("protocol: bad inline constA length")
	}
	constA = append([]byte(nil), p[i:i+aLen]...)
	i += aLen

	bLen := int(be.Uint16(p[i : i+2]))
	i += 2
	if bLen < 0 || i+bLen > len(p) {
		return 0, nil, nil, nil, fmt.Errorf("protocol: bad inline constB length")
	}
	constB = append([]byte(nil), p[i:i+bLen]...)
	i += bLen

	lLen := int(be.Uint16(p[i : i+2]))
	i += 2
	if lLen < 0 || i+lLen > len(p) {
		return 0, nil, nil, nil, fmt.Errorf("protocol: bad inline lane length")
	}
	lane = append([]byte(nil), p[i:i+lLen]...)

	return templateID, constA, constB, lane, nil
}
