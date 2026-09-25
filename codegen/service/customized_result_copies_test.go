package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
)

// TestCustomizedResultCopies checks that the renamed copies of a result type
// that customize its requiredness in different methods, and the result type
// itself, each get their own viewed result type and projected type, although
// the copies keep the identifier of the result type.
func TestCustomizedResultCopies(t *testing.T) {
	root := codegen.RunDSL(t, testdata.CustomizedResultCopiesDSL)
	services := NewServicesData(root)
	svc := services.Get("CustomizedResultCopies")
	require.NotNil(t, svc)

	cases := []struct {
		method    string
		result    string
		init      string
		projected string
		view      string
	}{
		{"M1", "MenuM1Result", "NewViewedMenuM1Result", "MenuM1ResultView", ""},
		{"M2", "MenuM2Result", "NewViewedMenuM2Result", "MenuM2ResultView", "tiny"},
		{"M3", "Menu", "NewViewedMenu", "MenuView", ""},
	}
	viewed := make(map[string]*ViewedResultTypeData, len(svc.viewedResultTypes))
	for _, vrt := range svc.viewedResultTypes {
		viewed[vrt.Name] = vrt
	}
	projected := make(map[string]*ProjectedTypeData, len(svc.projectedTypes))
	for _, pt := range svc.projectedTypes {
		projected[pt.Name] = pt
	}
	assert.Len(t, viewed, len(cases))
	assert.Len(t, projected, len(cases))
	for _, c := range cases {
		method := svc.Method(c.method)
		require.NotNil(t, method, c.method)
		require.NotNil(t, method.ViewedResult, c.method)
		assert.Equal(t, c.result, method.ViewedResult.Name, c.method)
		assert.Equal(t, c.init, method.ViewedResult.Init.Name, c.method)
		assert.Equal(t, c.view, method.ViewedResult.ViewName, c.method)
		assert.Same(t, method.ViewedResult, viewed[c.result], c.method)
		assert.NotNil(t, projected[c.projected], c.method)
	}

	code := renderServiceFile(t, root, services)
	for _, c := range cases {
		assert.Contains(t, code, "\nfunc "+c.init+"(", c.method)
	}
	views := renderViewsFile(t, root, services)
	for _, name := range []string{
		"type MenuM1Result struct",
		"type MenuM2Result struct",
		"type Menu struct",
		"type MenuM1ResultView struct",
		"type MenuM2ResultView struct",
		"type MenuView struct",
		"func ValidateMenuM1Result(",
		"func ValidateMenuM2Result(",
		"func ValidateMenu(",
		"func ValidateMenuM1ResultViewTiny(",
		"func ValidateMenuM2ResultViewTiny(",
		"func ValidateMenuViewTiny(",
	} {
		assert.Equal(t, 1, strings.Count(views, "\n"+name), name)
	}
}
