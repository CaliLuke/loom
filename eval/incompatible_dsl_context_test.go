package eval_test

import (
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestIncompatibleDSLIncludesTypeContext(t *testing.T) {
	dsl := func() {
		Type("SomeType", func() {
			Attribute("attr")
			View("default", func() {
				Attribute("attr")
			})
		})
	}

	err := expr.RunInvalidDSL(t, dsl)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "in type \"SomeType\"") {
		t.Fatalf("unexpected error: %s", err.Error())
	}
}
