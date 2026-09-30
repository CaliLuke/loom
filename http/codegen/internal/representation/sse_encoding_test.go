package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestMappedSSEEncodingMatchesNativeDispatch(t *testing.T) {
	type namedBytes []byte
	bytes := []byte("hi")
	for _, test := range []struct {
		name, typeRef string
		pointer       bool
		value         any
		codec         expr.ValueCodec
		wire          string
	}{
		{"native", "[]byte", false, bytes, expr.ValueCodecRaw, "hi"},
		{"named", "NamedBytes", false, namedBytes(bytes), expr.ValueCodecJSON, `"aGk="`},
		{"pointer", "[]byte", true, &bytes, expr.ValueCodecJSON, `"aGk="`},
		{"text", "string", false, "hi", expr.ValueCodecText, "hi"},
	} {
		t.Run(test.name, func(t *testing.T) {
			policy := MappedSSEEncoding(test.typeRef, test.pointer)
			require.Equal(t, test.codec, policy.Codec)
			encoded, err := loomhttp.EncodeSSEData(test.value)
			require.NoError(t, err)
			require.Equal(t, test.wire, encoded)
		})
	}
	require.True(t, MappedSSEEncoding("string", true).DereferenceString)
	require.False(t, MappedSSEEncoding("NamedString", true).DereferenceString)
}
