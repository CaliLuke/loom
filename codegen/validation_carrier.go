package codegen

// validationTargetIsPointer distinguishes the current value from its field
// layout. Unwrapping an object yields a concrete struct even when Pointer must
// remain set for that struct's fields. Child traversal resets validationValue.
func (a *AttributeContext) validationTargetIsPointer(required, hasDefault bool) bool {
	return !a.validationValue && (a.Pointer || (!required && (!hasDefault || !a.UseDefault)))
}
