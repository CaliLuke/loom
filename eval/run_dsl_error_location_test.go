package eval_test

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/eval"
)

func TestRunDSL_ReportErrorLocation(t *testing.T) {
	eval.SetupTestContext(t)

	var expectedFile string
	var expectedLine int
	expr := &runDSLExpr{
		name: "expr",
		dsl: func() {
			_, file, line, ok := runtime.Caller(0)
			eval.ReportError("boom")
			if !ok {
				t.Fatal("runtime.Caller failed")
			}
			expectedFile = relativeToWorkdir(t, file)
			expectedLine = line + 1
		},
	}

	if err := eval.Register(&runDSLRoot{expr: expr}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err := eval.RunDSL()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var merr eval.MultiError
	if !errors.As(err, &merr) {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if len(merr) != 1 {
		t.Fatalf("expected 1 error, got %d", len(merr))
	}

	got := merr[0]
	if got.File != expectedFile {
		t.Fatalf("unexpected file: got %q, expected %q", got.File, expectedFile)
	}
	if got.Line != expectedLine {
		t.Fatalf("unexpected line: got %d, expected %d", got.Line, expectedLine)
	}
}

func TestRunDSL_ValidationErrorLocation(t *testing.T) {
	eval.SetupTestContext(t)

	dsl := func() {}
	expr := &runDSLExpr{
		name:     "expr",
		dsl:      dsl,
		validate: errors.New("bad"),
	}

	if err := eval.Register(&runDSLRoot{expr: expr}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err := eval.RunDSL()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var merr eval.MultiError
	if !errors.As(err, &merr) {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if len(merr) != 1 {
		t.Fatalf("expected 1 error, got %d", len(merr))
	}

	got := merr[0]
	if got.File != "" {
		t.Fatalf("unexpected file: got %q, expected empty (validation errors are embedded)", got.File)
	}
	if got.Line != 0 {
		t.Fatalf("unexpected line: got %d, expected 0 (validation errors are embedded)", got.Line)
	}

	expectedFile, expectedLine := dslDeclLocation(t, dsl)
	expected := "[" + expectedFile + ":" + itoa(expectedLine) + "] expr: bad"
	if got := err.Error(); got != expected {
		t.Fatalf("unexpected error message:\n got: %q\nwant: %q", got, expected)
	}
}

func TestRunDSLExecutesRootsRegisteredDuringExecution(t *testing.T) {
	eval.SetupTestContext(t)

	var generatedRan bool
	generated := &runDSLExpr{
		name: "generated-expr",
		dsl: func() {
			generatedRan = true
		},
	}
	initial := &runDSLExpr{
		name: "initial-expr",
		dsl: func() {
			if err := eval.Register(&runDSLRoot{name: "generated", expr: generated}); err != nil {
				t.Fatalf("Register generated root failed: %v", err)
			}
		},
	}

	if err := eval.Register(&runDSLRoot{name: "initial", expr: initial}); err != nil {
		t.Fatalf("Register initial root failed: %v", err)
	}

	if err := eval.RunDSL(); err != nil {
		t.Fatalf("RunDSL failed: %v", err)
	}
	if !generatedRan {
		t.Fatal("generated root DSL was not executed")
	}
}

func TestRunDSLRecoversPanicAsError(t *testing.T) {
	eval.SetupTestContext(t)

	expr := &runDSLExpr{
		name: "panicking-expr",
		dsl: func() {
			panic("boom")
		},
	}
	if err := eval.Register(&runDSLRoot{name: "panicking", expr: expr}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err := eval.RunDSL()
	if err == nil {
		t.Fatal("expected panic to be reported as an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "panic: boom in panicking-expr") {
		t.Fatalf("expected panic diagnostic, got %q", got)
	}
}

func TestRunDSLRecoversLifecyclePanicsAsErrors(t *testing.T) {
	var prepareValidateRan bool
	cases := []struct {
		name string
		expr *runDSLExpr
		want string
	}{
		{
			name: "prepare",
			expr: &runDSLExpr{
				name: "prepare-expr",
				dsl:  func() {},
				prepare: func() {
					panic("prepare boom")
				},
				validateFunc: func() error {
					prepareValidateRan = true
					return nil
				},
			},
			want: "panic: prepare boom in prepare-expr",
		},
		{
			name: "validate",
			expr: &runDSLExpr{
				name: "validate-expr",
				dsl:  func() {},
				validateFunc: func() error {
					panic("validate boom")
				},
			},
			want: "panic: validate boom in validate-expr",
		},
		{
			name: "finalize",
			expr: &runDSLExpr{
				name: "finalize-expr",
				dsl:  func() {},
				finalize: func() {
					panic("finalize boom")
				},
			},
			want: "panic: finalize boom in finalize-expr",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eval.SetupTestContext(t)
			if err := eval.Register(&runDSLRoot{name: c.name, expr: c.expr}); err != nil {
				t.Fatalf("Register failed: %v", err)
			}

			err := eval.RunDSL()

			if err == nil {
				t.Fatal("expected panic to be reported as an error, got nil")
			}
			if got := err.Error(); !strings.Contains(got, c.want) {
				t.Fatalf("expected panic diagnostic %q, got %q", c.want, got)
			}
		})
	}
	if prepareValidateRan {
		t.Fatal("validation ran after prepare panic")
	}
}

type runDSLRoot struct {
	name string
	expr eval.Expression
}

func (r *runDSLRoot) EvalName() string {
	if r.name == "" {
		return "test"
	}
	return r.name
}
func (*runDSLRoot) DependsOn() []eval.Root {
	return nil
}
func (*runDSLRoot) Packages() []string {
	return nil
}
func (r *runDSLRoot) WalkSets(walk eval.SetWalker) {
	walk(eval.ExpressionSet{r.expr})
}

type runDSLExpr struct {
	name         string
	dsl          func()
	prepare      func()
	validate     error
	validateFunc func() error
	finalize     func()
}

func (e *runDSLExpr) EvalName() string { return e.name }
func (e *runDSLExpr) DSL() func()      { return e.dsl }
func (e *runDSLExpr) Prepare() {
	if e.prepare != nil {
		e.prepare()
	}
}
func (e *runDSLExpr) Validate() error {
	if e.validateFunc != nil {
		return e.validateFunc()
	}
	return e.validate
}
func (e *runDSLExpr) Finalize() {
	if e.finalize != nil {
		e.finalize()
	}
}

func dslDeclLocation(t *testing.T, fn func()) (file string, line int) {
	t.Helper()

	pc := reflect.ValueOf(fn).Pointer()
	f := runtime.FuncForPC(pc)
	if f == nil {
		t.Fatal("runtime.FuncForPC returned nil")
	}
	file, line = f.FileLine(pc)
	return relativeToWorkdir(t, file), line
}
