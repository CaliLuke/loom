package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
)

type requestPreparation struct {
	body      io.ReadCloser
	closeOnce sync.Once
	closeErr  error
}

// FinalizeRequestEncoding completes generated request preparation before the
// transport takes ownership of the body. It preserves a prior encoding error and
// closes an unpublished body on failure. A supplied Content-Length is reconciled
// with the final body and framing state after the encoder has run.
//
// Explicit zero lengths require EOF before publication. The bounded-memory probe
// uses Request.Body's standard concurrent Read/Close contract: cancellation closes
// the body to unblock Read. Only a successful EOF and Close replace the body and
// replay function with http.NoBody. Positive streams and absent lengths retain
// their body and replay function unchanged.
func FinalizeRequestEncoding(req *http.Request, service, method string, encodeErr error) error {
	preparation := &requestPreparation{body: req.Body}
	if encodeErr != nil {
		preparation.close()
		return errors.Join(encodeErr, preparation.closeErr, req.Context().Err())
	}
	err := req.Context().Err()
	if err == nil {
		err = preparation.finalize(req)
	}
	if err != nil || preparation.closeErr != nil {
		preparation.close()
		return ErrEncodingError(service, method, errors.Join(err, preparation.closeErr, req.Context().Err()))
	}
	return nil
}

func requestContentLength(header http.Header) (int64, bool, error) {
	var values []string
	present := false
	for name, entries := range header {
		if strings.EqualFold(name, "Content-Length") {
			present = true
			values = append(values, entries...)
		}
	}
	if !present {
		return 0, false, nil
	}
	if len(values) == 0 {
		return 0, true, fmt.Errorf("empty Content-Length")
	}
	first := textproto.TrimString(values[0])
	for _, value := range values[1:] {
		if textproto.TrimString(value) != first {
			return 0, true, fmt.Errorf("conflicting Content-Length values")
		}
	}
	length, err := strconv.ParseUint(first, 10, 63)
	if err != nil {
		return 0, true, fmt.Errorf("invalid Content-Length %q: %w", first, err)
	}
	return int64(length), true, nil
}

func (p *requestPreparation) close() {
	p.closeOnce.Do(func() {
		if p.body != nil {
			p.closeErr = p.body.Close()
		}
	})
}

func (p *requestPreparation) finalize(req *http.Request) error {
	length, present, err := requestContentLength(req.Header)
	if err != nil || !present {
		return err
	}
	if len(req.TransferEncoding) > 0 {
		return fmt.Errorf("Content-Length conflicts with Transfer-Encoding")
	}
	for name := range req.Header {
		if strings.EqualFold(name, "Transfer-Encoding") {
			return fmt.Errorf("Content-Length conflicts with Transfer-Encoding")
		}
	}
	if req.ContentLength > 0 && req.ContentLength != length {
		return fmt.Errorf("Content-Length %d conflicts with encoded body length %d", length, req.ContentLength)
	}
	if length > 0 {
		if req.Body == nil || req.Body == http.NoBody {
			return fmt.Errorf("Content-Length %d requires a nonempty body", length)
		}
		req.ContentLength = length
		return nil
	}
	switch req.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return fmt.Errorf("explicit Content-Length 0 cannot be sent for %q requests; omit the optional length value for an empty body", req.Method)
	}
	if err := p.verifyEmpty(req.Context()); err != nil || p.closeErr != nil {
		return err
	}
	req.ContentLength = 0
	req.Body = http.NoBody
	req.GetBody = func() (io.ReadCloser, error) {
		return http.NoBody, nil
	}
	return nil
}

func (p *requestPreparation) verifyEmpty(ctx context.Context) error {
	if p.body == nil || p.body == http.NoBody {
		return ctx.Err()
	}
	callbackDone := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		p.close()
		close(callbackDone)
	})
	readErr := p.probeEmpty(ctx)
	p.close()
	if !stop() {
		<-callbackDone
	}
	return errors.Join(readErr, ctx.Err())
}

func (p *requestPreparation) probeEmpty(ctx context.Context) error {
	var b [1]byte
	// Like the standard io package, stop a reader that repeatedly makes no progress.
	for range 100 {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := p.body.Read(b[:])
		if n != 0 {
			mismatch := fmt.Errorf("Content-Length 0 conflicts with a nonempty body")
			if err == io.EOF {
				err = nil
			}
			return errors.Join(mismatch, err)
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return io.ErrNoProgress
}
