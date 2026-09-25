package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestConvertMappedNamesGeneratedModule generates the ConvertTo and CreateFrom
// functions of types whose attributes are declared with an element name
// suffix, such as "text:string", compiles them in a temporary module and
// round-trips values through them. The suffix does not select the external Go
// field: struct:field:external or the attribute name does.
func TestConvertMappedNamesGeneratedModule(t *testing.T) {
	const modulePath = "example.com/convertmapped"
	root := codegen.RunDSL(t, convertMappedNamesDSL)
	services := NewServicesData(root)
	dir := t.TempDir()
	for _, service := range root.Services {
		files := Files(modulePath+"/gen", service, services, make(map[string][]string))
		convert, err := ConvertFiles(root, service, services)
		require.NoError(t, err)
		for _, file := range append(files, convert...) {
			_, err := file.Render(dir)
			require.NoError(t, err, file.Path)
		}
	}
	goMod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", modulePath, testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gen", "mapped", "convert_round_trip_test.go"), []byte(convertMappedNamesHarness), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}, {"test", "-count=1", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

// convertMappedNamesDSL declares a required attribute mapped to an external
// field with struct:field:external, an optional attribute and a required
// attribute that match the external field by attribute name.
func convertMappedNamesDSL() {
	var External = dsl.Type("External", func() {
		dsl.ConvertTo(testdata.ExternalNameT{})
		dsl.CreateFrom(testdata.ExternalNameT{})
		dsl.Attribute("text:string", dsl.String, func() {
			dsl.Meta("struct:field:external", "String")
		})
		dsl.Required("text:string")
	})
	var Optional = dsl.Type("Optional", func() {
		dsl.ConvertTo(testdata.ExternalNamePointerT{})
		dsl.CreateFrom(testdata.ExternalNamePointerT{})
		dsl.Attribute("string:s", dsl.String)
	})
	var Named = dsl.Type("Named", func() {
		dsl.ConvertTo(testdata.StringT{})
		dsl.CreateFrom(testdata.StringT{})
		dsl.Attribute("string:s", dsl.String)
		dsl.Required("string:s")
	})
	dsl.Service("mapped", func() {
		dsl.Method("convert_external", func() {
			dsl.Payload(External)
		})
		dsl.Method("convert_optional", func() {
			dsl.Payload(Optional)
		})
		dsl.Method("convert_named", func() {
			dsl.Payload(Named)
		})
	})
}

const convertMappedNamesHarness = `package mapped

import (
	"reflect"
	"testing"

	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestRoundTrip(t *testing.T) {
	text := "text"
	external := &External{Text: text}
	if got := external.ConvertToExternalNameT(); !reflect.DeepEqual(got, &testdata.ExternalNameT{String: text}) {
		t.Errorf("ConvertToExternalNameT: %+v", got)
	}
	var created External
	created.CreateFromExternalNameT(&testdata.ExternalNameT{String: text})
	if !reflect.DeepEqual(&created, external) {
		t.Errorf("CreateFromExternalNameT: %+v", created)
	}

	for _, optional := range []*Optional{{String: &text}, {}} {
		converted := optional.ConvertToExternalNamePointerT()
		if !reflect.DeepEqual(converted, &testdata.ExternalNamePointerT{String: optional.String}) {
			t.Errorf("ConvertToExternalNamePointerT: %+v", converted)
		}
		var back Optional
		back.CreateFromExternalNamePointerT(converted)
		if !reflect.DeepEqual(&back, optional) {
			t.Errorf("CreateFromExternalNamePointerT: %+v, want %+v", back, optional)
		}
	}

	named := &Named{String: text}
	if got := named.ConvertToStringT(); !reflect.DeepEqual(got, &testdata.StringT{String: text}) {
		t.Errorf("ConvertToStringT: %+v", got)
	}
	var namedBack Named
	namedBack.CreateFromStringT(&testdata.StringT{String: text})
	if !reflect.DeepEqual(&namedBack, named) {
		t.Errorf("CreateFromStringT: %+v", namedBack)
	}
}
`
