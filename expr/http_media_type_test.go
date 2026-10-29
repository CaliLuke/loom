package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPResponseMediaTypesDup(t *testing.T) {
	source := &HTTPResponseExpr{MediaTypes: []*HTTPMediaTypeExpr{{ContentType: "application/json", Branch: "document"}}}
	copy := source.Dup()
	require.Equal(t, source.MediaTypes, copy.MediaTypes)
	require.NotSame(t, source.MediaTypes[0], copy.MediaTypes[0])
	copy.MediaTypes[0].Branch = "other"
	require.Equal(t, "document", source.MediaTypes[0].Branch)
}
