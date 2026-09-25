package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestNestedUnionBodyFieldTypes checks that the request and response body
// types that hold a named union, directly or in an array or a map, refer to
// a union type that the transport package declares with the branch types of
// the same body, and that an alias user type in a request body keeps its
// underlying type as in a response body.
func TestNestedUnionBodyFieldTypes(t *testing.T) {
	root := RunHTTPDSL(t, nestedUnionBodyDSL)
	sd := CreateHTTPServices(root).Get("nested")
	branches := make(map[string][]string, len(sd.UnionTypes))
	for _, union := range sd.UnionTypes {
		for _, field := range union.Fields {
			branches[union.Name] = append(branches[union.Name], field.FieldType)
		}
	}
	cases := []struct {
		name   string
		types  []*TypeData
		holder string
		branch string
	}{
		{"server request", sd.ServerBodyAttributeTypes, "HolderRequestBody", "*LeafRequestBody"},
		{"client request", sd.ClientBodyAttributeTypes, "HolderRequestBody", "*LeafRequestBody"},
	}
	for _, c := range cases {
		var holder *TypeData
		for _, data := range c.types {
			if data.Name == c.holder {
				holder = data
			}
		}
		if !assert.NotNil(t, holder, "%s: no %s type", c.name, c.holder) {
			continue
		}
		fields := nestedUnionFieldTypes(holder.Def)
		for _, field := range []string{"Choice", "List", "Dict"} {
			union := fields[field]
			assert.Contains(t, branches, union, "%s: %s.%s refers to undeclared union %q", c.name, c.holder, field, union)
			assert.Contains(t, branches[union], c.branch, "%s: %s.%s union %q has branches %v", c.name, c.holder, field, union, branches[union])
		}
		assert.Equal(t, "string", fields["Nm"], "%s: %s.Nm", c.name, c.holder)
	}
}

