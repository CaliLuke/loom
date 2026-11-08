package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestHTTPBodyProjectionRecordsAuthoredSource checks that every body lowering
// records the authored source it flattens separately from the members it
// removes: implicit request and response bodies record the declaration itself,
// an explicit Body member records its selected member's declaration contract,
// and an explicit inline subset records a source that authors no composition.
func TestHTTPBodyProjectionRecordsAuthoredSource(t *testing.T) {
	for _, tc := range []struct {
		name     string
		body     func(root *expr.RootExpr) *expr.AttributeExpr
		source   func(root *expr.RootExpr) *expr.AttributeExpr
		excluded []string
		bases    bool
	}{
		{
			name:     "implicit request",
			body:     func(root *expr.RootExpr) *expr.AttributeExpr { return httpEndpoint(root, "submit").Body },
			source:   func(root *expr.RootExpr) *expr.AttributeExpr { return root.UserType("Child").Attribute() },
			excluded: []string{"id"},
			bases:    true,
		},
		{
			name:     "implicit response",
			body:     func(root *expr.RootExpr) *expr.AttributeExpr { return httpEndpoint(root, "show").Responses[0].Body },
			source:   func(root *expr.RootExpr) *expr.AttributeExpr { return root.UserType("Child").Attribute() },
			excluded: []string{"id"},
			bases:    true,
		},
		{
			name:  "explicit member",
			body:  func(root *expr.RootExpr) *expr.AttributeExpr { return httpEndpoint(root, "selected").Body },
			bases: true,
		},
		{
			name: "explicit inline subset",
			body: func(root *expr.RootExpr) *expr.AttributeExpr { return httpEndpoint(root, "subset").Body },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, bodyProjectionDesign)
			projection := tc.body(root).HTTPBodyProjection()
			require.NotNil(t, projection)
			if tc.source != nil {
				require.Same(t, tc.source(root), projection.Source)
			}
			require.Equal(t, tc.excluded, projection.Excluded)
			require.Equal(t, tc.bases, len(projection.Source.CompositionReferenceBases()) > 0)
			if len(projection.Excluded) > 0 {
				projection.Excluded[0] = "changed"
				require.Equal(t, tc.excluded, tc.body(root).HTTPBodyProjection().Excluded, "the record is detached")
			}
			require.Nil(t, root.UserType("Child").Attribute().HTTPBodyProjection(), "declarations carry no projection")
		})
	}
}

// TestProjectHTTPBodyContributorOwnContract projects Parent through a Child
// body that excludes id: the contributor keeps Parent's own members,
// requiredness and defaults, never Child's refinements, and records its own
// projection under Parent's identity.
func TestProjectHTTPBodyContributorOwnContract(t *testing.T) {
	root := expr.RunDSL(t, bodyProjectionDesign)
	parent := root.UserType("Parent")
	projection := httpEndpoint(root, "submit").Body.HTTPBodyProjection()
	contributor := expr.ProjectHTTPBodyContributor(parent, projection)
	named, ok := contributor.Type.(expr.UserType)
	require.True(t, ok)
	require.Equal(t, "Parent", named.Name())
	require.Equal(t, parent.ID(), named.ID())
	object := expr.AsObject(contributor.Type)
	require.NotNil(t, object.Attribute("kind"))
	require.NotNil(t, object.Attribute("n"))
	require.Nil(t, object.Attribute("id"), "excluded members stay excluded")
	require.Nil(t, object.Attribute("thumb"), "Child members are not Parent members")
	require.Equal(t, []string{"kind"}, contributor.Validation.Required, "the physical requiredness keeps Parent's own retained members")
	require.EqualValues(t, 2, object.Attribute("n").DefaultValue, "Parent keeps its own default")
	record := contributor.HTTPBodyProjection()
	require.Same(t, parent.Attribute(), record.Source)
	require.Equal(t, []string{"id"}, record.Excluded)
	require.Nil(t, parent.Attribute().HTTPBodyProjection(), "projection never mutates the declaration")
}

// TestDiscriminatorMemberBindsWireProperty binds a hint's literal schema
// property through the authored JSON wire name, before any transport renames.
func TestDiscriminatorMemberBindsWireProperty(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("Parent", func() {
			Attribute("kind", String, func() {
				Meta("struct:tag:json:name", "category")
			})
			Attribute("id", String)
		})
	})
	source := root.UserType("Parent").Attribute()
	for _, tc := range []struct {
		property, member string
		bound            bool
	}{
		{property: "category", member: "kind", bound: true},
		{property: "id", member: "id", bound: true},
		{property: "kind"},
		{property: "absent"},
	} {
		member, bound := expr.DiscriminatorMember(source, tc.property)
		require.Equal(t, tc.bound, bound, tc.property)
		require.Equal(t, tc.member, member, tc.property)
	}
}

func bodyProjectionDesign() {
	parent := Type("Parent", func() {
		Meta("openapi:discriminator:propertyName", "kind")
		Attribute("kind", String)
		Attribute("id", String)
		Attribute("n", Int, func() {
			Default(2)
		})
		Required("kind", "id")
	})
	child := Type("Child", func() {
		Extend(parent)
		Meta("openapi:allOf:reference", "true")
		Attribute("kind", String, func() {
			Meta("struct:tag:form", "k")
		})
		Attribute("n", Int, func() {
			Default(5)
		})
		Attribute("thumb", Bytes)
		Required("n")
	})
	Service("bodies", func() {
		Method("submit", func() {
			Payload(child)
			HTTP(func() {
				POST("/submit")
				Header("id")
			})
		})
		Method("show", func() {
			Result(child)
			HTTP(func() {
				GET("/show")
				Response(StatusOK, func() {
					Header("id")
				})
			})
		})
		Method("selected", func() {
			Payload(func() {
				Attribute("doc", child)
			})
			HTTP(func() {
				POST("/selected")
				Body("doc")
			})
		})
		Method("subset", func() {
			Payload(child)
			HTTP(func() {
				POST("/subset")
				Header("kind")
				Header("id")
				Header("n")
				Body(func() {
					Attribute("thumb")
				})
			})
		})
	})
}

func httpEndpoint(root *expr.RootExpr, name string) *expr.HTTPEndpointExpr {
	for _, endpoint := range root.API.HTTP.Services[0].HTTPEndpoints {
		if endpoint.Name() == name {
			return endpoint
		}
	}
	return nil
}
