package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestOrdinaryFormBodyPresence(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedFormDSL)
	functions := renderTransportFuncs(t, ServerFiles("", CreateHTTPServices(root)))
	for _, name := range []string{"PlainEmpty", "PlainFields", "PlainRequiredFields", "PlainSelectedRequired", "PlainOptional", "PlainSelected", "PlainSelectedEmpty"} {
		t.Run(name, func(t *testing.T) {
			decoder := functions["Decode"+name+"Request"]
			require.Contains(t, decoder, "loomhttp.ParseRawFormWithLimit")
			require.NotContains(t, decoder, "len(r.PostForm)")
			require.NotContains(t, decoder, "loom.MissingPayloadError()")
			if name != "PlainEmpty" && name != "PlainFields" && name != "PlainRequiredFields" && name != "PlainSelectedRequired" {
				require.Contains(t, decoder, "parameterFormPairs.HasInput()")
			}
		})
	}
}
