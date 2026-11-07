package http

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// MultipartPart retains a physical part and its uninterpreted payload.
	MultipartPart struct {
		// Name is the literal Content-Disposition form name.
		Name string
		// Filename is the optional uploaded filename.
		Filename string
		// ContentType is the part media label.
		ContentType string
		// Data contains the complete raw part payload.
		Data []byte
	}
	// MultipartForm is the parsed in-memory representation of a multipart
	// request body.
	MultipartForm struct {
		// Parts retains every physical part in source order, including empty names and files.
		Parts []MultipartPart
	}
	multipartSource struct {
		io.ReadCloser
		read    int64
		readErr error
	}
)

// ErrMultipartBodyAbsent identifies an absent multipart source. It unwraps to
// io.EOF for general body decoders; malformed envelopes and source failures never
// match this sentinel, even when their underlying error wraps io.EOF.
var ErrMultipartBodyAbsent = fmt.Errorf("multipart body is absent: %w", io.EOF)

// ReadMultipartForm reads the request's multipart body using RequestBodyLimit.
// The complete encoded body counts toward the limit, including headers,
// boundaries and epilogue. The caller retains ownership of Body.Close.
func ReadMultipartForm(req *http.Request) (*MultipartForm, error) {
	if req == nil {
		return nil, fmt.Errorf("multipart request cannot be nil")
	}
	return ReadMultipartFormWithLimit(req, RequestBodyLimit(req.Context()))
}

// ReadMultipartFormWithLimit reads a complete multipart request within maxBytes
// encoded source bytes. It returns no form on decoding or completion failure.
// The request must not already have been parsed; the caller owns Body.Close.
func ReadMultipartFormWithLimit(req *http.Request, maxBytes int64) (*MultipartForm, error) {
	if req == nil {
		return nil, fmt.Errorf("multipart request cannot be nil")
	}
	if req.Body == nil || req.Body == http.NoBody {
		return nil, ErrMultipartBodyAbsent
	}
	tracked := &multipartSource{ReadCloser: req.Body}
	request := *req
	request.Body = tracked
	var form *MultipartForm
	err := DecodeMultipartRequestWithLimit(&request, maxBytes, func(reader *multipart.Reader) error {
		var err error
		form, err = readMultipartForm(reader)
		return err
	})
	if err != nil {
		if errors.Is(err, io.EOF) && tracked.read == 0 && tracked.readErr == io.EOF { //nolint:errorlint // io.Reader absence requires bare EOF; wrapped EOF is a source failure.
			return nil, ErrMultipartBodyAbsent
		}
		return nil, err
	}
	return form, nil
}

// DecodeMultipartRequestWithLimit invokes decode with a multipart parser over
// a bounded request source. Before returning success it consumes any unread
// source, including epilogue, within maxBytes. A known oversized body is rejected
// before decode runs. Callback failures do not trigger extra reads; an observed
// size violation remains a request_too_large error. Request fields are unchanged
// and Body.Close remains caller-owned; decode must not retain the parser.
func DecodeMultipartRequestWithLimit(req *http.Request, maxBytes int64, decode func(*multipart.Reader) error) error {
	if req == nil {
		return fmt.Errorf("multipart request cannot be nil")
	}
	if req.Body == nil || req.Body == http.NoBody {
		return io.EOF
	}
	if maxBytes <= 0 {
		return fmt.Errorf("multipart request body limit must be positive")
	}
	if decode == nil {
		return fmt.Errorf("multipart decoder cannot be nil")
	}
	if req.ContentLength > maxBytes {
		return NormalizeRequestBodyDecodeError(errRequestBodyTooLarge)
	}
	body, installed := req.Body.(*requestBodyLimitReadCloser)
	if !installed || body.limit != maxBytes {
		body = &requestBodyLimitReadCloser{ReadCloser: req.Body, limit: maxBytes, remaining: maxBytes}
	}
	request := *req
	request.Body = body
	reader, err := request.MultipartReader()
	if err != nil {
		return NormalizeRequestBodyDecodeError(err)
	}
	decodeErr := decode(reader)
	if body.err != nil {
		return NormalizeRequestBodyDecodeError(body.err)
	}
	readErr := NormalizeRequestBodyDecodeError(body.readErr)
	var serviceErr *loom.ServiceError
	if errors.As(readErr, &serviceErr) && serviceErr.Name == loom.RequestBodyTooLarge {
		return readErr
	}
	if decodeErr != nil {
		return NormalizeRequestBodyDecodeError(decodeErr)
	}
	if readErr != nil {
		if readErr == io.EOF { //nolint:errorlint // io.Reader requires bare EOF; wrapped source errors remain failures.
			return nil
		}
		return readErr
	}
	_, err = io.Copy(io.Discard, body)
	return NormalizeRequestBodyDecodeError(err)
}

func readMultipartForm(mr *multipart.Reader) (*MultipartForm, error) {
	form := &MultipartForm{}
	for {
		part, err := mr.NextPart()
		if err != nil {
			if err == io.EOF {
				return form, nil
			}
			return nil, err
		}
		name := part.FormName()
		data, readErr := io.ReadAll(part)
		if readErr != nil {
			return nil, readErr
		}
		closeErr := part.Close()
		if closeErr != nil {
			return nil, closeErr
		}
		form.Parts = append(form.Parts, MultipartPart{
			Name: name, Filename: part.FileName(),
			ContentType: strings.TrimSpace(part.Header.Get("Content-Type")), Data: data,
		})
	}
}

func (s *multipartSource) Read(data []byte) (int, error) {
	count, err := s.ReadCloser.Read(data)
	s.read += int64(count)
	if err != nil && s.readErr == nil {
		s.readErr = err
	}
	return count, err
}
