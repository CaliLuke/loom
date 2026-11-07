//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

// admittedDefaultHelper identifies a changed conversion by the caller-owned
// selection and discovered occurrence path, never pointer addresses or values.
type admittedDefaultHelper struct {
	role, path, source, target, base string
}

func transformArray(source, target *expr.Array, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	if err := IsCompatible(source.ElemType.Type, target.ElemType.Type, sourceVar+"[0]", targetVar+"[0]"); err != nil {
		return nil, err
	}
	data := transformArrayRenderData{
		ElemTypeRef:    ta.TargetCtx.CollectionElementRef(target.ElemType, false),
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
		TargetPresence: ta.TargetCtx.CollectionElementPresence && !expr.ArrayElementsAllowNull(target),
	}
	return renderTransformGoArray(data)
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
		KeyTypeRef:     ta.TargetCtx.CollectionElementRef(target.KeyType, true),
		ElemTypeRef:    ta.TargetCtx.CollectionElementRef(target.ElemType, false),
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
		TargetPresence: ta.TargetCtx.CollectionElementPresence && !expr.MapValuesAllowNull(target),
	}
	if depth := MapDepth(target); depth > 0 {
		data.LoopVar = string(rune(97 + depth))
	}
	return renderTransformGoMap(data)
}

func renderTransformGoArray(data transformArrayRenderData) (*jen.Statement, error) {
	typeName := "[]" + collectionCarrierRef(data.ElemTypeRef, data.TargetPresence)
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
		target := data.TargetVar + "[" + data.LoopVar + "]"
		if data.TargetPresence {
			target = presenceTempName(target)
		}
		elemCode, err = transformCollectionElem(data.SourceElem, data.TargetElem, sourceElement, target, data.TargetPresence, data.TransformAttrs)
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
				ifGroup.Add(assignCollectionValue(data.TargetVar+"["+data.LoopVar+"]", jen.Nil(), data.ElemTypeRef, data.TargetPresence))
				ifGroup.Continue()
			})
		}
		if data.IsStruct && data.ElemHelper {
			value := jen.Id(transformHelperName(data.SourceElem, data.TargetElem, data.TransformAttrs)).
				Call(transformCallArguments(data.SourceElem, data.TargetElem, sourceElement, data.TransformAttrs)...)
			group.Add(assignCollectionValue(data.TargetVar+"["+data.LoopVar+"]", value, data.ElemTypeRef, data.TargetPresence))
			return
		}
		group.Add(elemCode)
		if data.TargetPresence {
			target := data.TargetVar + "[" + data.LoopVar + "]"
			group.Add(assignCollectionValue(target, Expr(presenceTempName(target)), data.ElemTypeRef, true))
		}
	})
	return renderCollectionConversion(data.SourceVar, data.TargetVar, typeName, data.NewVar, stmt), nil
}

func renderTransformGoMap(data transformMapRenderData) (*jen.Statement, error) {
	typeName := "map[" + data.KeyTypeRef + "]" + collectionCarrierRef(data.ElemTypeRef, data.TargetPresence)
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
		renderTransformMapEntry(group, data, sourceElement, keyCode, elemCode)
	})
	return renderCollectionConversion(data.SourceVar, data.TargetVar, typeName, data.NewVar, stmt), nil
}

