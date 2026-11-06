//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"bytes"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

func transformArray(source, target *expr.Array, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	if err := IsCompatible(source.ElemType.Type, target.ElemType.Type, sourceVar+"[0]", targetVar+"[0]"); err != nil {
		return nil, err
	}
	data := transformArrayRenderData{
		ElemTypeRef:    collectionTypeRef(target.ElemType, ta.TargetCtx, false),
		SourceElem:     source.ElemType,
		TargetElem:     target.ElemType,
		SourceVar:      sourceVar,
		TargetVar:      targetVar,
		NewVar:         newVar,
		TransformAttrs: ta,
		LoopVar:        string(rune(105 + strings.Count(targetVar, "["))),
		IsStruct:       expr.IsObject(target.ElemType.Type) && !expr.AllowsNull(target.ElemType),
		ElemHelper:     transformUsesHelper(source.ElemType, target.ElemType),
		SourcePresence: ta.SourceCtx.CollectionElementPresence && !expr.ArrayElementsAllowNull(source),
	}
	return renderTransformGoArray(data)
}

func collectionTypeRef(attribute *expr.AttributeExpr, context *AttributeContext, mapKey bool) string {
	pkg := ""
	targetPkg := context.Pkg(attribute)
	if context.SamePackageConversion {
		pkg = context.DefaultPkg
		targetPkg = ""
	}
	if pkg == "" && containsInlineObject(attribute) {
		return attributorCollectionElemTypeRef(attribute, context, targetPkg)
	}
	scope := context.Scope.Scope()
	if mapKey {
		return scope.mapKeyTypeDef(
			attribute,
			context.Pointer,
			context.UseDefault,
			pkg,
			targetPkg,
		)
	}
	return scope.collectionElemTypeDef(
		attribute,
		context.Pointer,
		context.UseDefault,
		pkg,
		targetPkg,
	)
}

// attributorCollectionElemTypeRef renders the type of a collection element
// that holds an inline object at any depth of arrays, maps and nullability.
// The inline struct comes from the context Attributor so that it matches the
// declaring type, which a transport may render differently from the shared
// scope. It returns the same rendering as NameScope.collectionElemTypeDef when
// the Attributor is AttributeScope.
func attributorCollectionElemTypeRef(attribute *expr.AttributeExpr, context *AttributeContext, targetPkg string) string {
	scope := context.Scope.Scope()
	fallback := func() string {
		return scope.collectionElemTypeDef(attribute, context.Pointer, context.UseDefault, "", targetPkg)
	}
	if metaType, _ := GetMetaType(attribute); metaType != "" {
		return fallback()
	}
	var value string
	switch actual := attribute.Type.(type) {
	case *expr.Object:
		name := context.Scope.Name(attribute, targetPkg, context.Pointer, context.UseDefault)
		if IsExplicitPresenceType(attribute) {
			// A presence type such as loom.Nullable[T] holds the struct value.
			return name
		}
		return "*" + name
	case *expr.Array:
		value = "[]" + attributorCollectionElemTypeRef(actual.ElemType, context, targetPkg)
	case *expr.Map:
		key := scope.mapKeyTypeDef(actual.KeyType, context.Pointer, context.UseDefault, "", targetPkg)
		value = "map[" + key + "]" + attributorCollectionElemTypeRef(actual.ElemType, context, targetPkg)
	default:
		return fallback()
	}
	switch {
	case expr.IsNullable(attribute):
		return "loom.Nullable[" + value + "]"
	case IsExplicitPresenceType(attribute):
		return fallback()
	default:
		return value
	}
}

// containsInlineObject reports whether att is an inline object or a
// collection whose elements hold one at any depth, without looking inside user
// types.
func containsInlineObject(att *expr.AttributeExpr) bool {
	switch actual := att.Type.(type) {
	case *expr.Object:
		return true
	case *expr.Array:
		return containsInlineObject(actual.ElemType)
	case *expr.Map:
		return containsInlineObject(actual.ElemType)
	default:
		return false
	}
}

// transformUsesHelper reports whether source converts to target through a
// generated helper function. collectHelpers generates helpers exactly for
// these pairs, and every call site uses this predicate, so that each call has
// a helper. Helpers need named types on both sides: a helper signature that
// names an inline object would not match the transport rendering of that
// object, so a pair with an inline object side is converted in place.
func transformUsesHelper(source, target *expr.AttributeExpr) bool {
	_, sourceUser := source.Type.(expr.UserType)
	_, targetUser := target.Type.(expr.UserType)
	return sourceUser && targetUser && expr.IsObject(source.Type) && expr.IsObject(target.Type)
}

