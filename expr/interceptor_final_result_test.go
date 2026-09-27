package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInterceptorResultBoundary validates access against the boundary that
// delivers the result, including the single final result of a client stream.
func TestInterceptorResultBoundary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		kind    StreamKind
		jsonrpc bool
	}{{"unary", NoStreamKind, false}, {"client", ClientStreamKind, false}, {"server", ServerStreamKind, false}, {"bidirectional", BidirectionalStreamKind, false}, {"jsonrpc-client", ClientStreamKind, true}} {
		stream := tc.kind
		for _, access := range []string{"read", "write", "read-stream", "write-stream"} {
			t.Run(tc.name+"/"+access, func(t *testing.T) {
				attribute := &AttributeExpr{Type: &Object{{Name: "id", Attribute: &AttributeExpr{Type: String}}}}
				interceptor := &InterceptorExpr{Name: "audit"}
				switch access {
				case "read":
					interceptor.ReadResult = attribute
				case "write":
					interceptor.WriteResult = attribute
				case "read-stream":
					interceptor.ReadStreamingResult = attribute
				case "write-stream":
					interceptor.WriteStreamingResult = attribute
				}
				method := &MethodExpr{Name: "upload", Service: &ServiceExpr{Name: "files"}, Stream: stream, Result: attribute}
				if tc.jsonrpc {
					method.Meta = MetaExpr{"jsonrpc": {}}
				}
				errors := interceptor.validate(method)
				streamingAccess := access == "read-stream" || access == "write-stream"
				if streamingAccess == (stream != NoStreamKind && !tc.jsonrpc) {
					require.Empty(t, errors.Errors)
				} else {
					require.Len(t, errors.Errors, 1)
					if stream == ClientStreamKind && !tc.jsonrpc {
						require.Contains(t, errors.Error(), "use ReadStreamingResult or WriteStreamingResult")
					}
				}
			})
		}
	}
}
