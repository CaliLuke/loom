package uniongen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/unionjson"
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
		// FieldType is the Go type used in the union struct field and public API.
		FieldType string
		// ValidateCode validates a decoded untagged branch held in v.
		ValidateCode string
		// ValidateRef is the Go expression that validates a decoded branch.
		ValidateRef string
		// FlatFormObject is true when the branch value is object-shaped and form
		// encoding should flatten its fields under the current prefix.
		FlatFormObject bool
		// FlatFormObjectAllowsEmpty is true when the flattened object branch may be
		// selected with only the discriminator because it has no required fields.
		FlatFormObjectAllowsEmpty bool
		// EmptyValueExpr is the Go expression that initializes an empty branch
		// value when FlatFormObjectAllowsEmpty is true.
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
)