func transformCollectionElem(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	if !expr.IsNullable(source) && !expr.IsNullable(target) && expr.IsUnion(source.Type) {
		targetPointer := false
		return transformUnion(source, target, sourceVar, targetVar, newVar, &targetPointer, ta)
	}
	return transformAttributeStmt(source, target, sourceVar, targetVar, newVar, ta)
}

// transformMap generates Go code to transform source map to target map.
func transformMap(source, target *expr.Map, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	if err := IsCompatible(source.KeyType.Type, target.KeyType.Type, sourceVar+"[key]", targetVar+"[key]"); err != nil {
		return nil, err
	}
	if err := IsCompatible(source.ElemType.Type, target.ElemType.Type, sourceVar+"[*]", targetVar+"[*]"); err != nil {
		return nil, err
	}
	data := transformMapRenderData{
		KeyTypeRef:     collectionTypeRef(target.KeyType, ta.TargetCtx, true),
		ElemTypeRef:    collectionTypeRef(target.ElemType, ta.TargetCtx, false),
		SourceKey:      source.KeyType,
		TargetKey:      target.KeyType,
		SourceElem:     source.ElemType,
		TargetElem:     target.ElemType,
		SourceVar:      sourceVar,
		TargetVar:      targetVar,
		NewVar:         newVar,
		TransformAttrs: ta,
		IsKeyStruct:    expr.IsObject(target.KeyType.Type) && !expr.AllowsNull(target.KeyType),
		IsElemStruct:   expr.IsObject(target.ElemType.Type) && !expr.AllowsNull(target.ElemType),
		ElemHelper:     transformUsesHelper(source.ElemType, target.ElemType),
		SourcePresence: ta.SourceCtx.CollectionElementPresence && !expr.MapValuesAllowNull(source),
	}
	if depth := MapDepth(target); depth > 0 {
		data.LoopVar = string(rune(97 + depth))
	}
	return renderTransformGoMap(data)
}

func renderTransformGoArray(data transformArrayRenderData) (*jen.Statement, error) {
	typeName := "[]" + data.ElemTypeRef
	if data.TypeAliasName != "" {
		typeName = data.TypeAliasName
	}
	var elemCode *jen.Statement
	sourceElement := "val"
	if data.SourcePresence {
		sourceElement = "actual"
	}
	if !data.IsStruct || !data.ElemHelper {
		var err error
		elemCode, err = transformCollectionElem(data.SourceElem, data.TargetElem, sourceElement, data.TargetVar+"["+data.LoopVar+"]", false, data.TransformAttrs)
		if err != nil {
			return nil, err
		}
	}
	stmt := &jen.Statement{}
	stmt.For(
		jen.List(Expr(data.LoopVar), jen.Id("val")).Op(":=").Range().Add(Expr(data.SourceVar)),
	).BlockFunc(func(group *jen.Group) {
		if data.SourcePresence {
			group.List(jen.Id("actual"), jen.Id("ok")).Op(":=").Id("val").Dot("Value").Call()
			group.If(jen.Op("!").Id("ok")).Block(jen.Continue())
		}
		if data.IsStruct {
			group.If(Expr(sourceElement + " == nil")).BlockFunc(func(ifGroup *jen.Group) {
				ifGroup.Add(Expr(data.TargetVar)).Index(Expr(data.LoopVar)).Op("=").Nil()
				ifGroup.Continue()
			})
		}
		if data.IsStruct && data.ElemHelper {
			group.Add(Expr(data.TargetVar)).
				Index(Expr(data.LoopVar)).
				Op("=").
				Id(transformHelperName(data.SourceElem, data.TargetElem, data.TransformAttrs)).
				Call(transformCallArguments(data.SourceElem, data.TargetElem, sourceElement, data.TransformAttrs)...)
			return
		}
		group.Add(elemCode)
	})
	return renderCollectionConversion(data.SourceVar, data.TargetVar, typeName, data.NewVar, stmt), nil
}

