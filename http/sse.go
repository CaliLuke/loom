package http

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	sse "github.com/tmaxmax/go-sse"
)

type (
	// SSEEvent represents a parsed server-sent event frame.
	SSEEvent struct {
		ID   string
		Type string
		Data string
	}

	// SSEMessage describes an SSE frame to write to an output stream.
	SSEMessage struct {
		ID          string
		Type        string
		Data        string
		RetryMillis int64
	}

	// SSEStreamReader reads framed Server-Sent Events from a response body.
	// Lines may end in LF, CR, or CRLF as defined by the WHATWG event-stream
	// format, and a block ends at the first blank line. A block longer than
	// bufio.MaxScanTokenSize bytes, the largest frame ParseSSEEvent accepts,
	// fails the stream with an error wrapping bufio.ErrTooLong instead of
	// being buffered without bound.
	SSEStreamReader struct {
		body   io.ReadCloser
		frames sseFrameScanner
		// readBuf is the reused body read buffer, guarded by readLock. A read
		// abandoned on cancellation may still write into it, which is safe
		// because cancellation closes the reader and no later read uses it.
		readBuf  []byte
		eof      bool
		readLock sync.Mutex
		lock     sync.Mutex
		closed   bool
	}

	sseReadResult struct {
		n   int
		err error
	}

	// sseFrameScanner splits buffered event-stream bytes into blocks. Its state
	// survives across reads so a line terminator split between two reads, such
	// as CR in one read and LF in the next, is interpreted exactly as if the
	// bytes had arrived together.
	sseFrameScanner struct {
		// buf holds unconsumed stream bytes starting at the current block.
		buf []byte
		// base is the backing array buf is compacted into so the window does
		// not slide into a reallocation on every read.
		base []byte
		// pos is the number of bytes of buf already scanned.
		pos int
		// lineLen is the length of the line being scanned.
		lineLen int
		// lines counts the non-blank lines of the current block.
		lines int
		// skipLF reports that the previous byte was a CR, so an LF that
		// follows completes the same CRLF terminator.
		skipLF bool
		// err is the sticky error recorded once a block exceeds
		// sseMaxFrameBytes.
		err error
	}
)

// sseMaxFrameBytes is the largest block SSEStreamReader buffers. It matches
// the bufio.Scanner token limit ParseSSEEvent parses frames with.
const sseMaxFrameBytes = bufio.MaxScanTokenSize

// sseUTF8BOM is the byte order mark go-sse strips from the first block.
const sseUTF8BOM = "\xEF\xBB\xBF"

// NewSSEStreamReader returns a reader for framed Server-Sent Events.
func NewSSEStreamReader(body io.ReadCloser) *SSEStreamReader {
	return &SSEStreamReader{
		body:    body,
		frames:  newSSEFrameScanner(),
		readBuf: make([]byte, 4096),
	}
}

// ReadEvent reads the next raw SSE event frame. Blocks that dispatch no event,
// such as keepalive comments, runs of blank lines, and blocks holding only
// retry or unknown fields, are consumed and skipped so every returned frame is
// accepted by ParseSSEEvent or reports why it is malformed.
func (r *SSEStreamReader) ReadEvent(ctx context.Context) ([]byte, error) {
	r.readLock.Lock()
	defer r.readLock.Unlock()

	for {
		frame, err := r.readFrame(ctx)
		if err != nil {
			return nil, err
		}
		if sseFrameDispatches(frame) {
			// frame aliases the scanner buffer; hand the caller its own copy.
			return append([]byte(nil), frame...), nil
		}
	}
}

// Close closes the SSE stream body.
func (r *SSEStreamReader) Close() error {
	r.lock.Lock()
	if r.closed {
		r.lock.Unlock()
		return nil
	}
	r.closed = true
	body := r.body
	r.lock.Unlock()
	return body.Close()
}

// newSSEFrameScanner returns an empty scanner with a 4 KiB buffer.
func newSSEFrameScanner() sseFrameScanner {
	base := make([]byte, 0, 4096)
	return sseFrameScanner{buf: base, base: base}
}

