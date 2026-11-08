package service

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPMetadataNativeReachability(t *testing.T) {
	for _, role := range []string{"result", "handshake", "informational"} {
		t.Run(role, func(t *testing.T) {
			var leaf expr.UserType
			root := codegen.RunDSL(t, func() {
				leaf = dsl.Type("MetadataLeaf", func() {
					dsl.Attribute("label", dsl.String)
				})
				carrier := dsl.Type("MetadataCarrier", func() {
					dsl.Meta("struct:pkg:path", "metadata")
					dsl.Attribute("trace", dsl.String)
					dsl.Attribute("detail", leaf)
				})
				dsl.Service("carriers", func() {
					dsl.Method("read", func() {
						if role == "result" {
							dsl.Result(carrier)
						}
						if role == "handshake" {
							dsl.StreamingResult(dsl.String)
						}
						dsl.HTTP(func() {
							dsl.GET("/read")
							switch role {
							case "result":
								dsl.Response(dsl.StatusOK)
							case "handshake":
								dsl.ServerSentEvents()
								dsl.Response(dsl.StatusCreated, carrier, func() {
									dsl.Header("trace:X-Trace")
								})
							case "informational":
								dsl.Response(dsl.StatusNoContent)
								dsl.Informational("Hints", dsl.StatusEarlyHints, carrier, func() {
									dsl.Header("trace:X-Trace")
								})
								dsl.Informational("Progress", dsl.StatusProcessing, carrier, func() {
									dsl.Header("trace:X-Trace")
								})
							}
						})
					})
				})
			})
			services := NewServicesData(root)
			data := services.Get("carriers")
			counts := make(map[string]int)
			for _, typ := range data.userTypes {
				counts[typ.Name]++
			}
			t.Logf("DSL admitted; role=%s native types=%v nested location=%v", role, counts, codegen.UserTypeLocation(leaf))
			require.Equal(t, 1, counts["MetadataLeaf"], "metadata-only nested declaration must be collected once")
			require.NotNil(t, codegen.UserTypeLocation(leaf), "nested declaration must inherit metadata package")
			require.Equal(t, "metadata", codegen.UserTypeLocation(leaf).RelImportPath)
			renamed := services.WithPackageNames(map[string]string{"metadata": "metadataAlias"}).Get("carriers")
			require.Len(t, renamed.userTypes, len(data.userTypes))
			for _, typ := range renamed.userTypes {
				reference := renamed.Scope.GoFullTypeRefWithPackages(&expr.AttributeExpr{Type: typ.Type}, renamed.PkgName, renamed.Scope.PackageName)
				require.Contains(t, reference, "metadataAlias.")
			}
			if role != "result" {
				require.Equal(t, 1, counts["MetadataCarrier"], "metadata carrier has no ordinary MethodData declaration owner")
			}
		})
	}
}

func TestHTTPMetadataInventory(t *testing.T) {
	t.Run("destination reservation", func(t *testing.T) {
		var left, right expr.UserType
		root := codegen.RunDSL(t, func() {
			left = dsl.Type("foo-bar", func() {
				dsl.Meta("struct:pkg:path", "metadata")
				dsl.Attribute("left", dsl.String)
			})
			right = dsl.Type("foo_bar", func() {
				dsl.Meta("struct:pkg:path", "metadata")
				dsl.Attribute("right", dsl.String)
			})
			dsl.Service("carriers", func() {
				dsl.Method("read", func() {
					dsl.HTTP(func() {
						dsl.GET("/read")
						dsl.Response(dsl.StatusNoContent)
						dsl.Informational("First", dsl.StatusProcessing, left)
						dsl.Informational("Second", dsl.StatusEarlyHints, right)
					})
				})
			})
		})
		owner := NewServicesData(root).generatedDestinationTypes()
		lname, lok := owner.lookup(left, codegen.UserTypeLocation(left))
		rname, rok := owner.lookup(right, codegen.UserTypeLocation(right))
		t.Logf("DSL admitted; reservations left=%q/%t right=%q/%t", lname, lok, rname, rok)
		require.True(t, lok, "metadata-only declaration missing from destination allocation")
		require.True(t, rok)
		require.NotEqual(t, lname, rname, "Goified collision must reserve distinct declaration names")
	})
	t.Run("own package", func(t *testing.T) {
		root := codegen.RunDSL(t, func() {
			carrier := dsl.Type("MetadataCarrier", func() {
				dsl.Meta("struct:pkg:path", "carriers")
				dsl.Attribute("trace", dsl.String)
			})
			dsl.Service("carriers", func() {
				dsl.Method("read", func() {
					dsl.HTTP(func() {
						dsl.GET("/read")
						dsl.Response(dsl.StatusNoContent)
						dsl.Informational("Hints", dsl.StatusEarlyHints, carrier, func() {
							dsl.Header("trace:X-Trace")
						})
					})
				})
			})
		})
		data := NewServicesData(root).Get("carriers")
		t.Logf("DSL admitted; own-package declaration=%v", data.ownPackageType)
		require.ErrorContains(t, checkUserTypePackages(data), "the service package cannot import itself")
	})
}

