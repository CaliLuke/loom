package http

import (
	"bytes"
	"mime/multipart"
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

		reader := multipart.NewReader(&buf, writer.Boundary())
		form, err := ReadMultipartForm(reader)
		require.NoError(t, err)
		require.Equal(t, []string{"brief"}, form.Values["title"])
		require.Equal(t, []string{"one", "two"}, form.Values["tags"])
		require.Len(t, form.Files["file"], 1)
		require.Equal(t, "brief.md", form.Files["file"][0].Filename)
		require.Equal(t, "application/octet-stream", form.Files["file"][0].ContentType)
		require.Equal(t, []byte("# hello"), form.Files["file"][0].Data)
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

		reader := multipart.NewReader(&buf, writer.Boundary())
		form, err := ReadMultipartForm(reader)
		require.NoError(t, err)
		require.Len(t, form.Files["file"], 2)
		require.Equal(t, "first.txt", form.Files["file"][0].Filename)
		require.Equal(t, "second.txt", form.Files["file"][1].Filename)
	})

	t.Run("nil reader", func(t *testing.T) {
		_, err := ReadMultipartForm(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be nil")
	})
}
