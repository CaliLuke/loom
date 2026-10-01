package codegen

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type cliExampleProjector struct {
	context   *codegen.Context
	attribute *expr.AttributeExpr
	service   string
	method    string
}

// cliBodyDefault returns the flag input default in the client body's JSON shape.
func cliBodyDefault(request *transportir.Request, body *expr.AttributeExpr) any {
	if request != nil && request.BodyOrigin != "" && !request.MustHaveBody &&
		(expr.IsArray(body.Type) || expr.IsMap(body.Type)) && !codegen.IsExplicitPresenceType(body) {
		return expr.CanonicalizeExample(body, request.Payload.GetDefault(request.BodyOriginKey))
	}
	return optionalBodyDefault(request)
}

func (b *payloadBuilder) clientCLIExample(target *transportir.ValueTarget) any {
	if cli.IsJSONFlagType(b.sd.Scope.GoTypeNameWithDefaults(b.bodyAttr)) {
		return b.cliExampleProjector().retainedJSON(target)
	}
	return expr.CanonicalizeExample(b.bodyAttr, b.bodyAttr.Example(b.sds.examplesFor(b.sd)))
}

func (b *payloadBuilder) cliExampleProjector() cliExampleProjector {
	return cliExampleProjector{
		context:   b.sds.Ctx,
		attribute: b.bodyAttr,
		service:   b.svc.Name,
		method:    b.endpointIR.MethodName,
	}
}

func (p cliExampleProjector) retainedJSON(target *transportir.ValueTarget) any {
	if target == nil {
		panic(codegen.NewError(p.context, p.attribute, errors.New("build HTTP CLI body example: missing runtime value target")))
	}
	if target.Error != nil {
		panic(codegen.NewError(p.context, p.attribute, fmt.Errorf("build HTTP CLI body example plan: %w", target.Error)))
	}
	if target.Source == nil || target.Source.Context == nil {
		panic(codegen.NewError(p.context, p.attribute, errors.New("build HTTP CLI body example: missing retained source owner")))
	}
	source := target.Source.Example
	switch source.Outcome() {
	case expr.ValueInvalid:
		if !source.Synthesized() {
			panic(codegen.NewError(p.context, p.attribute, valueExampleError("invalid authored HTTP CLI body example", source.Diagnostics())))
		}
		p.warnOmitted("invalid synthesized source", source.Diagnostics())
		return nil
	case expr.ValueIncomplete, expr.ValueAmbiguous, expr.ValueUnsupported:
		p.warnOmitted("source is not usable", source.Diagnostics())
		return nil
	case expr.ValueSuppressed:
		return nil
	case expr.ValueResolved:
	default:
		panic(codegen.NewError(p.context, p.attribute, fmt.Errorf("build HTTP CLI body example: invalid retained source outcome %d", source.Outcome())))
	}
	projected := target.Source.Context.ProjectJSON(source, target.Plan)
	if err := projected.Err(); err != nil {
		panic(codegen.NewError(p.context, p.attribute, fmt.Errorf("build HTTP CLI body example: %w", err)))
	}
	switch projected.Outcome() {
	case expr.ProjectionEmitted:
		wire, ok := projected.JSON()
		if !ok {
			panic(codegen.NewError(p.context, p.attribute, errors.New("build HTTP CLI body example: emitted projection has no JSON")))
		}
		return jsontext.Value(wire)
	case expr.ProjectionSuppressed:
		return nil
	case expr.ProjectionInvalidPlan:
		panic(codegen.NewError(p.context, p.attribute, valueExampleError("build HTTP CLI body example from invalid runtime plan", projected.Diagnostics())))
	case expr.ProjectionIncomplete, expr.ProjectionUnsupported, expr.ProjectionUnrepresentable:
		p.warnOmitted("runtime projection is not usable", projected.Diagnostics())
		return nil
	default:
		panic(codegen.NewError(p.context, p.attribute, fmt.Errorf("build HTTP CLI body example: invalid projection outcome %d", projected.Outcome())))
	}
}

func (p cliExampleProjector) warnOmitted(reason string, diagnostics []expr.ValueDiagnostic) {
	p.context.Warn(
		"omitting unusable HTTP CLI body example",
		"service", p.service,
		"method", p.method,
		"reason", reason,
		"diagnostics", diagnostics,
	)
}

func valueExampleError(prefix string, diagnostics []expr.ValueDiagnostic) error {
	if len(diagnostics) == 0 {
		return errors.New(prefix)
	}
	parts := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		path := strings.Join(diagnostic.Path, ".")
		if path == "" {
			parts = append(parts, diagnostic.Message)
			continue
		}
		parts = append(parts, path+": "+diagnostic.Message)
	}
	return fmt.Errorf("%s: %s", prefix, strings.Join(parts, "; "))
}
