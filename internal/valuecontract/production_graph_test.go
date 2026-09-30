package valuecontract

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type productionGraph struct {
	input        *productionInput
	t            *testing.T
	ids          map[expr.ValueIdentity]referenceIdentity
	declarations []any
	next         uint64
}

func newProductionGraph(t *testing.T, input *productionInput) *productionGraph {
	return &productionGraph{t: t, input: input, ids: make(map[expr.ValueIdentity]referenceIdentity)}
}

func (g *productionGraph) identity(id expr.ValueIdentity) referenceIdentity {
	if found, ok := g.ids[id]; ok {
		return found
	}
	g.next++
	found := referenceIdentity{Occurrence: g.next, Declaration: g.next}
	g.ids[id] = found
	return found
}

func productionLength(validation *expr.ValidationExpr) map[string]any {
	bounds := map[string]any{"minimum": nil, "maximum": nil}
	if validation != nil {
		bounds["minimum"], bounds["maximum"] = validation.MinLength, validation.MaxLength
	}
	return bounds
}

func productionRules(validation *expr.ValidationExpr, kind expr.Kind) map[string]any {
	format := "exact"
	if kind == expr.Float32Kind {
		format = "binary32"
	} else if kind == expr.Float64Kind {
		format = "binary64"
	}
	numeric := map[string]any{"minimum": nil, "maximum": nil, "exclusiveMinimum": false, "exclusiveMaximum": false}
	if validation != nil {
		if validation.Minimum != nil {
			numeric["minimum"] = referenceBinaryDecimal(*validation.Minimum)
		}
		if validation.Maximum != nil {
			numeric["maximum"] = referenceBinaryDecimal(*validation.Maximum)
		}
		if validation.ExclusiveMinimum != nil {
			numeric["minimum"], numeric["exclusiveMinimum"] = referenceBinaryDecimal(*validation.ExclusiveMinimum), true
		}
		if validation.ExclusiveMaximum != nil {
			numeric["maximum"], numeric["exclusiveMaximum"] = referenceBinaryDecimal(*validation.ExclusiveMaximum), true
		}
	}
	return map[string]any{"sourcePrimitive": productionSourcePrimitive(kind), "numericFormat": format, "integerFormat": "mathematical", "enumeration": nil, "length": productionLength(validation), "numeric": numeric, "externalChecks": []any{}}
}

func productionSourcePrimitive(kind expr.Kind) any {
	// Target-only fixtures use zero because source admission is inapplicable.
	if kind == 0 {
		return nil
	}
	for _, primitive := range []expr.Primitive{expr.Boolean, expr.String, expr.Bytes, expr.Int, expr.Int32, expr.Int64, expr.UInt, expr.UInt32, expr.UInt64, expr.Float32, expr.Float64, expr.Any} {
		if primitive.Kind() == kind {
			return primitive.Name()
		}
	}
	panic("unsupported source primitive")
}

// productionValidationEmpty protects vocabulary boundaries that this adapter
// cannot yet lower. It must not silently discard a declaration constraint.
func productionValidationEmpty(validation *expr.ValidationExpr) bool {
	return validation == nil || reflect.DeepEqual(*validation, expr.ValidationExpr{})
}

func productionScalarKind(kind expr.Kind) string {
	switch kind {
	case expr.BooleanKind:
		return "boolean"
	case expr.StringKind:
		return "string"
	case expr.BytesKind:
		return "bytes"
	case expr.Float32Kind, expr.Float64Kind:
		return "decimal"
	default:
		return "integer"
	}
}

