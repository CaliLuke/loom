package ir

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAcquisitionCheckpointDoesNotCopyPublishedIndexes(t *testing.T) {
	var allocations []float64
	for _, size := range []int{16, 2048} {
		analyzer := NewAnalyzer(expr.NewRandom("checkpoint"), false)
		analyzer.componentAnnotations = make(map[componentAnnotationIdentity]map[componentAnnotationPosition]componentExamples)
		for index := range size {
			name := fmt.Sprintf("prior-%d", index)
			analyzer.schemas[name] = &Schema{}
			analyzer.componentAnnotations[componentAnnotationIdentity{declaration: name}] = map[componentAnnotationPosition]componentExamples{
				{context: name}: {},
			}
		}
		allocated := testing.AllocsPerRun(5, func() {
			rollback := analyzer.acquisitionRollback()
			rollback()
		})
		allocations = append(allocations, allocated)
		t.Logf("published entries=%d checkpoint allocations=%.0f", size, allocated)
		require.Len(t, analyzer.schemas, size)
		require.Len(t, analyzer.componentAnnotations, size)
	}
	require.Less(t, allocations[1], 2*allocations[0], "a checkpoint must own changed entries, not copy prior published indexes")
}

func TestAcquisitionJournalScopeOutcomes(t *testing.T) {
	for _, outcome := range []string{"success", "error", "panic"} {
		t.Run(outcome, func(t *testing.T) {
			failure := errors.New("scope failure")
			a := NewAnalyzer(expr.NewRandom("journal"), false)
			prior := &Schema{Type: "string"}
			a.schemas["Prior"] = prior
			a.schemas["Nil"] = nil
			a.schemasByFingerprint["F"] = make([]schemaRef, 1, 8)
			a.schemasByFingerprint["F"][0] = schemaRef{ref: toRef("Prior"), explicitName: "Prior"}
			backing := a.schemasByFingerprint["F"][:8]
			priorBacking := slices.Clone(backing)
			references := slices.Clone(a.schemasByFingerprint["F"])
			identity := componentAnnotationIdentity{declaration: "Prior"}
			position := componentAnnotationPosition{context: "before"}
			a.componentAnnotations = map[componentAnnotationIdentity]map[componentAnnotationPosition]componentExamples{
				identity: {position: {representative: "prior"}},
			}
			annotations := maps.Clone(a.componentAnnotations[identity])
			apply := func() {
				defer a.schemaAnalysisScope()()
				setAcquisitionEntry(a, a.schemas, "Prior", &Schema{Type: "integer"})
				setAcquisitionEntry(a, a.schemas, "Nil", &Schema{Type: "integer"})
				setAcquisitionEntry(a, a.schemas, "New", &Schema{Type: "integer"})
				deleteAcquisitionEntry(a, a.schemas, "Prior")
				func() {
					defer a.schemaAnalysisScope()()
					setAcquisitionEntry(a, a.schemas, "Prior", &Schema{Type: "boolean"})
					a.registerComponentReference("F", toRef("Second"), "Second", false)
					a.registerComponentReference("F", toRef("Prior"), "Prior", true)
					setAcquisitionEntry(a, a.componentAnnotations[identity], position, componentExamples{representative: "changed"})
					a.addDiscriminatorChild("Prior", "Child")
				}()
				require.NotNil(t, a.acquisitionJournal, "nested scope cannot release the outer journal")
				switch outcome {
				case "error":
					a.err = failure
				case "panic":
					panic(failure)
				}
			}
			if outcome == "panic" {
				require.PanicsWithValue(t, failure, apply)
			} else {
				apply()
			}
			require.Zero(t, a.schemaDepth)
			require.Nil(t, a.acquisitionJournal, "completion must release undo captures")
			if outcome == "success" {
				require.Equal(t, "boolean", a.schemas["Prior"].Type)
				require.Equal(t, "Prior", a.schemasByFingerprint["F"][0].canonicalClaim)
				require.Len(t, a.schemasByFingerprint["F"], 2)
				require.NotNil(t, a.schemas["Nil"])
				require.Contains(t, a.schemas, "New")
				require.Equal(t, "changed", a.componentAnnotations[identity][position].representative)
				require.Equal(t, []string{"Child"}, a.discriminatorChildren["Prior"])
				return
			}
			require.Same(t, prior, a.schemas["Prior"])
			require.Contains(t, a.schemas, "Nil")
			require.Nil(t, a.schemas["Nil"])
			require.NotContains(t, a.schemas, "New")
			require.Equal(t, references, a.schemasByFingerprint["F"])
			require.Equal(t, priorBacking, backing, "rolled-back bindings must not remain retained in prior spare capacity")
			require.Equal(t, annotations, a.componentAnnotations[identity])
			require.Nil(t, a.discriminatorChildren, "rollback must restore a lazily allocated nil index")
			if outcome == "error" {
				require.ErrorIs(t, a.err, failure)
			}
		})
	}
}
