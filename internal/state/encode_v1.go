// internal/state/encode_v1.go
package state

import (
	"bufio"
	"bytes"
	"errors"
	"hash/fnv"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/httpio"
	"github.com/tamzrod/enzo/internal/protocol"
)

/*
LOCKED RULES (v1):

1) Byte-exact: Packet out must reconstruct packet in.
2) RAW fallback is always allowed.
3) Overlap is allowed.
4) Delayed promotion (economics first):
   - Hit 1 → RAW
   - Hit 2 → RAW
   - Hit 3 → DEFINE (once) + then REF if profitable (else RAW)
   - Hit ≥4 → REF if profitable (else RAW)
*/

const promoteAfterHits = 3

type spanStat struct {
	Hits    uint32
	Defined bool
}

func runEncodeTemplateV1(cc *connctx.ConnectionContext, stats *connStats) error {
	log.Printf("conn %d: encode v1 started (epoch=%d)", cc.ID, cc.Epoch)

	// Per-connection dictionary (decoder-visible state)
	d, _ := cc.Dictionary.(*Dict)
	if d == nil {
		d = NewDict()
		cc.Dictionary = d
	}

	// Per-connection encoder-only stats for promotion gating
	spanStats := make(map[uint16]*spanStat)

	// Read HTTP from raw TCP (chunked supported by httpio)
	r := bufio.NewReader(cc.Source)

	for {
		select {
		case <-cc.Ctx.Done():
			return cc.Ctx.Err()
		default:
		}

		meta, body, err := httpio.ReadHTTPRequestBody(
			r,
			64*1024,      // max header
			16*1024*1024, // max body
		)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		// Normalize headers:
		// - Remove Transfer-Encoding: chunked
		// - Force truthful Content-Length for the dechunked body
		hdr := normalizeHTTPHeaders(meta.HeaderBytes, len(body))

		// ---- PURE OBSERVER (Option B): observe reconstructed truth stream ----
		// This is the exact byte stream the backend will see after decode/expansion:
		// [normalized header bytes] + [dechunked body bytes]
		// It must NEVER influence wire flow.
		if cc.RawWindowFwd != nil {
			if len(hdr) > 0 {
				cc.RawWindowFwd.Append(hdr)
			}
			if len(body) > 0 {
				cc.RawWindowFwd.Append(body)
			}
		}

		// ---- HEADER COMPRESSION (DEFINE/REF) ----
		if err := encodeHeaderLines(cc, d, spanStats, stats, hdr); err != nil {
			return err
		}

		// ---- BODY COMPRESSION (Influx LP: key=value span factoring) ----
		if len(body) > 0 {
			stats.rawIn.Add(uint64(len(body)))
			if err := encodeBodyAsLines(cc, d, spanStats, stats, body); err != nil {
				return err
			}
		}
	}
}

// normalizeHTTPHeaders produces a valid HTTP header block ending in \r\n\r\n
// while removing Transfer-Encoding: chunked and forcing Content-Length to match bodyLen.
func normalizeHTTPHeaders(rawHdr []byte, bodyLen int) []byte {
	lines := bytes.Split(rawHdr, []byte("\r\n"))
	if len(lines) == 0 {
		return rawHdr
	}

	out := make([]byte, 0, len(rawHdr)+64)

	// request line
	out = append(out, lines[0]...)
	out = append(out, '\r', '\n')

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if len(line) == 0 {
			break
		}
		lower := strings.ToLower(string(line))
		if strings.HasPrefix(lower, "transfer-encoding:") {
			continue
		}
		if strings.HasPrefix(lower, "content-length:") {
			continue
		}
		out = append(out, line...)
		out = append(out, '\r', '\n')
	}

	out = append(out, []byte("Content-Length: ")...)
	out = append(out, []byte(strconv.Itoa(bodyLen))...)
	out = append(out, '\r', '\n')

	out = append(out, '\r', '\n')
	return out
}

/* ================= HEADER COMPRESSION ================= */

func encodeHeaderLines(
	cc *connctx.ConnectionContext,
	d *Dict,
	spanStats map[uint16]*spanStat,
	stats *connStats,
	hdr []byte,
) error {

	lines := bytes.Split(hdr, []byte("\r\n"))

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Re-append CRLF except final empty slice
		if i < len(lines)-1 {
			line = append(line, '\r', '\n')
		}
		if len(line) == 0 {
			continue
		}

		stats.rawIn.Add(uint64(len(line)))

		if constA, lane, constB, ok := trySplitHeaderLine(line); ok {
			if err := emitWithDelayedPromotionDefineRef(cc, d, spanStats, stats, constA, lane, constB); err != nil {
				return err
			}
			continue
		}

		if werr := writeFrameCount(cc, protocol.FrameRawData, line, stats); werr != nil {
			return werr
		}
	}

	return nil
}

func trySplitHeaderLine(
	line []byte,
) (constA []byte, lane []byte, constB []byte, ok bool) {

	if len(line) < 2 || !bytes.HasSuffix(line, []byte("\r\n")) {
		return nil, nil, nil, false
	}

	idx := bytes.Index(line, []byte(": "))
	if idx <= 0 {
		return nil, nil, nil, false
	}

	constA = append([]byte(nil), line[:idx+2]...)
	lane = append([]byte(nil), line[idx+2:len(line)-2]...)
	constB = []byte("\r\n")
	return constA, lane, constB, true
}

/* ================= BODY COMPRESSION (Influx LP) ================= */

