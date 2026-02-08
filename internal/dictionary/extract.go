// internal/dictionary/extract.go
package dictionary

// ExtractSpans scans a single packet and returns span observations.
// It is PURE: no side effects, no global state, no dictionary access.
//
// Rules enforced:
// - Packet-local only
// - Whole spans only
// - Overlap allowed
// - Intra-packet frequency counted
func ExtractSpans(ctx PacketContext) []SpanObservation {
	b := ctx.PacketBytes
	n := len(b)
	if n == 0 {
		return nil
	}

	// Map by (constA,constB) identity; VAR excluded.
	type key struct {
		a string
		b string
	}

	obs := make(map[key]*SpanObservation)

	// We discover spans by choosing a start (i) and an end (j),
	// and treating the middle as VAR. This is conservative:
	// - constA = b[i:iA]
	// - VAR    = b[iA:iB]
	// - constB = b[iB:j]
	//
	// To keep this bounded and sane for v1:
	// - constA and constB must be non-empty
	// - VAR must be non-empty
	//
	// NOTE: This is intentionally simple and byte-blind.
	for i := 0; i < n; i++ {
		// constA must be non-empty
		for iA := i + 1; iA < n-1; iA++ {
			// VAR must be non-empty; constB must be non-empty
			for iB := iA + 1; iB < n; iB++ {
				if iB+1 > n {
					continue
				}

				constA := b[i:iA]
				varPart := b[iA:iB]
				constB := b[iB : iB+1]

				if len(constA) == 0 || len(varPart) == 0 || len(constB) == 0 {
					continue
				}

				k := key{
					a: string(constA),
					b: string(constB),
				}

				o, ok := obs[k]
				if !ok {
					o = &SpanObservation{
						ConstA:         append([]byte(nil), constA...),
						ConstB:         append([]byte(nil), constB...),
						IntraHits:      0,
						SpanSize:       uint32(len(constA) + len(varPart) + len(constB)),
						LastSeenOffset: ctx.PacketOffset + uint64(iB),
					}
					obs[k] = o
				}

				o.IntraHits++
				// last occurrence wins
				o.LastSeenOffset = ctx.PacketOffset + uint64(iB)
			}
		}
	}

	// Flatten
	out := make([]SpanObservation, 0, len(obs))
	for _, v := range obs {
		out = append(out, *v)
	}
	return out
}
