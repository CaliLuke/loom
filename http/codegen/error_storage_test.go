package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func customErrorStorageDSL() {
	failure := Type("Failure", func() {
		Attribute("normal_header", Int)
		Attribute("normal_cookie", Int)
		Attribute("narrow_header", Int, func() {
			Meta("struct:field:type", "int8")
		})
		Attribute("narrow_cookie", Int, func() {
			Meta("struct:field:type", "int8")
		})
		Attribute("opaque_header", Int, func() {
			Meta("struct:field:type", "time.Duration", "time")
		})
		Attribute("opaque_cookie", Int, func() {
			Meta("struct:field:type", "time.Duration", "time")
		})
		Attribute("optional_header", Int, func() {
			Meta("struct:field:type", "int8")
		})
		Attribute("optional_cookie", Int, func() {
			Meta("struct:field:type", "int8")
		})
		Attribute("message", String)
		Required("normal_header", "normal_cookie", "narrow_header", "narrow_cookie", "opaque_header", "opaque_cookie", "message")
	})
	Service("errorstorage", func() {
		Method("show", func() {
			Error("bad", failure)
			HTTP(func() {
				GET("/")
				Response("bad", StatusBadRequest, func() {
					Header("normal_header:X-Normal")
					Header("narrow_header:X-Narrow")
					Header("opaque_header:X-Opaque")
					Cookie("normal_cookie:normal")
					Cookie("narrow_cookie:narrow")
					Cookie("opaque_cookie:opaque")
					Header("optional_header:X-Optional")
					Cookie("optional_cookie:optional")
					Body("message")
				})
			})
		})
	})
}

func TestCustomErrorStorageBindings(t *testing.T) {
	root := RunHTTPDSL(t, customErrorStorageDSL)
	response := CreateHTTPServices(root).Get("errorstorage").Endpoint("show").Errors[0].Errors[0].Response
	for _, arg := range response.ResultInit.ClientArgs {
		if arg.FieldName == "" {
			continue
		}
		t.Run(arg.Name, func(t *testing.T) {
			require.NotNil(t, arg.fieldAttribute)
			for _, header := range response.Headers {
				if header.Name == arg.Name {
					require.Same(t, header.fieldAttribute, arg.fieldAttribute)
				}
			}
			for _, cookie := range response.Cookies {
				if cookie.Name == arg.Name {
					require.Same(t, cookie.fieldAttribute, arg.fieldAttribute)
				}
			}
		})
	}
}

func TestCustomErrorStorageGenerated(t *testing.T) {
	root := RunHTTPDSL(t, customErrorStorageDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/errorstorage", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "error_storage_test.go"), []byte(customErrorStorageHarness), 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const customErrorStorageHarness = `package errorstorage_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	svc "example.com/errorstorage/gen/errorstorage"
	client "example.com/errorstorage/gen/http/errorstorage/client"
	server "example.com/errorstorage/gen/http/errorstorage/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestCustomErrorStorageRoundTrip(t *testing.T) {
	for _, value := range []int8{-128, 0, 127} {
		want := &svc.Failure{NormalHeader: 19, NormalCookie: 23, NarrowHeader: value, NarrowCookie: value, OpaqueHeader: time.Duration(41), OpaqueCookie: time.Duration(43), Message: "bad"}
		if value == 0 {
			want.OptionalHeader = &value
			want.OptionalCookie = &value
		}
		writer := httptest.NewRecorder()
		if err := server.EncodeShowError(loomhttp.ResponseEncoder, nil)(context.Background(), writer, want); err != nil {
			t.Fatal(err)
		}
		response := writer.Result()
		if response.StatusCode != http.StatusBadRequest || response.Header.Get("X-Normal") != "19" || response.Header.Get("X-Opaque") != "41" {
			t.Fatalf("response: status=%d headers=%v", response.StatusCode, response.Header)
		}
		_, err := client.DecodeShowResponse(loomhttp.ResponseDecoder, false)(response)
		var got *svc.Failure
		if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
			t.Fatalf("got=%+v err=%v want=%+v", got, err, want)
		}
	}
}

func TestCustomErrorStorageRejectsNarrowValues(t *testing.T) {
	for _, location := range []string{"header", "cookie"} {
		writer := httptest.NewRecorder()
		value := &svc.Failure{NormalHeader: 1, NormalCookie: 2, NarrowHeader: 3, NarrowCookie: 4, OpaqueHeader: 5, OpaqueCookie: 6, Message: "bad"}
		if err := server.EncodeShowError(loomhttp.ResponseEncoder, nil)(context.Background(), writer, value); err != nil {
			t.Fatal(err)
		}
		response := writer.Result()
		if location == "header" {
			response.Header.Set("X-Narrow", "200")
		} else {
			response.Header.Del("Set-Cookie")
			response.Header.Add("Set-Cookie", "normal=2")
			response.Header.Add("Set-Cookie", "narrow=200")
			response.Header.Add("Set-Cookie", "opaque=6")
		}
		_, err := client.DecodeShowResponse(loomhttp.ResponseDecoder, false)(response)
		var got *svc.Failure
		if err == nil || errors.As(err, &got) {
			t.Fatalf("%s: invalid integer became custom error: %v", location, err)
		}
	}
}
`
