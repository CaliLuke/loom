package codegen

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestSSEProjectionCodegen(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEVariantProjectionDSL)
	services := CreateHTTPServices(root)
	data := services.Get("SSEProjection")
	require.NotNil(t, data)
	require.Len(t, data.Endpoints, 1)
	endpoint := data.Endpoints[0]
	require.Len(t, endpoint.SSE.Projections, 2)

	serverCode := renderSection(t, serverSSESection(endpoint))
	require.Contains(t, serverCode, `switch v.EventType`)
	require.Contains(t, serverCode, `case "legacy":`)
	require.Contains(t, serverCode, `view = "legacy"`)
	require.Contains(t, serverCode, `view = "updated"`)
	require.Contains(t, serverCode, `invalid SSE projection discriminator`)
	require.Contains(t, serverCode, `payload = NewWatchResponseBodyLegacy(res.Projected)`)
	require.Contains(t, serverCode, `payload = NewWatchResponseBodyUpdated(res.Projected)`)

	clientCode := renderSection(t, sseClientSection(endpoint))
	require.Contains(t, clientCode, `switch parsed.Type`)
	require.Contains(t, clientCode, `invalid SSE projection discriminator`)
	require.Contains(t, clientCode, `projected := new(sseprojectionviews.ProjectionEventView)`)
	require.Contains(t, clientCode, `ValidateProjectionEvent(vres)`)
	require.Contains(t, clientCode, `event, err = sseprojection.NewProjectionEvent(vres)`)
}

func TestSSEProjectionResponseBodiesMatchViews(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEVariantProjectionDSL)
	services := CreateHTTPServices(root)
	file := serverType("gen", root.API.HTTP.Services[0], services)
	var buf bytes.Buffer
	for _, section := range file.AllSections()[1:] {
		require.NoError(t, section.Write(&buf))
	}
	code := codegen.FormatTestCode(t, "package server\n"+buf.String())

	legacy := generatedHTTPFunction(t, code, "NewWatchResponseBodyLegacy")
	updated := generatedHTTPFunction(t, code, "NewWatchResponseBodyUpdated")
	require.NotContains(t, legacy, "res.Event.Kind")
	require.Contains(t, legacy, ".Type")
	require.Contains(t, legacy, ".Payload")
	require.Contains(t, updated, "res.Event.Kind")
	require.NotContains(t, updated, ".Payload")
}

func TestSSEProjectionGeneratedModuleCompiles(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEVariantProjectionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/sseprojectionit", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "projection_integration_test.go"), []byte(sseProjectionIntegrationHarness), 0o644))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const sseProjectionIntegrationHarness = `package sseprojectionit

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	client "example.com/sseprojectionit/gen/http/sse_projection/client"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestProjectionClientDecodesMixedStream(t *testing.T) {
	frames := "event: legacy\ndata: {\"event_type\":\"legacy\",\"sequence\":1,\"type\":\"legacy\",\"payload\":{\"id\":\"a\"}}\n\n" +
		"event: updated\ndata: {\"event_type\":\"updated\",\"sequence\":2,\"event\":{\"type\":\"updated\",\"value\":{\"id\":\"b\"}}}\n\n"
	stream := client.NewWatchStream(
		&http.Response{Body: io.NopCloser(strings.NewReader(frames))},
		loomhttp.ResponseDecoder,
	)

	legacy, err := stream.Recv(context.Background())
	if err != nil {
		t.Fatalf("recv legacy: %v", err)
	}
	if legacy.EventType != "legacy" || legacy.Sequence != 1 || legacy.Type == nil || *legacy.Type != "legacy" {
		t.Fatalf("unexpected legacy event: %#v", legacy)
	}
	if legacy.Payload == nil || legacy.Payload.ID != "a" || legacy.Event != nil {
		t.Fatalf("unexpected legacy projection fields: %#v", legacy)
	}

	updated, err := stream.Recv(context.Background())
	if err != nil {
		t.Fatalf("recv updated: %v", err)
	}
	if updated.EventType != "updated" || updated.Sequence != 2 || updated.Event == nil {
		t.Fatalf("unexpected updated event: %#v", updated)
	}
	if string(updated.Event.Kind()) != "updated" || updated.Payload != nil {
		t.Fatalf("unexpected updated projection fields: %#v", updated)
	}
}

func TestProjectionClientRejectsUnknownDiscriminator(t *testing.T) {
	stream := client.NewWatchStream(
		&http.Response{Body: io.NopCloser(strings.NewReader("event: bogus\ndata: {}\n\n"))},
		loomhttp.ResponseDecoder,
	)
	_, err := stream.Recv(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid SSE projection discriminator") {
		t.Fatalf("expected discriminator error, got %v", err)
	}
}
`

func renderSection(t *testing.T, section codegen.Section) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, section.Write(&buf))
	return codegen.FormatTestCode(t, "package generated\n"+buf.String())
}