func renderTransformGoMap(data transformMapRenderData) (*jen.Statement, error) {
	typeName := "map[" + data.KeyTypeRef + "]" + data.ElemTypeRef
	if data.TypeAliasName != "" {
		typeName = data.TypeAliasName
	}
	var keyCode *jen.Statement
	if !data.IsKeyStruct {
		var err error
		keyCode, err = transformAttributeStmt(data.SourceKey, data.TargetKey, "key", "tk", true, data.TransformAttrs)
		if err != nil {
			return nil, err
		}
	}
	var elemCode *jen.Statement
	sourceElement := "val"
	if data.SourcePresence {
		sourceElement = "actual"
	}
	if !data.IsElemStruct || !data.ElemHelper {
		var err error
		temp := "tv" + data.LoopVar
		elemCode, err = transformCollectionElem(data.SourceElem, data.TargetElem, sourceElement, temp, true, data.TransformAttrs)
		if err != nil {
			return nil, err
		}
	}
	stmt := &jen.Statement{}
	stmt.For(
		jen.List(jen.Id("key"), jen.Id("val")).Op(":=").Range().Add(Expr(data.SourceVar)),
	).BlockFunc(func(group *jen.Group) {
		if data.SourcePresence {
			group.List(jen.Id("actual"), jen.Id("ok")).Op(":=").Id("val").Dot("Value").Call()
			group.If(jen.Op("!").Id("ok")).Block(jen.Continue())
		}
		if data.IsKeyStruct {
			group.Id("tk").Op(":=").Id(transformHelperName(data.SourceKey, data.TargetKey, data.TransformAttrs)).Call(transformCallArguments(data.SourceKey, data.TargetKey, "key", data.TransformAttrs)...)
		} else {
			group.Add(keyCode)
		}
		if data.IsElemStruct {
			group.If(Expr(sourceElement + " == nil")).BlockFunc(func(ifGroup *jen.Group) {
				ifGroup.Add(Expr(data.TargetVar)).Index(Expr("tk")).Op("=").Nil()
				ifGroup.Continue()
			})
		}
		if data.IsElemStruct && data.ElemHelper {
			group.Add(Expr(data.TargetVar)).
				Index(Expr("tk")).
				Op("=").
				Id(transformHelperName(data.SourceElem, data.TargetElem, data.TransformAttrs)).
				Call(transformCallArguments(data.SourceElem, data.TargetElem, sourceElement, data.TransformAttrs)...)
			return
		}
		temp := "tv" + data.LoopVar
		group.Add(elemCode)
		group.Add(Expr(data.TargetVar)).Index(Expr("tk")).Op("=").Add(Expr(temp))
	})
	return renderCollectionConversion(data.SourceVar, data.TargetVar, typeName, data.NewVar, stmt), nil
}

// renderCollectionConversion preserves native absence independently of the
// containing field or constructor. Existing destinations must also lose stale
// contents when the source is nil; supplied empty values still allocate storage.
func renderCollectionConversion(sourceVar, targetVar, typeName string, newVar bool, contents *jen.Statement) *jen.Statement {
	stmt := &jen.Statement{}
	if newVar {
		stmt.Var().Add(Expr(targetVar)).Add(TypeRef(typeName)).Line()
	}
	stmt.If(Expr(sourceVar).Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
		group.Add(Expr(targetVar)).Op("=").Make(TypeRef(typeName), jen.Len(Expr(sourceVar)))
		group.Add(contents)
	})
	if !newVar {
		stmt.Else().Block(Expr(targetVar).Op("=").Nil())
	}
	return stmt
}

func renderJenniferSnippet(stmt *jen.Statement) string {
	var buf bytes.Buffer
	_ = stmt.Render(&buf)
	rendered := buf.String()
	if rendered != "" && !strings.HasSuffix(rendered, "\n") {
		rendered += "\n"
	}
	return rendered
}

