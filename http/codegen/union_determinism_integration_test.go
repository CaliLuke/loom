package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedUnionMarshalersPreserveNestedMapOrdering(t *testing.T) {
	const modulePath = "example.com/uniondeterminism"
	root := RunHTTPDSL(t, unionDeterminismDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "gen", "union_determinism", "union_determinism_test.go"),
		[]byte(serviceUnionDeterminismHarness),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "gen", "http", "union_determinism", "client", "union_determinism_test.go"),
		[]byte(httpUnionDeterminismHarness),
		0o600,
	))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./gen/union_determinism", "./gen/http/union_determinism/client")
}

func unionDeterminismDSL() {
	tagged := Type("TaggedChoice", func() {
		OneOf("value", func() {
			Attribute("mapping", MapOf(String, String))
			Attribute("fallback", String)
		})
	})
	nestedMap := Type("NestedMap", func() {
		Attribute("values", MapOf(String, String))
		Required("values")
	})
	untaggedMap := Type("UntaggedMap", func() {
		Attribute("payload", nestedMap)
		Required("payload")
	})
	untaggedFallback := Type("UntaggedFallback", func() {
		Attribute("message", String)
		Required("message")
	})
	untaggedEnvelope := Type("UntaggedEnvelope", func() {
		Attribute("choice", OneOf(untaggedMap, untaggedFallback), func() {
			Untagged()
		})
		Required("choice")
	})

	Service("UnionDeterminism", func() {
		Method("Tagged", func() {
			Result(tagged)
			HTTP(func() {
				GET("/tagged")
				Response(StatusOK)
			})
		})
		Method("Untagged", func() {
			Result(untaggedEnvelope)
			HTTP(func() {
				GET("/untagged")
				Response(StatusOK)
			})
		})
	})
}

const serviceUnionDeterminismHarness = `package uniondeterminism

import (
	json "encoding/json/v2"
	"testing"
)

func TestServiceUnionDeterminism(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target any
		want   string
	}{
		{
			name:   "tagged",
			input:  "{\"value\":{\"type\":\"mapping\",\"value\":{\"zulu\":\"last\",\"alpha\":\"first\"}}}",
			target: new(TaggedChoice),
			want:   "{\"value\":{\"type\":\"mapping\",\"value\":{\"alpha\":\"first\",\"zulu\":\"last\"}}}",
		},
		{
			name:   "untagged",
			input:  "{\"choice\":{\"payload\":{\"values\":{\"zulu\":\"last\",\"alpha\":\"first\"}}}}",
			target: new(UntaggedEnvelope),
			want:   "{\"choice\":{\"payload\":{\"values\":{\"alpha\":\"first\",\"zulu\":\"last\"}}}}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertDeterministicUnion(t, test.input, test.target, test.want)
		})
	}
}

func assertDeterministicUnion(t *testing.T, input string, target any, want string) {
	t.Helper()
	if err := json.Unmarshal([]byte(input), target); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for range 20 {
		encoded, err := json.Marshal(target, json.Deterministic(true))
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(encoded) != want {
			t.Errorf("encoded = %s, want %s", encoded, want)
		}
	}
}
`

const httpUnionDeterminismHarness = `package client

import (
	json "encoding/json/v2"
	"testing"
)

func TestHTTPUnionDeterminism(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target any
		want   string
	}{
		{
			name:   "tagged",
			input:  "{\"value\":{\"type\":\"mapping\",\"value\":{\"zulu\":\"last\",\"alpha\":\"first\"}}}",
			target: new(TaggedResponseBody),
			want:   "{\"value\":{\"type\":\"mapping\",\"value\":{\"alpha\":\"first\",\"zulu\":\"last\"}}}",
		},
		{
			name:   "untagged",
			input:  "{\"choice\":{\"payload\":{\"values\":{\"zulu\":\"last\",\"alpha\":\"first\"}}}}",
			target: new(UntaggedResponseBody),
			want:   "{\"choice\":{\"payload\":{\"values\":{\"alpha\":\"first\",\"zulu\":\"last\"}}}}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertDeterministicUnion(t, test.input, test.target, test.want)
		})
	}
}

func assertDeterministicUnion(t *testing.T, input string, target any, want string) {
	t.Helper()
	if err := json.Unmarshal([]byte(input), target); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for range 20 {
		encoded, err := json.Marshal(target, json.Deterministic(true))
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(encoded) != want {
			t.Errorf("encoded = %s, want %s", encoded, want)
		}
	}
}
`
