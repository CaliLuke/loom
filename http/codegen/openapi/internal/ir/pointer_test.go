package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaReferencePointerSyntax(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  string
	}{
		{"Pet", "#/components/schemas/Pet"},
		{"Pet/~ID", "#/components/schemas/Pet~1~0ID"},
		{"Pet~1ID", "#/components/schemas/Pet~01ID"},
		{"café", "#/components/schemas/café"},
		{"", "#/components/schemas/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.ref, toRef(tc.name))
		})
	}
}
