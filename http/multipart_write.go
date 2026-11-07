package http

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

// WriteMultipartForm writes the retained ordered part set and closes its MIME
// envelope. Content labels and raw bytes are supplied by captured field codecs.
func WriteMultipartForm(writer *multipart.Writer, form *MultipartForm) error {
	if writer == nil || form == nil {
		return fmt.Errorf("multipart encoding requires a writer and present form")
	}
	for _, value := range form.Parts {
		params := map[string]string{"name": value.Name}
		if value.Filename != "" {
			params["filename"] = value.Filename
		}
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", mime.FormatMediaType("form-data", params))
		if value.ContentType != "" {
			header.Set("Content-Type", value.ContentType)
		}
		part, err := writer.CreatePart(header)
		if err != nil {
			return err
		}
		if err := writeMediaBytes(part, value.Data); err != nil {
			return err
		}
	}
	return writer.Close()
}

// SetMultipartRequest publishes a complete MIME body only after encoding succeeds.
// A present empty part set produces an empty envelope, distinct from no body.
func SetMultipartRequest(request *http.Request, form *MultipartForm) error {
	var data bytes.Buffer
	writer := multipart.NewWriter(&data)
	if err := WriteMultipartForm(writer, form); err != nil {
		return err
	}
	encoded := data.Bytes()
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.ContentLength = int64(len(encoded))
	request.Body = io.NopCloser(bytes.NewReader(encoded))
	request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(encoded)), nil
	}
	return nil
}
