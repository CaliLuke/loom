package codegen

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponseMetadataConstructors(t *testing.T) {
	root := RunHTTPDSL(t, responseMetadataConstructorDSL)
	services := CreateHTTPServices(root)
	data := services.Get("metadata")
	t.Run("informational", func(t *testing.T) {
		endpoint := data.Endpoint("read")
		require.Len(t, endpoint.Informational, 1)
		info := endpoint.Informational[0]
		require.Equal(t, "hints", info.Name)
		require.NotEmpty(t, info.Send)
		require.NotEmpty(t, info.Observe)
		require.Contains(t, info.Response.ResultInit.ReturnTypeRef, "Opening")
		require.Nil(t, info.Response.ClientBody)
		require.Empty(t, info.Response.ServerBody)
	})
	t.Run("handshake", func(t *testing.T) {
		response := data.Endpoint("watch").Result.Responses[0]
		require.True(t, response.Handshake)
		require.NotNil(t, response.ResultInit)
		require.Contains(t, response.ResultInit.ReturnTypeRef, "Opening")
		require.NotContains(t, response.ResultInit.ReturnTypeRef, "Event")
		require.Equal(t, "string", response.Headers[0].TypeRef)
	})
	t.Run("status_arguments", func(t *testing.T) {
		endpoint := data.Endpoint("read")
		for _, response := range []*ResponseData{endpoint.Result.Responses[0], endpoint.Errors[0].Errors[0].Response} {
			require.NotNil(t, response.ClientStatusField)
			require.NotNil(t, response.ResultInit)
			args := response.ResultInit.ClientArgs
			require.NotEmpty(t, args)
			last := args[len(args)-1]
			require.Equal(t, "resp.StatusCode", last.Ref)
			require.Equal(t, "Status", last.FieldName)
			require.Equal(t, "int", last.TypeRef)
			body := newConstructorPlan(response.ResultInit, true, false, data.Service.Scope).body
			require.Equal(t, 1, strings.Count(body, ".Status ="))
		}
	})
}

func responseMetadataConstructorDSL() {
	opening := Type("Opening", func() {
		Attribute("status", Int)
		Attribute("session", String)
		Required("status", "session")
	})
	value := Type("Value", func() {
		Attribute("status", Int)
		Attribute("message", String)
		Required("status", "message")
	})
	event := Type("Event", func() {
		Attribute("session", Int)
	})
	Service("metadata", func() {
		Method("read", func() {
			Result(value)
			Error("failure", value)
			HTTP(func() {
				GET("/read")
				Informational("hints", StatusEarlyHints, opening, func() {
					StatusField("status")
					Header("session:X-Session")
				})
				Response(StatusOK, func() {
					StatusField("status")
					Body("message")
				})
				Response("failure", StatusBadRequest, func() {
					StatusField("status")
					Body("message")
				})
			})
		})
		Method("watch", func() {
			StreamingResult(event)
			HTTP(func() {
				GET("/watch")
				ServerSentEvents()
				Response(StatusCreated, opening, func() {
					StatusField("status")
					Header("session:X-Session")
				})
			})
		})
	})
}

func TestResponseMetadataInformationalKeys(t *testing.T) {
	root := RunHTTPDSL(t, responseMetadataConstructorDSL)
	endpoint := CreateHTTPServices(root).Get("metadata").Endpoint("read")
	require.Equal(t, []expr.HTTPStatusSelector{{Code: 103}}, endpoint.InformationalKeys)
	require.Equal(t, []expr.HTTPStatusSelector{{Code: 200}, {Code: 400}}, endpoint.FinalKeys)
}

func TestResponseMetadataSymbols(t *testing.T) {
	var expected [2]string
	for _, reverse := range []bool{false, true} {
		root := RunHTTPDSL(t, func() {
			testdata.ConstructorOwnership("metadata", false, reverse)
		})
		services := CreateHTTPServices(root)
		data := services.Get("constructors")
		info := data.Endpoint("m").Informational[0]
		names := [2]string{info.Send, info.Observe}
		if !reverse {
			expected = names
		} else {
			require.Equal(t, expected, names)
		}
		opening := data.Endpoint("watch").Result.Responses[0].ResultInit
		require.Equal(t, info.Response.ResultInit.Name, opening.Name)
		require.Equal(t, info.Response.ResultInit.constructor.Hash(), opening.constructor.Hash())
		require.Equal(t, info.Response.ResultInit.constructor.body, opening.constructor.body)
		require.Equal(t, info.Response.ResultInit.ReturnTypeRef, opening.ReturnTypeRef)
		require.Contains(t, data.constructorReservations(false), info.Send)
		require.Contains(t, data.constructorReservations(true), info.Observe)
		seen := 0
		data.walkConstructors(true, func(init *InitData, _ bool) {
			if init == info.Response.ResultInit {
				seen++
			}
		})
		require.Equal(t, 1, seen)
		dir := t.TempDir()
		files := ClientTypeFiles("example.com/metadata/gen", services)
		files = append(files, ServerTypeFiles("example.com/metadata/gen", services)...)
		renderGeneratedFiles(t, dir, files)
		CheckConstructorPackageInventoryForTest(t, data, filepath.Join(dir, "gen", "http", data.Service.PathName))
		declarations := 0
		for _, file := range files {
			if !strings.Contains(file.Path, "/client/") {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, file.Path), nil, 0)
			require.NoError(t, err)
			for _, name := range packageDeclarationNames(parsed) {
				if name == info.Response.ResultInit.Name {
					declarations++
				}
			}
		}
		require.Equal(t, 1, declarations)
	}
	scope := codegen.NewNameScope()
	scope.Unique("SendMHints")
	allocated := httpInformationalSymbol(scope, "M", "hints", "send")
	require.NotEqual(t, "SendMHints", allocated)
	require.Equal(t, allocated, httpInformationalSymbol(scope, "M", "hints", "send"))
}

func TestResponseMetadataNativeTargets(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		result := ResultType("application/vnd.metadata-view", func() {
			Attributes(func() {
				Attribute("status", Int)
				Attribute("message", String)
				Required("status", "message")
			})
			View("default", func() {
				Attribute("status")
				Attribute("message")
			})
			View("alternate", func() {
				Attribute("status")
				Attribute("message")
			})
		})
		Service("viewed", func() {
			Method("read", func() {
				Result(result)
				HTTP(func() {
					GET("/")
					Response(StatusOK, func() {
						StatusField("status")
						Body("message")
					})
				})
			})
		})
	})
	data := CreateHTTPServices(root).Get("viewed")
	response := data.Endpoint("read").Result.Responses[0]
	require.NotNil(t, response.ViewedResult)
	require.NotNil(t, response.ClientStatusField)
	require.True(t, response.ClientStatusField.FieldPointer)
	require.Equal(t, "Status", response.ClientStatusField.FieldName)
	require.Equal(t, "int", response.ClientStatusField.TypeRef)
	require.Equal(t, "Message", response.ResultInit.ReturnTypeAttribute)
	plan := newConstructorPlan(response.ResultInit, true, false, data.Service.Scope)
	require.Equal(t, 1, strings.Count(plan.body, ".Status ="))
	require.Contains(t, plan.body, "&statusCode")
	for _, view := range []string{"default", "alternate"} {
		require.ElementsMatch(t, []string{"status", "message"}, response.Required[view])
	}
}
