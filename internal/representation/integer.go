package representation

import (
	"fmt"
	"go/types"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

// IntegerConversion describes the value-level contract of converting one
// integer representation to another. Storage overrides remain admitted; only
// values that the destination cannot represent are rejected at runtime.
type IntegerConversion struct {
	// Checked reports that some source value may be unrepresentable in the
	// target, so the conversion must round-trip and compare signs.
	Checked bool
	// SourceOpaque and TargetOpaque report external storage whose declaration
	// the generator cannot inspect. Emitters assert integer capability for them.
	SourceOpaque, TargetOpaque bool
}

// IntegerStorage returns the predeclared Go type that physically stores the
// attribute, following named declarations to their authored storage. It returns
// nil for external storage that is not a predeclared Go type.
func IntegerStorage(attribute *expr.AttributeExpr) *types.Basic {
	for {
		named, ok := attribute.Type.(expr.UserType)
		if !ok {
			break
		}
		attribute = named.Attribute()
	}
	name := encodingmeta.GoType(attribute.Meta).Name
	if name == "" {
		name = attribute.Type.Name()
	}
	object := types.Universe.Lookup(name)
	if object == nil {
		return nil
	}
	basic, _ := object.Type().Underlying().(*types.Basic)
	return basic
}

// IsIntegerKind reports whether the semantic type, after aliases, is one of the
// DSL integer primitives.
func IsIntegerKind(dt expr.DataType) bool {
	for {
		named, ok := dt.(expr.UserType)
		if !ok {
			break
		}
		dt = named.Attribute().Type
	}
	kind := dt.Kind()
	return kind >= expr.IntKind && kind <= expr.UInt64Kind
}

// ClassifyIntegerConversion reports the conversion contract between two
// integer attributes. ok is false when either side is not a semantic integer.
// Known non-integer storage cannot carry integer values and is an error.
// Conversions are checked when any value is lossy on the 386 or amd64 sizes.
func ClassifyIntegerConversion(source, target *expr.AttributeExpr) (conversion IntegerConversion, ok bool, err error) {
	if !IsIntegerKind(source.Type) || !IsIntegerKind(target.Type) {
		return IntegerConversion{}, false, nil
	}
	sourceType, targetType := IntegerStorage(source), IntegerStorage(target)
	for _, storage := range []*types.Basic{sourceType, targetType} {
		if storage != nil && storage.Info()&types.IsInteger == 0 {
			return IntegerConversion{}, true, fmt.Errorf("integer constructor conversion requires integer storage, got %s", storage.Name())
		}
	}
	conversion = IntegerConversion{SourceOpaque: sourceType == nil, TargetOpaque: targetType == nil}
	if sourceType == nil || targetType == nil {
		// External storage must satisfy the emitted integer capability assertion;
		// value representability is then checked at the actual conversion.
		conversion.Checked = true
		return conversion, true, nil
	}
	sourceUnsigned := sourceType.Info()&types.IsUnsigned != 0
	targetUnsigned := targetType.Info()&types.IsUnsigned != 0
	if !sourceUnsigned && targetUnsigned {
		conversion.Checked = true
		return conversion, true, nil
	}
	for _, arch := range []string{"386", "amd64"} {
		sizes := types.SizesFor("gc", arch)
		sourceBits, targetBits := sizes.Sizeof(sourceType)*8, sizes.Sizeof(targetType)*8
		if !targetUnsigned && sourceUnsigned {
			targetBits--
		}
		if sourceBits > targetBits {
			conversion.Checked = true
			return conversion, true, nil
		}
	}
	return conversion, true, nil
}

// IntegerCapabilityAssertion returns a declaration that compiles only when the
// referenced external type has integer storage.
func IntegerCapabilityAssertion(ref string) string {
	return fmt.Sprintf("const _ = ^%s(0)\n", ref)
}

// IntegerLossCondition returns the boolean expression that reports a value
// changed by converting value to converted, given the source reference.
func IntegerLossCondition(sourceRef, value, converted string) string {
	return fmt.Sprintf("%s(%s) != %s || (%s < 0) != (%s < 0)", sourceRef, converted, value, value, converted)
}

// IntegerLossError returns the error expression reporting that value cannot be
// represented by the target reference.
func IntegerLossError(targetRef, value string) string {
	return fmt.Sprintf("fmt.Errorf(\"value %%v cannot be represented as %s\", %s)", targetRef, value)
}
