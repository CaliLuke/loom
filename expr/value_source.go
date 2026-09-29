package expr

import "errors"

type (
	// ExamplePolicy applies reachability before source use. SuppressGenerated
	// suppresses synthesis only; it preserves the authored-example exception.
	ExamplePolicy struct {
		// Reachable indicates that the target consumes this occurrence.
		Reachable bool
		// SuppressGenerated disables synthesis when no authored source exists.
		SuppressGenerated bool
	}

	// ExampleState describes source selection without resolving or generating data.
	ExampleState uint8

	// ExampleSelection preserves the source and occurrence chosen before resolution.
	// Only an absent selection is eligible for synthesis.
	ExampleSelection struct {
		state      ExampleState
		occurrence ValueOccurrence
		source     ValueSource
	}
)

const (
	// ExampleExcluded is unreachable and cannot use authored or synthesized data.
	ExampleExcluded ExampleState = iota + 1
	// ExampleSuppressed has no authored source and disables synthesis.
	ExampleSuppressed
	// ExampleAbsent has no authored source and may attempt synthesis.
	ExampleAbsent
	// ExampleSelected identifies a supplied source, including invalid data or null.
	ExampleSelected
)

var errValueSourceOwnership = errors.New("example source is not owned by its semantic occurrence")

// SelectExample chooses the last authored example from the first extracted
// group, using local/reference/base/type precedence. It never resolves,
// synthesizes or invokes custom materialization.
func (c *ValueContext) SelectExample(occurrence ValueOccurrence, policy ExamplePolicy) ExampleSelection {
	selection := ExampleSelection{state: ExampleExcluded, occurrence: occurrence}
	if c == nil || occurrence.context != c.identity || occurrence.node == nil || !policy.Reachable {
		return selection
	}
	examples := occurrence.node.attribute.ExtractUserExamples()
	if len(examples) == 0 {
		selection.state = ExampleAbsent
		if policy.SuppressGenerated {
			selection.state = ExampleSuppressed
		}
		return selection
	}
	chosen := examples[len(examples)-1]
	for _, node := range occurrence.graph.nodes {
		for _, example := range node.examples {
			if example.example != chosen {
				continue
			}
			c.mu.Lock()
			source := c.examples[example.origin]
			if source == nil {
				source = &valueSourceData{context: c.identity, snapshot: example.source,
					explicitNull: example.example.ExplicitNull, origin: example.example.Summary}
				c.examples[example.origin] = source
			}
			c.mu.Unlock()
			selection.state = ExampleSelected
			selection.source = ValueSource{data: source}
			return selection
		}
	}
	// Every extracted source belongs to the occurrence's owned graph. An
	// unreachable descriptor indicates a malformed builder, never absence that
	// could authorize a substitute synthetic example.
	selection.state = ExampleSelected
	selection.source = ValueSource{data: &valueSourceData{context: c.identity,
		snapshot: valueSourceSnapshot{err: errValueSourceOwnership}}}
	return selection
}

// ContainsExampleSource reports whether a supplied authored example belongs to
// any node in this semantic graph, including selected-body children. It compares
// provenance identities only and never selects, resolves or materializes data.
func (c *ValueContext) ContainsExampleSource(occurrence ValueOccurrence, source ValueSource) bool {
	if c == nil || occurrence.context != c.identity || occurrence.graph == nil || source.data == nil || source.data.context != c.identity {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, node := range occurrence.graph.nodes {
		for _, example := range node.examples {
			if c.examples[example.origin] == source.data {
				return true
			}
		}
	}
	return false
}

// State returns the result of source selection, without interpreting source data.
func (s ExampleSelection) State() ExampleState {
	return s.state
}

// Source returns the supplied source only for an authored selection.
func (s ExampleSelection) Source() (ValueSource, bool) {
	return s.source, s.state == ExampleSelected
}
