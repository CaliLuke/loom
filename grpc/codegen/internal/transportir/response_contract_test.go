package transportir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAnalyzeResponseContractCasesUnary(t *testing.T) {
	headers := responseContractTestMetadata("version", "x-version", true)
	trailers := responseContractTestMetadata("checksum", "x-checksum", true)
	endpoint := &Endpoint{
		Name:    "Show",
		Service: &Service{Name: "Widgets"},
		Method:  &expr.MethodExpr{Stream: expr.NoStreamKind},
		Response: &Response{
			ProtoMessage: responseContractTestMessage("ShowResponse"),
			StatusCode:   0,
			Headers:      headers,
			Trailers:     trailers,
		},
		Errors: []*Error{
			{
				Name: "invalid",
				Type: &expr.Object{},
				Response: &Response{
					ProtoMessage: responseContractTestMessage("ShowInvalidError"),
					StatusCode:   3,
				},
			},
			{
				Name: "missing",
				Type: expr.ErrorResult,
				Response: &Response{
					StatusCode: 5,
				},
			},
		},
		Stream: &Stream{},
	}

	analysis := AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Equal(t, []*ResponseContractCase{
		{
			ID:               "Widgets.Show.success.0",
			Kind:             ResponseContractSuccess,
			StatusCode:       0,
			MessageType:      "ShowResponse",
			RequiredHeaders:  []string{"x-version"},
			RequiredTrailers: []string{"x-checksum"},
		},
		{
			ID:         "Widgets.Show.error.invalid.3",
			Kind:       ResponseContractError,
			StatusCode: 3,
			ErrorName:  "invalid",
			DetailType: "ShowInvalidError",
		},
		{
			ID:         "Widgets.Show.error.missing.5",
			Kind:       ResponseContractError,
			StatusCode: 5,
			ErrorName:  "missing",
			DetailType: "loompb.ErrorResponse",
		},
	}, analysis.Cases)
}

func TestAnalyzeResponseContractCasesServerStream(t *testing.T) {
	endpoint := &Endpoint{
		Name:     "Watch",
		Service:  &Service{Name: "Events"},
		Method:   &expr.MethodExpr{Stream: expr.ServerStreamKind},
		Response: &Response{ProtoMessage: responseContractTestMessage("WatchResponse")},
		Stream:   &Stream{IsStreaming: true},
	}

	analysis := AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Len(t, analysis.Cases, 1)
	require.Equal(t, &ResponseContractStream{Direction: "server", Terminal: "eof"}, analysis.Cases[0].Stream)
}

func TestAnalyzeResponseContractCasesRejectsUnsupportedStreams(t *testing.T) {
	tests := []struct {
		name string
		kind expr.StreamKind
	}{
		{name: "client", kind: expr.ClientStreamKind},
		{name: "bidirectional", kind: expr.BidirectionalStreamKind},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis := AnalyzeResponseContractCases(&Endpoint{
				Name:     "Sync",
				Service:  &Service{Name: "Events"},
				Method:   &expr.MethodExpr{Stream: test.kind},
				Response: &Response{ProtoMessage: responseContractTestMessage("SyncResponse")},
				Stream:   &Stream{IsStreaming: true},
			})
			require.False(t, analysis.Supported())
			require.Equal(t, []ResponseContractLimitation{{
				Code:   ResponseContractStreaming,
				Detail: "gRPC response contracts currently support unary and server-streaming endpoints",
			}}, analysis.Limitations)
		})
	}
}

func responseContractTestMessage(name string) *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{"struct:name:proto": []string{name}},
	}
}

func responseContractTestMetadata(name, wireName string, required bool) *expr.MappedAttributeExpr {
	attribute := &expr.AttributeExpr{
		Type: &expr.Object{{Name: name + ":" + wireName, Attribute: &expr.AttributeExpr{Type: expr.String}}},
	}
	if required {
		attribute.Validation = &expr.ValidationExpr{Required: []string{name}}
	}
	return expr.NewMappedAttributeExpr(attribute)
}
