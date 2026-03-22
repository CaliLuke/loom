package expr_test

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/v3/expr"
	"github.com/CaliLuke/loom/v3/expr/testdata"
)

func TestDuplicateUserTypeNames(t *testing.T) {
	err := expr.RunInvalidDSL(t, testdata.DuplicateUserTypeNamesDSL)
	if err == nil {
		t.Fatal("expected error, got none")
	}
	if !strings.Contains(err.Error(), `type "P" defined twice`) {
		t.Errorf("unexpected error:\n%s", err)
	}
}
