// internal/state/decode_v1.go
package state

import (
	"bytes"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/tamzrod/enzo/internal/connctx"
	"github.com/tamzrod/enzo/internal/protocol"
)

func runDecodeTemplateV1(cc *connctx.ConnectionContext) error {
	log.Printf("conn %d: decode v1 started", cc.ID)

	// Dictionary is decode-side state only.
	d, _ := cc.Dictionary.(*Dict)
	if d == nil {
		d = NewDict()
		cc.Dictionary = d
	}

	var req bytes.Buffer
	headerDone := false
	contentLen := -1
	bodyHave := 0

	resetReq := func() {
		req.Reset()
		headerDone = false
		contentLen = -1
		bodyHave = 0
	}

	flushReqIfComplete := func() error {
		if !headerDone || contentLen < 0 {
			return nil
		}
		if bodyHave < contentLen {
			return nil
		}

		// =========================
		// DEBUG START
		// =========================
		log.Printf("===== ENZO → INFLUX HTTP REQUEST BEGIN =====")
		b := req.Bytes()
		if len(b) > 800 {
			log.Printf("%s", b[:500])
			log.Printf("... (%d bytes total) ...", len(b))
			log.Printf("%s", b[len(b)-300:])
		} else {
			log.Printf("%s", b)
		}
		log.Printf("===== ENZO → INFLUX HTTP REQUEST END =====")
		// =========================
		// DEBUG END
		// =========================

		out := req.Bytes()
		for len(out) > 0 {
			n, err := cc.Destination.Write(out)
			if err != nil {
				return err
			}

			// ---- PURE OBSERVER (Option B) ----
			// Observe only bytes that were successfully written.
			if cc.RawWindowFwd != nil && n > 0 {
				cc.RawWindowFwd.Append(out[:n])
			}

			out = out[n:]
		}

		resetReq()
		return nil
	}

	feed := func(b []byte) error {
		if len(b) == 0 {
			return nil
		}

		req.Write(b)

		if !headerDone {
			all := req.Bytes()
			idx := bytes.Index(all, []byte("\r\n\r\n"))
			if idx >= 0 {
				headerDone = true
				hdr := string(all[:idx+4])
				contentLen = parseContentLengthFromHeaderString(hdr)
				if contentLen < 0 {
					return errors.New("decode protocol violation: missing Content-Length")
				}
				bodyHave = len(all) - (idx + 4)
			}
			return nil
		}

		all := req.Bytes()
		idx := bytes.Index(all, []byte("\r\n\r\n"))
		if idx < 0 {
			return errors.New("decode protocol violation: header marker lost")
		}
		bodyHave = len(all) - (idx + 4)
		return nil
	}

	for {
		select {
		case <-cc.Ctx.Done():
			return cc.Ctx.Err()
		default:
		}

		h, payload, err := protocol.ReadFrame(cc.Source)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		switch h.Type {

		case protocol.FrameRawData:
			if err := feed(payload); err != nil {
				return err
			}
			if err := flushReqIfComplete(); err != nil {
				return err
			}

		case protocol.FrameTemplateInline:
			tid, a, b, lane, perr := protocol.ParseTemplateInlinePayload(payload)
			if perr != nil {
				return perr
			}

			d.Templates[tid] = &Template{
				ID:     tid,
				ConstA: a,
				ConstB: b,
			}

			out := make([]byte, 0, len(a)+len(lane)+len(b))
			out = append(out, a...)
			out = append(out, lane...)
			out = append(out, b...)

			if err := feed(out); err != nil {
				return err
			}
			if err := flushReqIfComplete(); err != nil {
				return err
			}

		case protocol.FrameTemplateDefine:
			tid, a, b, perr := protocol.ParseTemplateDefinePayload(payload)
			if perr != nil {
				return perr
			}
			d.Templates[tid] = &Template{
				ID:     tid,
				ConstA: a,
				ConstB: b,
			}

		case protocol.FrameTemplateRef:
			tid, lane, perr := protocol.ParseTemplateRefPayload(payload)
			if perr != nil {
				return perr
			}
			t := d.Templates[tid]
			if t == nil {
				return errors.New("protocol violation: template ref before define/inline")
			}

			out := make([]byte, 0, len(t.ConstA)+len(lane)+len(t.ConstB))
			out = append(out, t.ConstA...)
			out = append(out, lane...)
			out = append(out, t.ConstB...)

			if err := feed(out); err != nil {
				return err
			}
			if err := flushReqIfComplete(); err != nil {
				return err
			}

		case protocol.FrameEpochReset:
			epochID, perr := protocol.ParseEpochResetPayload(payload)
			if perr != nil {
				return perr
			}

			cc.Epoch = epochID
			d.Templates = make(map[uint16]*Template)
			resetReq()

		default:
			return errors.New("protocol violation: unsupported frame type")
		}
	}
}

func parseContentLengthFromHeaderString(hdr string) int {
	lines := strings.Split(hdr, "\r\n")
	for _, line := range lines {
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "content-length:") {
			v := strings.TrimSpace(line[len("content-length:"):])
			n, err := strconv.Atoi(v)
			if err == nil && n >= 0 {
				return n
			}
		}
	}
	return -1
}
