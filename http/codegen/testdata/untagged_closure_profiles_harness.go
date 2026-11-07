package testdata

// UntaggedClosureProfilesHarness checks evaluated-member profiles through generated HTTP.
const UntaggedClosureProfilesHarness = `package rootunion_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/rootunion/gen/http/roots/client"
	server "example.com/rootunion/gen/http/roots/server"
	svc "example.com/rootunion/gen/roots"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestGeneratedClosureProfiles(t *testing.T) {
	for _, tc := range []struct {
		name, route, wire string
		status            int
		detail            string
	}{
		{"nested-ambiguous-three", "nested_three", "42", 400, "matched 3 branches in schema"},
		{"closure-open-pair-400", "closure_open_pair", "{\"x\":1}", 400, "matched 2 branches in schema"},
		{"closure-discriminated-400", "closure_discriminated", "{\"kind\":\"a\",\"x\":1}", 400, "matched 2 branches in schema"},
		{"closure-map-accepts-member", "closure_map", "{\"z\":1}", 200, ""},
		{"closure-any-rejects-member", "closure_any", "{\"z\":1}", 400, "unevaluated"},
		{"closure-any-scalar", "closure_any", "7", 200, ""},
		{"closure-nested-unevaluated-400", "closure_nested", "{\"x\":1,\"z\":2}", 400, "unevaluated"},
		{"closure-nested-selected", "closure_nested", "{\"x\":1}", 200, ""},
		{"closure-typed-additional", "closure_additional", "{\"x\":1,\"z\":2}", 200, ""},
		{"closure-explicit-open", "closure_explicit_open", "{\"z\":2}", 200, ""},
		{"closure-closed-object", "closure_closed", "{\"x\":1}", 200, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.detail != "" {
				require.ErrorContains(t, json.Unmarshal([]byte(tc.wire), closureProfileCandidate(tc.route)), tc.detail)
			}
			calls := 0
			var observed any
			echo := func(_ context.Context, value any) (any, error) {
				calls++
				observed = value
				return value, nil
			}
			mux := loomhttp.NewMuxer()
			endpoints := &svc.Endpoints{
				NestedThree: echo, ClosureOpenPair: echo, ClosureDiscriminated: echo,
				ClosureMap: echo, ClosureAny: echo, ClosureNested: echo,
				ClosureAdditional: echo, ClosureExplicitOpen: echo, ClosureClosed: echo,
			}
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			request := httptest.NewRequest(http.MethodPost, "/"+tc.route, strings.NewReader(tc.wire))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.status == 200 {
				require.Equal(t, 1, calls)
				require.NotNil(t, observed)
				actual, err := json.Marshal(observed, json.Deterministic(true))
				require.NoError(t, err)
				require.JSONEq(t, tc.wire, string(actual))
				require.JSONEq(t, tc.wire, response.Body.String())
			} else {
				require.Zero(t, calls)
				require.Nil(t, observed)
				var problem map[string]any
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, "decode_payload", problem["code"])
				require.Equal(t, "invalid request body", problem["detail"])
			}
		})
	}
}

func closureProfileCandidate(route string) any {
	switch route {
	case "nested_three":
		return new(server.NestedThreeRequestBody)
	case "closure_open_pair":
		return new(server.ClosureOpenPairRequestBody)
	case "closure_discriminated":
		return new(server.ClosureDiscriminatedRequestBody)
	case "closure_any":
		return new(server.ClosureAnyRequestBody)
	case "closure_nested":
		return new(server.ClosureNestedRequestBody)
	default:
		panic("unexpected closure profile: " + route)
	}
}

func TestGeneratedErrorEnvelopeBody(t *testing.T) {
	for _, wire := range []string{"\"denied\"", "42"} {
		t.Run("error-envelope-body/"+wire, func(t *testing.T) {
			var failure svc.ProfileErrorEnvelope
			require.NoError(t, json.Unmarshal([]byte("{\"body\":"+wire+",\"reason\":\"rejected\"}"), &failure))
			calls := 0
			endpoint := func(_ context.Context, value any) (any, error) {
				calls++
				payload, err := json.Marshal(value)
				require.NoError(t, err)
				require.JSONEq(t, "\"request\"", string(payload))
				return nil, &failure
			}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(&svc.Endpoints{ErrorEnvelope: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			request := httptest.NewRequest(http.MethodPost, "/error_envelope", strings.NewReader("\"request\""))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
			require.Equal(t, 1, calls)
			require.Equal(t, "profile_error", response.Header().Get("loom-error"))
			require.Equal(t, "rejected", response.Header().Get("X-Reason"))
			require.JSONEq(t, wire, response.Body.String())
			decoded, err := client.DecodeErrorEnvelopeResponse(loomhttp.ResponseDecoder, false)(response.Result())
			require.Nil(t, decoded)
			var actual *svc.ProfileErrorEnvelope
			require.ErrorAs(t, err, &actual)
			require.Equal(t, &failure, actual)
		})
	}
}
`
