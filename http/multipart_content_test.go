package http

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultipartContentApplication(t *testing.T) {
	var form MultipartForm
	present, err := form.AddJSON("json", "application/json", []string{"a+b%2C", "next"}, FieldItems)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, `"a+b%2C"`, string(form.Parts[0].Data))
	raw, matched, err := MultipartJSONValue(&form, "json", FieldItems)
	require.NoError(t, err)
	require.True(t, matched)
	require.JSONEq(t, `["a+b%2C","next"]`, string(raw))
	var decoded []string
	present, err = DecodeMultipartJSON(&form, "json", &decoded, false, FieldItems)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, []string{"a+b%2C", "next"}, decoded)
	present, err = form.AddBinary("file", "application/custom", [][]byte{{}, {0, 255, '+'}}, FieldItems)
	require.NoError(t, err)
	require.True(t, present)
	var files [][]byte
	present, err = DecodeMultipartBinary(&form, "file", &files, FieldItems)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, [][]byte{{}, {0, 255, '+'}}, files)
	_, err = DecodeMultipartBinary(&form, "file", new([]byte), FieldValue)
	require.ErrorContains(t, err, "multiple multipart files")
}

func TestMultipartContentRejectsInvalidApplicationBeforeAbsence(t *testing.T) {
	var target any
	_, err := DecodeMultipartJSON(nil, "field", &target, true, FieldApplication(99))
	require.Error(t, err)
	_, err = DecodeMultipartBinary(nil, "field", &target, FieldApplication(99))
	require.Error(t, err)
}
