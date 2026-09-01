package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInspectSourceRejectsMarshalWithoutDeterministicOption(t *testing.T) {
	source := []byte(`package specimen

import json "encoding/json/v2"

type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"zulu": true, "alpha": true})
}
`)

	findings, err := inspectSource("specimen.go", source)
	require.NoError(t, err)
	require.Equal(t, []finding{{Path: "specimen.go", Line: 8, Column: 9}}, findings)
}

func TestInspectSourceRejectsUnrelatedMarshalOption(t *testing.T) {
	source := []byte(`package specimen

import json "encoding/json/v2"

type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"zulu": true, "alpha": true}, json.FormatNilMapAsNull(true))
}
`)

	findings, err := inspectSource("specimen.go", source)
	require.NoError(t, err)
	require.Equal(t, []finding{{Path: "specimen.go", Line: 8, Column: 9}}, findings)
}

func TestInspectSourceAcceptsExplicitOptionsAndUnrelatedHelpers(t *testing.T) {
	source := []byte(`package specimen

import json "encoding/json/v2"

type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	if false {
		return helper.Marshal(map[string]any{"zulu": true, "alpha": true})
	}
	return json.Marshal(
		map[string]any{"zulu": true, "alpha": true},
		json.Deterministic(true),
	)
}
`)

	findings, err := inspectSource("specimen.go", source)
	require.NoError(t, err)
	require.Empty(t, findings)
}

func TestInspectSourceParsesGeneratedDeclarationFragments(t *testing.T) {
	source := []byte(`import json "encoding/json/v2"

type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"zulu": true, "alpha": true})
}
`)

	findings, err := inspectSource("specimen.golden", source)
	require.NoError(t, err)
	require.Equal(t, []finding{{Path: "specimen.golden", Line: 6, Column: 9}}, findings)
}

func TestInspectSourceChecksGeneratedFragmentsWithoutImports(t *testing.T) {
	source := []byte(`type Value struct{}

func (Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"zulu": true, "alpha": true})
}
`)

	findings, err := inspectSource("specimen.golden", source)
	require.NoError(t, err)
	require.Equal(t, []finding{{Path: "specimen.golden", Line: 4, Column: 9}}, findings)
}
