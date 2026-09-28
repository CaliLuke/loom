package testdata

// BodyTypeNamesHarness exercises colliding object, union and WebSocket body
// names through the generated transports of TypeIdentityDSL.
const BodyTypeNamesHarness = `package identity_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	object "example.com/identity/gen/bodycollision"
	objectclient "example.com/identity/gen/http/bodycollision/client"
	objectserver "example.com/identity/gen/http/bodycollision/server"
	union "example.com/identity/gen/unioncollision"
	unionclient "example.com/identity/gen/http/unioncollision/client"
	unionserver "example.com/identity/gen/http/unioncollision/server"
	stream "example.com/identity/gen/streamcollision"
	streamclient "example.com/identity/gen/http/streamcollision/client"
	streamserver "example.com/identity/gen/http/streamcollision/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type objectEcho struct{}

func (objectEcho) OtherEndpoint(_ context.Context, p *object.Envelope) (*object.Envelope, error) {
	return p, nil
}

type unionEcho struct{}

func (unionEcho) OtherEndpoint(_ context.Context, p *union.Choice) (*union.Choice, error) {
	return p, nil
}

type streamEcho struct{}

func (streamEcho) OtherEndpoint(ctx context.Context, s stream.OtherEndpointServerStream) error {
	p, err := s.RecvWithContext(ctx)
	if err != nil {
		return err
	}
	if err := s.SendWithContext(ctx, &stream.Envelope{Other: p}); err != nil {
		return err
	}
	return s.Close()
}

func TestCollidingBodyRoundTrips(t *testing.T) {
	mux := loomhttp.NewMuxer()
	objectserver.Mount(mux, objectserver.New(object.NewEndpoints(objectEcho{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	unionserver.Mount(mux, unionserver.New(union.NewEndpoints(unionEcho{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	streamserver.Mount(mux, streamserver.New(stream.NewEndpoints(streamEcho{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	host := strings.TrimPrefix(hs.URL, "http://")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	oc := objectclient.NewClient("http", host, hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, count := range []int{0, 7} {
		p := &object.Envelope{Other: &object.Other{Count: count}}
		got, err := oc.OtherEndpoint()(ctx, p)
		require.NoError(t, err)
		require.Equal(t, p, got)
	}
	uc := unionclient.NewClient("http", host, hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, p := range []union.Choice{
		union.NewChoiceLeaf(&union.Leaf{Name: "alpha"}),
		union.NewChoiceOther(&union.Other{Count: 7}),
		union.NewChoiceUnioncollisionOther(&union.UnioncollisionOther{Value: "collision"}),
	} {
		got, err := uc.OtherEndpoint()(ctx, &p)
		require.NoError(t, err)
		require.Equal(t, &p, got)
	}
	sc := streamclient.NewClient("ws", host, hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	value, err := sc.OtherEndpoint()(ctx, nil)
	require.NoError(t, err)
	connection := value.(stream.OtherEndpointClientStream)
	require.NoError(t, connection.SendWithContext(ctx, &stream.Other{Count: 11}))
	got, err := connection.RecvWithContext(ctx)
	require.NoError(t, err)
	require.Equal(t, &stream.Envelope{Other: &stream.Other{Count: 11}}, got)
	require.NoError(t, connection.Close())

	for _, path := range []string{"/object", "/union"} {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, hs.URL+path, strings.NewReader("{}"))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		response, err := hs.Client().Do(request)
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusBadRequest, response.StatusCode, string(body))
	}
}
`
