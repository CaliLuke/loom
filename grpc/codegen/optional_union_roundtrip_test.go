package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/loomsource"
)

// TestGeneratedOptionalUnionRoundTrip compiles a generated module whose
// payload and result carry optional unions directly, nested in an object, in
// an array and in a map. It round-trips the service values through the
// generated protobuf conversions with every union unset and set to each
// branch, and fails when a conversion panics or loses a value.
func TestGeneratedOptionalUnionRoundTrip(t *testing.T) {
	const modulePath = "example.com/grpcoptionalunion"
	root := RunGRPCDSL(t, optionalUnionDSL)
	dir := t.TempDir()
	loomSource := resolveGRPCLoomSource(t)
	renderGRPCModule(t, dir, modulePath, root, loomSource)
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(optionalUnionRoundTripHarness), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}

// TestOptionalUnionTransform checks that optional unions are nil-checked
// before conversion to protobuf and allocated before a branch setter runs on
// conversion from protobuf, while required unions keep value semantics.
func TestOptionalUnionTransform(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		Type("Mixed", func() {
			OneOf("optional_choice", func() {
				Field(1, "text", String)
				Field(2, "num", Int)
			})
			OneOf("required_choice", func() {
				Field(3, "flag", Boolean)
				Field(4, "count", Int)
			})
			Required("required_choice")
		})
	})
	sd := &ServiceData{Name: "Svc", Scope: codegen.NewNameScope()}
	svcCtx := serviceTypeContext("svc", sd.Scope)
	pbCtx := protoBufTypeContext("pb", sd.Scope, true)
	service := &expr.AttributeExpr{Type: root.UserType("Mixed")}
	message := makeProtoBufMessage(expr.DupAtt(service), service.Type.Name(), sd)

	cases := []struct {
		name      string
		toProto   bool
		contains  []string
		forbidden []string
	}{
		{
			name:    "service to protobuf",
			toProto: true,
			contains: []string{
				"if source.OptionalChoice != nil {",
				`if source.RequiredChoice.Kind() != "" {`,
			},
			forbidden: []string{`source.OptionalChoice.Kind() != ""`},
		},
		{
			name:    "protobuf to service",
			toProto: false,
			contains: []string{
				"u := &svc.OptionalChoice{}",
				"u := target.RequiredChoice",
			},
			forbidden: []string{"u := target.OptionalChoice"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			source, target, sourceCtx, targetCtx := message, service, pbCtx, svcCtx
			if c.toProto {
				source, target, sourceCtx, targetCtx = service, message, svcCtx, pbCtx
			}
			code, _, err := protoBufTransform(source, target, "source", "target", sourceCtx, targetCtx, c.toProto, true)
			require.NoError(t, err)
			for _, want := range c.contains {
				require.Contains(t, code, want)
			}
			for _, unwanted := range c.forbidden {
				require.NotContains(t, code, unwanted)
			}
		})
	}
}

// resolveGRPCLoomSource returns the Loom checkout selected by LOOM_DIR or the
// persisted loom-local/loom-remote mode.
func resolveGRPCLoomSource(t *testing.T) string {
	t.Helper()
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	source, err := loomsource.Resolve(repoRoot, filepath.Join(t.TempDir(), "loom-pinned"))
	require.NoError(t, err)
	t.Logf("using Loom source: %s", source)
	return source
}

func optionalUnionDSL() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Holder = Type("Holder", func() {
		Field(1, "label", String)
		OneOf("detail", func() {
			Field(2, "text", String)
			Field(3, "num", Int)
			Field(4, "leaf", Leaf)
		})
	})
	var Strict = Type("Strict", func() {
		OneOf("mode", func() {
			Field(1, "fast", String)
			Field(2, "slow", Int)
		})
		Required("mode")
	})
	var Envelope = Type("Envelope", func() {
		Field(1, "id", String)
		OneOf("choice", func() {
			Field(2, "text", String)
			Field(3, "num", Int)
			Field(4, "leaf", Leaf)
		})
		Field(5, "holder", Holder)
		Field(6, "holders", ArrayOf(Holder))
		Field(7, "holder_map", MapOf(String, Holder))
		Field(8, "strict", Strict)
	})
	Service("optunion", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			GRPC(func() {})
		})
	})
}

var optionalUnionRoundTripHarness = fmt.Sprintf(`package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	optunion "%[1]s/gen/optunion"
	"%[1]s/gen/grpc/optunion/client"
	"%[1]s/gen/grpc/optunion/server"
)

func choices() map[string]*optunion.Choice {
	text := &optunion.Choice{}
	text.SetText(optunion.ChoiceText("hello"))
	num := &optunion.Choice{}
	num.SetNum(optunion.ChoiceNum(42))
	leaf := &optunion.Choice{}
	leafName := "leaf"
	leaf.SetLeaf(&optunion.Leaf{Name: &leafName})
	return map[string]*optunion.Choice{"unset": nil, "text": text, "num": num, "leaf": leaf}
}

func holderChoices() map[string]*optunion.Detail {
	text := &optunion.Detail{}
	text.SetText(optunion.DetailText("hello"))
	num := &optunion.Detail{}
	num.SetNum(optunion.DetailNum(42))
	leaf := &optunion.Detail{}
	leafName := "leaf"
	leaf.SetLeaf(&optunion.Leaf{Name: &leafName})
	return map[string]*optunion.Detail{"unset": nil, "text": text, "num": num, "leaf": leaf}
}

func envelopes() map[string]*optunion.Envelope {
	id := "id"
	cases := map[string]*optunion.Envelope{}
	for name, choice := range choices() {
		cases["top/"+name] = &optunion.Envelope{ID: &id, Choice: choice}
	}
	for name, choice := range holderChoices() {
		label := "label"
		holder := &optunion.Holder{Label: &label, Detail: choice}
		cases["object/"+name] = &optunion.Envelope{Holder: holder}
		cases["array/"+name] = &optunion.Envelope{Holders: []*optunion.Holder{holder}}
		cases["map/"+name] = &optunion.Envelope{HolderMap: map[string]*optunion.Holder{"k": holder}}
	}
	fast := &optunion.Strict{}
	fast.Mode.SetFast(optunion.ModeFast("fast"))
	cases["required/fast"] = &optunion.Envelope{Strict: fast}
	slow := &optunion.Strict{}
	slow.Mode.SetSlow(optunion.ModeSlow(7))
	cases["required/slow"] = &optunion.Envelope{Strict: slow}
	return cases
}

func TestPayloadRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var decoded *optunion.Envelope
			require.NotPanics(t, func() {
				decoded = server.NewEchoPayload(client.NewProtoEchoRequest(envelope))
			})
			require.Equal(t, envelope, decoded)
		})
	}
}

func TestResultRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var decoded *optunion.Envelope
			require.NotPanics(t, func() {
				decoded = client.NewEchoResult(server.NewProtoEchoResponse(envelope))
			})
			require.Equal(t, envelope, decoded)
		})
	}
}
`, "example.com/grpcoptionalunion")
