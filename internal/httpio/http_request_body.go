// internal/httpio/http_request_body.go
package httpio

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	ErrHTTPBadRequest   = errors.New("http: bad request")
	ErrHTTPUnsupported  = errors.New("http: unsupported transfer mode")
	ErrHTTPBodyTooLarge = errors.New("http: body too large")
)

type RequestMeta struct {
	Method          string
	Path            string
	Proto           string
	TransferChunked bool
	ContentLength   int64 // -1 if unknown
	HeaderBytes     []byte
}

// ReadHTTPRequestBody reads exactly ONE HTTP request from r:
// - reads request line + headers
// - reads body (chunked OR content-length)
// - returns meta + body bytes (BODY ONLY; no chunk size lines; no CRLF framing)
//
// It does NOT attempt to parse HTTP semantics beyond framing correctness.
// This is an adapter function, not ENZO core logic.
func ReadHTTPRequestBody(r *bufio.Reader, maxHeaderBytes int, maxBodyBytes int) (RequestMeta, []byte, error) {
	meta := RequestMeta{
		ContentLength: -1,
	}

	// ---- Read request line
	reqLine, err := readLineCRLF(r, maxHeaderBytes)
	if err != nil {
		return meta, nil, err
	}
	meta.HeaderBytes = append(meta.HeaderBytes, reqLine...)

	// Parse: METHOD SP PATH SP PROTO
	parts := strings.Split(strings.TrimSpace(string(reqLine)), " ")
	if len(parts) < 3 {
		return meta, nil, ErrHTTPBadRequest
	}
	meta.Method = parts[0]
	meta.Path = parts[1]
	meta.Proto = parts[2]

	// ---- Read headers until CRLF CRLF
	var (
		transferEncoding string
		contentLengthStr string
	)

	for {
		line, err := readLineCRLF(r, maxHeaderBytes-len(meta.HeaderBytes))
		if err != nil {
			return meta, nil, err
		}
		meta.HeaderBytes = append(meta.HeaderBytes, line...)

		if bytes.Equal(line, []byte("\r\n")) {
			break
		}

		lower := strings.ToLower(string(bytes.TrimSpace(line)))
		if strings.HasPrefix(lower, "transfer-encoding:") {
			transferEncoding = strings.TrimSpace(strings.TrimPrefix(lower, "transfer-encoding:"))
		} else if strings.HasPrefix(lower, "content-length:") {
			contentLengthStr = strings.TrimSpace(strings.TrimPrefix(lower, "content-length:"))
		}
	}

	meta.TransferChunked = strings.Contains(transferEncoding, "chunked")

	if contentLengthStr != "" {
		n, err := strconv.ParseInt(contentLengthStr, 10, 64)
		if err != nil || n < 0 {
			return meta, nil, ErrHTTPBadRequest
		}
		meta.ContentLength = n
	}

	// ---- Read body
	if meta.TransferChunked {
		body, err := readChunkedBody(r, maxBodyBytes)
		return meta, body, err
	}

	if meta.ContentLength >= 0 {
		if meta.ContentLength > int64(maxBodyBytes) {
			return meta, nil, ErrHTTPBodyTooLarge
		}
		body := make([]byte, meta.ContentLength)
		_, err := io.ReadFull(r, body)
		if err != nil {
			return meta, nil, err
		}
		return meta, body, nil
	}

	// If neither chunked nor content-length, this proxy adapter doesn’t guess.
	return meta, nil, ErrHTTPUnsupported
}

func readChunkedBody(r *bufio.Reader, maxBodyBytes int) ([]byte, error) {
	var body bytes.Buffer

	for {
		// chunk-size line: <hex>\r\n
		line, err := readLineCRLF(r, 64) // chunk size lines are small
		if err != nil {
			return nil, err
		}
		sizeHex := strings.TrimSpace(string(line))
		// allow chunk extensions: "1a;foo=bar"
		if i := strings.IndexByte(sizeHex, ';'); i >= 0 {
			sizeHex = sizeHex[:i]
		}

		n, err := strconv.ParseInt(sizeHex, 16, 64)
		if err != nil || n < 0 {
			return nil, ErrHTTPBadRequest
		}

		if n == 0 {
			// Terminator: "0\r\n" then trailing headers (optional) then "\r\n".
			// We will consume until a blank line.
			for {
				trailer, err := readLineCRLF(r, 1024)
				if err != nil {
					return nil, err
				}
				if bytes.Equal(trailer, []byte("\r\n")) {
					break
				}
			}
			break
		}

		if body.Len()+int(n) > maxBodyBytes {
			return nil, ErrHTTPBodyTooLarge
		}

		chunk := make([]byte, n)
		_, err = io.ReadFull(r, chunk)
		if err != nil {
			return nil, err
		}
		body.Write(chunk)

		// trailing CRLF after chunk data
		crlf := make([]byte, 2)
		_, err = io.ReadFull(r, crlf)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(crlf, []byte("\r\n")) {
			return nil, ErrHTTPBadRequest
		}
	}

	return body.Bytes(), nil
}

func readLineCRLF(r *bufio.Reader, max int) ([]byte, error) {
	// ReadString('\n') allocates; ReadBytes also allocates.
	// This is fine for headers (small) and chunk-size lines.
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) > max {
		return nil, ErrHTTPBodyTooLarge
	}
	if len(line) < 2 || line[len(line)-2] != '\r' || line[len(line)-1] != '\n' {
		return nil, fmt.Errorf("%w: missing CRLF", ErrHTTPBadRequest)
	}
	return line, nil
}
