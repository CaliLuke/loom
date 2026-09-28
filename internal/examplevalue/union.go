// Package examplevalue carries generated example selections until projection.
package examplevalue

// Union retains a generated branch choice independently of its payload shape.
// It is an intermediate value, not a JSON envelope or an authored example.
type Union struct {
	// Branch is the zero-based index in the union's ordered branch list.
	Branch int
	// Value is the selected branch's unprojected example.
	Value any
}
