package codegen

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestViewedUnionDeclarations(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ViewedOmittedUnionDSL)
	services := CreateHTTPServices(root)
	files := ServerTypeFiles("example.com/viewunion/gen", services)
	var declarations strings.Builder
	for _, file := range files {
		for _, section := range file.Section("server-union-type") {
			declarations.WriteString(codegen.SectionCode(t, section))
		}
	}
	require.Empty(t, declarations.String(), "the selected server bodies contain no union")
}

func TestViewedUnionDeclarationsGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ViewedOmittedUnionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/viewunion", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
}

func TestViewedUnionPackageControls(t *testing.T) {
	for _, test := range []struct {
		name    string
		options testdata.ViewedUnionOptions
	}{
		{"included", testdata.ViewedUnionOptions{IncludeShape: true}},
		{"included map", testdata.ViewedUnionOptions{IncludeShape: true, MapItems: true}},
		{"included untagged", testdata.ViewedUnionOptions{IncludeShape: true, Untagged: true}},
		{"excluded request reuse", testdata.ViewedUnionOptions{ReuseRequest: true}},
		{"excluded untagged", testdata.ViewedUnionOptions{Untagged: true}},
		{"excluded map", testdata.ViewedUnionOptions{MapItems: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.ViewedUnionDSL(test.options))
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/viewunion", root)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
}

func TestViewedUnionPackageMembership(t *testing.T) {
	for _, test := range []struct {
		name        string
		options     testdata.ViewedUnionOptions
		serverCount int
		clientCount int
	}{
		{"omitted", testdata.ViewedUnionOptions{}, 0, 1},
		{"included", testdata.ViewedUnionOptions{IncludeShape: true}, 1, 1},
		{"request reuse", testdata.ViewedUnionOptions{ReuseRequest: true}, 1, 2},
		{"untagged omitted", testdata.ViewedUnionOptions{Untagged: true}, 0, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.ViewedUnionDSL(test.options))
			services := CreateHTTPServices(root)
			serverCount, clientCount := 0, 0
			for _, file := range ServerTypeFiles("example.com/viewunion/gen", services) {
				serverCount += len(file.Section("server-union-type"))
			}
			for _, file := range ClientTypeFiles("example.com/viewunion/gen", services) {
				clientCount += len(file.Section("client-union-type"))
			}
			require.Equal(t, test.serverCount, serverCount)
			require.Equal(t, test.clientCount, clientCount)
		})
	}
}

func TestPreparedBodyUnionAllocationAndMembership(t *testing.T) {
	for _, serverFirst := range []bool{false, true} {
		scope := codegen.NewNameScope()
		sd := &ServiceData{Scope: scope}
		shared := &expr.AttributeExpr{Type: makeUnionForOrderTest("Shared", "left", "right")}
		clientOnly := &expr.AttributeExpr{Type: makeUnionForOrderTest("ClientOnly", "one")}
		serverOnly := &expr.AttributeExpr{Type: makeUnionForOrderTest("ServerOnly", "two")}
		for _, server := range []bool{serverFirst, !serverFirst} {
			for range 2 {
				sd.registerBodyRoot(shared, "read", true, server, server)
			}
		}
		sd.registerBodyRoot(clientOnly, "read", false, false, true)
		sd.registerBodyRoot(serverOnly, "read", false, true, false)
		sd.collectBodyUnionTypes(false)
		require.ElementsMatch(t, []string{"Shared", "ClientOnly"}, httpUnionTypeNames(sd.ClientUnionTypes))
		require.ElementsMatch(t, []string{"Shared", "ServerOnly"}, httpUnionTypeNames(sd.ServerUnionTypes))
		clientShared := sd.ClientUnionTypes[1]
		serverShared := sd.ServerUnionTypes[1]
		require.Same(t, clientShared, serverShared, "package membership must reuse the allocated declaration")
		require.Equal(t, "SharedKindLeft", clientShared.Fields[0].KindConst)
	}
}

func TestViewedUntaggedUnionValidatorDependencies(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ViewedUnionDSL(testdata.ViewedUnionOptions{IncludeShape: true, Untagged: true}))
	sd := CreateHTTPServices(root).Get("svc")
	require.Len(t, sd.ServerUnionTypes, 1)
	require.True(t, sd.ServerUnionTypes[0].Untagged)
	validators := make(map[string]string)
	for _, typ := range sd.ServerBodyAttributeTypes {
		validators[typ.VarName] = typ.ValidateDef
	}
	for _, branch := range sd.ServerUnionTypes[0].Fields {
		require.NotEmpty(t, validators[strings.TrimPrefix(branch.FieldType, "*")])
	}
}

func TestViewedUnionWire(t *testing.T) {
	for _, included := range []bool{false, true} {
		t.Run(strconv.FormatBool(included), func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.ViewedUnionDSL(testdata.ViewedUnionOptions{IncludeShape: included}))
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/viewunion", root)
			harness := strings.ReplaceAll(viewedUnionWireHarness, "INCLUDED", strconv.FormatBool(included))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "view_test.go"), []byte(harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}

const viewedUnionWireHarness = `package viewunion_test
import (
 "context"
 "net/http"
 "net/http/httptest"
 "testing"
 "github.com/stretchr/testify/require"
 svc "example.com/viewunion/gen/svc"
 server "example.com/viewunion/gen/http/svc/server"
 client "example.com/viewunion/gen/http/svc/client"
 loomhttp "github.com/CaliLuke/loom/http"
)
func TestSelectedUnionWire(t *testing.T) {
 choice := svc.NewKindCircle(7)
 result := &svc.Wrap{ID:"wrap",Items:[]*svc.Item{{ID:"item",Shape:&svc.Shape{Kind:&choice}}}}
 for _, view := range []string{"default","tiny"} {
  projected,err := svc.NewViewedWrap(result,view)
  require.NoError(t,err)
  recorder:=httptest.NewRecorder()
  require.NoError(t,server.EncodeGetResponse(loomhttp.ResponseEncoder)(context.Background(),recorder,projected))
  require.Equal(t,http.StatusOK,recorder.Code)
  if view=="tiny" {
   require.JSONEq(t,"{\"id\":\"wrap\"}",recorder.Body.String())
  } else if INCLUDED {
   require.Contains(t,recorder.Body.String(),"\"shape\"")
   require.Contains(t,recorder.Body.String(),"\"circle\"")
  } else {
   require.JSONEq(t,"{\"id\":\"wrap\",\"items\":[{\"id\":\"item\"}]}",recorder.Body.String())
  }
  decoded,err:=client.DecodeGetResponse(loomhttp.ResponseDecoder,false)(recorder.Result())
  require.NoError(t,err)
  got:=decoded.(*svc.Wrap)
  require.Equal(t,"wrap",got.ID)
  if view=="tiny" {
   require.Empty(t,got.Items)
  } else {
   require.Len(t,got.Items,1)
   if INCLUDED {
    require.NotNil(t,got.Items[0].Shape)
    value,ok:=got.Items[0].Shape.Kind.AsCircle()
    require.True(t,ok)
    require.Equal(t,svc.KindCircle(7),value)
   } else {
    require.Nil(t,got.Items[0].Shape)
   }
  }
 }
}
`
