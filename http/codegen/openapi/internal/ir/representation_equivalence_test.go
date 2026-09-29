package ir

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testdata"
)

type representationAnnotationCounter struct {
	calls *int
}

func (c representationAnnotationCounter) MarshalJSON() ([]byte, error) {
	*c.calls++
	return []byte(`"owned"`), nil
}

func TestRepresentationEquivalenceMixedPublicFixture(t *testing.T) {
	root := codegen.RunDSL(t, testdata.BytesRepresentationDSL)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	node := document.Components.Schemas["RepresentationNode"]
	require.NotNil(t, node)
	require.Equal(t, toRef("RepresentationNode"), node.Properties["next"].Ref)
	for name := range document.Components.Schemas {
		require.False(t, strings.HasPrefix(name, "RepresentationNode_"), "equivalent recursive declaration duplicated as %s", name)
	}
	require.Equal(t, "base64", document.Components.Schemas["PublicRepresentationBlob"].ContentEncoding)
}

func TestRepresentationGraphCapturesReferencePaths(t *testing.T) {
	makeGraph := func(reverse bool) map[string]*Schema {
		properties := make(map[string]*Schema)
		if reverse {
			properties["second"] = &Schema{Ref: toRef("number")}
			properties["first"] = &Schema{Ref: toRef("text")}
		} else {
			properties["first"] = &Schema{Ref: toRef("text")}
			properties["second"] = &Schema{Ref: toRef("number")}
		}
		return map[string]*Schema{
			"root":   {Type: "object", Properties: properties},
			"text":   {Type: "string"},
			"number": {Type: "integer"},
		}
	}
	first, second := makeGraph(false), makeGraph(true)
	require.Equal(t, captureRepresentationGraph(first), captureRepresentationGraph(second))
	require.Equal(t, completeSchemaFingerprints(first), completeSchemaFingerprints(second))
	second["root"].Properties["first"], second["root"].Properties["second"] = second["root"].Properties["second"], second["root"].Properties["first"]
	require.NotEqual(t, completeSchemaFingerprints(first)["root"], completeSchemaFingerprints(second)["root"], "reference paths are part of the contract")
	first["root"] = &Schema{OneOf: []*Schema{{Ref: toRef("text")}, {Ref: toRef("number")}}}
	second["root"] = &Schema{OneOf: []*Schema{{Ref: toRef("number")}, {Ref: toRef("text")}}}
	require.NotEqual(t, completeSchemaFingerprints(first)["root"], completeSchemaFingerprints(second)["root"], "reference list order must survive capture")
}

func TestRepresentationGraphSnapshotsAnnotationsOnce(t *testing.T) {
	calls := 0
	schemas := map[string]*Schema{
		"first":  {Type: "object", Properties: map[string]*Schema{"next": {Ref: toRef("second")}}, Example: representationAnnotationCounter{calls: &calls}},
		"second": {Type: "object", Properties: map[string]*Schema{"next": {Ref: toRef("third")}}},
		"third":  {Type: "object", Properties: map[string]*Schema{"next": {Ref: toRef("third")}}},
	}
	fingerprints := completeSchemaFingerprints(schemas)
	require.Len(t, fingerprints, 3)
	require.Equal(t, 1, calls, "refinement and quotient traversal must not re-run annotation codecs")
	require.NotEqual(t, fingerprints["first"], fingerprints["second"])
	require.Equal(t, fingerprints["second"], fingerprints["third"])
}

func TestRepresentationEquivalencePreservesAcyclicNames(t *testing.T) {
	a := NewAnalyzer(nil, false)
	a.schemas = map[string]*Schema{
		"first":  {Type: "object", Properties: map[string]*Schema{"value": {Ref: toRef("childA")}}},
		"second": {Type: "object", Properties: map[string]*Schema{"value": {Ref: toRef("childB")}}},
		"childA": {Type: "string", Description: "22"},
		"childB": {Type: "integer"},
	}
	for _, name := range []string{"first", "second"} {
		a.schemaNames[name] = representationComponentName{desired: "Envelope", logical: name}
	}
	first, second := &Schema{Ref: toRef("first")}, &Schema{Ref: toRef("second")}
	a.occurrences = []schemaOccurrenceAnalysis{{first, a}, {second, a}}
	// These hashes are the prior exact acyclic serialization, with each local
	// reference replaced by "schema:" plus its child's complete fingerprint.
	fingerprints := completeSchemaFingerprints(a.schemas)
	require.Equal(t, "0fff809809bc720d6e2657c74ab6755413e71365a3d3a1bfbf4488bbf8fdfcd2", fingerprints["first"])
	require.Equal(t, "178fe49349f583acdb9b450c85923fa0b647f36046f505efe82c6f0e9441f038", fingerprints["second"])
	a.finalizeRepresentations()
	require.Equal(t, toRef("Envelope"), first.Ref)
	require.Equal(t, toRef("Envelope_178fe49349f583ac"), second.Ref)
	require.Equal(t, toRef("childA"), a.schemas["Envelope"].Properties["value"].Ref)
}

func TestRepresentationGraphReferenceSlotsPreserveAnnotations(t *testing.T) {
	lookalike := map[string]any{"Ref": toRef("target"), "slot": "local:0", "exact": jsontext.Value("9007199254740993")}
	schema := &Schema{
		Ref:           toRef("target"),
		Example:       lookalike,
		Items:         &Schema{Ref: toRef("target")},
		ContentSchema: &Schema{Ref: toRef("target")},
		Not:           &Schema{Ref: toRef("target")},
		Properties: map[string]*Schema{
			"escaped/~field": {Ref: toRef("target"), Example: lookalike},
		},
		Defs:                  map[string]*Schema{"local": {Ref: toRef("target")}},
		AllOf:                 []*Schema{{Ref: toRef("target")}},
		AnyOf:                 []*Schema{{Ref: toRef("target")}},
		OneOf:                 []*Schema{{Ref: toRef("target")}},
		AdditionalProperties:  &BoolOrSchema{Schema: &Schema{Ref: toRef("target")}},
		UnevaluatedProperties: &BoolOrSchema{Schema: &Schema{Ref: toRef("target")}},
		Discriminator:         &Discriminator{Mapping: map[string]string{"escaped/~tag": toRef("target")}},
	}
	graph := captureRepresentationGraph(map[string]*Schema{"root": schema, "target": {Type: "string"}})
	node := graph["root"]
	require.Equal(t, []string{
		"/Ref", "/Items/Ref", "/ContentSchema/Ref", "/Not/Ref",
		"/Properties/escaped~1~0field/Ref", "/Defs/local/Ref",
		"/AllOf/0/Ref", "/AnyOf/0/Ref", "/OneOf/0/Ref",
		"/AdditionalProperties/Schema/Ref", "/UnevaluatedProperties/Schema/Ref",
		"/Discriminator/Mapping/escaped~1~0tag",
	}, node.paths)
	encoded, err := json.Marshal(schema, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, string(encoded), rewriteCapturedReferences(node.encoded, nil))
	references := make(map[string]string, len(node.paths))
	for _, path := range node.paths {
		references[path] = "schema:child"
	}
	expected := mapSchemaReferences(schema, func(ref, _ string) string {
		if ref == toRef("target") {
			return "schema:child"
		}
		return ref
	})
	expectedJSON, err := json.Marshal(expected, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, string(expectedJSON), rewriteCapturedReferences(node.encoded, references))
	require.Contains(t, string(expectedJSON), `9007199254740993`)
	require.Contains(t, string(expectedJSON), `"Ref":"#/components/schemas/target"`, "annotation lookalikes stay literal")
}
