package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// nullableResponseProjection separates an outer presence carrier from its
// concrete field storage. Body selection needs a concrete parent; a whole-body
// null stays null and cannot supply or reconstruct transport metadata.
type nullableResponseProjection struct {
	selected     bool
	concreteBody bool
	metadataOnly bool
	statusBound  bool
	metadata     bool
	source       string
	required     bool
	concreteName string
}

func newNullableResponseProjection(attribute, body *expr.AttributeExpr, headers []*HeaderData, cookies []*CookieData, context *codegen.AttributeContext, viewed, statusBound bool) *nullableResponseProjection {
	selected := selectBodyField(attribute, body).Name
	concreteBody := nullableResponseRequiresConcrete(attribute, body)
	if !expr.IsNullable(attribute) || !expr.IsObject(attribute.Type) || (!concreteBody && len(headers)+len(cookies) == 0 && !statusBound) {
		return nil
	}
	plan := &nullableResponseProjection{
		selected:     selected != "",
		concreteBody: concreteBody,
		metadataOnly: body == nil || body.Type == expr.Empty,
		statusBound:  statusBound,
		metadata:     len(headers)+len(cookies) > 0 || statusBound,
		source:       responseNativeSource(viewed),
		concreteName: context.Scope.Name(representation.ConcreteAttribute(attribute), context.DefaultPkg, context),
	}
	for _, header := range headers {
		plan.required = plan.required || header.Required
	}
	for _, cookie := range cookies {
		plan.required = plan.required || cookie.Required
	}
	return plan
}

func (p *nullableResponseProjection) prepareConstructor(init *InitData, headers []*HeaderData, cookies []*CookieData) {
	if p == nil || init == nil || (p.concreteBody && !p.selected && len(headers)+len(cookies) == 0) {
		return
	}
	if !init.SkipFieldInit {
		init.nullableProjection = p
	}
	init.ClientErrorAware = true
	if p.selected || p.metadataOnly {
		if !init.SkipFieldInit {
			init.ReturnTypeName = p.concreteName
			init.ReturnIsStruct = true
		}
		return
	}
	// Presence belongs to the received wire bindings, not to decoded values:
	// defaults must not turn an explicit null body into a concrete parent.
	presence := make([]string, 0, len(headers)+len(cookies)+1)
	if p.statusBound {
		presence = append(presence, "true")
	}
	for _, header := range headers {
		presence = append(presence, fmt.Sprintf("len(resp.Header.Values(%q)) > 0", header.CanonicalName))
	}
	for _, cookie := range cookies {
		presence = append(presence, cookie.Locals.Present)
	}
	init.ClientArgs = append(init.ClientArgs, &InitArgData{
		Ref:           strings.Join(presence, " || "),
		AttributeData: &AttributeData{VarName: "responseMetadataPresent", TypeRef: "bool", Type: expr.Boolean},
	})
}

// SourcePreparation binds the concrete native response value and checks its
// presence before any body, header, cookie or status publication. onError owns
// the caller's return shape; nil emits a direct error return. bodyCondition selects
// the declared body, or is empty for unconditional selection. Replacement bodies
// never bypass native status, header or cookie prerequisites. Consumers run this
// after binding the original res carrier; status-bound values are already prepared.
func (plan *ResponseEncodePlan) SourcePreparation(onError func(string) string, bodyCondition string) string {
	if onError == nil {
		onError = func(value string) string {
			return "return " + value
		}
	}
	p := plan.sourceProjection
	if p == nil {
		return plan.selectedPreparation(onError, bodyCondition)
	}
	if p.statusBound {
		// Status selection captured this native value before choosing a branch.
		return plan.selectedPreparation(onError, bodyCondition)
	}
	return p.sourcePreparation(p.source, onError, bodyCondition) + plan.selectedPreparation(onError, bodyCondition)
}

// sourcePreparation captures one actual native value for status selection and
// subsequent body/metadata projection. Branch selection never reacquires it.
func (p *nullableResponseProjection) sourcePreparation(source string, onError func(string) string, bodyCondition string) string {
	prepare := "responseValue, responseHasValue := " + source + ".Value()\nif !" + source + ".Present() {\n" + onError("loom.ErrAbsentNullable") + "\n}\n"
	if p.concreteBody || p.metadataOnly || p.required || p.statusBound {
		condition := "!responseHasValue"
		if bodyCondition != "" && p.concreteBody && !p.metadata && !p.metadataOnly && !p.required {
			condition = "(" + bodyCondition + ") && (" + condition + ")"
		}
		prepare += "if " + condition + " {\n" + onError(`fmt.Errorf("null response cannot supply selected body or required metadata")`) + "\n}\n"
	}
	return prepare
}

func (p *nullableResponseProjection) encoderFields() (fieldSource, guard string) {
	if p == nil {
		return "", ""
	}
	if !p.concreteBody && !p.metadataOnly && !p.required && !p.statusBound {
		guard = "responseHasValue"
	}
	return "responseValue", guard
}

func (p *nullableResponseProjection) appendReconstruction(group *jen.Group, fields string) {
	group.List(jen.Id("responseValue"), jen.Id("responseHasValue")).Op(":=").Id("v").Dot("Value").Call()
	group.If(jen.Op("!").Id("v").Dot("Present").Call()).Block(
		jen.Op("*").Id("transformErr").Op("=").Id("loom").Dot("ErrAbsentNullable"),
	)
	group.If(jen.Op("!").Id("responseHasValue").Op("&&").Id("responseMetadataPresent")).Block(
		jen.Op("*").Id("transformErr").Op("=").Qual("fmt", "Errorf").Call(jen.Lit("null response body cannot carry transport metadata")),
	)
	group.If(jen.Id("responseHasValue")).BlockFunc(func(concrete *jen.Group) {
		appendHTTPRawBlock(concrete, fields)
		concrete.Id("v").Dot("SetValue").Call(jen.Id("responseValue"))
	})
}

// nullableResponseRequiresConcrete derives presence from the emitted body
// contract. A selected member or nonnullable object projection cannot be built
// from a null native parent; a nullable whole body retains its own presence.
func nullableResponseRequiresConcrete(attribute, body *expr.AttributeExpr) bool {
	if !expr.IsNullable(attribute) || !expr.IsObject(attribute.Type) || body == nil || body.Type == expr.Empty {
		return false
	}
	return selectBodyField(attribute, body).Name != "" || !expr.IsNullable(body)
}
