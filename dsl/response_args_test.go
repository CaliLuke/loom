package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPResponseArgumentGrammar(t *testing.T) {
	carrier := &expr.UserTypeExpr{TypeName: "Opening", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
	var nilCallback func()
	calls := 0
	callback := func() {
		calls++
		Description("mapped")
	}
	for _, tc := range []struct {
		name       string
		args       []any
		info       bool
		errorName  string
		status     expr.HTTPStatusSelector
		typed      bool
		callbacks  int
		diagnostic string
	}{
		{
			name:   "nil callback success",
			args:   []any{nilCallback},
			status: expr.HTTPStatusSelector{Code: StatusOK},
		},
		{
			name:   "nil callback after selector",
			args:   []any{StatusOK, nilCallback},
			status: expr.HTTPStatusSelector{Code: StatusOK},
		},
		{
			name:   "nil callback after carrier",
			args:   []any{StatusCreated, carrier, nilCallback},
			status: expr.HTTPStatusSelector{Code: StatusCreated},
			typed:  true,
		},
		{
			name:       "nil callback before carrier",
			args:       []any{nilCallback, carrier},
			diagnostic: "too many arguments",
		},
		{
			name:       "nil callback duplicate",
			args:       []any{nilCallback, callback},
			diagnostic: "too many arguments",
		},
		{
			name:       "selector nil callback before carrier",
			args:       []any{StatusOK, nilCallback, carrier},
			diagnostic: "too many arguments",
		},
		{
			name:       "selector nil callback duplicate",
			args:       []any{StatusOK, nilCallback, callback},
			diagnostic: "too many arguments",
		},
		{
			name:   "exact success",
			args:   []any{StatusOK},
			status: expr.HTTPStatusSelector{Code: StatusOK},
		},
		{
			name:      "callback success",
			args:      []any{callback},
			status:    expr.HTTPStatusSelector{Code: StatusOK},
			callbacks: 1,
		},
		{
			name:      "exact callback",
			args:      []any{StatusCreated, callback},
			status:    expr.HTTPStatusSelector{Code: StatusCreated},
			callbacks: 1,
		},
		{
			name:      "class callback",
			args:      []any{Status2XX, callback},
			status:    expr.HTTPStatusSelector{Class: Status2XX},
			callbacks: 1,
		},
		{
			name:      "fallback callback",
			args:      []any{StatusFallback, callback},
			status:    expr.HTTPStatusSelector{Class: StatusFallback},
			callbacks: 1,
		},
		{
			name:   "typed SSE",
			args:   []any{StatusCreated, carrier},
			status: expr.HTTPStatusSelector{Code: StatusCreated},
			typed:  true,
		},
		{
			name:      "typed SSE callback",
			args:      []any{Status2XX, carrier, callback},
			status:    expr.HTTPStatusSelector{Class: Status2XX},
			typed:     true,
			callbacks: 1,
		},
		{
			name:      "error name first",
			args:      []any{"failed", StatusBadRequest, callback},
			errorName: "failed",
			status:    expr.HTTPStatusSelector{Code: StatusBadRequest},
			callbacks: 1,
		},
		{
			name:      "error name second",
			args:      []any{StatusBadRequest, "failed", callback},
			errorName: "failed",
			status:    expr.HTTPStatusSelector{Code: StatusBadRequest},
			callbacks: 1,
		},
		{
			name:      "error default",
			args:      []any{"failed", callback},
			errorName: "failed",
			status:    expr.HTTPStatusSelector{Code: StatusBadRequest},
			callbacks: 1,
		},
		{
			name:      "class error first",
			args:      []any{"failed", Status4XX, callback},
			errorName: "failed",
			status:    expr.HTTPStatusSelector{Class: Status4XX},
			callbacks: 1,
		},
		{
			name:      "class error second",
			args:      []any{Status4XX, "failed", callback},
			errorName: "failed",
			status:    expr.HTTPStatusSelector{Class: Status4XX},
			callbacks: 1,
		},
		{
			name:      "fallback error",
			args:      []any{"failed", StatusFallback, callback},
			errorName: "failed",
			status:    expr.HTTPStatusSelector{Class: StatusFallback},
			callbacks: 1,
		},
		{
			name:       "too many named first",
			args:       []any{"failed", StatusOK, 1, 2},
			diagnostic: "too many arguments",
		},
		{
			name:       "too many named second",
			args:       []any{StatusOK, "failed", 1, 2},
			diagnostic: "too many arguments",
		},
		{
			name:       "too many success",
			args:       []any{StatusOK, carrier, callback, 1},
			diagnostic: "too many arguments",
		},
		{
			name:       "callback then carrier",
			args:       []any{callback, carrier},
			diagnostic: "too many arguments",
		},
		{
			name:       "callback then callback",
			args:       []any{callback, callback},
			diagnostic: "too many arguments",
		},
		{
			name:       "selector callback then carrier",
			args:       []any{StatusOK, callback, carrier},
			diagnostic: "too many arguments",
		},
		{
			name:       "duplicate carrier",
			args:       []any{StatusOK, carrier, carrier},
			diagnostic: "as type function",
		},
		{
			name:       "invalid typed callback",
			args:       []any{StatusOK, carrier, 1},
			diagnostic: "as type function",
		},
		{
			name:       "invalid value before callback",
			args:       []any{StatusOK, 1, callback},
			diagnostic: "handshake type or function",
		},
		{
			name:       "invalid selector",
			args:       []any{false, callback},
			diagnostic: "HTTP status code, class or function",
		},
		{
			name:       "named carrier forbidden",
			args:       []any{"failed", StatusBadRequest, carrier},
			diagnostic: "typed handshake is only valid on an SSE success Response",
		},
		{
			name:       "named carrier callback exceeds grammar",
			args:       []any{"failed", StatusBadRequest, carrier, callback},
			diagnostic: "too many arguments",
		},
		{
			name:   "informational",
			args:   []any{StatusEarlyHints, carrier},
			info:   true,
			status: expr.HTTPStatusSelector{Code: StatusEarlyHints},
			typed:  true,
		},
		{
			name:      "informational callback",
			args:      []any{Status1XX, carrier, callback},
			info:      true,
			status:    expr.HTTPStatusSelector{Class: Status1XX},
			typed:     true,
			callbacks: 1,
		},
		{
			name:   "observation only",
			args:   []any{StatusContinue, carrier},
			info:   true,
			status: expr.HTTPStatusSelector{Code: StatusContinue},
			typed:  true,
		},
		{
			name:       "informational too many",
			args:       []any{StatusEarlyHints, carrier, callback, 1},
			info:       true,
			diagnostic: "too many arguments",
		},
		{
			name:       "informational invalid carrier",
			args:       []any{StatusEarlyHints, 1},
			info:       true,
			diagnostic: "handshake type or function",
		},
		{
			name:       "informational invalid selector",
			args:       []any{false, carrier},
			info:       true,
			diagnostic: "HTTP status code, class or function",
		},
		{
			name:       "informational missing carrier",
			args:       []any{StatusEarlyHints, callback},
			info:       true,
			diagnostic: "requires a value type",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expr.SetupTestDSL(t)
			calls = 0
			endpoint := &expr.HTTPEndpointExpr{MethodExpr: &expr.MethodExpr{Name: "call"}}
			ok := eval.Execute(func() {
				if tc.info {
					Informational("hints", tc.args...)
				} else {
					Response(tc.args[0], tc.args[1:]...)
				}
			}, endpoint)
			if tc.diagnostic != "" {
				require.False(t, ok)
				require.Len(t, eval.Context.Errors, 1)
				require.ErrorContains(t, eval.Context.Errors, tc.diagnostic)
				require.Zero(t, calls, "malformed declarations must not execute mapping callbacks")
				return
			}
			require.True(t, ok, eval.Context.Errors)
			require.Equal(t, tc.callbacks, calls)
			var response *expr.HTTPResponseExpr
			var value *expr.AttributeExpr
			switch {
			case tc.info:
				require.Len(t, endpoint.Informational, 1)
				response = endpoint.Informational[0].HTTPResponseExpr
				value = endpoint.Informational[0].Value
			case tc.errorName != "":
				require.Len(t, endpoint.HTTPErrors, 1)
				require.Equal(t, tc.errorName, endpoint.HTTPErrors[0].Name)
				response = endpoint.HTTPErrors[0].Response
				value = response.Handshake
			default:
				require.Len(t, endpoint.Responses, 1)
				response = endpoint.Responses[0]
				value = response.Handshake
			}
			require.Equal(t, tc.status, response.Status)
			if tc.typed {
				require.NotNil(t, value)
				require.Same(t, carrier, value.Type)
			} else {
				require.Nil(t, value)
			}
			if tc.callbacks != 0 {
				require.Equal(t, "mapped", response.Description)
			}
		})
	}
}
