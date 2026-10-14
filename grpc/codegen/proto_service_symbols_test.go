package codegen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

const protoServiceSymbolHarness = `package protosymbols_test
import (
 "reflect"
 "testing"
 svc "example.com/protosymbols/gen/billing"
 client "example.com/protosymbols/gen/grpc/billing/client"
 server "example.com/protosymbols/gen/grpc/billing/server"
 pb "example.com/protosymbols/gen/grpc/billing/pb"
)
func TestCanonicalServiceAndConversion(t *testing.T) {
 text := "kept"
 value := &svc.MPayload{C: &svc.BillingClient{Name: &text}, Suffix: &svc.BillingClient2{Other: &text}}
 converted := server.NewMPayload(client.NewProtoMRequest(value))
 if !reflect.DeepEqual(value, converted) {
  t.Errorf("conversion = %#v, want %#v", converted, value)
 }
 if pb.Billing_M_FullMethodName != "/billing.Billing/M" || pb.Billing_ServiceDesc.ServiceName != "billing.Billing" {
  t.Error("service or RPC identity changed")
 }
}
`

func TestProtoServiceSymbolsInferred(t *testing.T) {
	for _, name := range []string{
		"BillingClient", "BillingServer", "UnimplementedBillingServer",
		"UnsafeBillingServer", "RegisterBillingServer", "NewBillingClient",
	} {
		t.Run(name, func(t *testing.T) {
			root := RunGRPCDSL(t, protoServiceSymbolsDSL(name, "", false))
			data := CreateGRPCServices(root).Get("billing")
			var allocated string
			for _, message := range data.Messages {
				if message.Type.Name() == name {
					allocated = message.VarName
				}
			}
			require.NotEmpty(t, allocated)
			require.NotEqual(t, name, allocated, "message must avoid fixed protoc service declaration")
			require.Equal(t, "Billing", data.GoName)
		})
	}
}

func TestProtoServiceSymbolsExplicit(t *testing.T) {
	for _, name := range []string{
		"BillingClient", "BillingServer", "UnimplementedBillingServer",
		"UnsafeBillingServer", "RegisterBillingServer", "NewBillingClient",
		"Billing_ServiceDesc", "Billing_M_FullMethodName",
		"Billing_MClient", "Billing_MServer",
	} {
		t.Run(name, func(t *testing.T) {
			root := RunGRPCDSL(t, protoServiceSymbolsDSL("Occupied", name, true))
			func() {
				defer func() {
					failure := recover()
					require.NotNil(t, failure, "explicit protobuf identity cannot silently change")
					require.Contains(t, fmt.Sprint(failure), "conflicts with")
					require.Contains(t, fmt.Sprint(failure), "billing")
					require.Contains(t, fmt.Sprint(failure), name)
				}()
				CreateGRPCServices(root).Get("billing")
			}()
		})
	}
}

func TestProtoServiceSymbolsGenerated(t *testing.T) {
	cases := []struct {
		name, explicit string
		stream         bool
	}{
		{name: "BillingClient"}, {name: "BillingServer"},
		{name: "UnimplementedBillingServer"}, {name: "UnsafeBillingServer"},
		{name: "RegisterBillingServer"}, {name: "NewBillingClient"},
		{name: "Ordinary"}, {name: "Streaming", stream: true},
		{name: "Explicit", explicit: "Billing_MClient"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunGRPCDSL(t, protoServiceSymbolsDSL(tc.name, tc.explicit, tc.stream))
			dir := t.TempDir()
			renderGRPCModule(t, dir, "example.com/protosymbols", root, resolveGRPCLoomSource(t))
			assertProtoServiceSymbolInventory(t, dir, CreateGRPCServices(root).Get("billing"))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "build", "./...")
			runGRPCGoCommand(t, dir, "vet", "./...")
			if tc.name == "BillingClient" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "conversion_test.go"), []byte(protoServiceSymbolHarness), 0o600))
				runGRPCGoCommand(t, dir, "test", "-race", "-count=1", ".")
			}
		})
	}
}

func assertProtoServiceSymbolInventory(t *testing.T, dir string, data *ServiceData) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "gen/grpc/billing/pb/*_grpc.pb.go"))
	require.NoError(t, err)
	require.Len(t, files, 1)
	file, err := parser.ParseFile(token.NewFileSet(), files[0], nil, 0)
	require.NoError(t, err)
	actual := make(map[string]bool)
	for _, declaration := range file.Decls {
		switch d := declaration.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				actual[d.Name.Name] = true
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					actual[s.Name.Name] = true
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.Name != "_" {
							actual[name.Name] = true
						}
					}
				}
			}
		}
	}
	expected := make(map[string]bool)
	for name := range data.protoServiceSymbols {
		expected[name] = true
	}
	require.Equal(t, expected, actual, "inventory must follow actual pinned protoc declarations")
}

func protoServiceSymbolsDSL(name, explicit string, stream bool) func() {
	return func() {
		typ := Type(name, func() {
			Field(1, "name", String)
			if explicit != "" {
				Meta("struct:name:proto", explicit)
			}
			if name == "BillingServer" {
				Meta("struct:pkg:path", "domain/types")
			}
		})
		var suffix any
		if name == "BillingClient" {
			suffix = Type("BillingClient2", func() {
				Field(1, "other", String)
			})
		}
		Service("billing", func() {
			Method("m", func() {
				Payload(func() {
					Field(1, "c", typ)
					if suffix != nil {
						Field(2, "suffix", suffix)
					}
				})
				if stream {
					StreamingResult(String)
				} else {
					Result(String)
				}
				GRPC(func() {
				})
			})
		})
	}
}
