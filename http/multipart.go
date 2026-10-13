package http

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// MultipartFile holds one multipart file part.
	MultipartFile struct {
		// Filename is the uploaded filename from the part headers.
		Filename string
		// ContentType is the part content type.
		ContentType string
		// Data contains the full part payload.
		Data []byte
	}

	// MultipartForm is the parsed in-memory representation of a multipart
	// request body.
	MultipartForm struct {
		// Values contains non-file part values keyed by form field name.
		Values url.Values
		// Files contains file parts keyed by form field name.
		Files map[string][]MultipartFile
	}
)

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
	var form *MultipartForm
	err := DecodeMultipartRequestWithLimit(req, maxBytes, func(reader *multipart.Reader) error {
		var err error
		form, err = readMultipartForm(reader)
		return err
	})
	if err != nil {
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
	if req == nil || req.Body == nil {
		return fmt.Errorf("multipart request and body cannot be nil")
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
	form := &MultipartForm{
		Values: url.Values{},
		Files:  make(map[string][]MultipartFile),
	}
	for {
		part, err := mr.NextPart()
		if err != nil {
			if err == io.EOF {
				return form, nil
			}
			return nil, err
		}
		name := part.FormName()
		if name == "" {
			_, readErr := io.Copy(io.Discard, part)
			if readErr != nil {
				return nil, readErr
			}
			if closeErr := part.Close(); closeErr != nil {
				return nil, closeErr
			}
			continue
		}
		data, readErr := io.ReadAll(part)
		if readErr != nil {
			return nil, readErr
		}
		closeErr := part.Close()
		if closeErr != nil {
			return nil, closeErr
		}
		if filename := part.FileName(); filename != "" {
			form.Files[name] = append(form.Files[name], MultipartFile{
				Filename:    filename,
				ContentType: strings.TrimSpace(part.Header.Get("Content-Type")),
				Data:        data,
			})
			continue
		}
		form.Values.Add(name, string(data))
	}
}
