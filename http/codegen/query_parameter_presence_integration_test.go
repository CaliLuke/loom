package codegen

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestOptionalQueryParameterPresenceIntegration(t *testing.T) {
	const modulePath = "example.com/querypresenceit"

	root := RunHTTPDSL(t, queryParameterPresenceDSL)
	spec := renderOpenAPIJSON(t, openapiv3.Files, root)
	parseOpenAPIV3Document(t, spec)

	var document struct {
		Paths map[string]struct {
			Get struct {
				Parameters []struct {
					Name            string `json:"name"`
					In              string `json:"in"`
					Required        bool   `json:"required"`
					AllowEmptyValue bool   `json:"allowEmptyValue"`
					Schema          struct {
						Type string `json:"type"`
					} `json:"schema"`
				} `json:"parameters"`
			} `json:"get"`
		} `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(spec, &document))
	parameters := document.Paths["/resource"].Get.Parameters
	require.Len(t, parameters, 1)
	require.Equal(t, "track_visit", parameters[0].Name)
	require.Equal(t, "query", parameters[0].In)
	require.False(t, parameters[0].Required)
	require.True(t, parameters[0].AllowEmptyValue)
	require.Equal(t, "string", parameters[0].Schema.Type)

	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "query_parameter_presence_test.go"),
		[]byte(queryParameterPresenceHarness),
		0o600,
	))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func queryParameterPresenceDSL() {
	Service("presence", func() {
		Method("inspect", func() {
			Payload(func() {
				Attribute("track_visit", String)
			})
			Result(String)
			HTTP(func() {
				GET("/resource")
				Param("track_visit")
			})
		})
	})
}

const queryParameterPresenceHarness = `package integration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	presence "example.com/querypresenceit/gen/presence"
	presenceclient "example.com/querypresenceit/gen/http/presence/client"
	presenceserver "example.com/querypresenceit/gen/http/presence/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type presenceService struct {
	present bool
	value   string
}

func (s *presenceService) Inspect(_ context.Context, payload *presence.InspectPayload) (string, error) {
	s.present = payload.TrackVisit != nil
	s.value = ""
	if s.present {
		s.value = *payload.TrackVisit
	}
	return "ok", nil
}

func TestGeneratedServerDistinguishesQueryParameterStates(t *testing.T) {
	service, server := newPresenceServer(t)
	tests := []struct {
		name        string
		query       string
		wantPresent bool
		wantValue   string
	}{
		{name: "omitted", wantPresent: false},
		{name: "present empty", query: "?track_visit=", wantPresent: true},
		{name: "present nonempty", query: "?track_visit=enabled", wantPresent: true, wantValue: "enabled"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := server.Client().Get(server.URL + "/resource" + test.query)
			if err != nil {
				t.Fatalf("GET resource: %v", err)
			}
			if _, err := io.Copy(io.Discard, response.Body); err != nil {
				t.Errorf("read response body: %v", err)
			}
			if err := response.Body.Close(); err != nil {
				t.Errorf("close response body: %v", err)
			}
			if response.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
			}
			if service.present != test.wantPresent {
				t.Errorf("parameter present = %t, want %t", service.present, test.wantPresent)
			}
			if service.value != test.wantValue {
				t.Errorf("parameter value = %q, want %q", service.value, test.wantValue)
			}
		})
	}
}

func TestGeneratedClientEmitsPresentEmptyQueryParameter(t *testing.T) {
	service, server := newPresenceServer(t)
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	doer := &queryRecordingDoer{base: server.Client()}
	client := presenceclient.NewClient(
		u.Scheme,
		u.Host,
		doer,
		loomhttp.RequestEncoder,
		loomhttp.ResponseDecoder,
		false,
	)
	empty := ""
	if _, err := client.Inspect()(t.Context(), &presence.InspectPayload{TrackVisit: &empty}); err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if !doer.present {
		t.Fatal("generated client omitted empty query parameter")
	}
	if doer.value != "" {
		t.Errorf("encoded query value = %q, want empty", doer.value)
	}
	if !service.present || service.value != "" {
		t.Errorf("decoded service state = (%t, %q), want (true, empty)", service.present, service.value)
	}
}

type queryRecordingDoer struct {
	base    loomhttp.Doer
	present bool
	value   string
}

func (d *queryRecordingDoer) Do(request *http.Request) (*http.Response, error) {
	values, present := request.URL.Query()["track_visit"]
	d.present = present
	if len(values) > 0 {
		d.value = values[0]
	}
	return d.base.Do(request)
}

func newPresenceServer(t *testing.T) (*presenceService, *httptest.Server) {
	t.Helper()
	service := &presenceService{}
	endpoints := presence.NewEndpoints(service)
	mux := loomhttp.NewMuxer()
	transport := presenceserver.New(
		endpoints,
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		nil,
		nil,
	)
	presenceserver.Mount(mux, transport)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return service, server
}
`
