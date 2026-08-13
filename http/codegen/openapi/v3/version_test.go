package openapiv3

import (
	"testing"

	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/stretchr/testify/require"
)

func TestConstructForVersion(t *testing.T) {
	tests := []struct {
		name         string
		target       openAPIVersion
		constructors []versionedConstructor[string]
		want         string
		wantOK       bool
	}{
		{
			name:   "additive feature omitted before minimum version",
			target: openAPIVersion31,
			constructors: []versionedConstructor[string]{
				{
					versions: versionRange{from: openAPIVersion32},
					construct: func() string {
						return "3.2 field"
					},
				},
			},
		},
		{
			name:   "additive feature emitted at minimum version",
			target: openAPIVersion32,
			constructors: []versionedConstructor[string]{
				{
					versions: versionRange{from: openAPIVersion32},
					construct: func() string {
						return "3.2 field"
					},
				},
			},
			want:   "3.2 field",
			wantOK: true,
		},
		{
			name:   "incompatible representation routes by version",
			target: openAPIVersion32,
			constructors: []versionedConstructor[string]{
				{
					versions: versionRange{from: openAPIVersion31, through: openAPIVersion31},
					construct: func() string {
						return "3.1 shape"
					},
				},
				{
					versions: versionRange{from: openAPIVersion32},
					construct: func() string {
						return "3.2 shape"
					},
				},
			},
			want:   "3.2 shape",
			wantOK: true,
		},
		{
			name:   "newer constructor overrides an open ended older constructor",
			target: openAPIVersion32 + 1,
			constructors: []versionedConstructor[string]{
				{
					versions: versionRange{from: openAPIVersion32},
					construct: func() string {
						return "3.2 shape"
					},
				},
				{
					versions: versionRange{from: openAPIVersion32 + 1},
					construct: func() string {
						return "future shape"
					},
				},
			},
			want:   "future shape",
			wantOK: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := constructForVersion(test.target, test.constructors...)
			require.Equal(t, test.wantOK, ok)
			require.Equal(t, test.want, got)
		})
	}
}

func TestFilterOpenAPI31PreservesContentSchema(t *testing.T) {
	contentSchema := &openapi.Schema{Type: openapi.Object}
	schema := &openapi.Schema{
		Type:          openapi.String,
		ContentSchema: contentSchema,
	}

	filterSchemaCompatibility(schema, make(map[*openapi.Schema]struct{}))

	require.Same(t, contentSchema, schema.ContentSchema)
}
