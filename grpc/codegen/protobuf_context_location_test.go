package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestProtoTransformPhysicalContexts(t *testing.T) {
	scope := codegen.NewNameScope()
	parent := func(path string) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Holder", AttributeExpr: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:pkg:path": {path}}}}}
	}
	source := codegen.NewAttributeContext(false, false, true, "service", scope)
	target := codegen.NewAttributeContext(true, true, false, "pb", scope)
	target.Scope = &protoBufScope{scope: scope}
	attrs := &transformAttrs{TransformAttrs: &codegen.TransformAttrs{SourceCtx: source, TargetCtx: target}, proto: true, errorAware: true}
	nested := transformTypeContexts(parent("common"), parent("messages"), attrs)
	require.NotSame(t, attrs.TransformAttrs, nested.TransformAttrs)
	require.Same(t, source, attrs.SourceCtx)
	require.Same(t, target, attrs.TargetCtx)
	require.Equal(t, "service", source.DefaultPkg)
	require.Equal(t, "pb", target.DefaultPkg)
	require.Equal(t, "common", nested.SourceCtx.DefaultPkg)
	require.Equal(t, "messages", nested.TargetCtx.DefaultPkg)
	require.IsType(t, &protoBufScope{}, nested.TargetCtx.Scope)
	require.IsType(t, &codegen.AttributeScope{}, nested.SourceCtx.Scope)
	require.True(t, nested.errorAware)
	require.True(t, nested.proto)
	require.Same(t, nested, transformTypeContexts(&expr.AttributeExpr{Type: expr.String}, &expr.AttributeExpr{Type: expr.String}, nested))
}
