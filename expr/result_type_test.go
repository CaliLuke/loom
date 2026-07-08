package expr_test

import (
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestDuplicateResultTypeNames(t *testing.T) {
	err := expr.RunInvalidDSL(t, testdata.DuplicateResultTypeNamesDSL)
	if err == nil {
		t.Fatal("expected error, got none")
	}
	// Root validation prefixes with the expression EvalName ("design").
	if !strings.Contains(err.Error(), `result type "A" defined twice`) {
		t.Errorf("unexpected error:\n%s", err)
	}
}

func TestGeneratedResultTypesAreAppendedToRootOnce(t *testing.T) {
	root := expr.RunDSL(t, func() {
		ResultType("item", func() {
			Attribute("name", String)
		})
		Service("generated result types", func() {
			Method("list", func() {
				Result(CollectionOf("item"))
			})
		})
	})

	seen := map[string]int{}
	for _, rt := range root.ResultTypes {
		seen[rt.Identifier]++
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("generated result type %q appeared %d times, expected once", id, count)
		}
	}
}
