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
		entries    []ExampleEntry
	}

	// ExampleEntry describes one captured source in the selected authored group.
	// Its value remains opaque; callers resolve Source against their retained
	// effective occurrence rather than inferring an owner from authored provenance.
	ExampleEntry struct {
		source      ValueSource
		summary     string
		description string
		meta        MetaExpr
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
	selection.state = ExampleSelected
	selection.entries = make([]ExampleEntry, 0, len(examples))
	for _, selected := range examples {
		entry, found := c.exampleEntry(occurrence, selected)
		if !found {
			entry = ExampleEntry{source: ValueSource{data: &valueSourceData{context: c.identity,
				snapshot: valueSourceSnapshot{err: errValueSourceOwnership}}}, summary: selected.Summary,
				description: selected.Description, meta: copyValueMeta(selected.Meta)}
		}
		selection.entries = append(selection.entries, entry)
	}
	selection.source = selection.entries[len(selection.entries)-1].source
	return selection
}

func (c *ValueContext) exampleEntry(occurrence ValueOccurrence, selected *ExampleExpr) (ExampleEntry, bool) {
	for _, node := range occurrence.graph.nodes {
		for _, example := range node.examples {
			if example.example != selected {
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
			return ExampleEntry{source: ValueSource{data: source}, summary: example.example.Summary,
				description: example.example.Description, meta: copyValueMeta(example.example.Meta)}, true
		}
	}
	return ExampleEntry{}, false
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

// Entries returns the selected authored group in declaration order. The result
// is empty for excluded, suppressed and absent selections. Editing the returned
// slice or metadata cannot alter the captured selection.
func (s ExampleSelection) Entries() []ExampleEntry {
	if s.state != ExampleSelected {
		return nil
	}
	return append([]ExampleEntry(nil), s.entries...)
}

// Source returns the opaque captured value source for this group entry.
func (e ExampleEntry) Source() ValueSource {
	return e.source
}

// Summary returns the authored short summary.
func (e ExampleEntry) Summary() string {
	return e.summary
}

// Description returns the authored long description.
func (e ExampleEntry) Description() string {
	return e.description
}

// Meta returns detached design-time metadata for this entry.
func (e ExampleEntry) Meta() MetaExpr {
	return copyValueMeta(e.meta)
}
