package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

const customizedResultValidationHarness = `package customizedviews_test

import (
	"testing"

	service "example.com/customizedviews/gen/customized_result_copies"
	views "example.com/customizedviews/gen/customized_result_copies/views"
)

func TestCustomResult(t *testing.T) {
	text := "value"
	for _, tc := range []struct {
		name, view string
		x, y       *string
		wantError  bool
	}{
		{"missing default", "default", nil, nil, true},
		{"present default", "default", &text, nil, false},
		{"missing tiny", "tiny", nil, nil, true},
		{"present tiny", "tiny", &text, nil, false},
		{"optional override", "optional", nil, nil, false},
		{"required override missing x", "required", nil, &text, true},
		{"required override missing y", "required", &text, nil, true},
		{"required override present", "required", &text, &text, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viewed := &views.MenuM1Result{View: tc.view, Projected: &views.MenuM1ResultView{X: tc.x, Y: tc.y}}
			err := views.ValidateMenuM1Result(viewed)
			var result *service.MenuM1Result
			if err == nil {
				result, err = service.NewMenuM1Result(viewed)
			}
			if (err != nil) != tc.wantError {
				t.Errorf("error = %v, wantError = %v", err, tc.wantError)
			}
			if tc.wantError && result != nil {
				t.Errorf("invalid result converted: %#v", result)
			}
			if !tc.wantError && tc.x != nil && (result == nil || result.X != text) {
				t.Errorf("decoded result = %#v", result)
			}
		})
	}
	for _, tc := range []struct {
		view      string
		wantError bool
	}{
		{"default", false},
		{"tiny", false},
		{"optional", false},
		{"required", true},
	} {
		original := &views.Menu{View: tc.view, Projected: &views.MenuView{}}
		if err := views.ValidateMenu(original); (err != nil) != tc.wantError {
			t.Errorf("original %s: error=%v, wantError=%v", tc.view, err, tc.wantError)
		}
	}
	if result, err := service.NewMenu(&views.Menu{View: "default", Projected: &views.MenuView{}}); err != nil || result == nil {
		t.Errorf("original conversion: result=%#v err=%v", result, err)
	}
	if err := views.ValidateMenuM2Result(&views.MenuM2Result{View: "tiny", Projected: &views.MenuM2ResultView{}}); err == nil {
		t.Error("fixed view accepted missing x")
	}
}
`

// TestCustomizedResultValidationGeneratedModule exercises the generated
// validation and conversion boundary of customized result types and views.
func TestCustomizedResultValidationGeneratedModule(t *testing.T) {
	const module = "example.com/customizedviews"
	root := codegen.RunDSL(t, testdata.CustomizedResultCopiesDSL)
	services := NewServicesData(root)
	dir := t.TempDir()
	for _, service := range root.Services {
		files := Files(module+"/gen", service, services, make(map[string][]string))
		if views := ViewsFile(module+"/gen", service, services); views != nil {
			files = append(files, views)
		}
		for _, file := range files {
			_, err := file.Render(dir)
			require.NoError(t, err)
		}
	}
	mod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "validation_test.go"), []byte(customizedResultValidationHarness), 0600))
	for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}, {"test", "-race", "-count=1", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}