// capture walks the supplied declaration and the opaque occurrence in lockstep.
// It does not call Resolve or infer a contract from a successful result.
func (g *productionGraph) capture(attribute *expr.AttributeExpr, occurrence expr.ValueOccurrence) referenceIdentity {
	identity := g.identity(occurrence.ID())
	for _, raw := range g.declarations {
		if raw.(map[string]any)["identity"] == identity {
			return identity
		}
	}
	declaration := map[string]any{"identity": identity, "expansionRank": uint64(0), "enumeration": nil}
	g.declarations = append(g.declarations, declaration)
	body := declaration
	if attribute.Nullable {
		g.next++
		child := referenceIdentity{Occurrence: g.next, Declaration: g.next}
		body = map[string]any{"identity": child, "expansionRank": uint64(0), "enumeration": nil}
		g.declarations = append(g.declarations, body)
		declaration["contract"] = referenceConstructor("nullable", map[string]any{"child": child})
		declaration["expansionRank"] = uint64(100)
	}
	switch typ := attribute.Type.(type) {
	case expr.UserType:
		if validation := attribute.Validation; validation != nil {
			local := *validation
			local.Values = nil
			require.Equal(g.t, expr.ValidationExpr{}, local, "alias-local constraints require independent node gates in the reference model")
		}
		child := g.capture(typ.Attribute(), occurrence.Underlying())
		body["contract"] = referenceConstructor("alias", map[string]any{"child": child})
		// Alias ranks are recomputed from graph edges once capture is complete.
	case *expr.Array:
		child := g.capture(typ.ElemType, occurrence.Element())
		if typ.NonNullableElems {
			g.next++
			wrapper := referenceIdentity{Occurrence: g.next, Declaration: g.next}
			g.declarations = append(g.declarations, map[string]any{"identity": wrapper, "expansionRank": uint64(1), "enumeration": nil, "contract": referenceConstructor("nonNull", map[string]any{"child": child})})
			child = wrapper
		}
		body["contract"] = referenceConstructor("array", map[string]any{"child": child, "length": productionLength(attribute.Validation)})
	case *expr.Map:
		child := g.capture(typ.ElemType, occurrence.Element())
		keyType := typ.KeyType.Type
		for named, ok := keyType.(expr.UserType); ok; named, ok = keyType.(expr.UserType) {
			require.True(g.t, productionValidationEmpty(named.Attribute().Validation), "named map-key constraints require explicit reference lowering")
			keyType = named.Attribute().Type
		}
		if validation := typ.KeyType.Validation; validation != nil {
			require.Nil(g.t, validation.Values, "map-key enums require explicit reference lowering")
		}
		var key any = "builtin"
		if keyType != expr.Any {
			key = referenceConstructor("scalar", map[string]any{"kind": productionScalarKind(keyType.Kind())})
		}
		body["contract"] = referenceConstructor("map", map[string]any{"key": key, "keyRules": productionRules(typ.KeyType.Validation, keyType.Kind()), "child": child, "length": productionLength(attribute.Validation)})
	case *expr.Object:
		members := make([]any, 0, len(*typ))
		aliases := make(map[string]bool)
		for index, member := range occurrence.Members() {
			require.False(g.t, aliases[member.WireName], "duplicate source wire aliases require explicit reference lowering")
			aliases[member.WireName] = true
			field := (*typ)[index]
			members = append(members, map[string]any{"identity": g.identity(member.ID), "sourceName": member.Name, "wireAlias": member.WireName, "required": attribute.IsRequired(field.Name), "child": g.capture(field.Attribute, member.Occurrence)})
		}
		open := true
		if value, ok := attribute.Meta.Last("openapi:additionalProperties"); ok {
			open = value != "false"
		}
		body["contract"] = referenceConstructor("object", map[string]any{"members": members, "isOpen": open})
	case *expr.Union:
		branches := make([]any, 0, len(typ.Values))
		for index, branch := range occurrence.Branches() {
			branches = append(branches, map[string]any{"identity": g.identity(branch.ID), "name": branch.Name, "child": g.capture(typ.Values[index].Attribute, branch.Occurrence)})
		}
		body["contract"] = referenceConstructor("union", map[string]any{"occurrence": identity, "alternatives": branches})
	default:
		if attribute.Type == expr.Any {
			body["contract"] = "any"
		} else {
			body["contract"] = referenceConstructor("scalar", map[string]any{"kind": productionScalarKind(attribute.Type.Kind()), "rules": productionRules(attribute.Validation, attribute.Type.Kind())})
		}
	}

	if attribute.Validation != nil && attribute.Validation.Values != nil {
		values := make([]any, 0, len(attribute.Validation.Values))
		for _, raw := range attribute.Validation.Values {
			if value, accepted := g.literalMaybe(attribute, raw); accepted {
				values = append(values, value)
			}
		}
		declaration["enumeration"] = values
	}
	return identity
}

func (g *productionGraph) ranks() {
	var rank func(referenceIdentity, map[referenceIdentity]bool) uint64
	rank = func(identity referenceIdentity, active map[referenceIdentity]bool) uint64 {
		require.False(g.t, active[identity], "nonconsuming declaration cycle")
		active[identity] = true
		defer delete(active, identity)
		for _, raw := range g.declarations {
			declaration := raw.(map[string]any)
			if declaration["identity"] != identity {
				continue
			}
			contract, ok := declaration["contract"].(map[string]any)
			if !ok {
				return 0
			}
			for _, name := range []string{"alias", "nullable", "nonNull"} {
				if fields, ok := contract[name].(map[string]any); ok {
					return rank(fields["child"].(referenceIdentity), active) + 1
				}
			}
			if union, ok := contract["union"].(map[string]any); ok {
				var maximum uint64
				for _, rawBranch := range union["alternatives"].([]any) {
					branch := rawBranch.(map[string]any)
					maximum = max(maximum, rank(branch["child"].(referenceIdentity), active)+1)
				}
				return maximum
			}
			return 0
		}
		g.t.Fatalf("missing declaration %v", identity)
		return 0
	}
	for _, raw := range g.declarations {
		declaration := raw.(map[string]any)
		declaration["expansionRank"] = rank(declaration["identity"].(referenceIdentity), make(map[referenceIdentity]bool))
	}
}

