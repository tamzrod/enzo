// internal/dictionary/span.go
package dictionary

// PacketContext describes a single packet presented
// to the span extractor.
type PacketContext struct {
	// PacketBytes is the complete, atomic packet payload.
	// Span extraction MUST NOT read outside this buffer.
	PacketBytes []byte

	// PacketOffset is the starting RAW-window offset
	// corresponding to PacketBytes[0].
	PacketOffset uint64

	// PacketID is a monotonically increasing identifier
	// for packet-level de-duplication.
	PacketID uint64
}

// SpanObservation represents one discovered candidate span
// inside a single packet.
type SpanObservation struct {
	// Pattern identity (VAR excluded)
	ConstA []byte
	ConstB []byte

	// Intra-packet frequency
	// Number of full-span occurrences inside this packet.
	IntraHits uint16

	// SpanSize is total byte size of CONST_A + VAR + CONST_B
	SpanSize uint32

	// LastSeenOffset is where the LAST occurrence of this
	// span ended in the RAW window.
	LastSeenOffset uint64
}
