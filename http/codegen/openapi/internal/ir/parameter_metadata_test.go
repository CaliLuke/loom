package ir

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestParameterEffectiveSerializationMetadata(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ParameterMetadataDSL)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	for _, tc := range []struct {
		name     string
		reserved bool
	}{
		{"inherited", true}, {"overridden", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operation := document.Paths["/"+tc.name].Operations["GET"]
			parameter := operation.Parameters[0].Value
			require.NotNil(t, parameter)
			require.Equal(t, tc.reserved, parameter.AllowReserved)
			response := operation.Responses["204"].Value
			require.NotNil(t, response)
			require.Equal(t, tc.reserved, response.Headers["X-Value"].Value.AllowReserved)
			form := document.Paths["/"+tc.name+"/form"].Operations["POST"].RequestBody.Value.Content["application/x-www-form-urlencoded"]
			require.Equal(t, []string{strconv.FormatBool(tc.reserved)}, form.Metadata["openapi:encoding:labels:allowReserved"])
			require.Equal(t, []string{"form"}, form.Metadata["openapi:encoding:labels:style"])
			require.NotContains(t, form.Metadata, "openapi:encoding:labels:contentType")
		})
	}
}
