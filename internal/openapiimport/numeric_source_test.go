package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNumericSourcePreservesYAMLComposition(t *testing.T) {
	for _, test := range []struct {
		name, source, want string
	}{
		{"scalar alias", "value: &bound 0x20000000000001\nminimum: *bound\n", "9007199254740993"},
		{"mapping merge", "value: &bound {minimum: 0x20000000000001}\n<<: *bound\n", "9007199254740993"},
		{"local override", "value: &bound {minimum: 1}\nminimum: 2\n<<: *bound\n", "2"},
		{"first merge wins", "first: &first {minimum: 1}\nsecond: &second {minimum: 2}\n<<: [*first, *second]\n", "1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			values, err := numericSources([]byte(test.source))
			require.NoError(t, err)
			source, present := values["#/minimum"]
			require.True(t, present)
			number, err := source.number()
			require.NoError(t, err)
			require.Equal(t, test.want, number.String())
		})
	}
}

func TestNumericSourceKeepsProtocolKinds(t *testing.T) {
	for _, test := range []struct {
		name, source string
		accepted     bool
	}{
		{"JSON number beyond finite carrier", `{"minimum":1e309}`, true},
		{"JSON quoted number", `{"minimum":"42"}`, false},
		{"YAML quoted number", "minimum: \"42\"\n", false},
		{"YAML explicit string", "minimum: !!str 42\n", false},
		{"YAML explicit float", "minimum: !!float \"42\"\n", true},
		{"YAML empty float", "minimum: !!float \"\"\n", false},
		{"YAML sign float", "minimum: !!float \"+\"\n", false},
		{"YAML dot float", "minimum: !!float \".\"\n", false},
		{"YAML plain tiny", "minimum: 1e-1000001\n", true},
		{"YAML hex float string", "minimum: 0x1.4p0\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			values, err := numericSources([]byte(test.source))
			require.NoError(t, err)
			_, err = values["#/minimum"].number()
			if test.accepted {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestNumericSourceReplacesMergedMappings(t *testing.T) {
	for _, source := range []string{
		"base: &base {schema: {minimum: 5, maximum: 10}}\n<<: *base\nschema: {maximum: 20}\n",
		"first: &first {schema: {maximum: 20}}\nsecond: &second {schema: {minimum: 5, maximum: 10}}\n<<: [*first, *second]\n",
	} {
		values, err := numericSources([]byte(source))
		require.NoError(t, err)
		require.NotContains(t, values, "#/schema/minimum")
		require.Equal(t, "20", values["#/schema/maximum"].text)
	}
}

func TestNumericSourceYAMLFloatSpellings(t *testing.T) {
	for _, test := range []struct{ source, want string }{
		{"01.5", "1.5"}, {"01e2", "100"}, {"-01.5", "-1.5"},
		{"+0001.50", "1.5"}, {"00.000", "0"}, {"01.5e-1000001", "15e-1000002"},
	} {
		t.Run(test.source, func(t *testing.T) {
			values, err := numericSources([]byte("minimum: !!float " + test.source + "\n"))
			require.NoError(t, err)
			number, err := values["#/minimum"].number()
			require.NoError(t, err)
			require.Equal(t, test.want, number.String())
		})
	}
}

func TestAnalyzeYAMLNumericSourceNormalization(t *testing.T) {
	prefix := "openapi: 3.1.1\ninfo: {title: Probe, version: '1'}\npaths: {}\n"
	for _, token := range []string{"01.5", "01e2", "-01.5"} {
		document, diagnostics, err := Analyze([]byte(prefix + "components:\n  schemas:\n    Bound: {type: number, minimum: " + token + "}\n"))
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		require.NotNil(t, document.Components.Schemas[0].Schema.Minimum)
	}
	document, diagnostics, err := Analyze([]byte(prefix + "x-base: &base\n  schemas:\n    Bound: {type: integer, minimum: 5, maximum: 10}\ncomponents:\n  <<: *base\n  schemas:\n    Bound: {type: integer, maximum: 20}\n"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Nil(t, document.Components.Schemas[0].Schema.Minimum)
	require.Equal(t, "20", document.Components.Schemas[0].Schema.Maximum.String())
}

func TestNumericSourceAliasedMergeSequences(t *testing.T) {
	for _, test := range []struct {
		name, source, want string
	}{
		{"alias sequence", "base: &base {minimum: 5}\nsources: &sources [*base]\n<<: *sources\n", "5"},
		{"first mapping wins", "first: &first {schema: {maximum: 20}}\nsecond: &second {schema: {minimum: 5}}\nsources: &sources [*first, *second]\n<<: *sources\n", ""},
		{"explicit subtree wins", "base: &base {schema: {minimum: 5}}\nsources: &sources [*base]\n<<: *sources\nschema: {maximum: 20}\n", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			values, err := numericSources([]byte(test.source))
			require.NoError(t, err)
			if test.want != "" {
				require.Equal(t, test.want, values["#/minimum"].text)
			} else {
				require.NotContains(t, values, "#/schema/minimum")
				require.Equal(t, "20", values["#/schema/maximum"].text)
			}
		})
	}
	_, err := numericSources([]byte("base: &base {<<: *base}\n<<: *base\n"))
	require.ErrorContains(t, err, "cyclic merge")
	_, err = numericSources([]byte("base: &base [{minimum: 5}]\nsources: &sources [*base]\n<<: *sources\n"))
	require.ErrorContains(t, err, "merge requires a mapping")
}

func TestAnalyzeYAMLAliasedMergeSequence(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(`openapi: 3.1.1
info: {title: Probe, version: '1'}
paths: {}
x-base: &base
  schemas:
    Bound: {type: integer, minimum: 5, maximum: 10}
x-sources: &sources [*base]
components:
  <<: *sources
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Equal(t, "5", document.Components.Schemas[0].Schema.Minimum.String())
	require.Equal(t, "10", document.Components.Schemas[0].Schema.Maximum.String())
}