// transformAttributeHelpers returns the Go transform functions and their definitions
// that may be used in code produced by Transform. It returns an error if source and
// target are incompatible (different types, fields of different type etc).
// transformAttributeHelpers recurses through the attribute types and calls
// collectHelpers for each child attribute. collectHelpers actually produces the
// transform helper functions for the given attribute.
//
// source, target are the source and target attributes used in transformation
//
// ta holds the transform attributes
//
// seen keeps track of generated transform functions to avoid infinite recursion.
func transformAttributeHelpers(source, target *expr.AttributeExpr, ta *TransformAttrs, seen map[string]*TransformFunctionData) (helpers []*TransformFunctionData, err error) {
	if expr.IsNullable(source) || expr.IsNullable(target) {
		concreteSource := concretePresenceAttribute(source)
		concreteTarget := concretePresenceAttribute(target)
		if nullablePhysicalTypeRef(source, ta.SourceCtx) != nullablePhysicalTypeRef(target, ta.TargetCtx) &&
			transformUsesHelper(concreteSource, concreteTarget) {
			helper, helperErr := generateHelper(concreteSource, concreteTarget, true, ta, seen)
			if helperErr != nil {
				return nil, helperErr
			}
			if helper != nil {
				helpers = append(helpers, helper)
			}
		}
	}
	// Non-nullable top-level user types are transformed inline and do not need
	// their own helper. Nullable named object roots use a helper inside the
	// presence transform and are collected above.
	nested, err := appendNestedHelpers(source, target, true, ta, seen)
	return append(helpers, nested...), err
}

// collectHelpers recurses through the given attributes and returns the transform
// helper functions required to generate the transform code. If the attributes type
// is array or map then the recursion is done via transformAttributeHelpers so that
// the tope level conversion function is skipped as the generate code does not make
// use of it (since it inlines that top-level transformation).
func collectHelpers(source, target *expr.AttributeExpr, req bool, ta *TransformAttrs, seen map[string]*TransformFunctionData) (helpers []*TransformFunctionData, err error) {
	if expr.IsNullable(source) || expr.IsNullable(target) {
		source = concretePresenceAttribute(source)
		target = concretePresenceAttribute(target)
	}
	name := transformHelperName(source, target, ta)
	if _, ok := seen[name]; ok {
		return helpers, err
	}
	if transformUsesHelper(source, target) {
		var h *TransformFunctionData
		h, err = generateHelper(source, target, req, ta, seen)
		if err != nil {
			return helpers, err
		}
		if h != nil {
			helpers = append(helpers, h)
		}
	}
	var other []*TransformFunctionData
	if other, err = appendNestedHelpers(source, target, req, ta, seen); err != nil {
		return helpers, err
	}
	helpers = append(helpers, other...)
	return helpers, err
}

func appendNestedHelpers(source, target *expr.AttributeExpr, req bool, ta *TransformAttrs, seen map[string]*TransformFunctionData) (helpers []*TransformFunctionData, err error) {
	if matchingExplicitType(source, target) {
		return helpers, nil
	}
	var other []*TransformFunctionData
	switch {
	case expr.IsArray(source.Type):
		if other, err = collectHelpers(expr.AsArray(source.Type).ElemType, expr.AsArray(target.Type).ElemType, req, ta, seen); err == nil {
			helpers = append(helpers, other...)
		}
	case expr.IsMap(source.Type):
		sm, tm := expr.AsMap(source.Type), expr.AsMap(target.Type)
		if other, err = collectHelpers(sm.ElemType, tm.ElemType, req, ta, seen); err == nil {
			helpers = append(helpers, other...)
			if other, err = collectHelpers(sm.KeyType, tm.KeyType, req, ta, seen); err == nil {
				helpers = append(helpers, other...)
			}
		}
	case expr.IsUnion(source.Type):
		tt := expr.AsUnion(target.Type)
		if tt == nil {
			return helpers, err
		}
		for i, st := range expr.AsUnion(source.Type).Values {
			if other, err = collectHelpers(st.Attribute, tt.Values[i].Attribute, req, ta, seen); err == nil {
				helpers = append(helpers, other...)
			}
		}
	case expr.IsObject(source.Type):
		if expr.IsUnion(target.Type) {
			return helpers, err
		}
		walkMatches(source, target, func(srcMatt, _ *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, n string) {
			if err != nil {
				return
			}
			if other, err = collectHelpers(srcc, tgtc, srcMatt.IsRequired(n), ta, seen); err == nil {
				helpers = append(helpers, other...)
			}
		})
		// The additional-members map is a physical child of the object, just
		// like its fields. Discover the helpers used by transformAdditionalMembers
		// even when no declared or synthesized field reaches its value type.
		if err == nil {
			sourceCarrier, targetCarrier := source.AdditionalMembersCarrier(), target.AdditionalMembersCarrier()
			if sourceCarrier != nil && targetCarrier != nil {
				if other, err = collectHelpers(sourceCarrier, targetCarrier, false, ta, seen); err == nil {
					helpers = append(helpers, other...)
				}
			}
		}
	}
	return helpers, err
}

