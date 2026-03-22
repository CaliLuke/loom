package framework

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderWebSocketStreamSupportsNonStringStreamingResults(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind string
	}{
		{name: "array", kind: TypeArray},
		{name: "object", kind: TypeObject},
		{name: "map", kind: TypeMap},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rendered := renderWebSocketStream(&MethodImplData{
				MethodData: &MethodData{
					GoName: "Stream" + tc.name,
					Info: MethodInfo{
						Action: ActionStream,
						Type:   tc.kind,
					},
				},
				ServicePackage: "testws",
			})

			require.Contains(t, rendered, "stream.SendNotification")
			require.Contains(t, rendered, "stream.SendResponse")
			require.NotEqual(t, "\treturn nil", rendered)
		})
	}
}
