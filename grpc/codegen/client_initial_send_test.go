package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestClientInitialSend(t *testing.T) {
	cases := []struct {
		name        string
		design      func()
		initialSend bool
	}{
		{"client-payload", testdata.ClientStreamingRPCWithPayloadDSL, true},
		{"bidi-payload", testdata.BidirectionalStreamingRPCWithPayloadDSL, true},
		{"client-no-payload", testdata.ClientStreamingRPCDSL, false},
		{"bidi-no-payload", testdata.BidirectionalStreamingRPCDSL, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.design)
			services := CreateGRPCServices(root)
			code := codegen.SectionsCode(t, ClientFiles("", services)[1].Section("remote-method-builder"))
			if c.initialSend {
				require.Contains(t, code, "err != nil && !errors.Is(err, io.EOF)")
				require.Contains(t, code, "return stream, nil")
			} else {
				require.NotContains(t, code, "stream.Send(")
			}
			testutil.AssertGo(t, "testdata/golden/client_initial_send_"+c.name+".go.golden", code)
		})
	}
}
