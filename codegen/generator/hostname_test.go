package generator

import (
	"testing"

	"github.com/CaliLuke/loom/dsl"
)

func TestHostnameValidation(t *testing.T) {
	runDesignHarness(t, "example.com/hostname", func() {
		dsl.Service("hosts", func() {
			dsl.Method("create", func() {
				dsl.Payload(func() {
					dsl.Attribute("host", dsl.String, func() {
						dsl.Format(dsl.FormatHostname)
					})
					dsl.Required("host")
				})
				dsl.HTTP(func() {
					dsl.POST("/hosts")
				})
			})
		})
	}, hostnameHarness)
}

const hostnameHarness = `package hostname_test

import (
	"context"
	"encoding/json/v2"
	"net/http/httptest"
	"strings"
	"testing"

	svc "example.com/hostname/gen/hosts"
	server "example.com/hostname/gen/http/hosts/server"
	lh "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestHostnameRequest(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{"a.b", true}, {"3a.example.", true},
		{"ab!", false}, {"ab..cd", false}, {"ab-.cd", false},
		{strings.Repeat("a", 64) + ".example", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			calls := 0
			endpoint := func(_ context.Context, payload any) (any, error) {
				calls++
				require.Equal(t, tc.value, payload.(*svc.CreatePayload).Host)
				return nil, nil
			}
			body, err := json.Marshal(map[string]string{"host": tc.value})
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/hosts", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			server.NewCreateHandler(endpoint, lh.NewMuxer(), lh.RequestDecoder, lh.ResponseEncoder, nil, nil).ServeHTTP(rec, req)
			if tc.valid {
				require.Equal(t, 204, rec.Code)
				require.Equal(t, 1, calls)
			} else {
				require.Equal(t, 400, rec.Code)
				require.Zero(t, calls)
				var problem map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
				require.Equal(t, "invalid_format", problem["code"])
				require.Equal(t, "validation error", problem["detail"])
			}
		})
	}
}
`
