package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

const namedMetadataHarness = `package namedmetadata_test

import (
 "context"
 "testing"

 "github.com/stretchr/testify/require"
 "google.golang.org/grpc/metadata"

 svc "example.com/namedmetadata/gen/named_metadata"
 client "example.com/namedmetadata/gen/grpc/named_metadata/client"
 server "example.com/namedmetadata/gen/grpc/named_metadata/server"
)

func TestNamedMetadata(t *testing.T) {
 optional := svc.Label("optional")
 for _, value := range []*svc.Label{nil, &optional} {
  input := &svc.Value{Body:"body", Label:"label", Optional:value}
  ctx := context.Background()
  requestMetadata := metadata.MD{}
  request, err := client.EncodeEchoRequest(ctx, input, &requestMetadata)
  require.NoError(t, err)
  decoded, err := server.DecodeEchoRequest(ctx, request, requestMetadata)
  require.NoError(t, err)
  require.Equal(t, input, decoded)
  headers, trailers := metadata.MD{}, metadata.MD{}
  response, err := server.EncodeEchoResponse(ctx, input, &headers, &trailers)
  require.NoError(t, err)
  result, err := client.DecodeEchoResponse(ctx, response, headers, trailers)
  require.NoError(t, err)
  require.Equal(t, input, result)
 }
}
`

func TestMappedMetadata(t *testing.T) {
	for _, suffix := range []string{"", ":json_name"} {
		for _, required := range []bool{false, true} {
			for _, defaults := range []bool{false, true} {
				for _, explicit := range []bool{false, true} {
					t.Run(fmt.Sprintf("suffix=%s/required=%t/default=%t/explicit=%t", suffix, required, defaults, explicit), func(t *testing.T) {
						root := RunGRPCDSL(t, func() {
							testdata.MappedMetadataCaseDSL("mapped", suffix, required, defaults, explicit)
						})
						e := root.API.GRPC.Service("mapped").GRPCEndpoints[0]
						for _, msg := range []*expr.AttributeExpr{e.Request, e.Response.Message} {
							require.Len(t, *expr.AsObject(msg.Type), 1)
							key, body := msg.FindAttribute("body")
							require.NotNil(t, body)
							require.Equal(t, required, msg.IsRequired(key))
						}
						services := CreateGRPCServices(root)
						endpoint := services.Get("mapped").Endpoint("echo")
						for _, metadata := range [][]*MetadataData{endpoint.Request.Metadata, endpoint.Response.Headers, endpoint.Response.Trailers} {
							for _, field := range metadata {
								require.Equal(t, required, field.Required)
								require.Equal(t, !required && !defaults, field.Pointer)
								if defaults {
									require.Equal(t, "fallback", field.DefaultValue)
								}
							}
						}
						require.Equal(t, "x-token", endpoint.Request.Metadata[0].Name)
						require.Equal(t, "x-tail", endpoint.Response.Trailers[0].Name)
						// The service contract retains its authored HTTP/JSON names.
						require.NotNil(t, e.MethodExpr.Payload.Find("tok"+suffix))
					})
				}
			}
		}
	}
}

func TestMappedMetadataGeneratedModule(t *testing.T) {
	root := RunGRPCDSL(t, testdata.MappedMetadataDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/mapped", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata_test.go"), []byte(mappedMetadataHarness()), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", ".")
}

