package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// RequestBodyPolicy is an immutable request-body size policy. Its Handler
	// method applies the limit before generated or application code reads the
	// request body.
	RequestBodyPolicy struct {
		maxBytes int64
	}

	requestBodyLimitContextKey struct{}

	requestBodyLimitReadCloser struct {
		io.ReadCloser
		limit     int64
		remaining int64
		tooLarge  bool
		err       error
		readErr   error
	}
)

// NewRequestBodyPolicy validates maxBytes and returns an immutable request-body
// policy. maxBytes must be positive.
func NewRequestBodyPolicy(maxBytes int64) (RequestBodyPolicy, error) {
	if maxBytes <= 0 {
		return RequestBodyPolicy{}, fmt.Errorf("loom http request body limit must be positive")
	}
	return RequestBodyPolicy{maxBytes: maxBytes}, nil
}

// MaxBytes returns the largest request body accepted by the policy.
func (p RequestBodyPolicy) MaxBytes() int64 {
	return p.maxBytes
}

// Handler wraps next with the request-body limit. Generated JSON, text, form,
// and multipart decoders preserve an exceeded limit as Loom's standard 413
// service error. Raw-body services receive the same error when reading Body.
// Handler panics when p was not constructed by NewRequestBodyPolicy.
func (p RequestBodyPolicy) Handler(next http.Handler) http.Handler {
	if p.maxBytes <= 0 {
		panic("loom: use NewRequestBodyPolicy to construct a request body policy")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), requestBodyLimitContextKey{}, p.maxBytes)
		r = r.WithContext(ctx)
		if r.Body != nil {
			r.Body = &requestBodyLimitReadCloser{
				ReadCloser: r.Body,
				limit:      p.maxBytes,
				remaining:  p.maxBytes,
				tooLarge:   r.ContentLength > p.maxBytes,
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequestBodyLimit returns the configured limit from ctx. It returns
// DefaultMaxRequestBodyBytes when no RequestBodyPolicy is active.
func RequestBodyLimit(ctx context.Context) int64 {
	if ctx != nil {
		if limit, ok := ctx.Value(requestBodyLimitContextKey{}).(int64); ok && limit > 0 {
			return limit
		}
	}
	return DefaultMaxRequestBodyBytes
}

// NormalizeRequestBodyDecodeError preserves request-too-large errors across
// standard-library form and multipart wrappers.
func NormalizeRequestBodyDecodeError(err error) error {
	if err == nil {
		return nil
	}
	var serviceErr *loom.ServiceError
	if errors.As(err, &serviceErr) {
		return err
	}
	var maxBytesErr *http.MaxBytesError
	if errors.Is(err, errRequestBodyTooLarge) || errors.As(err, &maxBytesErr) {
		return loom.RequestBodyTooLargeError()
	}
	return err
}

func (r *requestBodyLimitReadCloser) Read(buffer []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	if r.tooLarge {
		r.err = loom.RequestBodyTooLargeError()
		return 0, r.err
	}
	if len(buffer) == 0 {
		return r.readSource(buffer)
	}
	if r.remaining == 0 {
		var probe [1]byte
		n, err := r.readSource(probe[:])
		if n > 0 {
			r.err = loom.RequestBodyTooLargeError()
			return 0, r.err
		}
		return 0, err
	}
	readLength := int64(len(buffer))
	if readLength > r.remaining {
		readLength = r.remaining
	}
	n, err := r.readSource(buffer[:readLength])
	r.remaining -= int64(n)
	return n, err
}

// readSource retains the first source error for decoder completion without
// changing Read retry behavior for policy-wrapped raw-body consumers.
func (r *requestBodyLimitReadCloser) readSource(buffer []byte) (int, error) {
	n, err := r.ReadCloser.Read(buffer)
	if r.readErr == nil {
		r.readErr = err
	}
	return n, err
}