// TestNestedUnionBodyGeneratedIntegration compiles and vets services whose
// request and response bodies hold a named union in a user type, directly
// and in an array and a map, next to an alias user type with a validation,
// both as the whole body and as an explicit Body attribute. It round-trips
// every branch through the generated client and server, sends invalid
// request bodies and checks the status, the problem code and detail, and
// that the service is not invoked.
func TestNestedUnionBodyGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/unionnestedbody"

	root := RunHTTPDSL(t, nestedUnionBodyDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union_nested_body_test.go"), []byte(nestedUnionBodyHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

// nestedUnionFieldTypes returns the union or value type of each field of the
// Go struct definition def, without the presence wrappers and collection
// syntax around it.
func nestedUnionFieldTypes(def string) map[string]string {
	fields := make(map[string]string)
	for _, line := range strings.Split(def, "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		typ, trimmed := parts[1], true
		for trimmed {
			trimmed = false
			for _, prefix := range []string{"loom.Optional[", "loom.Nullable[", "*", "[]", "map[string]"} {
				if strings.HasPrefix(typ, prefix) {
					typ, trimmed = strings.TrimPrefix(typ, prefix), true
				}
			}
		}
		fields[parts[0]] = strings.TrimRight(typ, "]")
	}
	return fields
}

func nestedUnionBodyDSL() {
	leaf, other := unionBodyBranchTypes()
	choice := Type("Choice", OneOf(leaf, other))
	name := Type("Name", String, func() {
		MinLength(2)
	})
	holder := Type("Holder", func() {
		Attribute("choice", choice)
		Attribute("nm", name)
		Attribute("list", ArrayOf(choice))
		Attribute("dict", MapOf(String, choice))
		Required("choice")
	})
	Service("nested", func() {
		Method("put", func() {
			Payload(func() {
				Attribute("holder", holder)
				Required("holder")
			})
			Result(holder)
			HTTP(func() {
				POST("/put")
			})
		})
		Method("body", func() {
			Payload(func() {
				Attribute("q", String)
				Attribute("holder", holder)
				Required("holder")
			})
			Result(holder)
			HTTP(func() {
				POST("/body")
				Param("q")
				Body("holder")
			})
		})
	})
}

const nestedUnionBodyHarness = `package unionnestedbody

import (
	"context"
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	nestedclient "example.com/unionnestedbody/gen/http/nested/client"
	nestedserver "example.com/unionnestedbody/gen/http/nested/server"
	nested "example.com/unionnestedbody/gen/nested"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []*nested.Holder
}

func (s *service) record(h *nested.Holder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, h)
}

func (s *service) take() []*nested.Holder {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

func (s *service) Put(_ context.Context, p *nested.PutPayload) (*nested.Holder, error) {
	s.record(p.Holder)
	return p.Holder, nil
}

func (s *service) Body(_ context.Context, p *nested.BodyPayload) (*nested.Holder, error) {
	s.record(p.Holder)
	return p.Holder, nil
}

func ptr[T any](v T) *T { return &v }

func holders() []*nested.Holder {
	name := nested.Name("ab")
	return []*nested.Holder{
		{Choice: nested.NewChoiceLeaf(&nested.Leaf{Name: "a"})},
		{
			Choice: nested.NewChoiceOther(&nested.Other{Count: ptr(3)}),
			Nm:     &name,
			List: []nested.Choice{
				nested.NewChoiceLeaf(&nested.Leaf{Name: "b"}),
				nested.NewChoiceOther(&nested.Other{}),
			},
			Dict: map[string]nested.Choice{
				"k": nested.NewChoiceLeaf(&nested.Leaf{Name: "c"}),
			},
		},
	}
}

func TestRoundTrip(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	nestedserver.Mount(mux, nestedserver.New(nested.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := nestedclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()
	for i, h := range holders() {
		calls := []struct {
			name string
			call func() (any, error)
		}{
			{"put", func() (any, error) { return c.Put()(ctx, &nested.PutPayload{Holder: h}) }},
			{"body", func() (any, error) { return c.Body()(ctx, &nested.BodyPayload{Q: ptr("q"), Holder: h}) }},
		}
		for _, call := range calls {
			res, err := call.call()
			if err != nil {
				t.Fatalf("%s %d: %v", call.name, i, err)
			}
			if !reflect.DeepEqual(res, h) {
				t.Errorf("%s %d: result %#v, want %#v", call.name, i, res, h)
			}
			if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], h) {
				t.Errorf("%s %d: server received %+v, want %+v", call.name, i, seen, h)
			}
		}
	}

	cases := []struct {
		name   string
		holder string
		status int
		code   string
		detail string
	}{
		{"leaf", ` + "`" + `{"choice":{"type":"Leaf","value":{"name":"x"}},"nm":"ab","list":[{"type":"Other","value":{}}],"dict":{"k":{"type":"Leaf","value":{"name":"y"}}}}` + "`" + `, http.StatusOK, "", ""},
		{"missing choice", ` + "`" + `{"nm":"ab"}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: choice"},
		{"unknown branch", ` + "`" + `{"choice":{"type":"Nope","value":{}}}` + "`" + `, http.StatusBadRequest, "invalid_enum_value", ` + "`" + `invalid value for "type": got "Nope", expected one of "Leaf", "Other"` + "`" + `},
		{"invalid branch", ` + "`" + `{"choice":{"type":"Leaf","value":{}}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name"},
		{"invalid list branch", ` + "`" + `{"choice":{"type":"Other","value":{}},"list":[{"type":"Leaf","value":{}}]}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name"},
		{"invalid dict branch", ` + "`" + `{"choice":{"type":"Other","value":{}},"dict":{"k":{"type":"Leaf"}}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: value"},
		{"short alias", ` + "`" + `{"choice":{"type":"Other","value":{}},"nm":"a"}` + "`" + `, http.StatusBadRequest, "invalid_length", "validation error"},
	}
	bodies := []struct {
		path string
		wrap func(string) string
	}{
		{"/put", func(h string) string { return ` + "`" + `{"holder":` + "`" + ` + h + "}" }},
		{"/body", func(h string) string { return h }},
	}
	for _, b := range bodies {
		for _, tc := range cases {
			body := b.wrap(tc.holder)
			resp, err := hs.Client().Post(hs.URL+b.path, "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatalf("%s %s: %v", b.path, tc.name, err)
			}
			raw, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("%s %s: read body: %v", b.path, tc.name, err)
			}
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("%s %s: close body: %v", b.path, tc.name, err)
			}
			if resp.StatusCode != tc.status {
				t.Errorf("%s %s: status %d, want %d (%s)", b.path, tc.name, resp.StatusCode, tc.status, raw)
			}
			seen := svc.take()
			if tc.status == http.StatusOK {
				if len(seen) != 1 || seen[0] == nil || seen[0].Choice.Kind() != nested.ChoiceKindLeaf || len(seen[0].List) != 1 || len(seen[0].Dict) != 1 {
					t.Errorf("%s %s: service received %+v", b.path, tc.name, seen)
				}
				if got := strings.TrimSpace(string(raw)); got != tc.holder {
					t.Errorf("%s %s: response body %s, want %s", b.path, tc.name, got, tc.holder)
				}
				continue
			}
			var problem loomhttp.ProblemResponse
			if err := json.Unmarshal(raw, &problem); err != nil {
				t.Errorf("%s %s: decode problem %q: %v", b.path, tc.name, raw, err)
			}
			if problem.Code != tc.code || problem.Detail != tc.detail {
				t.Errorf("%s %s: problem (%q, %q), want (%q, %q)", b.path, tc.name, problem.Code, problem.Detail, tc.code, tc.detail)
			}
			if len(seen) != 0 {
				t.Errorf("%s %s: service invoked with %+v", b.path, tc.name, seen)
			}
		}
	}
}
`
