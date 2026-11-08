package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestJSONQueryCarrierDeclarationClosure renders every transport package that
// receives a named JSON query value and requires each package-local identifier
// used by its decoder, constructor and CLI loader to be declared in that package.
func TestJSONQueryCarrierDeclarationClosure(t *testing.T) {
	cases := []struct {
		name   string
		design func()
	}{
		{"plain union", jsonQueryUnionDSL(false)},
		{"hinted union", jsonQueryUnionDSL(true)},
		{"map of objects", jsonQueryMapDSL},
		{"shared body and query", jsonQueryCarrierDSL},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.design)
			dir := t.TempDir()
			renderHTTPModuleFiles(t, dir, "example.com/carrier/gen", root)
			for _, pkg := range []string{"server", "client"} {
				undefined := undefinedPackageIdentifiers(t, filepath.Join(dir, "gen", "http", "carrier", pkg))
				if len(undefined) > 0 {
					t.Errorf("%s package references undeclared identifiers: %v", pkg, undefined)
				}
			}
		})
	}
}

func TestJSONQueryCarrierGeneratedWire(t *testing.T) {
	root := RunHTTPDSL(t, jsonQueryCarrierDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/carrier", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte(jsonQueryCarrierHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	t.Log(runCommand(t, dir, "go", "test", "-count=1", "-v", "./..."))
}

func jsonQueryPets() (any, any) {
	cat := Type("Cat", func() {
		Attribute("kind", String, func() {
			Enum("cat")
		})
		Attribute("name", String, func() {
			MinLength(2)
		})
		Required("kind", "name")
	})
	dog := Type("Dog", func() {
		Attribute("kind", String, func() {
			Enum("dog")
		})
		Attribute("age", Int, func() {
			Minimum(0)
		})
		Required("kind", "age")
	})
	return cat, dog
}

func jsonQueryChoice(name string, cat, dog any, hinted bool) any {
	return Type(name, OneOf(func() {
		Untagged()
		Attribute("cat", cat)
		Attribute("dog", dog)
	}), func() {
		if hinted {
			Meta("openapi:discriminator:propertyName", "kind")
		}
	})
}

func jsonQueryChoiceParam() {
	Param("choice", func() {
		Meta("openapi:contentType", "application/json")
	})
}

func jsonQueryUnionDSL(hinted bool) func() {
	return func() {
		cat, dog := jsonQueryPets()
		choice := jsonQueryChoice("Choice", cat, dog, hinted)
		Service("carrier", func() {
			Method("read", func() {
				Payload(func() {
					Attribute("choice", choice)
				})
				HTTP(func() {
					GET("/")
					jsonQueryChoiceParam()
					Response(StatusNoContent)
				})
			})
		})
	}
}

func jsonQueryMapDSL() {
	cat, _ := jsonQueryPets()
	Service("carrier", func() {
		Method("pets", func() {
			Payload(func() {
				Attribute("cats", MapOf(String, cat))
			})
			HTTP(func() {
				GET("/pets")
				Param("cats")
				Response(StatusNoContent)
			})
		})
	})
}

// jsonQueryCarrierDSL combines plain and hinted untagged unions, a recursive
// union, a map of objects, the ordinary scalar map control, a body sharing the
// declarations and service declarations generated in a struct:pkg:path package.
func jsonQueryCarrierDSL() {
	cat, dog := jsonQueryPets()
	choice := jsonQueryChoice("Choice", cat, dog, false)
	hinted := jsonQueryChoice("HintedChoice", cat, dog, true)
	leaf := Type("Leaf", func() {
		Attribute("value", String, func() {
			MinLength(1)
		})
		Required("value")
	})
	branch := Type("Branch", func() {
		Attribute("label", String)
		Attribute("child", "Tree")
		Required("label")
	})
	tree := Type("Tree", OneOf(func() {
		Untagged()
		Attribute("leaf", leaf)
		Attribute("branch", branch)
	}))
	packagedCat := Type("PackagedCat", func() {
		Meta("struct:pkg:path", "types")
		Attribute("kind", String, func() {
			Enum("cat")
		})
		Attribute("name", String)
		Required("kind", "name")
	})
	packagedDog := Type("PackagedDog", func() {
		Meta("struct:pkg:path", "types")
		Attribute("kind", String, func() {
			Enum("dog")
		})
		Attribute("age", Int, func() {
			Minimum(0)
		})
		Required("kind", "age")
	})
	packaged := Type("PackagedChoice", OneOf(func() {
		Untagged()
		Attribute("cat", packagedCat)
		Attribute("dog", packagedDog)
	}), func() {
		Meta("struct:pkg:path", "types")
	})
	Service("carrier", func() {
		Method("packaged", func() {
			Payload(func() {
				Attribute("choice", packaged)
				Attribute("cats", MapOf(String, packagedCat))
			})
			HTTP(func() {
				GET("/packaged")
				jsonQueryChoiceParam()
				Param("cats")
				Response(StatusNoContent)
			})
		})
		Method("plain", func() {
			Payload(func() {
				Attribute("choice", choice)
			})
			HTTP(func() {
				GET("/plain")
				jsonQueryChoiceParam()
				Response(StatusNoContent)
			})
		})
		Method("hinted", func() {
			Payload(func() {
				Attribute("choice", hinted)
				Required("choice")
			})
			HTTP(func() {
				GET("/hinted")
				jsonQueryChoiceParam()
				Response(StatusNoContent)
			})
		})
		Method("pets", func() {
			Payload(func() {
				Attribute("cats", MapOf(String, cat))
				Attribute("labels", MapOf(String, String))
			})
			HTTP(func() {
				GET("/pets")
				Param("cats")
				Param("labels")
				Response(StatusNoContent)
			})
		})
		Method("tree", func() {
			Payload(func() {
				Attribute("tree", tree)
			})
			HTTP(func() {
				GET("/tree")
				Param("tree", func() {
					Meta("openapi:contentType", "application/json")
				})
				Response(StatusNoContent)
			})
		})
		Method("write", func() {
			Payload(func() {
				Attribute("choice", choice)
				Attribute("cat", cat)
			})
			HTTP(func() {
				POST("/write")
				Response(StatusNoContent)
			})
		})
	})
}

// undefinedPackageIdentifiers returns the identifiers that no file of the
// package in dir, its imports or the universe scope declares.
func undefinedPackageIdentifiers(t *testing.T, dir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	require.NotEmpty(t, paths, dir)
	fset := token.NewFileSet()
	declared := make(map[string]bool)
	var unresolved []*ast.Ident
	for _, path := range paths {
		file, err := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, err)
		for _, spec := range file.Imports {
			name := filepath.Base(spec.Path.Value[1 : len(spec.Path.Value)-1])
			if spec.Name != nil {
				name = spec.Name.Name
			}
			declared[name] = true
		}
		for name := range file.Scope.Objects {
			declared[name] = true
		}
		unresolved = append(unresolved, file.Unresolved...)
	}
	missing := make(map[string]bool)
	for _, ident := range unresolved {
		if !declared[ident.Name] && types.Universe.Lookup(ident.Name) == nil {
			missing[ident.Name] = true
		}
	}
	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

const jsonQueryCarrierHarness = `package carrier_test

import (
	"context"
	"encoding/json/v2"
	"net/http/httptest"
	"net/url"
	"testing"

	client "example.com/carrier/gen/http/carrier/client"
	server "example.com/carrier/gen/http/carrier/server"
	service "example.com/carrier/gen/carrier"
	types "example.com/carrier/gen/types"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cat(name string) *service.Cat {
	return &service.Cat{Kind: "cat", Name: name}
}

func plainCat(name string) *service.Choice {
	value := service.NewChoiceCat(cat(name))
	return &value
}

func plainDog(age int) *service.Choice {
	value := service.NewChoiceDog(&service.Dog{Kind: "dog", Age: age})
	return &value
}

func hintedCat(name string) service.HintedChoice {
	return service.NewHintedChoiceCat(cat(name))
}

func TestPlainUnionRoundTrip(t *testing.T) {
	for _, test := range []struct {
		payload *service.PlainPayload
		wire    string
	}{
		{&service.PlainPayload{Choice: plainCat("Tom")}, "{\"kind\":\"cat\",\"name\":\"Tom\"}"},
		{&service.PlainPayload{Choice: plainDog(3)}, "{\"kind\":\"dog\",\"age\":3}"},
		{&service.PlainPayload{}, ""},
	} {
		request := httptest.NewRequest("GET", "/plain", nil)
		require.NoError(t, client.EncodePlainRequest(nil)(request, test.payload))
		want := url.Values{}
		if test.wire != "" {
			want.Set("choice", test.wire)
		}
		assert.Equal(t, want.Encode(), request.URL.RawQuery)
		decoded, err := server.DecodePlainRequest(loomhttp.NewMuxer(), nil)(request)
		require.NoError(t, err)
		assert.Equal(t, test.payload, decoded)
	}
}

func TestHintedUnionRoundTrip(t *testing.T) {
	payload := &service.HintedPayload{Choice: hintedCat("Tom")}
	request := httptest.NewRequest("GET", "/hinted", nil)
	require.NoError(t, client.EncodeHintedRequest(nil)(request, payload))
	assert.Equal(t, url.Values{"choice": {"{\"kind\":\"cat\",\"name\":\"Tom\"}"}}.Encode(), request.URL.RawQuery)
	decoded, err := server.DecodeHintedRequest(loomhttp.NewMuxer(), nil)(request)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

func TestPetsRoundTrip(t *testing.T) {
	payload := &service.PetsPayload{
		Cats:   map[string]*service.Cat{"a": cat("Tom")},
		Labels: map[string]string{"k": "v"},
	}
	request := httptest.NewRequest("GET", "/pets", nil)
	require.NoError(t, client.EncodePetsRequest(nil)(request, payload))
	want := url.Values{"cats": {"{\"a\":{\"kind\":\"cat\",\"name\":\"Tom\"}}"}, "labels": {"{\"k\":\"v\"}"}}
	assert.Equal(t, want.Encode(), request.URL.RawQuery)
	decoded, err := server.DecodePetsRequest(loomhttp.NewMuxer(), nil)(request)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

func TestSharedBodyRoundTrip(t *testing.T) {
	payload := &service.WritePayload{Choice: plainDog(4), Cat: cat("Tom")}
	request := httptest.NewRequest("POST", "/write", nil)
	require.NoError(t, client.EncodeWriteRequest(loomhttp.RequestEncoder)(request, payload))
	decoded, err := server.DecodeWriteRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)(request)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

func TestPackagedRoundTrip(t *testing.T) {
	choice := types.NewPackagedChoiceDog(&types.PackagedDog{Kind: "dog", Age: 1})
	payload := &service.PackagedPayload{
		Choice: &choice,
		Cats:   map[string]*types.PackagedCat{"a": {Kind: "cat", Name: "Tom"}},
	}
	request := httptest.NewRequest("GET", "/packaged", nil)
	require.NoError(t, client.EncodePackagedRequest(nil)(request, payload))
	want := url.Values{"choice": {"{\"kind\":\"dog\",\"age\":1}"}, "cats": {"{\"a\":{\"kind\":\"cat\",\"name\":\"Tom\"}}"}}
	assert.Equal(t, want.Encode(), request.URL.RawQuery)
	decoded, err := server.DecodePackagedRequest(loomhttp.NewMuxer(), nil)(request)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
	built, err := client.BuildPackagedPayload(want.Get("choice"), want.Get("cats"))
	require.NoError(t, err)
	assert.Equal(t, payload, built)
	_, err = client.BuildPackagedPayload("", "{\"a\":{\"name\":\"Tom\"}}")
	assert.Error(t, err)
}

func TestRecursiveUnionRoundTrip(t *testing.T) {
	leaf := service.NewTreeLeaf(&service.Leaf{Value: "v"})
	root := service.NewTreeBranch(&service.Branch{Label: "root", Child: &leaf})
	payload := &service.TreePayload{Tree: &root}
	raw := "{\"label\":\"root\",\"child\":{\"value\":\"v\"}}"
	request := httptest.NewRequest("GET", "/tree", nil)
	require.NoError(t, client.EncodeTreeRequest(nil)(request, payload))
	assert.Equal(t, url.Values{"tree": {raw}}.Encode(), request.URL.RawQuery)
	decoded, err := server.DecodeTreeRequest(loomhttp.NewMuxer(), nil)(request)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
	built, err := client.BuildTreePayload(raw)
	require.NoError(t, err)
	assert.Equal(t, payload, built)
	_, err = client.BuildTreePayload("{\"label\":\"root\",\"child\":{\"value\":\"\"}}")
	assert.Error(t, err)
}

func TestRejectedQueryValues(t *testing.T) {
	for _, test := range []struct {
		path, field, raw, code string
	}{
		{"/plain", "choice", "{", "decode_payload"},
		{"/plain", "choice", "null", "decode_payload"},
		{"/plain", "choice", "{\"kind\":\"cat\"}", "decode_payload"},
		{"/plain", "choice", "{\"kind\":\"cat\",\"name\":\"T\"}", "decode_payload"},
		{"/plain", "choice", "{\"kind\":\"dog\",\"age\":-1}", "decode_payload"},
		{"/hinted", "", "", "missing_field"},
		{"/pets", "cats", "{\"a\":{\"name\":\"Tom\"}}", "missing_field"},
		{"/pets", "cats", "{\"a\":{\"kind\":\"dog\",\"name\":\"Tom\"}}", "invalid_enum_value"},
		{"/pets", "cats", "{\"a\":{\"kind\":\"cat\",\"name\":\"T\"}}", "invalid_length"},
		{"/pets", "cats", "{\"a\":null}", "decode_payload"},
		{"/packaged", "choice", "{\"kind\":\"dog\",\"age\":-1}", "decode_payload"},
		{"/packaged", "cats", "{\"a\":{\"name\":\"Tom\"}}", "missing_field"},
		{"/pets", "labels", "{\"k\":null}", "decode_payload"},
		{"/tree", "tree", "{\"label\":\"root\",\"child\":{\"value\":\"\"}}", "decode_payload"},
	} {
		t.Run(test.path+test.raw, func(t *testing.T) {
			calls := 0
			endpoint := func(context.Context, any) (any, error) {
				calls++
				return nil, nil
			}
			mux := loomhttp.NewMuxer()
			endpoints := &service.Endpoints{Packaged: endpoint, Plain: endpoint, Hinted: endpoint, Pets: endpoint, Tree: endpoint, Write: endpoint}
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			target := test.path
			if test.field != "" {
				target += "?" + url.Values{test.field: {test.raw}}.Encode()
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest("GET", target, nil))
			require.Equal(t, 400, response.Code, response.Body.String())
			require.Zero(t, calls)
			var problem loomhttp.ProblemResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
			assert.Equal(t, test.code, problem.Code, response.Body.String())
		})
	}
}

func TestMountedQueryValues(t *testing.T) {
	var received any
	endpoint := func(_ context.Context, payload any) (any, error) {
		received = payload
		return nil, nil
	}
	mux := loomhttp.NewMuxer()
	endpoints := &service.Endpoints{Packaged: endpoint, Plain: endpoint, Hinted: endpoint, Pets: endpoint, Tree: endpoint, Write: endpoint}
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	response := httptest.NewRecorder()
	target := "/hinted?" + url.Values{"choice": {"{\"kind\":\"dog\",\"age\":2}"}}.Encode()
	mux.ServeHTTP(response, httptest.NewRequest("GET", target, nil))
	require.Equal(t, 204, response.Code, response.Body.String())
	dog := service.NewHintedChoiceDog(&service.Dog{Kind: "dog", Age: 2})
	assert.Equal(t, &service.HintedPayload{Choice: dog}, received)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/plain", nil))
	require.Equal(t, 204, response.Code, response.Body.String())
	assert.Equal(t, &service.PlainPayload{}, received)
}

func TestCLIQueryValues(t *testing.T) {
	plain, err := client.BuildPlainPayload("{\"kind\":\"cat\",\"name\":\"Tom\"}")
	require.NoError(t, err)
	assert.Equal(t, &service.PlainPayload{Choice: plainCat("Tom")}, plain)
	empty, err := client.BuildPlainPayload("")
	require.NoError(t, err)
	assert.Equal(t, &service.PlainPayload{}, empty)
	hinted, err := client.BuildHintedPayload("{\"kind\":\"cat\",\"name\":\"Tom\"}")
	require.NoError(t, err)
	assert.Equal(t, &service.HintedPayload{Choice: hintedCat("Tom")}, hinted)
	pets, err := client.BuildPetsPayload("{\"a\":{\"kind\":\"cat\",\"name\":\"Tom\"}}", "{\"k\":\"v\"}")
	require.NoError(t, err)
	assert.Equal(t, &service.PetsPayload{Cats: map[string]*service.Cat{"a": cat("Tom")}, Labels: map[string]string{"k": "v"}}, pets)
	for _, invalid := range []func() error{
		func() error { _, err := client.BuildPlainPayload("{\"kind\":\"cat\"}"); return err },
		func() error { _, err := client.BuildPlainPayload("{\"kind\":\"cat\",\"name\":\"T\"}"); return err },
		func() error { _, err := client.BuildPlainPayload("null"); return err },
		func() error { _, err := client.BuildPetsPayload("null", ""); return err },
		func() error { _, err := client.BuildPetsPayload("", "null"); return err },
		func() error { _, err := client.BuildPetsPayload("{\"a\":{\"name\":\"Tom\"}}", ""); return err },
	} {
		assert.Error(t, invalid())
	}
}
`
