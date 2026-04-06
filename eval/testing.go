package eval

import "testing"

// SetupTestContext installs a fresh DSL evaluation context for tests.
func SetupTestContext(t testing.TB) {
	t.Helper()
	Context = NewContext()
}