func TestMappedMetadataRejectsInvalidReferences(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mapping func()
		want    string
	}{
		{"missing", func() {
			dsl.Metadata(func() {
				dsl.Attribute("missing")
			})
		}, "metadata attribute \"missing\" is not found"},
		{"request overlap", func() {
			dsl.Message(func() {
				dsl.Attribute("tok:message_name")
			})
			dsl.Metadata(func() {
				dsl.Attribute("tok:header_name")
			})
		}, "defined in both request message and metadata"},
		{"response overlap", func() {
			dsl.Response(dsl.CodeOK, func() {
				dsl.Message(func() {
					dsl.Attribute("tok:message_name")
				})
				dsl.Headers(func() {
					dsl.Attribute("tok:header_name")
				})
			})
		}, "defined in both response message and header metadata"},
		{"three component overlap", func() {
			dsl.Response(dsl.CodeOK, func() {
				dsl.Message(func() {
					dsl.Attribute("body")
				})
				dsl.Headers(func() {
					dsl.Attribute("tok:header_name")
				})
				dsl.Trailers(func() {
					dsl.Attribute("tok:trailer_name")
				})
			})
		}, "defined in both response trailer metadata and header metadata"},
		{"metadata overlap", func() {
			dsl.Response(dsl.CodeOK, func() {
				dsl.Headers(func() {
					dsl.Attribute("tok:header_name")
				})
				dsl.Trailers(func() {
					dsl.Attribute("tok:trailer_name")
				})
			})
		}, "defined in both response trailer metadata and header metadata"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				value := dsl.Type("Value", func() {
					dsl.Field(1, "tok:json_name", dsl.String)
					dsl.Field(2, "body", dsl.String)
				})
				dsl.Service("mapped", func() {
					dsl.Method("echo", func() {
						dsl.Payload(value)
						dsl.Result(value)
						dsl.GRPC(tc.mapping)
					})
				})
			})
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func mappedMetadataHarness() string {
	var source strings.Builder
	source.WriteString(`package mappedtest
import (
 "context"
 "reflect"
 "testing"
 "github.com/stretchr/testify/require"
 "google.golang.org/grpc/metadata"
 "google.golang.org/protobuf/proto"
`)
	for i := range 16 {
		fmt.Fprintf(&source, "svc%d \"example.com/mapped/gen/mapped%d\"\nclient%d \"example.com/mapped/gen/grpc/mapped%d/client\"\nserver%d \"example.com/mapped/gen/grpc/mapped%d/server\"\n", i, i, i, i, i, i)
	}
	source.WriteString(")\nfunc TestMetadataRoundTrips(t *testing.T) {\n")
	for i := range 16 {
		fmt.Fprintf(&source, `t.Run("mapped%[1]d",func(t *testing.T) {
 input := &svc%[1]d.Mapped%[1]dValue{}
 fields := reflect.ValueOf(input).Elem()
 for _, name := range []string{"Body","Tok","Tail"} {
  field := fields.FieldByName(name)
  value := name+"-value"
  if field.Kind() == reflect.Pointer {
   field.Set(reflect.ValueOf(&value))
  } else {
   field.SetString(value)
  }
 }
 ctx := context.Background()
 md := metadata.MD{}
 request, err := client%[1]d.EncodeEchoRequest(ctx,input,&md)
 require.NoError(t,err)
 require.Equal(t,[]string{"Tok-value"},md.Get("x-token"))
 require.Equal(t,[]string{"Tail-value"},md.Get("tail"))
 require.Equal(t,1,request.(proto.Message).ProtoReflect().Descriptor().Fields().Len())
 decoded,err := server%[1]d.DecodeEchoRequest(ctx,request,md)
 require.NoError(t,err)
 require.Equal(t,input,decoded)
 headers,trailers := metadata.MD{},metadata.MD{}
 response,err := server%[1]d.EncodeEchoResponse(ctx,input,&headers,&trailers)
 require.NoError(t,err)
 require.Equal(t,[]string{"Tok-value"},headers.Get("x-token"))
 require.Equal(t,[]string{"Tail-value"},trailers.Get("x-tail"))
 require.Equal(t,1,response.(proto.Message).ProtoReflect().Descriptor().Fields().Len())
 result,err := client%[1]d.DecodeEchoResponse(ctx,response,headers,trailers)
 require.NoError(t,err)
 require.Equal(t,input,result)
 _, requestErr := server%[1]d.DecodeEchoRequest(ctx,request,nil)
 _, responseErr := client%[1]d.DecodeEchoResponse(ctx,response,nil,nil)
 if %[2]t {
  require.ErrorContains(t,requestErr,"x-token")
  require.ErrorContains(t,responseErr,"x-token")
 } else {
  require.NoError(t,requestErr)
  require.NoError(t,responseErr)
 }
})
`, i, i%8 >= 4)
	}
	source.WriteString("}\n")
	return source.String()
}

func TestNamedMetadataGeneratedModule(t *testing.T) {
	root := RunGRPCDSL(t, testdata.NamedMetadataDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/namedmetadata", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata_test.go"), []byte(namedMetadataHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", "-run", "^TestNamedMetadata$", "./...")
}
