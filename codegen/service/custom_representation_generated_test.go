package service

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestCustomRepresentationViewsGeneratedModule exercises semantic validation,
// selected default materialization and nested failure propagation without HTTP.
func TestCustomRepresentationViewsGeneratedModule(t *testing.T) {
	const module = "example.com/customrepresentation"
	root := codegen.RunDSL(t, testdata.CustomRepresentationViewsDSL(module))
	services := NewServicesData(root)
	dir := t.TempDir()
	for _, service := range root.Services {
		files := Files(module+"/gen", service, services, make(map[string][]string))
		if views := ViewsFile(module+"/gen", service, services); views != nil {
			files = append(files, views)
		}
		for _, file := range files {
			if header := file.HeaderTemplate(); header != nil {
				AddServiceDataMetaTypeImports(header, services.Get(service.Name))
			}
			_, err := file.Render(dir)
			require.NoError(t, err)
		}
	}
	mod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0o700))
	for name, source := range map[string]string{
		"scalar/scalar.go":       customRepresentationScalarSource,
		"representation_test.go": customRepresentationViewsHarness,
	} {
		formatted, err := format.Source([]byte(source))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), formatted, 0o600))
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}, {"test", "-race", "-count=1", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

const customRepresentationScalarSource = `package scalar

import (
 "errors"
 "strings"
 "sync/atomic"
)

// ErrRejected identifies a failed authored default parser.
var ErrRejected = errors.New("authored parser rejected text")

// StringCalls counts semantic projections, independently from JSON codecs.
var StringCalls atomic.Int64

// ParseCalls counts default materialization attempts.
var ParseCalls atomic.Int64

// Token is an opaque String representation with explicit semantic capabilities.
type Token struct {
 // Text is the authored semantic value.
 Text string
}

// String exposes the semantic value independently of JSON.
func (t Token) String() string {
 StringCalls.Add(1)
 return t.Text
}

// UnmarshalText parses defaults and deliberately mutates before failure.
func (t *Token) UnmarshalText(raw []byte) error {
 ParseCalls.Add(1)
 t.Text = "partial receiver mutation"
 if !strings.HasPrefix(string(raw), "ok-") {
  return ErrRejected
 }
 t.Text = string(raw)
 return nil
}

// MarshalJSON rejects implicit use as a semantic String projection.
func (Token) MarshalJSON() ([]byte, error) {
 return nil, errors.New("JSON codec must not project String semantics")
}

// UnmarshalJSON rejects implicit use as a String default materializer.
func (*Token) UnmarshalJSON([]byte) error {
 return errors.New("JSON codec must not materialize String defaults")
}

// Label is a defined string with no authored text methods.
type Label string

// Opaque has no text or JSON capabilities and is used without semantic operations.
type Opaque struct {
 // Number is copied without inferring a semantic text representation.
 Number int
}
`

const customRepresentationViewsHarness = `package customrepresentation_test

import (
 "errors"
 "strings"
 "testing"

 svc "example.com/customrepresentation/gen/customviews"
 views "example.com/customrepresentation/gen/customviews/views"
 "example.com/customrepresentation/scalar"
 "github.com/google/uuid"
)

// Native-only selected conversions remain infallible. A recursive reachable
// parser default adds an error only to the selected conversion that can use it.
var _ func(*views.DefaultsView) *svc.Defaults = svc.NewDefaultsFromDefaultsViewTiny
var _ func(*views.BranchView) *svc.Branch = svc.NewBranchFromBranchViewTiny
var _ func(*views.BranchView) (*svc.Branch, error) = svc.NewBranchFromBranchView
var _ func(*views.DefaultsView) (*svc.Defaults, error) = svc.NewDefaultsFromDefaultsView

func pointer[T any](value T) *T {
 return &value
}

func TestSemanticValidation(t *testing.T) {
 id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
 for _, tc := range []struct {
  name string
  token, email *scalar.Token
  label *scalar.Label
  wantError bool
  wantMessage string
 }{
  {name:"optional absent"},
  {name:"format valid", email:pointer(scalar.Token{Text:"a@b.example"})},
  {name:"format invalid", email:pointer(scalar.Token{Text:"invalid"}), wantError:true, wantMessage:"must be formatted"},
  {name:"valid", token:pointer(scalar.Token{Text:"ok-one"}), label:pointer(scalar.Label("label"))},
  {name:"minimum", token:pointer(scalar.Token{Text:"ok-"}), wantError:true, wantMessage:"greater or equal than 6"},
  {name:"maximum", token:pointer(scalar.Token{Text:"ok-too-long"}), wantError:true, wantMessage:"lesser or equal than 6"},
  {name:"pattern", token:pointer(scalar.Token{Text:"no-one"}), wantError:true, wantMessage:"must match the regexp"},
  {name:"enum", token:pointer(scalar.Token{Text:"ok-six"}), wantError:true, wantMessage:"expected one of"},
  {name:"defined string invalid", label:pointer(scalar.Label("x")), wantError:true, wantMessage:"greater or equal than 2"},
  {name:"defined string rune length", label:pointer(scalar.Label("é")), wantError:true, wantMessage:"greater or equal than 2"},
 } {
  t.Run(tc.name, func(t *testing.T) {
   scalar.StringCalls.Store(0)
   input := &views.Record{View:"default", Projected:&views.RecordView{ID:&id,Token:tc.token,Label:tc.label,Email:tc.email}}
   err := views.ValidateRecord(input)
   if (err != nil) != tc.wantError {
    t.Fatalf("validation error=%v wantError=%t",err,tc.wantError)
   }
   if tc.wantMessage != "" && (err == nil || !strings.Contains(err.Error(),tc.wantMessage)) {
    t.Fatalf("authored constraint was not preserved: error=%v want=%q",err,tc.wantMessage)
   }
   wantCalls := int64(0)
   if tc.token != nil {
    wantCalls++
   }
   if tc.email != nil {
    wantCalls++
   }
   if got := scalar.StringCalls.Load(); got != wantCalls {
    t.Fatalf("semantic projection count=%d want=%d",got,wantCalls)
   }
  })
 }
 if err := views.ValidateRecord(&views.Record{View:"default",Projected:&views.RecordView{}}); err == nil {
  t.Fatal("missing required UUID accepted")
 }
 opaque := &scalar.Opaque{Number:7}
 result, err := svc.NewRecord(&views.Record{View:"tiny",Projected:&views.RecordView{Opaque:opaque}})
 if err != nil || result == nil || result.Opaque == nil || result.Opaque.Number != 7 {
  t.Fatalf("unconstrained opaque copy: result=%#v error=%v",result,err)
 }
 viewed, err := svc.NewViewedRecord(&svc.Record{ID:id,Token:pointer(scalar.Token{Text:"ok-two"})},"default")
 if err != nil || viewed == nil || viewed.Projected.Token.Text != "ok-two" {
  t.Fatalf("service projection validation: value=%#v error=%v",viewed,err)
 }
 if _, err := svc.NewViewedRecord(&svc.Record{ID:id,Token:pointer(scalar.Token{Text:"bad"})},"default"); err == nil {
  t.Fatal("service projection skipped custom validation")
 }
}

func TestCustomDefaults(t *testing.T) {
 scalar.ParseCalls.Store(0)
 result, err := svc.NewDefaults(&views.Defaults{View:"default",Projected:&views.DefaultsView{}})
 if err != nil || result == nil || result.ID.String() != "550e8400-e29b-41d4-a716-446655440000" || result.Token.Text != "ok-default" || result.Label != "native-label" {
  t.Fatalf("absent defaults: value=%#v error=%v",result,err)
 }
 if scalar.ParseCalls.Load() != 1 {
  t.Fatalf("excluded bad default parsed: %d",scalar.ParseCalls.Load())
 }
 zeroID, zeroToken, zeroLabel := uuid.Nil, scalar.Token{}, scalar.Label("")
 scalar.ParseCalls.Store(0)
 result, err = svc.NewDefaults(&views.Defaults{View:"default",Projected:&views.DefaultsView{ID:&zeroID,Token:&zeroToken,Label:&zeroLabel}})
 if err != nil || result == nil || result.ID != uuid.Nil || result.Token.Text != "" || result.Label != "" {
  t.Fatalf("supplied zero overwritten: value=%#v error=%v",result,err)
 }
 if scalar.ParseCalls.Load() != 0 || zeroToken.Text != "" {
  t.Fatal("supplied value parsed or mutated")
 }
 scalar.ParseCalls.Store(0)
 result, err = svc.NewDefaults(&views.Defaults{View:"tiny",Projected:&views.DefaultsView{}})
 if err != nil || result == nil || result.Native != "native-default" || scalar.ParseCalls.Load() != 0 {
  t.Fatalf("native selected view parsed excluded custom defaults: value=%#v error=%v",result,err)
 }
 result, err = svc.NewDefaults(&views.Defaults{View:"bad",Projected:&views.DefaultsView{}})
 if !errors.Is(err,scalar.ErrRejected) || result != nil {
  t.Fatalf("authored parser error lost or partial result published: value=%#v error=%v",result,err)
 }
 result, err = svc.NewDefaults(&views.Defaults{View:"baduuid",Projected:&views.DefaultsView{}})
 if err == nil || result != nil {
  t.Fatalf("invalid UUID default accepted: value=%#v error=%v",result,err)
 }
}

func TestRecursiveDefaultFailures(t *testing.T) {
 supplied := scalar.Token{Text:"ok-supplied"}
 for _, tc := range []struct {
  name string
  input *views.BranchView
 }{
  {"recursive",&views.BranchView{Head:&views.NodeView{Token:&supplied,Next:&views.NodeView{}}}},
  {"array",&views.BranchView{Children:[]*views.NodeView{{}}}},
  {"map",&views.BranchView{Lookup:map[string]*views.NodeView{"leaf":{}}}},
 } {
  t.Run(tc.name,func(t *testing.T) {
   result, err := svc.NewBranch(&views.Branch{View:"default",Projected:tc.input})
   if !errors.Is(err,scalar.ErrRejected) || result != nil {
    t.Fatalf("nested failure lost or partial result published: value=%#v error=%v",result,err)
   }
   if supplied.Text != "ok-supplied" {
    t.Fatal("conversion mutated supplied input")
   }
  })
 }
 source := &views.BranchView{Head:&views.NodeView{},Tag:pointer("kept")}
 result, err := svc.NewBranch(&views.Branch{View:"tiny",Projected:source})
 if err != nil || result == nil || result.Head != nil || result.Tag == nil || *result.Tag != "kept" {
  t.Fatalf("excluded recursive defaults leaked: value=%#v error=%v",result,err)
 }
 result, err = svc.NewBranch(&views.Branch{View:"default",Projected:&views.BranchView{}})
 if err != nil || result == nil {
  t.Fatalf("absent optional recursive carriers must not parse: value=%#v error=%v",result,err)
 }
}
`
