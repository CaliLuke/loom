// Package unionjson projects evaluated union contracts into typed JSON adapters.
// Service and transport generators share this analysis and rendering boundary.
package unionjson

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// Union describes generated typed adapters for one untagged occurrence.
	Union struct {
		// Name is the allocated Go union type name.
		Name string
		// Branches follow declaration order and carry allocated Go references.
		Branches []*Branch
	}
	// Branch binds the normalized predicates to a generated Go representation.
	Branch struct {
		// Type is the NameScope-allocated Go type reference.
		Type string
		// Field is the allocated union payload field.
		Field string
		// Kind is the allocated discriminator constant.
		Kind string
		// Validate is a statement block using v and named error result err.
		Validate string
		// Schema and Runtime check the respective finalized representations.
		Schema, Runtime *loom.JSONShape
	}
)

// Analyze captures branch predicates using the existing occurrence/value-plan
// owner. fieldName selects service or transport member spelling before capture.
func Analyze(name string, union *expr.Union, branches []*Branch, fieldName func(string) string, closedObjects bool) *Union {
	for index, branch := range union.Values {
		for _, target := range []struct {
			schema bool
			result **loom.JSONShape
		}{
			{true, &branches[index].Schema}, {false, &branches[index].Runtime},
		} {
			plan, err := Plan(branch.Attribute, fieldName, target.schema, closedObjects)
			if err != nil {
				panic(err)
			}
			*target.result, err = plan.JSONShape()
			if err != nil {
				panic(err)
			}
		}
	}
	return &Union{Name: name, Branches: branches}
}

// Plan captures the finalized JSON occurrence with explicit member spelling and visibility.
func Plan(source *expr.AttributeExpr, fieldName func(string) string, schema, closedObjects bool) (expr.ValuePlan, error) {
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	if err != nil {
		return expr.ValuePlan{}, fmt.Errorf("untagged branch occurrence: %w", err)
	}
	target := expr.DupAtt(source)
	request := expr.ValuePlanRequest{Target: target, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation}
	seen := make(map[*expr.AttributeExpr]bool)
	var walk func(*expr.AttributeExpr)
	walk = func(att *expr.AttributeExpr) {
		if seen[att] {
			return
		}
		seen[att] = true
		if ut, ok := att.Type.(expr.UserType); ok {
			walk(ut.Attribute())
			return
		}
		if array := expr.AsArray(att.Type); array != nil {
			walk(array.ElemType)
			return
		}
		if object := expr.AsObject(att.Type); object != nil {
			if schema && closedObjects {
				if att.Meta == nil {
					att.Meta = make(expr.MetaExpr)
				}
				att.Meta["openapi:additionalProperties"] = []string{"false"}
			}
			for _, field := range *object {
				visible := true
				if hidden, ok := field.Attribute.Meta.Last("openapi:generate"); schema && ok && hidden == "false" {
					visible = false
				}
				var numeric expr.Kind
				if expr.IsPrimitive(field.Attribute.Type) {
					kind := field.Attribute.Type.Kind()
					if kind >= expr.IntKind && kind <= expr.Float64Kind {
						numeric = kind
					}
				}
				request.Fields = append(request.Fields, expr.ValueFieldPolicy{
					Parent: att, Target: field.Attribute, Name: field.Name,
					WireName: expr.JSONFieldName(fieldName(field.Name), field.Attribute),
					Visible:  visible, Required: att.IsRequired(field.Name),
					Presence: expr.ValueFieldRetain, NumericKind: numeric,
				})
				walk(field.Attribute)
			}
		}
	}
	walk(target)
	plan, err := context.NewValuePlan(occurrence, request)
	if err != nil {
		return expr.ValuePlan{}, fmt.Errorf("untagged branch plan: %w", err)
	}
	return plan, nil
}

// Marshal renders an adapter that retains selected identity through matching.
func (u *Union) Marshal() string {
	var b strings.Builder
	b.WriteString("if err := u.Validate(); err != nil {\nreturn nil, err\n}\nvar data []byte\nvar err error\nselected := -1\nswitch u.kind {\n")
	for index, branch := range u.Branches {
		fmt.Fprintf(&b, "case %s:\nselected = %d\ndata, err = json.Marshal(u.%s, loom.JSONOptions(), json.Deterministic(true))\n", branch.Kind, index, branch.Field)
	}
	b.WriteString("}\nif err != nil {\nreturn nil, err\n}\n")
	b.WriteString("_, matched, err := u.matchJSON(data)\n")
	b.WriteString("if err != nil {\nreturn nil, err\n}\nif matched != selected {\nreturn nil, fmt.Errorf(\"untagged union changed selected branch\")\n}\nreturn data, nil")
	return b.String()
}

// Unmarshal renders transactional assignment after independent unique matches.
func (u *Union) Unmarshal() string {
	return "matched, _, err := u.matchJSON(data)\nif err != nil {\nreturn err\n}\n*u = matched\nreturn nil"
}

// Matcher renders the single typed matching adapter shared by both directions.
func (u *Union) Matcher() string {
	var b strings.Builder
	fmt.Fprintf(&b, "func (u %s) matchJSON(data []byte) (%s, int, error) {\n", u.Name, u.Name)
	u.candidates(&b)
	fmt.Fprintf(&b, "matched, err := loom.MatchUntaggedJSON(%q, data, candidates)\n", u.Name)
	fmt.Fprintf(&b, "if err != nil {\nreturn %s{}, -1, err\n}\nswitch matched {\n", u.Name)
	for index, branch := range u.Branches {
		fmt.Fprintf(&b, "case %d:\nreturn %s{kind: %s, %s: value%d}, matched, nil\n", index, u.Name, branch.Kind, branch.Field, index)
	}
	fmt.Fprintf(&b, "}\nreturn %s{}, -1, fmt.Errorf(\"invalid untagged union match index\")\n}", u.Name)
	return b.String()
}

func (u *Union) candidates(b *strings.Builder) {
	for index, branch := range u.Branches {
		fmt.Fprintf(b, "var value%d %s\n", index, branch.Type)
		renderShape(b, fmt.Sprintf("schema%d", index), branch.Schema)
		renderShape(b, fmt.Sprintf("runtime%d", index), branch.Runtime)
	}
	b.WriteString("candidates := []loom.JSONUnionBranch{\n")
	for index, branch := range u.Branches {
		fmt.Fprintf(b, "{Schema: schema%d, Decode: func(wire jsontext.Value) (bool, error) {\n", index)
		fmt.Fprintf(b, "matched, err := runtime%d.Match(wire)\nif err != nil || !matched {\nreturn matched, err\n}\n", index)
		fmt.Fprintf(b, "var v %s\nmatched, err = loom.DecodeJSONCandidate(wire, &v)\nif err != nil || !matched {\nreturn matched, err\n}\n", branch.Type)
		if branch.Validate != "" {
			fmt.Fprintf(b, "if err := func() (err error) {\n%s\nreturn\n}(); err != nil {\nreturn false, nil\n}\n", branch.Validate)
		}
		fmt.Fprintf(b, "value%d = v\nreturn true, nil\n}},\n", index)
	}
	b.WriteString("}\n")
}
