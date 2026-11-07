package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func constructorSourceStorageDSL() {
	small := Type("Small", Int, func() {
		Meta("struct:field:type", "int8")
	})
	state := Type("State", Int64, func() {
		Meta("struct:field:type", "http.ConnState", "net/http")
	})
	out := Type("Out", func() {
		Attribute("small", small)
		Attribute("state", state)
		Attribute("message", String)
		Required("small", "state", "message")
	})
	Service("storage", func() {
		Method("list", func() {
			Payload(func() {
				Attribute("limit", small)
				Attribute("state", state)
				Required("limit", "state")
			})
			Result(out)
			HTTP(func() {
				GET("/")
				Param("limit")
				Header("state:X-State")
				Response(StatusOK, func() {
					Header("small:X-Small")
					Header("state:X-State")
					Body("message")
				})
			})
		})
	})
}

func TestConstructorSourceStorage(t *testing.T) {
	root := RunHTTPDSL(t, constructorSourceStorageDSL)
	data := CreateHTTPServices(root).Get("storage")
	init := data.Endpoint("list").Payload.Request.PayloadInit
	for _, arg := range init.ServerArgs {
		if arg.FieldName == "" {
			continue
		}
		require.NotNil(t, arg.attribute, arg.VarName)
		require.NotNil(t, arg.fieldAttribute, arg.VarName)
	}
}

func TestConstructorSourceStorageGenerated(t *testing.T) {
	root := RunHTTPDSL(t, constructorSourceStorageDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/storage", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "storage_test.go"), []byte(constructorSourceStorageHarness), 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const constructorSourceStorageHarness = `package storage_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	client "example.com/storage/gen/http/storage/client"
	server "example.com/storage/gen/http/storage/server"
	svc "example.com/storage/gen/storage"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestWireStorage(t *testing.T) {
	request, err := http.NewRequest("GET", "http://example.com/?limit=7", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-State", "2")
	payload, err := server.DecodeListRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)(request)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Limit != 7 || payload.State != 2 {
		t.Fatalf("payload=%+v", payload)
	}
	response := &http.Response{StatusCode: 200, Header: http.Header{"X-Small": {"7"}, "X-State": {"2"}, "Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader("\"ok\""))}
	value, err := client.DecodeListResponse(loomhttp.ResponseDecoder, false)(response)
	if err != nil {
		t.Fatal(err)
	}
	result := value.(*svc.Out)
	if result.Small != 7 || result.State != 2 || result.Message != "ok" {
		t.Fatalf("result=%+v", result)
	}
}
func TestCLIStorage(t *testing.T) {
	value, err := client.BuildListPayload("7", "2")
	if err != nil {
		t.Fatal(err)
	}
	if value.Limit != 7 || value.State != 2 {
		t.Fatalf("payload=%+v", value)
	}
	if _, err = client.BuildListPayload("200", "2"); err == nil {
		t.Error("out-of-range narrow CLI input accepted")
	}
}
`