// sseFrameDispatches reports whether ParseSSEEvent yields an event or an error
// for frame, as opposed to a block that dispatches nothing, without parsing
// it. It mirrors go-sse v0.11.0: leading blank lines are skipped and a BOM
// that follows them is ignored; a data or event field, or an id field whose
// value has no NUL, dispatches (a line without a colon is a field with an
// empty value); retry, comment, and unknown lines do not; and an
// unterminated final line is an error that ParseSSEEvent reports.
func sseFrameDispatches(frame []byte) bool {
	rest := bytes.TrimLeft(frame, "\r\n")
	rest = bytes.TrimPrefix(rest, []byte(sseUTF8BOM))
	for len(rest) > 0 {
		end := bytes.IndexAny(rest, "\r\n")
		if end < 0 {
			return true
		}
		line := rest[:end]
		next := end + 1
		if rest[end] == '\r' && next < len(rest) && rest[next] == '\n' {
			next++
		}
		rest = rest[next:]
		name, value, _ := bytes.Cut(line, []byte(":"))
		switch string(name) {
		case "data", "event":
			return true
		case "id":
			if bytes.IndexByte(value, 0) < 0 {
				return true
			}
		}
	}
	return false
}

// readFrame returns the next complete block. At the end of the stream it
// returns any trailing incomplete block once, then io.EOF. The returned slice
// aliases scanner memory and is valid only until the next readFrame call.
func (r *SSEStreamReader) readFrame(ctx context.Context) ([]byte, error) {
	buf := r.readBuf
	for {
		frame, ok, err := r.frames.next()
		if err != nil {
			return nil, err
		}
		if ok {
			return frame, nil
		}
		if r.eof {
			return r.frames.rest()
		}

		body, err := r.currentBody(ctx)
		if err != nil {
			return nil, err
		}
		if body == nil {
			r.eof = true
			continue
		}

		n, err := r.readChunk(ctx, body, buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		r.frames.reserve(n)
		r.frames.buf = append(r.frames.buf, buf[:n]...)
		if errors.Is(err, io.EOF) {
			r.eof = true
		}
	}
}

// currentBody returns the body to read from, or nil once the reader has been
// closed.
func (r *SSEStreamReader) currentBody(ctx context.Context) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closed {
		return nil, nil
	}
	return r.body, nil
}

func (r *SSEStreamReader) readChunk(ctx context.Context, body io.Reader, buf []byte) (int, error) {
	readc := make(chan sseReadResult, 1)
	go func() {
		n, err := body.Read(buf)
		readc <- sseReadResult{n: n, err: err}
	}()

	select {
	case result := <-readc:
		return result.n, result.err
	case <-ctx.Done():
		select {
		case result := <-readc:
			return result.n, result.err
		default:
			if err := r.Close(); err != nil {
				// Preserve the client contract: cancellation is the
				// observable cause even when closing the body fails.
				return 0, ctx.Err()
			}
			return 0, ctx.Err()
		}
	}
}

// next returns the next complete block from the buffered bytes, including its
// terminating blank line. Blank lines that precede a block are discarded. It
// reports false when more bytes are needed, and an error once the current
// block exceeds sseMaxFrameBytes. The returned block aliases buf: appends only
// write past the consumed region, so it stays intact until the caller reads
// more input.
func (s *sseFrameScanner) next() ([]byte, bool, error) {
	for s.err == nil && s.pos < len(s.buf) {
		b := s.buf[s.pos]
		s.pos++
		if s.pos > sseMaxFrameBytes {
			s.err = fmt.Errorf("loom http: SSE event exceeds %d bytes: %w", sseMaxFrameBytes, bufio.ErrTooLong)
			s.buf = nil
			break
		}
		if s.skipLF {
			s.skipLF = false
			if b == '\n' {
				if s.lines == 0 && s.lineLen == 0 {
					s.consume()
				}
				continue
			}
		}
		if b != '\r' && b != '\n' {
			s.lineLen++
			continue
		}
		s.skipLF = b == '\r'
		if s.lineLen > 0 {
			s.lines++
			s.lineLen = 0
			continue
		}
		if s.lines == 0 {
			s.consume()
			continue
		}
		frame := s.buf[:s.pos]
		s.consume()
		s.lines = 0
		return frame, true, nil
	}
	return nil, false, s.err
}

