package testdata

import . "github.com/CaliLuke/loom/dsl"

// CollectionDeclarationIdentityDSL exercises allocated names in payload/result collections.
func CollectionDeclarationIdentityDSL(reserved bool) func() {
	return func() {
		firstName, secondName := "item", "Item"
		if reserved {
			firstName, secondName = "Control", "Use"
		}
		first := Type(firstName, func() {
			Attribute("a", String)
		})
		second := Type(secondName, func() {
			Attribute("a", String)
		})
		root := Type("Root", func() {
			Attribute("control", first)
			Attribute("value", second)
			Attribute("items", ArrayOf(second))
			Attribute("values", MapOf(String, second))
			Attribute("nested", ArrayOf(MapOf(String, second)))
		})
		Service("identity", func() {
			Method("echo", func() {
				Payload(root)
				Result(root)
				HTTP(func() {
					POST("/echo")
				})
			})
		})
	}
}

// CollectionDeclarationIdentityHarness checks generated request/result conversions.
const CollectionDeclarationIdentityHarness = `package identity_test

import (
	"context"
	"encoding/json/v2"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/collectionidentity/gen/http/identity/client"
	server "example.com/collectionidentity/gen/http/identity/server"
	svc "example.com/collectionidentity/gen/identity"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type implementation struct {
	calls int
}

func (s *implementation) Echo(_ context.Context, payload *svc.Root) (*svc.Root, error) {
	s.calls++
	return payload, nil
}

func TestCollectionRoundTrip(t *testing.T) {
	var payload svc.Root
	require.NoError(t, json.Unmarshal([]byte("{\"control\":{\"a\":\"first\"},\"value\":{\"a\":\"second\"},\"items\":[{\"a\":\"array\"}],\"values\":{\"key\":{\"a\":\"map\"}},\"nested\":[{\"key\":{\"a\":\"nested\"}}]}"), &payload))
	service := new(implementation)
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(svc.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	actual, err := c.Echo()(t.Context(), &payload)
	require.NoError(t, err)
	require.Equal(t, &payload, actual)
	require.Equal(t, 1, service.calls)
}
`
