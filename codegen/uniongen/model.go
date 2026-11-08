// Package uniongen renders Go sum types for service, transport and plugin generators.
package uniongen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/unionjson"
	"github.com/CaliLuke/loom/internal/formgen"
)

type (
	// Type describes a generated sum-type union for service and transport code.
	Type struct {
		// Name is the Go type name of the union struct.
		Name string
		// KindName is the Go type name of the discriminator kind.
		KindName string
		// Fields describes each union branch.
		Fields []*Field
		// Loc defines the file and Go package of the union type if overridden via
		// Meta. When nil the type is generated in the default service file.
		Loc *codegen.Location
		// TypeKey is the discriminator field name for JSON marshaling (defaults to "type").
		TypeKey string
		// ValueKey is the value field name for JSON marshaling (defaults to "value").
		ValueKey string
		// Untagged reports whether JSON marshaling emits the selected branch value
		// directly instead of the canonical discriminator/value wrapper.
		Untagged bool
		// Closed rejects unevaluated JSON members at this union occurrence.
		Closed bool
		// JSONOnly omits form codecs for transports that only consume JSON.
		// The zero value preserves both JSON and form emission.
		JSONOnly bool
		// JSON holds shared untagged JSON analysis; nil for tagged unions.
		JSON *unionjson.Union
		// HasScalarFormBranch is true when at least one branch keeps canonical
		// type/value form encoding.
		HasScalarFormBranch bool
	}

	// Field describes a single branch of a union.
	Field struct {
		// Name is the branch name as defined in the DSL.
		Name string
		// KindConst is the Go identifier for the kind constant of this branch.
		KindConst string
		// FieldName is the struct field name in the union.
		FieldName string
		// MethodName is the required caller-supplied allocated Go suffix for
		// constructors, accessors and setters. Builders obtain it from
		// unionnames.Resolve together with FieldName; it is not derived during emission.
		MethodName string
		// FieldType is the Go type used in the union struct field and public API.
		FieldType string
		// AllowsNull reports whether the branch contract admits an explicit JSON null.
		AllowsNull bool
		// ValidateCode validates a decoded untagged branch held in v.
		ValidateCode string
		// ValidateRef is the Go expression that validates a decoded branch.
		ValidateRef string
		// FlatFormObject is true when the branch value is object-shaped and form
		// encoding should flatten its fields under the current prefix.
		FlatFormObject bool
		// FormValue retains the captured nonobject branch codec.
		FormValue formgen.Field
		// FormFields are the flattened object's visible captured members.
		FormFields []*FormField
		// FormError describes an unsupported form contract on a reusable type.
		FormError string
		// FormBranchAllowsEmpty reports whether the selected branch can be
		// materialized from only its discriminator; constraints still validate it.
		FormBranchAllowsEmpty bool
		// EmptyValueExpr is the Go expression that initializes an empty branch
		// value when FormBranchAllowsEmpty is true.
		EmptyValueExpr string
		// EmitPrimitiveAlias is true when the branch uses a generated primitive alias
		// that must be declared in the same file as the union type. Service data
		// sets it only for bare Any branches, whose alias is loom.JSONValue.
		EmitPrimitiveAlias bool
		// PrimitiveAliasType is the underlying Go type used by the generated branch
		// alias (for example "string" or "float64").
		PrimitiveAliasType string
		// TypeTag is the JSON "type" discriminator value for this branch.
		TypeTag string
	}
	// FormField combines captured policy with a declaration-owned Go selector.
	FormField struct {
		// Field retains the resolved codec and nullability.
		formgen.Field
		// Name is the captured form key before a caller prefix.
		Name string
		// Selector is the allocated exported Go field name.
		Selector string
		// Presence uses value as the containing Go object expression.
		Presence string
		// Required requires this member's own wire evidence.
		Required bool
	}
)