// literal encodes an independently authored enum fixture. Unsupported fixture
// shapes fail loudly instead of consulting the production resolver for an oracle.
func (g *productionGraph) literal(attribute *expr.AttributeExpr, raw any) any {
	value, accepted := g.literalMaybe(attribute, raw)
	require.True(g.t, accepted, "canonical fixture has inadmissible %T for %s", raw, attribute.Type.Name())
	return value
}

// literalMaybe applies declaration admission before normalization, independently
// of Resolve. Rejected enum members stay rejected rather than being coerced into
// admissible values; an enum whose members are all rejected remains an empty enum.
func (g *productionGraph) literalMaybe(attribute *expr.AttributeExpr, raw any) (any, bool) {
	if raw == nil && attribute.Nullable {
		return "null", true
	}
	switch typ := attribute.Type.(type) {
	case expr.UserType:
		return g.literalMaybe(typ.Attribute(), raw)
	case *expr.Array:
		input := reflect.ValueOf(raw)
		if !input.IsValid() || (input.Kind() != reflect.Array && input.Kind() != reflect.Slice) {
			return nil, false
		}
		items := make([]any, input.Len())
		for index := range items {
			item, accepted := g.literalMaybe(typ.ElemType, input.Index(index).Interface())
			if !accepted {
				return nil, false
			}
			items[index] = item
		}
		return referenceConstructor("array", map[string]any{"items": items}), true
	case *expr.Map:
		return g.mapLiteral(typ, raw)
	case expr.Primitive:
		if !typ.IsCompatible(raw) {
			return nil, false
		}
		if typ == expr.Any {
			return referenceConstructor("any", map[string]any{"payload": g.input.canonicalRaw(g.input.encode(raw))}), true
		}
		if typ == expr.Bytes {
			if text, ok := raw.(string); ok {
				raw = []byte(text)
			}
		}
		if typ == expr.Float32 || typ == expr.Float64 {
			bits := 64
			if typ == expr.Float32 {
				bits = 32
			}
			value, err := strconv.ParseFloat(fmt.Sprint(raw), bits)
			require.NoError(g.t, err)
			raw = value
			if bits == 32 {
				raw = float32(value)
			}
		}
		return referenceConstructor("scalar", map[string]any{"value": g.input.scalar(raw)}), true
	default:
		g.t.Fatalf("enum fixture requires an explicit independent literal adapter for %T", typ)
		return nil, false
	}
}

func (g *productionGraph) mapLiteral(typ *expr.Map, raw any) (any, bool) {
	input := reflect.ValueOf(raw)
	if !input.IsValid() || input.Kind() != reflect.Map {
		return nil, false
	}
	entries := make([]any, 0, input.Len())
	for _, key := range productionMapKeys(input) {
		var scalar any
		if typ.KeyType.Type == expr.Any {
			scalar = g.input.scalar(key.Interface())
		} else {
			literal, accepted := g.literalMaybe(typ.KeyType, key.Interface())
			if !accepted {
				return nil, false
			}
			scalar = literal.(map[string]any)["scalar"].(map[string]any)["value"]
		}
		value, accepted := g.literalMaybe(typ.ElemType, input.MapIndex(key).Interface())
		if !accepted {
			return nil, false
		}
		entries = append(entries, []any{scalar, value})
	}
	return referenceConstructor("map", map[string]any{"entries": entries}), true
}

func TestProductionConcreteSourcePolicies(t *testing.T) {
	require.Nil(t, productionRules(nil, 0)["sourcePrimitive"])
	for _, tc := range []struct {
		kind expr.Kind
		want string
	}{
		{expr.BooleanKind, "boolean"}, {expr.StringKind, "string"}, {expr.BytesKind, "bytes"},
		{expr.IntKind, "int"}, {expr.Int32Kind, "int32"}, {expr.Int64Kind, "int64"},
		{expr.UIntKind, "uint"}, {expr.UInt32Kind, "uint32"}, {expr.UInt64Kind, "uint64"},
		{expr.Float32Kind, "float32"}, {expr.Float64Kind, "float64"}, {expr.AnyKind, "any"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			require.Equal(t, tc.want, productionRules(nil, tc.kind)["sourcePrimitive"])
		})
	}
}

