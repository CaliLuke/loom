package service

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/expr"
)

func TestInterceptorContractMethodStorage(t *testing.T) {
	for _, mode := range []string{"normal", "required", "renamed"} {
		t.Run(mode, func(t *testing.T) {
			root := runDSL(t, testdata.InterceptorContractStorageDSL(mode))
			services := NewServicesData(root)
			var getter, setter string
			for _, file := range InterceptorsFiles("example.com/interceptorcontracts/gen", root.Services[0], services) {
				var source bytes.Buffer
				for _, section := range file.AllSections() {
					require.NoError(t, section.Write(&source))
				}
				fset := token.NewFileSet()
				parsed, err := parser.ParseFile(fset, file.Path, source.Bytes(), 0)
				require.NoError(t, err)
				for _, decl := range parsed.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Recv == nil {
						continue
					}
					var receiver, body bytes.Buffer
					require.NoError(t, format.Node(&receiver, fset, fn.Recv.List[0].Type))
					if receiver.String() != "*auditSecondPayload" {
						continue
					}
					require.NoError(t, format.Node(&body, fset, fn.Body))
					if fn.Name.Name == "Value" {
						getter = body.String()
					}
					if fn.Name.Name == "SetValue" {
						setter = body.String()
					}
				}
			}
			require.NotEmpty(t, getter)
			require.NotEmpty(t, setter)
			switch mode {
			case "normal":
				require.Contains(t, getter, "return *p.payload.Value")
				require.Contains(t, setter, "p.payload.Value = &v")
			case "required":
				require.Contains(t, getter, "return p.payload.Value")
				require.Contains(t, setter, "p.payload.Value = v")
			case "renamed":
				require.Contains(t, getter, "return *p.payload.StoredValue")
				require.Contains(t, setter, "p.payload.StoredValue = &v")
			}
		})
	}
}

func TestInterceptorContractRejectsIncompatibleSignature(t *testing.T) {
	for _, mode := range []string{"incompatible", "incompatible-disjoint", "nominal-incompatible", "nominal-incompatible-disjoint", "nullable-incompatible", "nullable-incompatible-disjoint"} {
		t.Run(mode, func(t *testing.T) {
			root := runDSL(t, testdata.InterceptorContractStorageDSL(mode))
			services := NewServicesData(root)
			var failure any
			func() {
				defer func() {
					failure = recover()
				}()
				services.Get("contracts")
			}()
			require.NotNil(t, failure, "incompatible signatures must fail analysis before files are emitted")
			message := fmt.Sprint(failure)
			for _, detail := range []string{"Audit", "first", "second", "value", "incompatible signatures"} {
				require.Contains(t, message, detail)
			}
			failureError, ok := failure.(error)
			require.True(t, ok)
			var attributed *codegen.Error
			require.ErrorAs(t, failureError, &attributed)
			require.Equal(t, "contracts", attributed.Service)
			require.Equal(t, "second", attributed.Method)
			require.Same(t, root.Services[0].Methods[1], attributed.Expr)
		})
	}
}

func TestInterceptorContractStreamingBindings(t *testing.T) {
	root := runDSL(t, testdata.InterceptorAccessorCoverageDSL([]string{"a", "b"}, []string{"b", "c"}, true))
	method := root.Services[0].Methods[1]
	for _, parent := range []*expr.AttributeExpr{method.Payload, method.StreamingPayload, method.EffectiveStreamingResult()} {
		parent.Validation.Required = nil
		parent.Find("value").Meta = expr.MetaExpr{"struct:field:name": {"StoredValue"}}
	}
	services := NewServicesData(root)
	var rendered string
	for _, file := range InterceptorsFiles("example.com/coverage/gen", root.Services[0], services) {
		var source bytes.Buffer
		for _, section := range file.AllSections() {
			require.NoError(t, section.Write(&source))
		}
		rendered += source.String()
	}
	for _, receiver := range []string{"p.payload", "r.result"} {
		require.Contains(t, rendered, "return *"+receiver+".StoredValue")
		require.Contains(t, rendered, receiver+".StoredValue = &v")
	}
	// Each independently declared payload/message accessor must use its method's
	// physical selector while the shared interface retains the logical name.
	for _, name := range []string{"auditBPayload", "auditBStreamingPayload", "auditBStreamingResult"} {
		require.Contains(t, rendered, "*"+name+") Value() string")
		require.Contains(t, rendered, "*"+name+") SetValue(v string)")
	}
}