// generateHelper generates the code that transform instances of source into
// target. Both source and targe must be user types or generateHelper panics.
// generateHelper returns nil if a helper has already been generated for the
// pair source, target.
func generateHelper(source, target *expr.AttributeExpr, req bool, ta *TransformAttrs, seen map[string]*TransformFunctionData) (*TransformFunctionData, error) {
	name := transformHelperName(source, target, ta)
	if _, ok := seen[name]; ok {
		return nil, nil
	}
	nested := *ta
	nested.SourceCtx = ta.SourceCtx.Dup()
	nested.TargetCtx = ta.TargetCtx.Dup()
	ta = &nested

	// When transforming into a user type defined in an external package, assume
	// nested anonymous types (e.g., union sum types) belong to the same target
	// package unless they explicitly specify a different location.
	prevDefaultPkg := ta.TargetCtx.DefaultPkg
	prevSamePackageConversion := ta.TargetCtx.SamePackageConversion
	if pkg := ta.TargetCtx.Pkg(target); pkg != "" && pkg != prevDefaultPkg {
		ta.TargetCtx.DefaultPkg = pkg
		ta.TargetCtx.SamePackageConversion = false
		defer func() {
			ta.TargetCtx.DefaultPkg = prevDefaultPkg
			ta.TargetCtx.SamePackageConversion = prevSamePackageConversion
		}()
	}

	code, err := transformAttribute(source, target, "v", "res", true, ta)
	if err != nil {
		return nil, err
	}
	if ta.SourceCtx.Pointer && !expr.IsPrimitive(source.Type) {
		code = "if v == nil {\n\treturn nil\n}\n" + code
	} else if !req && !expr.IsPrimitive(source.Type) {
		code = "if v == nil {\n\treturn nil\n}\n" + code
	}
	tfd := &TransformFunctionData{
		Name:          name,
		ParamTypeRef:  ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source)),
		ResultTypeRef: ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target)),
		Code:          code,
		ErrorAware:    ta.failures.fallible(source, target, ta),
	}
	seen[name] = tfd
	return tfd, nil
}

// walkMatches iterates through the attributes of source and looks for
// attributes with identical names in target. walkMatches calls the walker
// function for each pair of matched attributes. Both source and target must be
// objects or else walkMatches panics.
func walkMatches(source, target *expr.AttributeExpr, walker func(src, tgt *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, n string)) {
	srcMatt := expr.NewMappedAttributeExpr(source)
	tgtMatt := expr.NewMappedAttributeExpr(target)
	srcObj := expr.AsObject(srcMatt.Type)
	tgtObj := expr.AsObject(tgtMatt.Type)
	for _, nat := range *srcObj {
		if att := tgtObj.Attribute(nat.Name); att != nil {
			walker(srcMatt, tgtMatt, nat.Attribute, att, nat.Name)
		}
	}
	// Members that a typed object requires without declaring them have
	// physical fields on both sides.
	targets := make(map[string]*expr.AttributeExpr)
	for _, member := range target.SynthesizedMembers() {
		targets[member.Name] = member.Attribute
	}
	for _, member := range source.SynthesizedMembers() {
		if att := targets[member.Name]; att != nil {
			walker(srcMatt, tgtMatt, member.Attribute, att, member.Name)
		}
	}
}

// transformHelperName returns the transformation function name to initialize a
// target user type from an instance of a source user type.
func transformHelperName(source, target *expr.AttributeExpr, ta *TransformAttrs) string {
	if ta.Options.HelperName != nil {
		if name := ta.Options.HelperName(source, target); name != "" {
			return name
		}
	}
	var (
		sname  string
		tname  string
		prefix string
	)
	{
		sname = Goify(ta.SourceCtx.Scope.Name(source, ta.SourceCtx.NamePkg(source), ta.SourceCtx.Pointer, ta.SourceCtx.UseDefault), true)
		tname = Goify(ta.TargetCtx.Scope.Name(target, ta.TargetCtx.NamePkg(target), ta.TargetCtx.Pointer, ta.TargetCtx.UseDefault), true)
		prefix = ta.Prefix
		if prefix == "" {
			prefix = "transform"
		}
	}
	return Goify(prefix+sname+"To"+tname, false)
}
