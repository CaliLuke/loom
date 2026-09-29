package valuecontract

import (
	"encoding/json/jsontext"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type productionTargetCase struct {
	name    string
	command any
	decoded any
	schema  bool
}

func checkProductionTargetNumericConformance(t *testing.T, executable string) {
	t.Helper()
	texts := []string{"0", "-0", "+1", "01", "1.0", "1e0", "0.1", "1.0000000000000001", "2147483647", "2147483648", "-2147483649", "4294967295", "4294967296", "9223372036854775807", "9223372036854775808", "18446744073709551615", "18446744073709551616", "-1", "1e39", "1e309", "1e-999"}
	policies := productionNumericPolicies()
	cases := make([]productionTargetCase, 0, len(texts)*20)
	for _, text := range texts {
		for _, mapping := range []bool{false, true} {
			for _, policy := range policies {
				cases = append(cases, productionTargetNumericCase(t, text, []productionNumericPolicy{policy}, mapping))
			}
			for _, pair := range [][2]int{{0, 1}, {2, 0}, {4, 5}, {1, 5}} {
				cases = append(cases, productionTargetNumericCase(t, text, []productionNumericPolicy{policies[pair[0]], policies[pair[1]]}, mapping))
			}
		}
	}
	commands := make([]any, len(cases))
	for index, tc := range cases {
		commands[index] = tc.command
	}
	results := runReference(t, executable, commands)
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := referenceDecode[map[string]jsontext.Value](t, results[index])
			require.Equal(t, "true", string(actual["targetsValid"]), "missing input/invalid target: %s", results[index])
			require.Equal(t, productionJSON(t, map[string]any{"ok": tc.schema}), productionCanonical(t, actual["schema"]))
			require.Equal(t, productionCanonical(t, jsontext.Value(productionJSON(t, map[string]any{"ok": tc.decoded}))), productionCanonical(t, actual["decoded"]))
		})
	}
}

func productionTargetNumericCase(t *testing.T, text string, policies []productionNumericPolicy, mapping bool) productionTargetCase {
	t.Helper()
	identity := func(index uint64) referenceIdentity {
		return referenceIdentity{Occurrence: index, Declaration: index}
	}
	root := identity(1)
	var wire any = referenceConstructor("number", map[string]any{"text": text})
	texts := []string{text}
	if mapping {
		wire = referenceConstructor("object", map[string]any{"fields": []any{[]any{text, referenceConstructor("text", map[string]any{"value": "value"})}}})
		texts = append(texts, "value")
	}
	codecs := productionTargetCodecs(t, texts, policies)
	targets := make([]any, 0, len(policies)+2)
	alternatives := make([]any, 0, len(policies))
	matches := []any{}
	schemaMatches := 0
	name := fmt.Sprintf("wire=%s/map=%t", text, mapping)
	for index, policy := range policies {
		name += "/" + policy.name
		id := root
		if len(policies) > 1 {
			id = identity(uint64(index) + 2)
		}
		rules := productionRules(nil, 0)
		rules["numericFormat"], rules["integerFormat"] = policy.decimalFormat(), policy.integerFormat()
		var target any = referenceConstructor("scalar", map[string]any{"encoding": "json", "kind": policy.kind(), "rules": rules})
		decoded, accepted := policy.scalar(text)
		schemaAccepted := productionNumericSchemaAccepted(t, text, policy)
		var value any
		if mapping {
			target = referenceConstructor("map", map[string]any{"key": referenceConstructor("scalar", map[string]any{"kind": policy.kind()}), "keyRules": rules, "child": identity(100), "length": productionLength(nil)})
			decoded, accepted = policy.key(text)
			schemaAccepted = true
			if accepted {
				scalar := productionDecodedScalar(t, decoded).(map[string]any)["scalar"].(map[string]any)["value"]
				value = referenceConstructor("map", map[string]any{"entries": []any{[]any{scalar, referenceConstructor("scalar", map[string]any{"value": referenceConstructor("string", map[string]any{"value": "value"})})}}})
			}
		} else if accepted {
			value = productionDecodedScalar(t, decoded)
		}
		branch := identity(uint64(index) + 10)
		if accepted {
			if len(policies) > 1 {
				value = referenceConstructor("union", map[string]any{"occurrence": root, "branch": branch, "payload": value})
			}
			matches = append(matches, value)
		}
		if schemaAccepted {
			schemaMatches++
		}
		targets = append(targets, productionTargetDeclaration(id, target, 0))
		alternatives = append(alternatives, map[string]any{"identity": branch, "wireName": policy.name, "child": id})
	}
	if mapping {
		stringTarget := referenceConstructor("scalar", map[string]any{"encoding": "json", "kind": "string", "rules": productionRules(nil, 0)})
		targets = append(targets, productionTargetDeclaration(identity(100), stringTarget, 0))
	}
	if len(policies) > 1 {
		targets = append(targets, productionTargetDeclaration(root, referenceConstructor("union", map[string]any{"occurrence": root, "style": "untagged", "alternatives": alternatives}), 1))
	}
	var decoded any
	if len(matches) == 1 {
		decoded = matches[0]
	}
	command := referenceConstructor("decode", map[string]any{"request": map[string]any{"targets": targets, "root": root, "wire": wire, "codecs": codecs, "checks": []any{}}})
	return productionTargetCase{name: name, command: command, decoded: decoded, schema: schemaMatches == 1}
}

func productionTargetDeclaration(identity referenceIdentity, target any, rank int) any {
	return map[string]any{"identity": identity, "expansionRank": rank, "target": target, "enumeration": nil, "schemaEnumeration": nil, "schemaAllowsUnknown": true, "decoderRejectsUnknown": false}
}
