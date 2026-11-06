package httpencoding

import (
	"fmt"
	"mime"
	"strings"

	"github.com/CaliLuke/loom/internal/httpcodec"
)

// MultipartContentType resolves a field's part media type and verifies that an
// authored override agrees with the native text, raw byte, or JSON field codec.
func MultipartContentType(shape Shape, binary bool, authored string) (string, error) {
	media := "text/plain"
	if binary {
		media = "application/octet-stream"
	} else if shape == Object || shape == ScalarMap || shape == Nested {
		media = "application/json"
	}
	if authored == "" {
		return media, nil
	}
	base, _, err := mime.ParseMediaType(authored)
	if err != nil {
		return "", fmt.Errorf("invalid multipart content type %q", authored)
	}
	if !strings.Contains(base, "/") || strings.Contains(base, "*") {
		return "", fmt.Errorf("multipart content type %q must be a concrete type/subtype", authored)
	}
	kind := httpcodec.RequestContentType(base).Kind
	if !binary && (media == "application/json" && kind != httpcodec.JSON || media == "text/plain" && kind != httpcodec.Text) {
		return "", fmt.Errorf("multipart content type %q disagrees with its generated codec", authored)
	}
	return authored, nil
}

// MatchMultipart checks that authored style flags retain the existing native
// text/repeated-part or JSON-part codec. RFC 6570 multipart expansion is separate.
func MatchMultipart(value BranchShape, source Serialization) error {
	codec, _, err := Resolve(FormField, "part", value, source)
	if err != nil {
		return err
	}
	if codec.Style == JSON || codec.Style == Form && (value.Shape == Scalar || value.Shape == Sequence && codec.Explode) {
		return nil
	}
	return fmt.Errorf("authored serialization does not match the multipart part codec")
}
