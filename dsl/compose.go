package dsl

// composeDSL combines deferred DSL declarations in declaration order.
func composeDSL(first, next func()) func() {
	if first == nil {
		return next
	}
	if next == nil {
		return first
	}
	return func() {
		first()
		next()
	}
}
