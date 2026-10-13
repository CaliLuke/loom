package codegen

const transportMapCarrierHarness = `package varcollision_test

import (
	"context"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	server "example.com/varcollision/gen/http/var_collision/server"
	service "example.com/varcollision/gen/var_collision"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestCollisionMapCarrier(t *testing.T) {
	for _, tc := range []struct {
		name, raw        string
		present, invalid bool
	}{
		{name: "unrelated query only"},
		{name: "valid map", raw: ` + "`" + `{"key":17}` + "`" + `, present: true},
		{name: "empty map", raw: ` + "`" + `{}` + "`" + `, present: true},
		{name: "null member", raw: ` + "`" + `{"key":null}` + "`" + `, present: true, invalid: true},
		{name: "malformed map", raw: ` + "`" + `[` + "`" + `, present: true, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var received *service.CollidingPayload
			endpoints := &service.Endpoints{Colliding: func(_ context.Context, p any) (any, error) {
				calls++
				received = p.(*service.CollidingPayload)
				return &service.CollidingResult{}, nil
			}}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			query := url.Values{"qp": {"unrelated"}}
			if tc.present {
				query.Set("val", tc.raw)
			}
			request := httptest.NewRequest("POST", "/items/path/segment/1,2/tail?"+query.Encode(), strings.NewReader(` + "`" + `"payload"` + "`" + `))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Val", "29")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if tc.invalid {
				require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
				require.Zero(t, calls)
				require.Nil(t, received)
				var problem loomhttp.ProblemResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, "decode_payload", problem.Code)
				require.Equal(t, "invalid request body", problem.Detail)
				return
			}
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.Equal(t, 1, calls)
			require.NotNil(t, received)
			require.NotNil(t, received.ValHeader)
			require.Equal(t, 29, *received.ValHeader)
			require.NotNil(t, received.Qp)
			require.Equal(t, "unrelated", *received.Qp)
			var want map[string]int
			if tc.present {
				want = map[string]int{}
				if tc.raw != "{}" {
					want["key"] = 17
				}
			}
			require.Equal(t, want, received.Val)
		})
	}
}
`