func encodeBodyAsLines(
	cc *connctx.ConnectionContext,
	d *Dict,
	spanStats map[uint16]*spanStat,
	stats *connStats,
	body []byte,
) error {

	i := 0
	for i < len(body) {
		j := bytes.IndexByte(body[i:], '\n')
		if j < 0 {
			return writeFrameCount(cc, protocol.FrameRawData, body[i:], stats)
		}

		j = i + j + 1
		line := body[i:j]
		i = j

		stats.rawIn.Add(uint64(len(line)))

		if err := encodeInfluxLineKV(cc, d, spanStats, stats, line); err != nil {
			return err
		}
	}
	return nil
}

func encodeInfluxLineKV(
	cc *connctx.ConnectionContext,
	d *Dict,
	spanStats map[uint16]*spanStat,
	stats *connStats,
	line []byte,
) error {

	if len(line) < 2 || line[len(line)-1] != '\n' {
		return writeFrameCount(cc, protocol.FrameRawData, line, stats)
	}

	// We operate purely on bytes:
	// repeatedly factor "CONST_A...=" + LANE(value) + CONST_B(delim)
	// until we hit ' ' (timestamp separator) then RAW tail (timestamp + '\n').

	cur := 0

	for {
		eq := bytes.IndexByte(line[cur:], '=')
		if eq < 0 {
			return writeFrameCount(cc, protocol.FrameRawData, line[cur:], stats)
		}
		eq = cur + eq

		constA := append([]byte(nil), line[cur:eq+1]...)
		valStart := eq + 1

		nextComma := bytes.IndexByte(line[valStart:], ',')
		nextSpace := bytes.IndexByte(line[valStart:], ' ')
		nextNL := bytes.IndexByte(line[valStart:], '\n')

		delimIdx := -1
		delimByte := byte(0)

		selectDelim := func(idx int, b byte) {
			if idx < 0 {
				return
			}
			abs := valStart + idx
			if delimIdx < 0 || abs < delimIdx {
				delimIdx = abs
				delimByte = b
			}
		}

		selectDelim(nextComma, ',')
		selectDelim(nextSpace, ' ')
		selectDelim(nextNL, '\n')

		if delimIdx < 0 || delimIdx <= valStart {
			return writeFrameCount(cc, protocol.FrameRawData, line[cur:], stats)
		}

		lane := append([]byte(nil), line[valStart:delimIdx]...)
		constB := []byte{delimByte}

		if err := emitWithDelayedPromotionDefineRef(cc, d, spanStats, stats, constA, lane, constB); err != nil {
			return err
		}

		if delimByte == ',' {
			// ✅ FIX:
			// The comma is already emitted as constB of the current segment.
			// Next constA must start AFTER the comma to avoid ",," in reconstruction.
			cur = delimIdx + 1
			continue
		}

		if delimByte == ' ' {
			// After space is timestamp + '\n' => keep RAW for now
			return writeFrameCount(cc, protocol.FrameRawData, line[delimIdx+1:], stats)
		}

		// '\n' end
		return nil
	}
}

/* ================= DEFINE/REF EMIT (Delayed Promotion) ================= */

func emitWithDelayedPromotionDefineRef(
	cc *connctx.ConnectionContext,
	d *Dict,
	spanStats map[uint16]*spanStat,
	stats *connStats,
	constA []byte,
	lane []byte,
	constB []byte,
) error {

	raw := make([]byte, 0, len(constA)+len(lane)+len(constB))
	raw = append(raw, constA...)
	raw = append(raw, lane...)
	raw = append(raw, constB...)

	if len(constA)+len(constB) <= len(lane) {
		return writeFrameCount(cc, protocol.FrameRawData, raw, stats)
	}

	tid := templateIDFor(constA, constB)

	st, ok := spanStats[tid]
	if !ok {
		st = &spanStat{}
		spanStats[tid] = st
	}
	st.Hits++

	if st.Hits < promoteAfterHits {
		return writeFrameCount(cc, protocol.FrameRawData, raw, stats)
	}

	if !st.Defined {
		defPayload, derr := protocol.BuildTemplateDefinePayload(tid, constA, constB)
		if derr != nil {
			return writeFrameCount(cc, protocol.FrameRawData, raw, stats)
		}

		if werr := writeFrameCount(cc, protocol.FrameTemplateDefine, defPayload, stats); werr != nil {
			return werr
		}

		d.Templates[tid] = &Template{ID: tid, ConstA: constA, ConstB: constB}
		st.Defined = true

		refPayload, rerr := protocol.BuildTemplateRefPayload(tid, lane)
		if rerr == nil && len(refPayload) < len(raw) {
			return writeFrameCount(cc, protocol.FrameTemplateRef, refPayload, stats)
		}
		return writeFrameCount(cc, protocol.FrameRawData, raw, stats)
	}

	refPayload, rerr := protocol.BuildTemplateRefPayload(tid, lane)
	if rerr == nil && len(refPayload) < len(raw) {
		return writeFrameCount(cc, protocol.FrameTemplateRef, refPayload, stats)
	}

	return writeFrameCount(cc, protocol.FrameRawData, raw, stats)
}

/* ================= HELPERS ================= */

func templateIDFor(constA, constB []byte) uint16 {
	h := fnv.New32a()
	_, _ = h.Write(constA)
	_, _ = h.Write(constB)
	return uint16(h.Sum32() & 0xFFFF)
}

// Kept for reference/debug; adapter already provides contentLen.
func parseContentLength(hdr []byte) int {
	s := string(hdr)
	lines := strings.Split(s, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			v := strings.TrimSpace(line[len("content-length:"):])
			n, err := strconv.Atoi(v)
			if err == nil && n >= 0 {
				return n
			}
		}
	}
	return -1
}