func TestHTTPMetadataNormalizedConflict(t *testing.T) {
	for _, metadata := range []bool{false, true} {
		name := "ordinary"
		if metadata {
			name = "informational"
		}
		t.Run(name, func(t *testing.T) {
			leaf := &expr.UserTypeExpr{TypeName: "MetadataLeaf", UID: "leaf", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
			carrier := func(name, path string) *expr.AttributeExpr {
				return &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: name, UID: name, AttributeExpr: &expr.AttributeExpr{
					Type: &expr.Object{&expr.NamedAttributeExpr{Name: "detail", Attribute: &expr.AttributeExpr{Type: leaf}}},
					Meta: expr.MetaExpr{"struct:pkg:path": []string{path}},
				}}}
			}
			left, right := carrier("LeftMetadata", "left"), carrier("RightMetadata", "right")
			method := &expr.MethodExpr{Name: "read"}
			service := &expr.ServiceExpr{Name: "carriers", Methods: []*expr.MethodExpr{method}}
			endpoint := &expr.HTTPEndpointExpr{MethodExpr: method}
			root := &expr.RootExpr{Services: []*expr.ServiceExpr{service}, API: &expr.APIExpr{HTTP: &expr.HTTPExpr{
				Services: []*expr.HTTPServiceExpr{{ServiceExpr: service, HTTPEndpoints: []*expr.HTTPEndpointExpr{endpoint}}},
			}}}
			if metadata {
				endpoint.Informational = []*expr.HTTPInformationalExpr{
					{Name: "First", Value: left, HTTPResponseExpr: &expr.HTTPResponseExpr{Status: expr.HTTPStatusSelector{Code: 102}}},
					{Name: "Second", Value: right, HTTPResponseExpr: &expr.HTTPResponseExpr{Status: expr.HTTPStatusSelector{Code: 103}}},
				}
			} else {
				method.Payload, method.Result = left, right
			}
			require.Nil(t, codegen.UserTypeLocation(leaf), "normalized owner input deliberately has no explicit child package")
			require.PanicsWithValue(t, `user type "MetadataLeaf" is transitively required by shared packages "left" and "right"; set struct:pkg:path metadata on the type to select its package`, func() {
				promoteSharedTypeLocations(root)
			})
		})
	}
}

func TestHTTPMetadataCarrierOccurrences(t *testing.T) {
	method := &expr.MethodExpr{Name: "read"}
	handshake := &expr.HTTPResponseExpr{Handshake: &expr.AttributeExpr{Type: &expr.Object{}}}
	info := &expr.HTTPInformationalExpr{Name: "Hints", Value: &expr.AttributeExpr{Type: &expr.Object{}}, HTTPResponseExpr: &expr.HTTPResponseExpr{}}
	endpoint := &expr.HTTPEndpointExpr{MethodExpr: method, Responses: []*expr.HTTPResponseExpr{handshake}, Informational: []*expr.HTTPInformationalExpr{info}}
	root := &expr.RootExpr{API: &expr.APIExpr{HTTP: &expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{{HTTPEndpoints: []*expr.HTTPEndpointExpr{endpoint}}}}}}
	want := []httpMetadataCarrier{
		{response: handshake, attribute: handshake.Handshake, role: "handshake"},
		{response: info.HTTPResponseExpr, attribute: info.Value, role: "informational", name: "Hints"},
	}
	require.Equal(t, want, httpMetadataCarriers(root, method))
	require.Empty(t, httpMetadataCarriers(root, &expr.MethodExpr{Name: "read"}), "method names do not establish occurrence ownership")
	for _, absent := range []*expr.RootExpr{nil, {}, {API: &expr.APIExpr{}}} {
		require.Empty(t, httpMetadataCarriers(absent, method))
	}
	endpoint.Meta = expr.MetaExpr{"jsonrpc": {"true"}}
	require.Empty(t, httpMetadataCarriers(root, method), "JSON-RPC occurrences cannot publish HTTP metadata carriers")
}

