package httpencoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentMediaAgreement(t *testing.T) {
	for _, tc := range []struct {
		media string
		shape Shape
		style Style
		part  PartKind
		err   string
	}{
		{"application/json; charset=utf-8", Object, JSON, JSONPart, ""},
		{"application/example+json", Object, JSON, JSONPart, ""},
		{"Application/JSON", Object, JSON, JSONPart, ""},
		{"text/plain; charset=utf-8", Scalar, Form, TextPart, ""},
		{"text/html", Scalar, Form, TextPart, ""},
		{"application/json; bad", Object, 0, 0, "invalid"},
		{"application/*+json", Object, 0, 0, "concrete"},
		{"application/xml", Object, 0, 0, "no equivalent"},
	} {
		for _, items := range []bool{false, true} {
			name := tc.media + "/value"
			shape := BranchShape{Shape: tc.shape}
			application := Value
			if items {
				name = tc.media + "/items"
				item := shape
				shape = BranchShape{Shape: UnsupportedShape, SourceType: "array", Items: &item}
				application = Items
			}
			t.Run(name, func(t *testing.T) {
				source := Serialization{MediaType: tc.media}
				field, _, formErr := Resolve(FormField, "value", shape, source)
				part, multipartErr := ResolveMultipart("value", shape, source)
				if tc.err != "" {
					require.ErrorContains(t, formErr, tc.err)
					require.ErrorContains(t, multipartErr, tc.err)
					return
				}
				require.NoError(t, formErr)
				require.NoError(t, multipartErr)
				require.Equal(t, tc.style, field.Style)
				require.Equal(t, tc.part, part.Kind)
				require.Equal(t, application, field.Application)
				require.Equal(t, application, part.Application)
				require.Equal(t, tc.media, part.ContentType)
			})
		}
	}
}