// rest returns the buffered bytes of a trailing incomplete block, or io.EOF
// when none remain.
func (s *sseFrameScanner) rest() ([]byte, error) {
	if len(s.buf) == 0 {
		return nil, io.EOF
	}
	frame := s.buf
	s.buf = nil
	s.pos, s.lineLen, s.lines = 0, 0, 0
	return frame, nil
}

// reserve makes room to append n bytes to buf. When the spare capacity after
// buf is too small, it moves the live bytes to the front of base, growing base
// only when they do not fit; live data never exceeds sseMaxFrameBytes plus one
// read, which bounds base. Moving overwrites memory that blocks returned by
// next may alias. That is safe because a returned block is valid only until
// the next readFrame call, and ReadEvent copies any block it returns before
// reading again.
func (s *sseFrameScanner) reserve(n int) {
	if cap(s.buf)-len(s.buf) >= n {
		return
	}
	live := len(s.buf)
	if cap(s.base) < live+n {
		s.base = make([]byte, 0, max(min(2*cap(s.base), sseMaxFrameBytes+n), live+n))
	}
	s.buf = s.base[:copy(s.base[:live], s.buf)]
}

// consume discards the scanned bytes of buf.
func (s *sseFrameScanner) consume() {
	s.buf = s.buf[s.pos:]
	s.pos = 0
}

// ParseSSEEvent parses a single SSE event frame.
func ParseSSEEvent(data []byte) (SSEEvent, error) {
	events, err := ParseSSEStream(bytes.NewReader(data))
	if err != nil {
		return SSEEvent{}, err
	}
	if len(events) == 0 {
		return SSEEvent{}, fmt.Errorf("ParseSSEEvent: no event found")
	}
	if len(events) != 1 {
		return SSEEvent{}, fmt.Errorf("ParseSSEEvent: expected 1 event, got %d", len(events))
	}
	return events[0], nil
}

// ParseSSEStream parses all SSE events from the reader.
func ParseSSEStream(r io.Reader) ([]SSEEvent, error) {
	events := make([]SSEEvent, 0)
	for event, err := range sse.Read(r, nil) {
		if err != nil {
			return nil, err
		}
		events = append(events, SSEEvent{
			ID:   event.LastEventID,
			Type: event.Type,
			Data: event.Data,
		})
	}
	return events, nil
}

// WriteJSONSSEEvent marshals payload as JSON and writes it as an SSE event.
func WriteJSONSSEEvent(w io.Writer, msg SSEMessage, payload any) error {
	byts, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg.Data = string(byts)
	return WriteSSEEvent(w, msg)
}

// EncodeSSEData encodes a payload as an SSE data field.
func EncodeSSEData(payload any) (string, error) {
	switch v := payload.(type) {
	case nil:
		return "null", nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case int8:
		return fmt.Sprintf("%d", v), nil
	case int16:
		return fmt.Sprintf("%d", v), nil
	case int32:
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case uint:
		return fmt.Sprintf("%d", v), nil
	case uint8:
		return fmt.Sprintf("%d", v), nil
	case uint16:
		return fmt.Sprintf("%d", v), nil
	case uint32:
		return fmt.Sprintf("%d", v), nil
	case uint64:
		return fmt.Sprintf("%d", v), nil
	case float32:
		return fmt.Sprintf("%g", v), nil
	case float64:
		return fmt.Sprintf("%g", v), nil
	default:
		byts, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		return string(byts), nil
	}
}

// WriteSSEEvent writes a single SSE event frame.
func WriteSSEEvent(w io.Writer, msg SSEMessage) error {
	event := sse.Message{}
	if msg.ID != "" {
		id, err := sse.NewID(msg.ID)
		if err != nil {
			return err
		}
		event.ID = id
	}
	if msg.Type != "" {
		typ, err := sse.NewType(msg.Type)
		if err != nil {
			return err
		}
		event.Type = typ
	}
	if msg.RetryMillis > 0 {
		event.Retry = time.Duration(msg.RetryMillis) * time.Millisecond
	}
	event.AppendData(msg.Data)
	_, err := event.WriteTo(w)
	return err
}