func TestHTTPMetadataAnonymousOccurrencePreservation(t *testing.T) {
	child := &expr.AttributeExpr{Type: expr.String, DefaultValue: "child-default", Meta: expr.MetaExpr{"struct:tag:json": {"trace,omitempty"}}, Validation: &expr.ValidationExpr{MinLength: new(2)}}
	object := &expr.Object{{Name: "trace", Attribute: child}}
	first := &expr.AttributeExpr{Type: object, Validation: &expr.ValidationExpr{Required: []string{"trace"}}, DefaultValue: map[string]any{"trace": "first-default"}, UserExamples: []*expr.ExampleExpr{{Value: map[string]any{"trace": "first-example"}}}}
	second := expr.DupAtt(first)
	second.Type = object
	second.DefaultValue = map[string]any{"trace": "second-default"}
	second.UserExamples = []*expr.ExampleExpr{{Value: map[string]any{"trace": "second-example"}}}
	third := expr.DupAtt(first)
	third.Type, third.DefaultValue, third.UserExamples = object, nil, nil
	thirdExample := valueDataExample(newValueData(expr.NewValueContext(), third, expr.NewRandom("metadata")))
	method := &expr.MethodExpr{Name: "read"}
	service := &expr.ServiceExpr{Name: "carriers", Methods: []*expr.MethodExpr{method}}
	endpoint := &expr.HTTPEndpointExpr{MethodExpr: method}
	root := &expr.RootExpr{Services: []*expr.ServiceExpr{service}, API: &expr.APIExpr{HTTP: &expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{{ServiceExpr: service, HTTPEndpoints: []*expr.HTTPEndpointExpr{endpoint}}}}}}
	for _, attribute := range []*expr.AttributeExpr{first, second, third} {
		endpoint.Responses = append(endpoint.Responses, &expr.HTTPResponseExpr{Handshake: attribute})
	}
	normalizeHTTPMetadataCarriers(root, service, codegen.NewNameScope())
	declaration, ok := first.Type.(*expr.UserTypeExpr)
	require.True(t, ok, "anonymous metadata needs a native named declaration")
	require.Same(t, declaration, second.Type)
	require.Same(t, declaration, third.Type)
	require.Nil(t, declaration.DefaultValue, "a shared declaration cannot inherit an occurrence's default")
	require.Empty(t, declaration.UserExamples, "a shared declaration cannot inherit an occurrence's example")
	require.Equal(t, []string{"trace"}, declaration.Validation.Required)
	declaredChild := expr.AsObject(declaration).Attribute("trace")
	require.Equal(t, child.DefaultValue, declaredChild.DefaultValue)
	require.Equal(t, child.Meta, declaredChild.Meta)
	require.Equal(t, child.Validation, declaredChild.Validation)
	for i, attribute := range []*expr.AttributeExpr{first, second} {
		prefix := []string{"first", "second"}[i]
		require.Equal(t, map[string]any{"trace": prefix + "-default"}, attribute.DefaultValue)
		require.Equal(t, []string{"trace"}, attribute.Validation.Required)
		value := newValueData(expr.NewValueContext(), attribute, expr.NewRandom("metadata"))
		resolvedDefault, present := value.Occurrence.EffectiveConstraints().Default()
		require.True(t, present)
		require.Equal(t, attribute.DefaultValue, resolvedDefault)
		require.Equal(t, map[string]any{"trace": prefix + "-example"}, valueDataExample(value))
	}
	require.Nil(t, third.DefaultValue)
	require.Empty(t, third.UserExamples)
	value := newValueData(expr.NewValueContext(), third, expr.NewRandom("metadata"))
	_, hasDefault := value.Occurrence.EffectiveConstraints().Default()
	require.False(t, hasDefault, "unannotated occurrence has no inherited root default")
	require.Equal(t, thirdExample, valueDataExample(value), "unannotated occurrence retains its own synthesis, not first response values")
	normalizeHTTPMetadataCarriers(root, service, codegen.NewNameScope())
	require.Same(t, declaration, first.Type, "reanalyzing package aliases cannot create another declaration")
}

