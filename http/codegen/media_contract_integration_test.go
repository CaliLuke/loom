package codegen

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

//go:embed testdata/media_contract_harness.go.txt
var mediaContractHarness string

func TestGeneratedMediaContracts(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MultipleMediaContractsDSL)
	directory := t.TempDir()
	renderHTTPModule(t, directory, "example.com/mediacontracts", root)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "media_contract_test.go"), []byte(mediaContractHarness), 0o600))
	runGoCommand(t, directory, "mod", "tidy")
	runGoCommand(t, directory, "test", "-race", "-count=1", ".")
}

func TestGeneratedStyledMediaContracts(t *testing.T) {
	root := RunHTTPDSL(t, testdata.StyledMediaContractsDSL)
	directory := t.TempDir()
	renderHTTPModule(t, directory, "example.com/mediacontracts", root)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "media_contract_test.go"), []byte(mediaContractHarness+styledMediaContractHarness), 0o600))
	runGoCommand(t, directory, "mod", "tidy")
	runGoCommand(t, directory, "test", "-race", "-count=1", ".")
}

const styledMediaContractHarness = `
type statusRecorder struct {
 *httptest.ResponseRecorder
 committed bool
}

func (w *statusRecorder) WriteHeader(status int) {
 w.committed = true
 w.ResponseRecorder.WriteHeader(status)
}

func TestRequiredStyledMedia(t *testing.T) {
 value := &service.MediaEnvelope{Body: service.NewBodyRequiredForm(&service.BodyRequiredForm{Labels: map[string]string{}})}
 request := httptest.NewRequest("POST", "/media-contracts", nil)
 require.ErrorContains(t, client.EncodeExchangeRequest(loomhttp.RequestEncoder)(request, value), "required value produces no wire key")
 require.Zero(t, request.ContentLength)
 writer := &statusRecorder{ResponseRecorder: httptest.NewRecorder()}
 require.ErrorContains(t, server.EncodeExchangeResponse(loomhttp.ResponseEncoder)(context.Background(), writer, value), "required value produces no wire key")
 require.False(t, writer.committed)
 require.Empty(t, writer.Body.String())
}

func TestStyledMediaWire(t *testing.T) {
	value := &service.MediaEnvelope{Body: service.NewBodyForm(&service.BodyForm{Numbers: []int{1, 2}, Labels: map[string]string{"a": "b"}})}
	request := httptest.NewRequest("POST", "/media-contracts", nil)
	require.NoError(t, client.EncodeExchangeRequest(loomhttp.RequestEncoder)(request, value))
	wire, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	require.NoError(t, request.Body.Close())
	values, err := url.ParseQuery(string(wire))
	require.NoError(t, err)
	require.Equal(t, url.Values{"labels[a]": {"b"}, "numbers": {"1,2"}}, values)
	replay, err := request.GetBody()
	require.NoError(t, err)
	replayed, err := io.ReadAll(replay)
	require.NoError(t, err)
	require.NoError(t, replay.Close())
	require.Equal(t, wire, replayed)
	// An encoder may reuse the request with a different form representation.
	next := "next"
	choice := service.NewChoiceForm(&service.ChoiceForm{Name: &next})
	require.NoError(t, client.EncodeOptionalRequest(loomhttp.RequestEncoder)(request, &service.OptionalPayload{Choice: &choice}))
	reused, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	require.NoError(t, request.Body.Close())
	require.Equal(t, "name=next", string(reused))
	mux := loomhttp.NewMuxer()
	var calls atomic.Int32
	endpoint := func(_ context.Context, payload any) (any, error) {
		calls.Add(1)
		require.Equal(t, value.Body.Form, payload.(*service.MediaEnvelope).Body.Form)
		return payload, nil
	}
	server.Mount(mux, server.New(&service.Endpoints{Exchange: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	incoming := httptest.NewRequest("POST", "/media-contracts", strings.NewReader("labels[a]=b&numbers=1,2"))
	incoming.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, incoming)
	require.Equal(t, 200, response.Code, response.Body.String())
	responseValues, err := url.ParseQuery(response.Body.String())
	require.NoError(t, err)
	require.Equal(t, values, responseValues)
	decoded, err := client.DecodeExchangeResponse(loomhttp.ResponseDecoder, false)(response.Result())
	require.NoError(t, err)
	require.Equal(t, value.Body.Form, decoded.(*service.MediaEnvelope).Body.Form)
	for _, malformed := range []string{"numbers=1,2&numbers=3", "labels[a]=b&labels[a]=c"} {
		incoming := httptest.NewRequest("POST", "/media-contracts", strings.NewReader(malformed))
		incoming.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, incoming)
		require.Equal(t, 400, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), "decode_payload")
	}
	require.EqualValues(t, 1, calls.Load())
}
`