func renderTransformMapEntry(group *jen.Group, data transformMapRenderData, sourceElement string, keyCode, elemCode *jen.Statement) {
	if data.IsKeyStruct {
		group.Id("tk").Op(":=").Id(transformHelperName(data.SourceKey, data.TargetKey, data.TransformAttrs)).Call(transformCallArguments(data.SourceKey, data.TargetKey, "key", data.TransformAttrs)...)
	} else {
		group.Add(keyCode)
	}
	if data.SourcePresence {
		group.List(jen.Id("actual"), jen.Id("ok")).Op(":=").Id("val").Dot("Value").Call()
		group.If(jen.Op("!").Id("ok")).BlockFunc(func(ifGroup *jen.Group) {
			// Keep invalid absent slots visible instead of dropping their keys.
			// Native destinations cannot retain presence, but retain membership.
			missing := presenceTempName(data.TargetVar + "Missing")
			ifGroup.Var().Id(missing).Add(TypeRef(collectionCarrierRef(data.ElemTypeRef, data.TargetPresence)))
			ifGroup.Add(Expr(data.TargetVar + "[tk]")).Op("=").Id(missing)
			ifGroup.Continue()
		})
	}
	if data.IsElemStruct {
		group.If(Expr(sourceElement + " == nil")).BlockFunc(func(ifGroup *jen.Group) {
			ifGroup.Add(assignCollectionValue(data.TargetVar+"[tk]", jen.Nil(), data.ElemTypeRef, data.TargetPresence))
			ifGroup.Continue()
		})
	}
	if data.IsElemStruct && data.ElemHelper {
		value := jen.Id(transformHelperName(data.SourceElem, data.TargetElem, data.TransformAttrs)).
			Call(transformCallArguments(data.SourceElem, data.TargetElem, sourceElement, data.TransformAttrs)...)
		group.Add(assignCollectionValue(data.TargetVar+"[tk]", value, data.ElemTypeRef, data.TargetPresence))
		return
	}
	temp := "tv" + data.LoopVar
	group.Add(elemCode)
	group.Add(assignCollectionValue(data.TargetVar+"[tk]", Expr(temp), data.ElemTypeRef, data.TargetPresence))
}

func collectionCarrierRef(valueRef string, presence bool) string {
	if presence {
		return "loom.Optional[" + valueRef + "]"
	}
	return valueRef
}

