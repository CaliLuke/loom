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
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestCustomRepresentationExternalConversion exercises the public conversion
// methods at their actual error and receiver-publication boundaries.
func TestCustomRepresentationExternalConversion(t *testing.T) {
	const module = "example.com/customrepresentation"
	root := codegen.RunDSL(t, func() {
		record := dsl.Type("Record", func() {
			dsl.Attribute("String", dsl.String, func() {
				dsl.Meta("struct:field:type", "scalar.Token", module+"/scalar")
			})
			dsl.Required("String")
			dsl.ConvertTo(testdata.StringT{})
			dsl.CreateFrom(testdata.StringT{})
		})
		dsl.Service("conversions", func() {
			dsl.Method("show", func() {
				dsl.Payload(record)
			})
		})
	})
	services := NewServicesData(root)
	dir := t.TempDir()
	for _, svc := range root.Services {
		files := Files(module+"/gen", svc, services, make(map[string][]string))
		conversions, err := ConvertFiles(root, svc, services)
		require.NoError(t, err)
		files = append(files, conversions...)
		for _, file := range files {
			if header := file.HeaderTemplate(); header != nil {
				AddServiceDataMetaTypeImports(header, services.Get(svc.Name))
			}
			_, err := file.Render(dir)
			require.NoError(t, err)
		}
	}
	mod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0o700))
	for name, source := range map[string]string{
		"scalar/scalar.go": customRepresentationScalarSource,
		"convert_test.go":  customRepresentationConvertHarness,
	} {
		formatted, err := format.Source([]byte(source))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), formatted, 0o600))
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"test", "-race", "-count=1", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

const customRepresentationConvertHarness = `package customrepresentation_test

import (
 "errors"
 "testing"

 svc "example.com/customrepresentation/gen/conversions"
 "example.com/customrepresentation/scalar"
 "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestConvertMethods(t *testing.T) {
 record := &svc.Record{String: scalar.Token{Text: "ok-value"}}
 converted, err := record.ConvertToStringT()
 if err != nil || converted == nil || converted.String != "ok-value" {
  t.Fatalf("conversion: %#v %v", converted, err)
 }
 for _, value := range []string{"ok-next", "invalid"} {
  before := *record
  err := record.CreateFromStringT(&testdata.StringT{String: value})
  if value == "invalid" {
   if !errors.Is(err, scalar.ErrRejected) || *record != before {
    t.Fatalf("failed conversion changed receiver or lost cause: %#v %v", record, err)
   }
  } else if err != nil || record.String.Text != value {
   t.Fatalf("creation: %#v %v", record, err)
  }
 }
}
`
