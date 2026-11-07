package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicetestdata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestNullableErrorGenerated(t *testing.T) {
	root := RunHTTPDSL(t, servicetestdata.NullableErrorsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/nullableerrors", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nullable_test.go"), []byte(nullableErrorHTTPHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const nullableErrorHTTPHarness = `package nullableerrors_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	contracts "example.com/nullableerrors/gen/contracts"
	client "example.com/nullableerrors/gen/http/nullable/client"
	server "example.com/nullableerrors/gen/http/nullable/server"
	service "example.com/nullableerrors/gen/nullable"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestNullableHTTPErrorRoundTrip(t *testing.T) {
	for _, null := range []bool{false, true} {
		payload := loom.NullableValue(service.Details{Message: "detail"})
		if null {
			payload = loom.NullValue[service.Details]()
		}
		failure, err := service.NewDetailsError(payload)
		require.NoError(t, err)
		require.Equal(t, "details", failure.LoomErrorName())
		require.Equal(t, "retry_details", failure.LoomErrorRemedy().Code)
		rec := httptest.NewRecorder()
		require.NoError(t, server.EncodeObjectError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, fmt.Errorf("wrapped: %w", failure)))
		require.Equal(t, 409, rec.Code)
		require.Equal(t, "details", rec.Header().Get("loom-error"))
		if null {
			require.Equal(t, "null", strings.TrimSpace(rec.Body.String()))
		} else {
			require.JSONEq(t, ` + "`" + `{"message":"detail"}` + "`" + `, rec.Body.String())
		}
		_, got := client.DecodeObjectResponse(loomhttp.ResponseDecoder, false)(rec.Result())
		var decoded *service.DetailsError
		require.ErrorAs(t, got, &decoded)
		roundTrip, err := decoded.Payload()
		require.NoError(t, err)
		require.True(t, roundTrip.Present())
		require.Equal(t, null, roundTrip.IsNull())
		if !null {
			value, ok := roundTrip.Value()
			require.True(t, ok)
			require.Equal(t, "detail", value.Message)
			require.Equal(t, "detail", decoded.Error())
		}
	}
}

func TestNullableScalarAndIdentity(t *testing.T) {
	for _, payload := range []loom.Nullable[service.Short]{loom.NullValue[service.Short](), loom.NullableValue(service.Short("short"))} {
		failure, err := service.NewShortError(payload)
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		require.NoError(t, server.EncodeScalarError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, failure))
		_, got := client.DecodeScalarResponse(loomhttp.ResponseDecoder, false)(rec.Result())
		var decoded *service.ShortError
		require.ErrorAs(t, got, &decoded)
		roundTrip, err := decoded.Payload()
		require.NoError(t, err)
		require.Equal(t, payload.IsNull(), roundTrip.IsNull())
	}
	for _, name := range []string{"first", "second", ""} {
		failure, err := service.NewFirstError(loom.NullableValue(service.Identified{Name: name, Message: "message"}))
		if name != "first" {
			require.Error(t, err)
			require.Nil(t, failure)
			continue
		}
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		require.NoError(t, server.EncodeIdentifiedError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, failure))
		_, got := client.DecodeIdentifiedResponse(loomhttp.ResponseDecoder, false)(rec.Result())
		var decoded *service.FirstError
		require.ErrorAs(t, got, &decoded)
	}
	failure, err := service.NewFirstError(loom.NullValue[service.Identified]())
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	require.NoError(t, server.EncodeIdentifiedError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, failure))
	require.Equal(t, "null", strings.TrimSpace(rec.Body.String()))
	_, got := client.DecodeIdentifiedResponse(loomhttp.ResponseDecoder, false)(rec.Result())
	var decoded *service.FirstError
	require.ErrorAs(t, got, &decoded)
	resp := &http.Response{StatusCode: 400, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(` + "`" + `{"name":"second","message":"bad"}` + "`" + `))}
	_, got = client.DecodeIdentifiedResponse(loomhttp.ResponseDecoder, false)(resp)
	var invalid *loomhttp.ClientError
	require.ErrorAs(t, got, &invalid)
	require.Equal(t, "validation_error", invalid.Name)
}

func TestNullableAbsentAndInvalidOwner(t *testing.T) {
	failure, err := service.NewShortError(loom.Nullable[service.Short]{})
	require.NoError(t, err)
	payload, err := failure.Payload()
	require.NoError(t, err)
	require.False(t, payload.Present())
	rec := httptest.NewRecorder()
	err = server.EncodeScalarError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, failure)
	require.ErrorIs(t, err, loom.ErrAbsentNullable)
	var nilOwner *service.FirstError
	rec = httptest.NewRecorder()
	err = server.EncodeIdentifiedError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, nilOwner)
	require.Error(t, err)
	require.Empty(t, rec.Body.String())
	require.Empty(t, rec.Header().Get("loom-error"))
	valid, err := service.NewFirstError(loom.NullValue[service.Identified]())
	require.NoError(t, err)
	rec = httptest.NewRecorder()
	require.NoError(t, server.EncodeIdentifiedError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, errors.Join(valid, errors.New("other"))))
	require.Equal(t, 500, rec.Code)
	require.Empty(t, rec.Header().Get("loom-error"))
}

func TestNullableInheritedSharedPackage(t *testing.T) {
	failure, err := service.NewInheritedError(loom.NullValue[contracts.Inherited]())
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	require.NoError(t, server.EncodeInheritedError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, failure))
	require.Equal(t, "null", strings.TrimSpace(rec.Body.String()))
	_, err = client.DecodeInheritedResponse(loomhttp.ResponseDecoder, false)(rec.Result())
	var decoded *service.InheritedError
	require.ErrorAs(t, err, &decoded)
	payload, err := decoded.Payload()
	require.NoError(t, err)
	require.True(t, payload.IsNull())
}
`
