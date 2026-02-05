package state

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/protocol"
)

/*
HTTP helper functions
- no parsing beyond CRLF
- no semantics beyond Content-Length
- preserves byte correctness
*/

// readHTTPHeaderBlock reads until "\r\n\r\n".
// Returns:
//   hdr   = full header bytes including CRLFCRLF
//   cl    = Content-Length or -1 if missing/invalid
//   extra = bytes already read past header (start of body)
func readHTTPHeaderBlock(r io.Reader) (hdr []byte, cl int, extra []byte, err error) {
	cl = -1

	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 2048)

	for {
		if i := bytes.Index(buf, []byte("\r\n\r\n")); i >= 0 {
			end := i + 4
			hdr = append([]byte(nil), buf[:end]...)
			if end < len(buf) {
				extra = append([]byte(nil), buf[end:]...)
			}
			cl = parseContentLength(hdr)
			return
		}

		n, e := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > 256*1024 {
				return nil, -1, nil, errors.New("http header too large")
			}
		}
		if e != nil {
			return nil, -1, nil, e
		}
	}
}

// readExactBody reads exactly Content-Length bytes.
// Uses `extra` first if present.
func readExactBody(r io.Reader, contentLen int, extra []byte) ([]byte, error) {
	if contentLen <= 0 {
		return []byte{}, nil
	}

	body := make([]byte, 0, contentLen)

	if len(extra) > 0 {
		if len(extra) >= contentLen {
			return append(body, extra[:contentLen]...), nil
		}
		body = append(body, extra...)
	}

	remain := contentLen - len(body)
	if remain > 0 {
		tmp := make([]byte, remain)
		if _, err := io.ReadFull(r, tmp); err != nil {
			return nil, err
		}
		body = append(body, tmp...)
	}

	return body, nil
}

// rewriteContentLength rewrites or inserts Content-Length to match newLen.
func rewriteContentLength(hdr []byte, newLen int) []byte {
	term := []byte("\r\n\r\n")
	i := bytes.Index(hdr, term)
	if i < 0 {
		return hdr
	}

	head := hdr[:i]
	tail := hdr[i:]

	lines := bytes.Split(head, []byte("\r\n"))
	found := false

	for idx, l := range lines {
		if len(l) >= 15 && strings.EqualFold(string(l[:15]), "Content-Length:") {
			lines[idx] = []byte("Content-Length: " + strconv.Itoa(newLen))
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, []byte("Content-Length: "+strconv.Itoa(newLen)))
	}

	out := bytes.Join(lines, []byte("\r\n"))
	out = append(out, tail...)
	return out
}

// passThroughRawStreamWithPrefix sends prefix bytes then continues RAW passthrough.
func passThroughRawStreamWithPrefix(
	cc *connctx.ConnectionContext,
	stats *connStats,
	hdr []byte,
	extra []byte,
) error {

	if len(hdr) > 0 {
		stats.rawIn.Add(uint64(len(hdr)))
		if err := writeFrameCount(cc, protocol.FrameRawData, hdr, stats); err != nil {
			return err
		}
	}
	if len(extra) > 0 {
		stats.rawIn.Add(uint64(len(extra)))
		if err := writeFrameCount(cc, protocol.FrameRawData, extra, stats); err != nil {
			return err
		}
	}

	buf := make([]byte, 32*1024)
	for {
		n, err := cc.Source.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			stats.rawIn.Add(uint64(n))
			if err := writeFrameCount(cc, protocol.FrameRawData, chunk, stats); err != nil {
				return err
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