func assignCollectionValue(target string, value jen.Code, valueRef string, presence bool) *jen.Statement {
	if presence {
		value = jen.Id("loom.OptionalValue").Types(TypeRef(valueRef)).Call(value)
	}
	return Expr(target).Op("=").Add(value)
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
	ta = transformPresenceContext(source, target, ta)
	if expr.IsNullable(source) || expr.IsNullable(target) {
		concreteSource, concreteTarget := concreteTransformPair(source, target, ta)
		if (nullablePhysicalTypeRef(source, ta.SourceCtx) != nullablePhysicalTypeRef(target, ta.TargetCtx) || ta.failures.sourceDefaultNode(source, target, ta) != nil) &&
			transformUsesHelper(concreteSource, concreteTarget) {
			helper, helperErr := generateHelper(concreteSource, concreteTarget, ta, seen)
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
	nested, err := appendNestedHelpers(source, target, ta, seen)
	return append(helpers, nested...), err
}

// collectHelpers recurses through the given attributes and returns the transform
// helper functions required to generate the transform code. If the attributes type
// is array or map then the recursion is done via transformAttributeHelpers so that
// the tope level conversion function is skipped as the generate code does not make
// use of it (since it inlines that top-level transformation).
func collectHelpers(source, target *expr.AttributeExpr, ta *TransformAttrs, seen map[string]*TransformFunctionData) (helpers []*TransformFunctionData, err error) {
	ta = transformPresenceContext(source, target, ta)
	if expr.IsNullable(source) || expr.IsNullable(target) {
		source, target = concreteTransformPair(source, target, ta)
	}
	name := transformHelperName(source, target, ta)
	if _, ok := seen[name]; ok {
		return helpers, err
	}
	if transformUsesHelper(source, target) {
		var h *TransformFunctionData
		h, err = generateHelper(source, target, ta, seen)
		if err != nil {
			return helpers, err
		}
		if h != nil {
			helpers = append(helpers, h)
		}
	}
	var other []*TransformFunctionData
	if other, err = appendNestedHelpers(source, target, ta, seen); err != nil {
		return helpers, err
	}
	helpers = append(helpers, other...)
	return helpers, err
}

func appendNestedHelpers(source, target *expr.AttributeExpr, ta *TransformAttrs, seen map[string]*TransformFunctionData) (helpers []*TransformFunctionData, err error) {
	ta = transformPresenceContext(source, target, ta)
	if matchingExplicitType(source, target) {
		return helpers, nil
	}
	var other []*TransformFunctionData
	switch {
	case expr.IsArray(source.Type):
		if other, err = collectHelpers(expr.AsArray(source.Type).ElemType, expr.AsArray(target.Type).ElemType, ta, seen); err == nil {
			helpers = append(helpers, other...)
		}
	case expr.IsMap(source.Type):
		sm, tm := expr.AsMap(source.Type), expr.AsMap(target.Type)
		if other, err = collectHelpers(sm.ElemType, tm.ElemType, ta, seen); err == nil {
			helpers = append(helpers, other...)
			if other, err = collectHelpers(sm.KeyType, tm.KeyType, ta, seen); err == nil {
				helpers = append(helpers, other...)
			}
		}
	case expr.IsUnion(source.Type):
		tt := expr.AsUnion(target.Type)
		if tt == nil {
			return helpers, err
		}
		for i, st := range expr.AsUnion(source.Type).Values {
			if other, err = collectHelpers(st.Attribute, tt.Values[i].Attribute, ta, seen); err == nil {
				helpers = append(helpers, other...)
			}
		}
	case expr.IsObject(source.Type):
		if expr.IsUnion(target.Type) {
			return helpers, err
		}
		walkMatches(source, target, func(_, _ *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, _ string) {
			if err != nil {
				return
			}
			if other, err = collectHelpers(srcc, tgtc, ta, seen); err == nil {
				helpers = append(helpers, other...)
			}
		})
		// The additional-members map is a physical child of the object, just
		// like its fields. Discover the helpers used by transformAdditionalMembers
		// even when no declared field reaches its value type.
		if err == nil {
			sourceCarrier, targetCarrier := source.AdditionalMembersCarrier(), target.AdditionalMembersCarrier()
			if sourceCarrier != nil && targetCarrier != nil {
				if other, err = collectHelpers(sourceCarrier, targetCarrier, ta, seen); err == nil {
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
func generateHelper(source, target *expr.AttributeExpr, ta *TransformAttrs, seen map[string]*TransformFunctionData) (*TransformFunctionData, error) {
	ta = transformPresenceContext(source, target, ta)
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
	// A helper is shared by all occurrences of the same type pair. Requiredness
	// cannot change its conversion semantics or make its behavior depend on the
	// first occurrence visited. Preserve nil from the actual pointer carrier;
	// callers and validators still own whether that absence is admitted.
	paramTypeRef := ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source))
	if strings.HasPrefix(paramTypeRef, "*") {
		code = "if v == nil {\n\treturn nil\n}\n" + code
	}
	tfd := &TransformFunctionData{
		Name:          name,
		ParamTypeRef:  paramTypeRef,
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
	srcMatt := transformMappedAttribute(source)
	tgtMatt := transformMappedAttribute(target)
	srcObj := expr.AsObject(srcMatt.Type)
	tgtObj := expr.AsObject(tgtMatt.Type)
	for _, nat := range *srcObj {
		if att := tgtObj.Attribute(nat.Name); att != nil {
			walker(srcMatt, tgtMatt, nat.Attribute, att, nat.Name)
		}
	}
}

// transformMappedAttribute detaches name and parent facts while retaining the
// actual selected child occurrences. Transform consumers only read these children;
// replacing them with deep copies would split discovery from helper emission.
func transformMappedAttribute(attribute *expr.AttributeExpr) *expr.MappedAttributeExpr {
	mapped := expr.NewMappedAttributeExpr(attribute)
	for _, field := range *expr.AsObject(attribute.Type) {
		expr.AsObject(mapped.Type).Set(expr.AttributeName(field.Name), field.Attribute)
	}
	return mapped
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
		sname = Goify(ta.SourceCtx.Scope.Name(source, ta.SourceCtx.NamePkg(source), ta.SourceCtx), true)
		tname = Goify(ta.TargetCtx.Scope.Name(target, ta.TargetCtx.NamePkg(target), ta.TargetCtx), true)
		prefix = ta.Prefix
		if prefix == "" {
			prefix = "transform"
		}
	}
	name := Goify(prefix+sname+"To"+tname, false)
	if node := ta.failures.sourceDefaultNode(source, target, ta); node != nil {
		key := admittedDefaultHelper{role: ta.Options.AdmittedSource, path: node.path,
			source: ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source)),
			target: ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target)), base: name}
		return ta.SourceCtx.Scope.Scope().HashedUnique(key, name+"Admitted")
	}
	return name
}

func (key admittedDefaultHelper) Hash() string {
	return fmt.Sprintf("admitted-default/%q/%q/%q/%q/%q", key.role, key.path, key.source, key.target, key.base)
}
