package expr_test

import (
	"strings"
	"testing"

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
