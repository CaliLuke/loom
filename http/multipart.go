package http

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"
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

// ReadMultipartForm reads all multipart parts from mr into an in-memory form
// representation suitable for generated request decoding.
func ReadMultipartForm(mr *multipart.Reader) (*MultipartForm, error) {
	if mr == nil {
		return nil, fmt.Errorf("multipart reader cannot be nil")
	}
	form := &MultipartForm{
		Values: url.Values{},
		Files:  make(map[string][]MultipartFile),
	}
	remaining := int64(DefaultMaxRequestBodyBytes)
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
			data, readErr := readAllLimited(part, remaining)
			if readErr != nil {
				return nil, requestBodyDecodeError(readErr, true)
			}
			remaining -= int64(len(data))
			if closeErr := part.Close(); closeErr != nil {
				return nil, closeErr
			}
			continue
		}
		if remaining <= 0 {
			return nil, requestBodyDecodeError(errRequestBodyTooLarge, true)
		}
		data, readErr := readAllLimited(part, remaining)
		if readErr != nil {
			return nil, requestBodyDecodeError(readErr, true)
		}
		closeErr := part.Close()
		if closeErr != nil {
			return nil, closeErr
		}
		remaining -= int64(len(data))
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
