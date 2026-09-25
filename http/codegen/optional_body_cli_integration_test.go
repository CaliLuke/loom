package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestOptionalBodyCLIGeneratedIntegration generates services whose request
// body is a union, object or nullable payload attribute selected with Body,
// compiles and vets them in a temporary module, and calls the generated CLI
// payload builders. An empty body flag leaves an optional attribute nil or
// absent, a set flag is decoded, invalid JSON fails, and the example that
// the error reports decodes to a concrete attribute.
func TestOptionalBodyCLIGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/optbodycli"

	root := RunHTTPDSL(t, optionalBodyCLIIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_body_cli_test.go"), []byte(optionalBodyCLIHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-count=1", ".")
}

func optionalBodyCLIIntegrationDSL() {
	var Leaf = Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	var Other = Type("Other", func() {
		Attribute("count", Int)
		Required("count")
	})
	services := []struct {
		name      string
		required  bool
		attribute func()
	}{
		{"optunion", false, func() { Attribute("b", OneOf(Leaf, Other)) }},
		{"requnion", true, func() { Attribute("b", OneOf(Leaf, Other)) }},
		{"optnunion", false, func() { Attribute("b", OneOf(Leaf, Other), func() { Nullable() }) }},
		{"optobj", false, func() { Attribute("b", Leaf) }},
		{"optnobj", false, func() { Attribute("b", Leaf, func() { Nullable() }) }},
	}
	for _, s := range services {
		Service(s.name, func() {
			Method("pick", func() {
				Payload(func() {
					Attribute("q", String)
					s.attribute()
					if s.required {
						Required("b")
					}
				})
				HTTP(func() {
					POST("/" + s.name)
					Param("q")
					Body("b")
				})
			})
		})
	}
}

const optionalBodyCLIHarness = `package optbodycli

import (
	"strings"
	"testing"

	optnobjclient "example.com/optbodycli/gen/http/optnobj/client"
	optnunionclient "example.com/optbodycli/gen/http/optnunion/client"
	optobjclient "example.com/optbodycli/gen/http/optobj/client"
	optunionclient "example.com/optbodycli/gen/http/optunion/client"
	requnionclient "example.com/optbodycli/gen/http/requnion/client"
)

// example returns the example of valid JSON reported by err, without the
// shell quotes.
func example(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("invalid JSON: no error")
	}
	_, ex, ok := strings.Cut(err.Error(), "example of valid JSON:\n")
	if !ok {
		t.Fatalf("error %q has no example", err)
	}
	return strings.Trim(ex, "'")
}

func TestOptionalUnion(t *testing.T) {
	p, err := optunionclient.BuildPickPayload("", "a")
	if err != nil || p.B != nil || p.Q == nil || *p.Q != "a" {
		t.Errorf("empty flag: payload %+v, error %v, want nil union", p, err)
	}
	p, err = optunionclient.BuildPickPayload(` + "`" + `{"type":"Leaf","value":{"name":"x"}}` + "`" + `, "")
	if err != nil || p.B == nil || p.B.Kind() != "Leaf" {
		t.Errorf("leaf flag: payload %+v, error %v, want Leaf", p, err)
	}
	_, err = optunionclient.BuildPickPayload("{x}", "")
	p, err = optunionclient.BuildPickPayload(example(t, err), "")
	if err != nil || p.B == nil || p.B.Kind() == "" {
		t.Errorf("example flag: payload %+v, error %v, want a union", p, err)
	}
}

func TestRequiredUnion(t *testing.T) {
	_, err := requnionclient.BuildPickPayload("", "")
	if err == nil {
		t.Errorf("empty flag: no error")
	}
	_, err = requnionclient.BuildPickPayload("{x}", "")
	p, err := requnionclient.BuildPickPayload(example(t, err), "")
	if err != nil {
		t.Fatalf("example flag: %v", err)
	}
	if p.B.Kind() == "" {
		t.Errorf("example flag: payload %+v, error %v, want a union", p, err)
	}
}

func TestOptionalNullableUnion(t *testing.T) {
	p, err := optnunionclient.BuildPickPayload("", "")
	if err != nil || p.B.Present() {
		t.Errorf("empty flag: payload %+v, error %v, want absent union", p, err)
	}
	p, err = optnunionclient.BuildPickPayload("null", "")
	if err != nil || !p.B.IsNull() {
		t.Errorf("null flag: payload %+v, error %v, want null union", p, err)
	}
	_, err = optnunionclient.BuildPickPayload("{x}", "")
	p, err = optnunionclient.BuildPickPayload(example(t, err), "")
	if err != nil {
		t.Fatalf("example flag: %v", err)
	}
	if v, ok := p.B.Value(); !ok || v.Kind() == "" {
		t.Errorf("example flag: payload %+v, error %v, want a union", p, err)
	}
}

func TestOptionalObject(t *testing.T) {
	p, err := optobjclient.BuildPickPayload("", "")
	if err != nil || p.B != nil {
		t.Errorf("empty flag: payload %+v, error %v, want nil object", p, err)
	}
	_, err = optobjclient.BuildPickPayload("{x}", "")
	p, err = optobjclient.BuildPickPayload(example(t, err), "")
	if err != nil || p.B == nil || p.B.Name == "" {
		t.Errorf("example flag: payload %+v, error %v, want an object", p, err)
	}
}

func TestOptionalNullableObject(t *testing.T) {
	p, err := optnobjclient.BuildPickPayload("", "")
	if err != nil || p.B.Present() {
		t.Errorf("empty flag: payload %+v, error %v, want absent object", p, err)
	}
	p, err = optnobjclient.BuildPickPayload("null", "")
	if err != nil || !p.B.IsNull() {
		t.Errorf("null flag: payload %+v, error %v, want null object", p, err)
	}
	_, err = optnobjclient.BuildPickPayload("{x}", "")
	p, err = optnobjclient.BuildPickPayload(example(t, err), "")
	if err != nil {
		t.Fatalf("example flag: %v", err)
	}
	if v, ok := p.B.Value(); !ok || v.Name == "" {
		t.Errorf("example flag: payload %+v, error %v, want an object", p, err)
	}
}
`