// Conflicting roots here are synthetic normalized-owner inputs. Public raw
// Response datatype arguments construct bare roots, not these refinements.
func TestHTTPMetadataAnonymousStorageConflict(t *testing.T) {
	for _, kind := range []string{"required", "additional", "destination"} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reverse=%t", kind, reverse), func(t *testing.T) {
				object := &expr.Object{{Name: "trace", Attribute: &expr.AttributeExpr{Type: expr.String}}}
				left, right := &expr.AttributeExpr{Type: object}, &expr.AttributeExpr{Type: object}
				switch kind {
				case "required":
					left.Validation = &expr.ValidationExpr{Required: []string{"trace"}}
				case "additional":
					left.AdditionalAttributes = &expr.AttributeExpr{Type: expr.String}
				case "destination":
					left.Meta = expr.MetaExpr{"struct:pkg:path": {"left"}}
					right.Meta = expr.MetaExpr{"struct:pkg:path": {"right"}}
				}
				if reverse {
					left, right = right, left
				}
				unrelated := &expr.AttributeExpr{Type: &expr.Object{{Name: "other", Attribute: &expr.AttributeExpr{Type: expr.String}}}}
				original := unrelated.Type
				root, service := anonymousMetadataRoot(unrelated, left, right)
				scope := codegen.NewNameScope()
				var failure any
				func() {
					defer func() {
						failure = recover()
					}()
					normalizeHTTPMetadataCarriers(root, service, scope)
				}()
				if failure == nil {
					t.Error("incompatible anonymous storage must reject before publication")
				} else {
					err, ok := failure.(*codegen.Error)
					require.True(t, ok, "existing contextual codegen diagnostic required")
					require.Equal(t, "carriers", err.Service)
					require.Equal(t, "read", err.Method)
					require.Contains(t, err.Error(), "incompatible anonymous HTTP metadata")
				}
				require.Same(t, original, unrelated.Type, "earlier compatible group cannot be partially published")
				require.Same(t, object, left.Type)
				require.Same(t, object, right.Type)
			})
		}
	}
}

func TestHTTPMetadataAnonymousLocalPredicates(t *testing.T) {
	child := &expr.AttributeExpr{Type: expr.String, DefaultValue: "child", Validation: &expr.ValidationExpr{MinLength: new(2)}, Meta: expr.MetaExpr{"audit:child": {"shared"}}}
	object := &expr.Object{{Name: "trace", Attribute: child}}
	attributes := []*expr.AttributeExpr{
		{Type: object, Validation: &expr.ValidationExpr{Required: []string{"trace"}, Values: []any{map[string]any{"trace": "first"}}}, Meta: expr.MetaExpr{"audit:root": {"first"}}, Description: "first root"},
		{Type: object, Validation: &expr.ValidationExpr{Required: []string{"trace"}, Values: []any{map[string]any{"trace": "second"}}}, Meta: expr.MetaExpr{"audit:root": {"second"}}, Description: "second root"},
	}
	before := make([]*expr.ValidationExpr, len(attributes))
	for i, attribute := range attributes {
		facts, err := expr.EffectiveConstraintFactsFor(attribute)
		require.NoError(t, err)
		before[i] = facts.Validation().Lowered()
	}
	root, service := anonymousMetadataRoot(attributes...)
	normalizeHTTPMetadataCarriers(root, service, codegen.NewNameScope())
	declaration := attributes[0].Type.(*expr.UserTypeExpr)
	require.Same(t, declaration, attributes[1].Type)
	require.Empty(t, declaration.Validation.Enums(), "root predicates belong to occurrences, not shared storage")
	require.Empty(t, declaration.Meta)
	require.Empty(t, declaration.Description)
	for i, attribute := range attributes {
		facts, err := expr.EffectiveConstraintFactsFor(attribute)
		require.NoError(t, err)
		require.Equal(t, before[i], facts.Validation().Lowered())
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		for j, raw := range []any{map[string]any{"trace": "first"}, map[string]any{"trace": "second"}} {
			resolved := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: raw}), expr.ValueRoleExample)
			if i == j {
				require.Equal(t, expr.ValueResolved, resolved.Outcome())
			} else {
				require.Equal(t, expr.ValueInvalid, resolved.Outcome())
			}
		}
		currentChild := expr.AsObject(attribute.Type).Attribute("trace")
		require.Equal(t, child.DefaultValue, currentChild.DefaultValue)
		require.Equal(t, child.Validation, currentChild.Validation)
		require.Equal(t, child.Meta, currentChild.Meta)
	}
}