func TestProductionEnumAdmission(t *testing.T) {
	type namedInt int
	array := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}}
	mapping := &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Int}, ElemType: &expr.AttributeExpr{Type: expr.String}}}
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		raw       any
		accepted  bool
	}{
		{"int", &expr.AttributeExpr{Type: expr.Int}, int(1), true},
		{"wrong integer width", &expr.AttributeExpr{Type: expr.Int}, int64(1), false},
		{"defined integer", &expr.AttributeExpr{Type: expr.Int}, namedInt(1), false},
		{"integer to float", &expr.AttributeExpr{Type: expr.Float32}, int64(1), true},
		{"text bytes", &expr.AttributeExpr{Type: expr.Bytes}, "hi", true},
		{"valid array", array, []int{1}, true},
		{"invalid array member", array, []int64{1}, false},
		{"valid map", mapping, map[int]string{1: "x"}, true},
		{"invalid map key", mapping, map[int64]string{1: "x"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := newProductionGraph(t, newProductionInput(t))
			_, accepted := graph.literalMaybe(tc.attribute, tc.raw)
			require.Equal(t, tc.accepted, accepted)
		})
	}
	t.Run("rejected enum remains present", func(t *testing.T) {
		attribute := &expr.AttributeExpr{Type: expr.Int, Validation: &expr.ValidationExpr{Values: []any{int64(1)}}}
		require.Equal(t, []any{int64(1)}, attribute.Validation.Values)
		_, err := expr.EffectiveConstraintsFor(attribute)
		require.EqualError(t, err, `enum member 1 declared by "int" violates the effective contract for "int"`)
		require.Equal(t, []any{int64(1)}, attribute.Validation.Values,
			"rejection must not erase the authored declaration")
	})
	t.Run("rejected map enum key remains present", func(t *testing.T) {
		raw := map[int64]string{1: "x"}
		attribute := &expr.AttributeExpr{
			Type:       &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Int}, ElemType: &expr.AttributeExpr{Type: expr.String}},
			Validation: &expr.ValidationExpr{Values: []any{raw}},
		}
		require.Equal(t, []any{raw}, attribute.Validation.Values)
		_, err := expr.EffectiveConstraintsFor(attribute)
		require.EqualError(t, err, `enum member map[int64]string{1:"x"} declared by "map" violates the effective contract for "map"`)
		require.Equal(t, []any{raw}, attribute.Validation.Values,
			"rejection must not erase the authored declaration")
	})
	t.Run("Any canonical values have no source tags", func(t *testing.T) {
		graph := newProductionGraph(t, newProductionInput(t))
		value := graph.literal(&expr.AttributeExpr{Type: expr.Any}, map[any]any{int64(1): []any{int(2), "x"}})
		text := productionJSON(t, value)
		require.NotContains(t, text, "primitive")
		require.Contains(t, text, "host")
		require.Contains(t, text, "integer")
	})
}

func TestProductionUnsupportedKeyConstraints(t *testing.T) {
	one := 1
	for _, tc := range []struct {
		name       string
		validation *expr.ValidationExpr
		empty      bool
	}{
		{"absent", nil, true},
		{"empty", &expr.ValidationExpr{}, true},
		{"length", &expr.ValidationExpr{MinLength: &one}, false},
		{"enum", &expr.ValidationExpr{Values: []any{"x"}}, false},
		{"empty enum", &expr.ValidationExpr{Values: []any{}}, false},
		{"pattern", &expr.ValidationExpr{Pattern: "x"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.empty, productionValidationEmpty(tc.validation))
		})
	}
}

func TestProductionConstraintGuardFailsClosed(t *testing.T) {
	if mode := os.Getenv("LOOM_ADAPTER_GUARD_CONTROL"); mode != "" {
		key := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"x"}}}
		if mode == "named" {
			one := 1
			key = &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Key", AttributeExpr: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{MinLength: &one}}}}
		}
		attribute := &expr.AttributeExpr{Type: &expr.Map{KeyType: key, ElemType: &expr.AttributeExpr{Type: expr.String}}}
		if mode == "aliases" {
			attribute = &expr.AttributeExpr{Type: &expr.Object{
				{Name: "left:wire", Attribute: &expr.AttributeExpr{Type: expr.String}},
				{Name: "right:wire", Attribute: &expr.AttributeExpr{Type: expr.String}},
			}}
		}
		occurrence, err := expr.NewValueContext().NewOccurrence(attribute)
		require.NoError(t, err)
		newProductionGraph(t, newProductionInput(t)).capture(attribute, occurrence)
		return
	}
	for _, mode := range []string{"enum", "named", "aliases"} {
		t.Run(mode, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestProductionConstraintGuardFailsClosed$")
			command.Env = append(os.Environ(), "LOOM_ADAPTER_GUARD_CONTROL="+mode)
			output, err := command.CombinedOutput()
			require.Error(t, err, "unsupported constraints must not pass capture")
			require.Contains(t, string(output), "require explicit reference lowering")
		})
	}
}
