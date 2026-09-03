package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedArbitraryJSONRoundTripPreservesNumbers(t *testing.T) {
	const modulePath = "example.com/arbitraryjsonit"

	root := RunHTTPDSL(t, arbitraryJSONDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	serviceCode := readGeneratedGo(t, filepath.Join(dir, "gen", "arbitrary_json"))
	require.Contains(t, serviceCode, "Value loom.JSONValue")

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "arbitrary_json_test.go"),
		[]byte(arbitraryJSONHarness),
		0o600,
	))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func arbitraryJSONDSL() {
	var ArbitraryPayload = Type("ArbitraryPayload", func() {
		Attribute("value", Any)
		Required("value")
	})

	Service("ArbitraryJSON", func() {
		Method("Get", func() {
			Result(ArbitraryPayload)
			HTTP(func() {
				GET("/value")
				Response(StatusOK)
			})
		})

		Method("Query", func() {
			Payload(func() {
				Attribute("value", Any)
				Required("value")
			})
			Result(Any)
			HTTP(func() {
				GET("/query")
				Param("value")
				Response(StatusOK)
			})
		})
	})
}

const arbitraryJSONHarness = `package arbitraryjsonit_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http/httptest"
	"net/url"
	"testing"

	arbitraryjson "example.com/arbitraryjsonit/gen/arbitrary_json"
	arbitraryjsonclient "example.com/arbitraryjsonit/gen/http/arbitrary_json/client"
	arbitraryjsonserver "example.com/arbitraryjsonit/gen/http/arbitrary_json/server"
	loom "github.com/CaliLuke/loom/pkg"
	loomhttp "github.com/CaliLuke/loom/http"
)

const arbitraryValue = ` + "`" + `{"positive":9007199254740993,"negative":-9007199254740993,"decimal":1.234567890123456789,"nested":[{"null":null,"bool":true,"string":"value"}]}` + "`" + `
const serviceResult = ` + "`" + `{"value":` + "`" + ` + arbitraryValue + "}"

func TestArbitraryJSONServerAndClientRoundTrip(t *testing.T) {
	endpoints := &arbitraryjson.Endpoints{
		Get: func(context.Context, any) (any, error) {
			var result arbitraryjson.ArbitraryPayload
			if err := json.Unmarshal([]byte(serviceResult), &result); err != nil {
				return nil, err
			}
			return &result, nil
		},
		Query: func(_ context.Context, payload any) (any, error) {
			return payload.(*arbitraryjson.QueryPayload).Value, nil
		},
	}
	mux := loomhttp.NewMuxer()
	generated := arbitraryjsonserver.New(
		endpoints,
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		nil,
		nil,
	)
	arbitraryjsonserver.Mount(mux, generated)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	response, err := httpServer.Client().Get(httpServer.URL + "/value")
	if err != nil {
		t.Fatalf("get generated server response: %v", err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read generated server response: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close generated server response: %v", err)
	}
	if got := string(body); got != serviceResult+"\n" {
		t.Errorf("server response = %s, want %s", got, serviceResult)
	}

	serverURL, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatalf("parse generated server URL: %v", err)
	}
	client := arbitraryjsonclient.NewClient(
		serverURL.Scheme,
		serverURL.Host,
		httpServer.Client(),
		loomhttp.RequestEncoder,
		loomhttp.ResponseDecoder,
		false,
	)
	value, err := client.Get()(t.Context(), nil)
	if err != nil {
		t.Fatalf("call generated client: %v", err)
	}
	result, ok := value.(*arbitraryjson.ArbitraryPayload)
	if !ok {
		t.Fatalf("generated client result type = %T", value)
	}
	roundTrip, err := json.Marshal(result, json.Deterministic(true))
	if err != nil {
		t.Fatalf("marshal generated client result: %v", err)
	}
	if got := string(roundTrip); got != serviceResult {
		t.Errorf("client result = %s, want %s", got, serviceResult)
	}

	queryInput := loom.JSONValueFromString("query-value")
	queryResult, err := client.Query()(t.Context(), &arbitraryjson.QueryPayload{Value: queryInput})
	if err != nil {
		t.Fatalf("call generated scalar Any query client: %v", err)
	}
	queryValue, ok := queryResult.(loom.JSONValue)
	if !ok {
		t.Fatalf("generated query result type = %T", queryResult)
	}
	if got := loom.JSONValueString(queryValue); got != "query-value" {
		t.Errorf("generated scalar Any query round trip = %q", got)
	}
}
`
