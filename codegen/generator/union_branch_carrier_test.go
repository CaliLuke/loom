package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

func TestUnionBranchCarrierValidation(t *testing.T) {
	runDesignHarness(t, "example.com/branchcarrier", unionBranchCarrierDSL, unionBranchCarrierHarness)
}

func unionBranchCarrierDSL() {
	dsl.API("branchcarrier", func() {
	})
	leaf := dsl.Type("Leaf", func() {
		dsl.Attribute("name", dsl.String, func() {
			dsl.MinLength(1)
		})
		dsl.Required("name")
	})
	emptyObject := dsl.Type("EmptyObject", func() {
	})
	choice := dsl.Type("Choice", dsl.OneOf(leaf, dsl.String))
	list := dsl.Type("List", dsl.ArrayOf(dsl.String), func() {
		dsl.MaxLength(2)
	})
	tightList := dsl.Type("TightList", dsl.ArrayOf(dsl.String), func() {
		dsl.MinLength(1)
	})
	dict := dsl.Type("Dict", dsl.MapOf(dsl.String, dsl.String), func() {
		dsl.MaxLength(2)
	})
	tightDict := dsl.Type("TightDict", dsl.MapOf(dsl.String, dsl.String), func() {
		dsl.MinLength(1)
	})
	holder := dsl.Type("Holder", func() {
		dsl.OneOf("branch", func() {
			dsl.Attribute("nested", choice)
			dsl.Attribute("nullable", choice, func() {
				dsl.Nullable()
			})
			dsl.Attribute("list", list)
			dsl.Attribute("tight_list", tightList)
			dsl.Attribute("dict", dict)
			dsl.Attribute("tight_dict", tightDict)
			dsl.Attribute("text", dsl.String, func() {
				dsl.MinLength(1)
			})
			dsl.Attribute("object", leaf)
			dsl.Attribute("empty_object", emptyObject)
		})
		dsl.Required("branch")
	})
	dsl.Service("carrier", func() {
		dsl.Method("show", func() {
			dsl.Payload(holder)
			dsl.Result(holder)
			dsl.HTTP(func() {
				dsl.POST("/show")
			})
		})
	})
}

const unionBranchCarrierHarness = `package branchcarrier

import (
	"encoding/json/v2"
	"strings"
	"testing"

	client "example.com/branchcarrier/gen/http/carrier/client"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestConstructedBranches(t *testing.T) {
	name := "valid"
	empty := ""
	invalidChoice := client.NewChoiceResponseBodyLeaf(&client.LeafResponseBody{Name: &empty})
	choice := client.NewChoiceResponseBodyLeaf(&client.LeafResponseBody{Name: &name})
	for _, test := range []struct {
		name    string
		branch  client.Branch2
		invalid bool
	}{
		{"nested_nil", client.NewBranch2Nested(nil), true},
		{"nested_invalid", client.NewBranch2Nested(&invalidChoice), true},
		{"nested_valid", client.NewBranch2Nested(&choice), false},
		{"empty_object_nil", client.NewBranch2EmptyObject(nil), true},
		{"empty_object_valid", client.NewBranch2EmptyObject(&client.EmptyObjectResponseBody{}), false},
		{"object_nil", client.NewBranch2Object(nil), true},
		{"object_invalid", client.NewBranch2Object(&client.LeafResponseBody{Name: &empty}), true},
		{"object_valid", client.NewBranch2Object(&client.LeafResponseBody{Name: &name}), false},
		{"nullable_absent", client.NewBranch2Nullable(loom.Nullable[client.ChoiceResponseBody]{}), true},
		{"nullable_null", client.NewBranch2Nullable(loom.NullValue[client.ChoiceResponseBody]()), false},
		{"nullable_invalid", client.NewBranch2Nullable(loom.NullableValue(invalidChoice)), true},
		{"nullable_value", client.NewBranch2Nullable(loom.NullableValue(choice)), false},
		{"list_nil", client.NewBranch2List(nil), false},
		{"list_empty", client.NewBranch2List(client.ListResponseBody{}), false},
		{"list_long", client.NewBranch2List(client.ListResponseBody{loom.OptionalValue("a"), loom.OptionalValue("b"), loom.OptionalValue("c")}), true},
		{"tight_list_nil", client.NewBranch2TightList(nil), true},
		{"tight_list_empty", client.NewBranch2TightList(client.TightListResponseBody{}), true},
		{"tight_list_valid", client.NewBranch2TightList(client.TightListResponseBody{loom.OptionalValue("ok")}), false},
		{"dict_nil", client.NewBranch2Dict(nil), false},
		{"dict_empty", client.NewBranch2Dict(client.DictResponseBody{}), false},
		{"dict_long", client.NewBranch2Dict(client.DictResponseBody{"a": loom.OptionalValue("a"), "b": loom.OptionalValue("b"), "c": loom.OptionalValue("c")}), true},
		{"tight_dict_nil", client.NewBranch2TightDict(nil), true},
		{"tight_dict_empty", client.NewBranch2TightDict(client.TightDictResponseBody{}), true},
		{"tight_dict_valid", client.NewBranch2TightDict(client.TightDictResponseBody{"key": loom.OptionalValue("ok")}), false},
		{"text_empty", client.NewBranch2Text(""), true},
		{"text_valid", client.NewBranch2Text("ok"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var err error
			require.NotPanics(t, func() {
				err = client.ValidateShowResponseBody(&client.ShowResponseBody{Branch: test.branch})
			})
			if test.invalid {
				require.Error(t, err)
				var serviceError *loom.ServiceError
				require.ErrorAs(t, err, &serviceError)
				if strings.HasPrefix(test.name, "tight_") || strings.HasSuffix(test.name, "_long") {
					require.Equal(t, loom.InvalidLength, serviceError.Name)
				}
				if test.name == "nested_nil" || test.name == "object_nil" || test.name == "empty_object_nil" || test.name == "nullable_absent" {
					require.Equal(t, loom.MissingField, serviceError.Name)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCollectionWireValues(t *testing.T) {
	for _, test := range []struct {
		name     string
		branch   client.Branch2
		expected string
	}{
		{"nil_list", client.NewBranch2List(nil), "{\"type\":\"list\",\"value\":[]}"},
		{"empty_list", client.NewBranch2List(client.ListResponseBody{}), "{\"type\":\"list\",\"value\":[]}"},
		{"nil_dict", client.NewBranch2Dict(nil), "{\"type\":\"dict\",\"value\":{}}"},
		{"empty_dict", client.NewBranch2Dict(client.DictResponseBody{}), "{\"type\":\"dict\",\"value\":{}}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			wire, err := json.Marshal(test.branch)
			require.NoError(t, err)
			require.JSONEq(t, test.expected, string(wire))
			var decoded client.Branch2
			require.NoError(t, json.Unmarshal(wire, &decoded))
			require.NoError(t, client.ValidateShowResponseBody(&client.ShowResponseBody{Branch: decoded}))
		})
	}
}

func TestWireNullBranches(t *testing.T) {
	for _, test := range []struct {
		name, wire string
		invalid    bool
	}{
		{"nested", "{\"type\":\"nested\",\"value\":null}", true},
		{"object", "{\"type\":\"object\",\"value\":null}", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var branch client.Branch2
			err := json.Unmarshal([]byte(test.wire), &branch)
			if test.invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NoError(t, client.ValidateShowResponseBody(&client.ShowResponseBody{Branch: branch}))
			}
		})
	}
}
`
