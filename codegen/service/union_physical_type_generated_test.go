package service_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestUnionPhysicalTypeGenerated(t *testing.T) {
	for _, customType := range []string{"", "scalar.Text", "scalar.Label", "*scalar.Text"} {
		name := map[string]string{"": "native", "scalar.Text": "struct", "scalar.Label": "defined-string", "*scalar.Text": "pointer"}[customType]
		t.Run(name, func(t *testing.T) {
			const module = "example.com/unionphysical"
			root := codegen.RunDSL(t, func() {
				choice := dsl.Type("Choice", func() {
					dsl.OneOf("value", func() {
						dsl.Attribute("text", dsl.String, func() {
							if customType != "" {
								dsl.Meta("struct:field:type", customType, module+"/scalar")
							}
						})
						dsl.Attribute("other", dsl.String, func() {
							if customType != "" {
								dsl.Meta("struct:field:type", customType, module+"/scalar")
							}
						})
						dsl.Attribute("number", dsl.Int)
					})
				})
				result := dsl.ResultType("application/vnd.choice-result", func() {
					dsl.TypeName("ChoiceResult")
					dsl.Extend(choice)
					dsl.View("default", func() {
						dsl.Attribute("value")
					})
				})
				dsl.Service("choices", func() {
					dsl.Method("echo", func() {
						dsl.Payload(choice)
						dsl.Result(result)
						dsl.HTTP(func() {
							dsl.POST("/echo")
						})
					})
				})
			})
			services := servicecodegen.NewServicesData(root)
			dir := t.TempDir()
			files := make([]*codegen.File, 0)
			for _, service := range root.Services {
				require.NoError(t, servicecodegen.SetUserTypeImports(module+"/gen", services.Get(service.Name)))
				files = append(files, servicecodegen.Files(module+"/gen", service, services, make(map[string][]string))...)
				files = append(files, servicecodegen.EndpointFile(module+"/gen", service, services))
				if views := servicecodegen.ViewsFile(module+"/gen", service, services); views != nil {
					files = append(files, views)
				}
			}
			httpServices := httpcodegen.NewServicesData(services, root.API.HTTP)
			files = append(files, httpcodegen.PathFiles(httpServices)...)
			files = append(files, httpcodegen.ServerFiles(module+"/gen", httpServices)...)
			files = append(files, httpcodegen.ServerTypeFiles(module+"/gen", httpServices)...)
			files = append(files, httpcodegen.ClientFiles(module+"/gen", httpServices)...)
			files = append(files, httpcodegen.ClientTypeFiles(module+"/gen", httpServices)...)
			for _, file := range files {
				if header := file.HeaderTemplate(); header != nil {
					servicecodegen.AddServiceDataMetaTypeImports(header, services.Get("choices"))
				}
				_, err := file.Render(dir)
				require.NoError(t, err)
			}
			repository, err := filepath.Abs("../..")
			require.NoError(t, err)
			mod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, repository)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0o750))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "scalar", "text.go"), []byte(unionPhysicalText), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "union_test.go"), []byte(strings.ReplaceAll(unionPhysicalHarness, "CUSTOM_TYPE", customType)), 0o600))
			for _, args := range [][]string{{"mod", "tidy"}, {"test", "-race", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}

const unionPhysicalText = `package scalar

import "errors"

var ErrText = errors.New("invalid authored text")

type Text struct {
	Value string
}

func (v Text) String() string {
	return v.Value
}
func (v Text) MarshalText() ([]byte, error) {
	return []byte(v.Value), nil
}
func (v *Text) UnmarshalText(text []byte) error {
	if string(text) == "bad" {
		return ErrText
	}
	v.Value = string(text)
	return nil
}

type Label string

func (v Label) String() string {
	return string(v)
}
func (v Label) MarshalText() ([]byte, error) {
	return []byte(v), nil
}
func (v *Label) UnmarshalText(text []byte) error {
	if string(text) == "bad" {
		return ErrText
	}
	*v = Label(text)
	return nil
}
`

const unionPhysicalHarness = `package unionphysical_test

import (
	"context"
	"encoding"
	"encoding/json/v2"
	"errors"
	choices "example.com/unionphysical/gen/choices"
	client "example.com/unionphysical/gen/http/choices/client"
	server "example.com/unionphysical/gen/http/choices/server"
	scalar "example.com/unionphysical/scalar"
	"fmt"
	loomhttp "github.com/CaliLuke/loom/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
)

func TestAuthoredBranchIdentity(t *testing.T) {
	var union choices.Value
	custom := "CUSTOM_TYPE"
	if custom != "" {
		expected := map[string]reflect.Type{"scalar.Text": reflect.TypeOf(scalar.Text{}), "scalar.Label": reflect.TypeOf(scalar.Label("")), "*scalar.Text": reflect.TypeOf((*scalar.Text)(nil))}[custom]
		if reflect.TypeOf(union.Text) != expected {
			t.Errorf("branch type = %T, want authored %v", union.Text, expected)
		}
		if _, ok := any(union.Text).(fmt.Stringer); !ok {
			t.Errorf("branch %T lost fmt.Stringer", union.Text)
		}
		if custom == "*scalar.Text" {
			reflect.ValueOf(&union.Text).Elem().Set(reflect.New(expected.Elem()))
		}
		parser, ok := any(union.Text).(encoding.TextUnmarshaler)
		if !ok {
			parser, ok = any(&union.Text).(encoding.TextUnmarshaler)
		}
		if !ok {
			t.Fatalf("branch %T lost encoding.TextUnmarshaler", &union.Text)
		}
		if err := parser.UnmarshalText([]byte("valid")); err != nil {
			t.Fatal(err)
		}
		if text := any(union.Text).(fmt.Stringer).String(); text != "valid" {
			t.Errorf("text = %q", text)
		}
		if err := parser.UnmarshalText([]byte("bad")); !errors.Is(err, scalar.ErrText) {
			t.Errorf("parser failure = %v", err)
		}
	}
	union.SetText(union.Text)
	if union.Kind() != choices.ValueKindText {
		t.Errorf("wrong tag: %s", union.Kind())
	}
	value, ok := union.AsText()
	if !ok {
		t.Fatal("text branch not selected")
	}
	raw, err := json.Marshal(union)
	if err != nil {
		t.Fatal(err)
	}
	var copy choices.Value
	if err := json.Unmarshal(raw, &copy); err != nil {
		t.Fatal(err)
	}
	if decoded, ok := copy.AsText(); !ok || !reflect.DeepEqual(value, decoded) {
		t.Errorf("round trip: %#v %#v", value, decoded)
	}
	service := &echoService{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(choices.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	endpointURL, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	transport := client.NewClient(endpointURL.Scheme, endpointURL.Host, httpServer.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	response, err := transport.Echo()(context.Background(), &choices.Choice{Value: &union})
	if err != nil {
		t.Fatal(err)
	}
	result := response.(*choices.ChoiceResult)
	decoded, ok := result.Value.AsText()
	if !ok || !reflect.DeepEqual(value, decoded) || service.calls.Load() != 1 {
		t.Fatalf("HTTP view roundtrip: %#v calls=%d", result, service.calls.Load())
	}
	union.SetOther(union.Other)
	if union.Kind() != choices.ValueKindOther {
		t.Errorf("duplicate physical type lost distinct tag: %s", union.Kind())
	}
	if _, ok := union.AsText(); ok {
		t.Fatal("other branch still selects text")
	}
}

type echoService struct {
	calls atomic.Int64
}

func (s *echoService) Echo(_ context.Context, input *choices.Choice) (*choices.ChoiceResult, error) {
	s.calls.Add(1)
	return &choices.ChoiceResult{Value: input.Value}, nil
}
`
