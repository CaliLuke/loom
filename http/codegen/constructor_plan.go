package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

type (
	// constructorPlan is the complete declaration contract, independent of its
	// allocated name and of expressions passed by individual callers.
	constructorPlan struct {
		parameters []constructorParameter
		result     string
		errorAware bool
		body       string
	}
	constructorParameter struct {
		name, typ string
	}
)

func newConstructorPlan(init *InitData, client, body bool, pkgs *codegen.NameScope) *constructorPlan {
	args, code := initRenderData(init, client)
	plan := &constructorPlan{result: init.ReturnTypeRef, errorAware: init.ServerErrorAware}
	if client {
		plan.errorAware = init.ClientErrorAware
	}
	for _, arg := range args {
		plan.parameters = append(plan.parameters, constructorParameter{arg.VarName, arg.TypeRef})
	}
	var fields codegen.TransformPlan
	if !body && !init.SkipFieldInit {
		fields = fieldCode(init, initRenderTarget(client), pkgs)
		plan.errorAware = plan.errorAware || fields.ErrorAware
	}
	if client {
		init.ClientErrorAware = plan.errorAware
	} else {
		init.ServerErrorAware = plan.errorAware
	}
	plan.body = fmt.Sprintf("%#v", jen.BlockFunc(func(group *jen.Group) {
		if plan.errorAware {
			group.Id("transformErr").Op(":=").New(jen.Error())
		}
		if body {
			if code != "" {
				appendHTTPRawBlock(group, code)
			}
			plan.returnValue(group, "body")
			return
		}
		fieldInitCode := ""
		if !init.SkipFieldInit {
			fieldInitCode = strings.TrimRight(fields.Code, "\n\t ")
		}
		appendInitResult(group, init, code, client)
		if fieldInitCode != "" {
			appendHTTPRawBlock(group, fieldInitCode)
		}
		if code != "" || fieldInitCode != "" {
			group.Line()
		}
		if init.ReturnTypeAttribute != "" {
			plan.returnValue(group, "res")
		} else {
			plan.returnValue(group, "v")
		}
	}))
	return plan
}

// Hash uses exactly the declaration rendered below. A signature alone cannot
// distinguish two conversions that read different fields of the same argument.
func (p *constructorPlan) Hash() string {
	return fmt.Sprintf("%#v", p.declaration("constructor"))
}

func (p *constructorPlan) declaration(name string) *jen.Statement {
	declaration := jen.Func().Id(name).ParamsFunc(func(group *jen.Group) {
		for _, arg := range p.parameters {
			group.Id(arg.name).Add(codegen.TypeRef(arg.typ))
		}
	})
	if p.errorAware {
		declaration.Params(codegen.TypeRef(p.result), jen.Error())
	} else {
		declaration.Add(codegen.TypeRef(p.result))
	}
	return declaration.Add(codegen.Expr(p.body))
}

func constructorSection(name string, init *InitData, client, body bool, pkgs *codegen.NameScope) codegen.Section {
	plan := init.constructor
	if plan == nil {
		// Direct builder tests can render an isolated constructor before service
		// finalization. They use the same lowering as the package owner.
		plan = newConstructorPlan(init, client, body, pkgs)
	}
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, init.Description)
		stmt.Add(plan.declaration(init.Name))
		stmt.Line()
	})
}

func (p *constructorPlan) returnValue(group *jen.Group, value string) {
	if !p.errorAware {
		group.Return(jen.Id(value))
		return
	}
	group.If(jen.Op("*").Id("transformErr").Op("!=").Nil()).Block(
		jen.Var().Id("zero").Add(codegen.TypeRef(p.result)),
		jen.Return(jen.Id("zero"), jen.Op("*").Id("transformErr")),
	)
	group.Return(jen.Id(value), jen.Nil())
}
