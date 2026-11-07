package http

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadMultipartForm(t *testing.T) {
	t.Run("scalar and file parts", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		require.NoError(t, writer.WriteField("title", "brief"))
		require.NoError(t, writer.WriteField("tags", "one"))
		require.NoError(t, writer.WriteField("tags", "two"))

		part, err := writer.CreateFormFile("file", "brief.md")
		require.NoError(t, err)
		_, err = part.Write([]byte("# hello"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		form, err := ReadMultipartForm(req)
		require.NoError(t, err)
		require.Equal(t, []string{"brief"}, multipartValues(form, "title"))
		require.Equal(t, []string{"one", "two"}, multipartValues(form, "tags"))
		require.Len(t, form.Parts, 4)
		require.Equal(t, "brief.md", form.Parts[3].Filename)
		require.Equal(t, "application/octet-stream", form.Parts[3].ContentType)
		require.Equal(t, []byte("# hello"), form.Parts[3].Data)
	})

	t.Run("multiple files for same field", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		first, err := writer.CreateFormFile("file", "first.txt")
		require.NoError(t, err)
		_, err = first.Write([]byte("first"))
		require.NoError(t, err)

		second, err := writer.CreateFormFile("file", "second.txt")
		require.NoError(t, err)
		_, err = second.Write([]byte("second"))
		require.NoError(t, err)

		require.NoError(t, writer.Close())

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		form, err := ReadMultipartForm(req)
		require.NoError(t, err)
		require.Len(t, form.Parts, 2)
		require.Equal(t, "first.txt", form.Parts[0].Filename)
		require.Equal(t, "second.txt", form.Parts[1].Filename)
	})

	t.Run("nil request", func(t *testing.T) {
		_, err := ReadMultipartForm(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be nil")
	})
}

func TestMultipartRetainsEmptyPhysicalNames(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	original := &MultipartForm{Parts: []MultipartPart{
		{Name: "", Data: []byte("unknown")},
		{Name: "value", Filename: "file.txt", Data: []byte("file")},
		{Name: "value", Data: []byte("text")},
	}}
	require.NoError(t, SetMultipartRequest(request, original))
	decoded, err := ReadMultipartForm(request)
	require.NoError(t, err)
	require.Equal(t, original, decoded)
}
