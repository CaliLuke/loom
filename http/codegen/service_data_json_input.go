package codegen

import (
	"fmt"
	"sort"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

// jsonBodyInput owns a detached decoding representation and its declaration
// closure. Its physical role does not depend on client/server package placement.
type jsonBodyInput struct {
	init      *InitData
	argument  *InitArgData
	attribute *expr.AttributeExpr
	context   *codegen.AttributeContext
	root      *TypeData
	types     []*TypeData
	unions    []*uniongen.Type
	helpers   []*codegen.TransformFunctionData
}

// cliBodyInputArgument retains the public --body or --p flag spelling while
// giving both structured and bare collection input the same decoding plan.
func cliBodyInputArgument(ctx *codegen.Context, endpoint *EndpointData) (*InitData, *InitArgData) {
	init := endpoint.Payload.Request.PayloadInit
	if init != nil {
		for _, arg := range init.ClientArgs {
			if arg.VarName == "body" && cli.IsJSONFlagType(arg.TypeName) {
				return init, arg
			}
		}
	}
	if !cli.IsJSONFlagType(endpoint.Method.PayloadRef) || init != nil && len(init.ClientArgs) != 0 {
		return nil, nil
	}
	arg := &InitArgData{Ref: "p", AttributeData: &AttributeData{
		Name: "p", VarName: "p", Required: true,
		TypeName: endpoint.Method.PayloadRef, TypeRef: endpoint.Method.PayloadRef,
		Type:         endpoint.Payload.Request.PayloadType,
		Description:  endpoint.Method.PayloadDesc,
		DefaultValue: endpoint.Method.PayloadDefault,
		Example: (cliExampleProjector{
			context: ctx, attribute: endpoint.valueTransport.Request.Body,
			service: endpoint.ServiceName, method: endpoint.Method.Name,
		}).retainedJSON(endpoint.Payload.Request.ClientBody.Value),
	}}
	return &InitData{ClientArgs: []*InitArgData{arg}, ReturnTypeRef: endpoint.Payload.Ref}, arg
}

func newJSONBodyInput(attribute *expr.AttributeExpr, endpoint string, sd *ServiceData, closed bool, names map[bodyTypeKey]string, allocated map[string]*uniongen.Type) *jsonBodyInput {
	attribute = expr.DupAtt(attribute)
	// Names are allocated after all wire types. Renaming the detached graph
	// preserves recursive edges while keeping encoder declarations untouched.
	walkBodyTypes(attribute.Type, func(typ expr.DataType) {
		key := bodyTypeIdentity(typ)
		if key == (bodyTypeKey{}) {
			return
		}
		name, exists := names[key]
		if !exists {
			name = sd.Scope.PeekUnique("CLI" + codegen.Goify(typ.Name(), true))
			names[key] = name
		}
		switch actual := typ.(type) {
		case expr.UserType:
			delete(actual.Attribute().Meta, "http:media:form-root")
			actual.Rename(name)
		case *expr.Union:
			actual.TypeName = name
		}
		sd.Scope.GoTypeName(&expr.AttributeExpr{Type: typ})
	})
	ctx := bodyCodecContext(sd.Scope, true)
	ctx.JSONPresence = true
	ctx.CollectionElementPresence = true
	layout := representation.NewLayout()
	layout.RecordAll(attribute, true, true, false)
	layout.Bind(ctx)
	input := &jsonBodyInput{attribute: attribute, context: ctx}
	input.root = jsonInputTypeData(attribute, endpoint, sd, ctx)
	branches := make(map[string]struct{})
	collectUnionBranchUserTypes(attribute, branches)
	seen := map[string]bool{input.root.VarName: true}
	representation.WalkUserTypes(attribute.Type, func(typ expr.UserType) {
		att := &expr.AttributeExpr{Type: typ}
		ref := sd.Scope.GoValueTypeName(att)
		if seen[ref] {
			return
		}
		seen[ref] = true
		data := jsonInputTypeData(att, endpoint, sd, ctx)
		completeUnionBranchTypeData(data, typ, branches)
		input.types = append(input.types, data)
	})
	unions := make(map[string]*uniongen.Type)
	collectHTTPUnionTypes(attribute, sd.Scope, unions, make(map[string]struct{}), closed, allocated)
	for _, union := range unions {
		input.unions = append(input.unions, union)
	}
	sort.Slice(input.unions, func(i, j int) bool {
		return input.unions[i].Name < input.unions[j].Name
	})
	return input
}

func jsonInputTypeData(attribute *expr.AttributeExpr, endpoint string, sd *ServiceData, ctx *codegen.AttributeContext) *TypeData {
	details := buildRequestBodyTypeDetails(attribute, endpoint, false, true, true, sd, ctx)
	valueRef := details.valueRef
	if valueRef == "" {
		valueRef = details.varName
	}
	return &TypeData{
		Name:               attribute.Type.Name(),
		VarName:            details.varName,
		Description:        details.description,
		Def:                details.definition,
		Ref:                goBodyTypeRef(sd.Scope, attribute, ctx),
		ValueRef:           valueRef,
		ValidateDef:        details.validateDefinition,
		ValidateRef:        details.validateReference,
		validationFunction: codegen.NamedValidationFunction(attribute, ctx, "body"),
	}
}

func (sds *ServicesData) prepareCLIBodyInputs(sd *ServiceData, closed bool) {
	names := make(map[bodyTypeKey]string)
	allocated := make(map[string]*uniongen.Type)
	for _, endpoint := range sd.Endpoints {
		request := endpoint.Payload.Request
		if request.ClientBody == nil {
			continue
		}
		init, arg := cliBodyInputArgument(sds.Ctx, endpoint)
		if arg == nil {
			continue
		}
		source := endpoint.valueTransport.Request
		input := newJSONBodyInput(source.Body, endpoint.Method.Name, sd, closed, names, allocated)
		input.init, input.argument = init, arg
		target := source.Payload
		if source.BodyOrigin != "" {
			_, target = target.FindAttribute(source.BodyOrigin)
			target = serviceFieldTransformAttribute(source.Payload, source.BodyOrigin, target)
		}
		pkg := sd.Service.LocationPackageName(endpoint.Method.PayloadLoc)
		plan, err := unmarshal(input.attribute, target, arg.VarName, input.context, serviceContext(pkg, sd.Service.Scope))
		if err != nil {
			panic(codegen.NewError(sds.Ctx, source.Body, fmt.Errorf("build CLI JSON input conversion: %w", err)))
		}
		input.helpers = plan.Helpers
		init.ClientCode = plan.Code
		init.ClientErrorAware = plan.ErrorAware
		arg.Type = input.attribute.Type
		arg.TypeName = input.root.ValueRef
		arg.TypeRef = input.root.Ref
		arg.Validate = input.root.ValidateRef
		if arg.VarName != "body" {
			arg.Validate = codegen.ValidationCode(input.attribute, nil, input.context, true, expr.IsAlias(input.attribute.Type), false, arg.VarName)
		}
		request.cliInput = input
		sd.cliBodyInputs = append(sd.cliBodyInputs, input)
	}
}

func (r *RequestData) cliPayloadInit() *InitData {
	if r.cliInput != nil {
		return r.cliInput.init
	}
	return r.PayloadInit
}
