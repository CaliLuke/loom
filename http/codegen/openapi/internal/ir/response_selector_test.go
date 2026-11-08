package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestOpenAPIResponseSelectorKeys(t *testing.T) {
	root := codegen.RunDSL(t, testdata.OpenAPIStatusSelectorDSL)
	var doc *Document
	var err error
	require.NotPanics(t, func() {
		doc, err = BuildDocument(root.API, root.Types, root.ResultTypes)
	})
	require.NoError(t, err)
	for _, path := range []string{"/first", "/second"} {
		responses := doc.Paths[path].Operations["GET"].Responses
		require.Len(t, responses, 4)
		for _, key := range []string{"204", "401", "4XX", "default"} {
			require.Contains(t, responses, key)
		}
		require.NotContains(t, responses, "0")
	}
}