func TestHTTPMetadataAnonymousSharedShape(t *testing.T) {
	object := &expr.Object{{Name: "left", Attribute: &expr.AttributeExpr{Type: expr.String}}, {Name: "right", Attribute: &expr.AttributeExpr{Type: expr.String}}}
	additional := &expr.AttributeExpr{Type: expr.Int}
	first := &expr.AttributeExpr{Type: object, AdditionalAttributes: additional, Validation: &expr.ValidationExpr{Required: []string{"left", "right"}}, Meta: expr.MetaExpr{"struct:pkg:path": {"metadata"}, "audit:root": {"first"}}}
	second := &expr.AttributeExpr{Type: object, AdditionalAttributes: additional, Validation: &expr.ValidationExpr{Required: []string{"right", "left"}}, Meta: expr.MetaExpr{"struct:pkg:path": {"metadata"}, "audit:root": {"second"}}}
	root, service := anonymousMetadataRoot(first, second)
	normalizeHTTPMetadataCarriers(root, service, codegen.NewNameScope())
	declaration := first.Type.(*expr.UserTypeExpr)
	require.Same(t, declaration, second.Type)
	require.Same(t, additional, declaration.AdditionalAttributes)
	require.Equal(t, []string{"left", "right"}, declaration.Validation.Required)
	require.Equal(t, "metadata", codegen.UserTypeLocation(declaration).RelImportPath)
	require.NotContains(t, declaration.Meta, "audit:root")
	require.Equal(t, "first", first.Meta["audit:root"][0])
	require.Equal(t, "second", second.Meta["audit:root"][0])
}

func anonymousMetadataRoot(attributes ...*expr.AttributeExpr) (*expr.RootExpr, *expr.ServiceExpr) {
	method := &expr.MethodExpr{Name: "read"}
	service := &expr.ServiceExpr{Name: "carriers", Methods: []*expr.MethodExpr{method}}
	endpoint := &expr.HTTPEndpointExpr{MethodExpr: method}
	for _, attribute := range attributes {
		endpoint.Responses = append(endpoint.Responses, &expr.HTTPResponseExpr{Handshake: attribute})
	}
	root := &expr.RootExpr{Services: []*expr.ServiceExpr{service}, API: &expr.APIExpr{HTTP: &expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{{ServiceExpr: service, HTTPEndpoints: []*expr.HTTPEndpointExpr{endpoint}}}}}}
	return root, service
}

func TestHTTPMetadataPublicRawRootControl(t *testing.T) {
	for _, role := range []string{"informational", "handshake"} {
		t.Run(role, func(t *testing.T) {
			child := &expr.AttributeExpr{Type: expr.String, DefaultValue: "child", Validation: &expr.ValidationExpr{MinLength: new(2)}, Meta: expr.MetaExpr{"audit:child": {"preserved"}}}
			object := &expr.Object{{Name: "trace", Attribute: child}}
			root := codegen.RunDSL(t, func() {
				dsl.Service("carriers", func() {
					dsl.Method("read", func() {
						if role == "handshake" {
							dsl.StreamingResult(dsl.String)
						}
						dsl.HTTP(func() {
							dsl.GET("/read")
							if role == "handshake" {
								dsl.ServerSentEvents()
								dsl.Response(201, object, func() {
									dsl.Header("trace:X-Trace")
									dsl.Meta("audit:root", "response-meta")
								})
								return
							}
							dsl.Response(204)
							dsl.Informational("First", 102, object, func() {
								dsl.Header("trace:X-Trace")
								dsl.Meta("audit:root", "first-response-meta")
							})
							dsl.Informational("Second", 103, object, func() {
								dsl.Header("trace:X-Trace")
								dsl.Meta("audit:root", "second-response-meta")
							})
						})
					})
				})
			})
			service := root.Services[0]
			carriers := httpMetadataCarriers(root, service.Methods[0])
			before := make([]*expr.ValidationExpr, len(carriers))
			definitions := make([]string, len(carriers))
			for i, carrier := range carriers {
				before[i] = metadataRootValidation(t, carrier.attribute)
				definitions[i] = codegen.NewNameScope().GoValueTypeDef(carrier.attribute, false, true)
				t.Logf("DSL admitted %s occurrence %d root=%+v responseMeta=%v", role, i, before[i], carrier.response.Meta)
			}
			normalizeHTTPMetadataCarriers(root, service, codegen.NewNameScope())
			for i, carrier := range carriers {
				assert.Equal(t, before[i], metadataRootValidation(t, carrier.attribute))
				declaration := carrier.attribute.Type.(*expr.UserTypeExpr)
				assert.Equal(t, definitions[i], codegen.NewNameScope().GoValueTypeDef(declaration.AttributeExpr, false, true), "native declaration bytes must survive normalization")
				currentChild := expr.AsObject(carrier.attribute.Type).Attribute("trace")
				assert.Equal(t, child.DefaultValue, currentChild.DefaultValue)
				assert.Equal(t, child.Validation, currentChild.Validation)
				assert.Equal(t, child.Meta, currentChild.Meta)
			}
		})
	}
}

func metadataRootValidation(t *testing.T, attribute *expr.AttributeExpr) *expr.ValidationExpr {
	t.Helper()
	facts, err := expr.EffectiveConstraintFactsFor(attribute)
	require.NoError(t, err)
	return facts.Validation().Lowered()
}
