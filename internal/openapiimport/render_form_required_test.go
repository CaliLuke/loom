package openapiimport

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormBodyConjunctiveRequiredSelection(t *testing.T) {
	base := &Schema{Type: "object", Required: []string{"kind"}, Properties: []NamedProperty{{Name: "kind", Schema: &Schema{Type: "string"}}}}
	derived := &Schema{Type: "object", Bases: []*Schema{{Ref: "#/components/schemas/Base"}}, Properties: []NamedProperty{{Name: "note", Schema: &Schema{Type: "string"}}}}
	plain := &Schema{Type: "object", Properties: []NamedProperty{{Name: "note", Schema: &Schema{Type: "string"}}}}
	cycle := &Schema{Type: "object", Bases: []*Schema{{Ref: "#/components/schemas/Cycle"}, {Ref: "#/components/schemas/Base"}}}
	emptyCycle := &Schema{Type: "object", Bases: []*Schema{{Ref: "#/components/schemas/EmptyCycle"}}}
	schemas := map[string]NamedSchema{
		"Base":       {Name: "Base", Schema: base},
		"Derived":    {Name: "Derived", Schema: derived},
		"Cycle":      {Name: "Cycle", Schema: cycle},
		"EmptyCycle": {Name: "EmptyCycle", Schema: emptyCycle},
	}
	for _, tc := range []struct {
		name     string
		schema   *Schema
		required bool
		media    []string
		want     requestBodyMode
		error    string
	}{
		{name: "inherited", schema: derived, want: requestBodyRaw},
		{name: "named inherited", schema: &Schema{Ref: "#/components/schemas/Derived"}, want: requestBodyRaw},
		{name: "explicit", schema: base, want: requestBodyRaw},
		{name: "optional no required", schema: plain, want: requestBodyForm},
		{name: "required form", schema: derived, required: true, want: requestBodyForm},
		{name: "nested required is not root required", schema: &Schema{Type: "object", Properties: []NamedProperty{{Name: "nested", Schema: base}}}, want: requestBodyForm},
		{name: "alternatives are not conjunctions", schema: &Schema{OneOf: []*Schema{base, plain}}, want: requestBodyForm},
		{name: "cycle with required base", schema: cycle, want: requestBodyRaw},
		{name: "cycle without required base", schema: emptyCycle, want: requestBodyForm},
		{name: "missing base", schema: &Schema{Type: "object", Bases: []*Schema{{Ref: "#/components/schemas/Missing"}}}, error: `render OpenAPI design: #/paths/~1value/post/requestBody/schema/allOf/0 schema reference "#/components/schemas/Missing" does not resolve`},
		{name: "malformed base", schema: &Schema{Type: "object", Bases: []*Schema{{Ref: "#/other/Base"}}}, error: `#/paths/~1value/post/requestBody/schema/allOf/0 schema reference "#/other/Base"`},
		{name: "multipart unchanged", schema: derived, media: []string{"multipart/form-data"}, want: requestBodyMultipart},
		{name: "json unchanged", schema: derived, media: []string{"application/json"}, want: requestBodyJSON},
		{name: "multiple media unchanged", schema: derived, media: []string{"application/x-www-form-urlencoded", "application/json"}, want: requestBodyJSON},
	} {
		t.Run(tc.name, func(t *testing.T) {
			media := tc.media
			if len(media) == 0 {
				media = []string{"application/x-www-form-urlencoded"}
			}
			localRequired := slices.Clone(tc.schema.Required)
			localBases := slices.Clone(tc.schema.Bases)
			baseRequired := slices.Clone(base.Required)
			r := renderer{schemas: schemas}
			body, err := r.requestBody(&RequestBody{Schema: tc.schema, ContentTypes: media, Required: tc.required}, "#/paths/~1value/post/requestBody")
			if tc.error != "" {
				require.ErrorContains(t, err, tc.error)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, body.mode)
			}
			require.Equal(t, localRequired, tc.schema.Required)
			require.Equal(t, localBases, tc.schema.Bases)
			require.Equal(t, baseRequired, base.Required)
		})
	}
}
