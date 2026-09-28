package testdata

// TypeIdentityHarness verifies request validators and generated result/error
// conversions when their Go body types share a design identifier.
const TypeIdentityHarness = `package identity_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	client "example.com/identity/gen/http/probe/client"
	server "example.com/identity/gen/http/probe/server"
	svc "example.com/identity/gen/probe"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type implementation struct {
	calls atomic.Int32
}

func (s *implementation) Put(_ context.Context, records []*svc.Record) (*svc.Record, error) {
	s.calls.Add(1)
	if len(records) == 0 {
		return nil, errors.New("missing record")
	}
	if records[0].ID == "bad" {
		return nil, records[0]
	}
	return records[0], nil
}

func (s *implementation) Puts(_ context.Context, records []*svc.Record) (svc.RecordCollection, error) {
	s.calls.Add(1)
	if len(records) == 0 {
		return nil, errors.New("missing record")
	}
	if records[0].ID == "bad" {
		return nil, records[0]
	}
	return svc.RecordCollection(records), nil
}

func TestIdentityBodyRoundTrips(t *testing.T) {
	service := new(implementation)
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(svc.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, tc := range []struct {
		name       string
		call       loom.Endpoint
		collection bool
	}{{"single", c.Put(), false}, {"collection", c.Puts(), true}} {
		t.Run(tc.name, func(t *testing.T) {
			for _, id := range []string{"ok", "bad"} {
				payload := []*svc.Record{{ID: id, Required: svc.NewChoiceLeaf(&svc.Leaf{Name: "alpha"})}}
				optional := svc.NewChoiceOther(&svc.Other{Count: 7})
				payload[0].Optional = &optional
				before := service.calls.Load()
				value, err := tc.call(t.Context(), payload)
				require.Equal(t, before+1, service.calls.Load())
				if id == "bad" {
					var failure *svc.Record
					require.ErrorAs(t, err, &failure)
					require.Equal(t, payload[0], failure)
					require.Nil(t, value)
				} else if tc.collection {
					require.NoError(t, err)
					require.Equal(t, svc.RecordCollection(payload), value)
				} else {
					require.NoError(t, err)
					require.Equal(t, payload[0], value)
				}
			}
		})
	}
	for _, path := range []string{"/put", "/puts"} {
		before := service.calls.Load()
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, host.URL+path, strings.NewReader("[{\"id\":\"invalid\"}]"))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		response, err := host.Client().Do(request)
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusBadRequest, response.StatusCode, string(body))
		require.Equal(t, before, service.calls.Load())
		var problem map[string]any
		require.NoError(t, json.Unmarshal(body, &problem))
		require.Equal(t, "missing_field", problem["code"], string(body))
		require.Contains(t, problem["detail"], "required")
	}
}
`
