package dsl_test

import (
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestFormat(t *testing.T) {
	cases := map[string]struct {
		Format expr.ValidationFormat
	}{
		"date":      {expr.FormatDate},
		"date-time": {expr.FormatDateTime},
		"uuid":      {expr.FormatUUID},
		"email":     {expr.FormatEmail},
		"hostname":  {expr.FormatHostname},
		"ipv4":      {expr.FormatIPv4},
		"ipv6":      {expr.FormatIPv6},
		"ip":        {expr.FormatIP},
		"uri":       {expr.FormatURI},
		"mac":       {expr.FormatMAC},
		"cidr":      {expr.FormatCIDR},
		"regexp":    {expr.FormatRegexp},
		"json":      {expr.FormatJSON},
		"rfc1123":   {expr.FormatRFC1123},
	}

	for k, tc := range cases {
		eval.SetupTestContext(t)
		expr := &expr.AttributeExpr{}
		eval.Execute(func() { Format(tc.Format) }, expr)
		if eval.Context.Errors != nil {
			t.Errorf("%s: Format failed unexpectedly with %s", k, eval.Context.Errors)
		}
		if expr.Validation == nil {
			t.Errorf("%s: Format not initialized Validation in %+v", k, expr)
		} else if expr.Validation.Format != tc.Format {
			t.Errorf("%s: Format not set on %+v, expected %s, got %+v", k, expr, tc.Format, expr.Validation.Format)
		}
	}
}

func TestRequired(t *testing.T) {
	att := &expr.AttributeExpr{
		Type: &expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Object{
					{Name: "foo", Attribute: &expr.AttributeExpr{Type: String}},
					{Name: "bar", Attribute: &expr.AttributeExpr{Type: String}},
				},
			},
			TypeName: "Foo",
		},
	}
	eval.SetupTestContext(t)
	eval.Execute(func() { Required("foo") }, att)
	if eval.Context.Errors != nil {
		t.Errorf("Required failed unexpectedly with %s", eval.Context.Errors)
		return
	}
	if len(att.Validation.Required) == 0 {
		t.Errorf("Required not set on %+v", att)
		return
	}
	if att.Validation.Required[0] != "foo" {
		t.Errorf("Required invalid on %+v, expected foo, got %+v", att, att.Validation.Required)
	}
	uattr := att.Type.(*expr.UserTypeExpr).AttributeExpr
	if uattr.Validation == nil {
		t.Fatalf("Required not set on %+v", uattr)
	}
	if len(uattr.Validation.Required) == 0 {
		t.Errorf("Required not set on %+v, got %+v", uattr, uattr.Validation.Required)
	}
	if uattr.Validation.Required[0] != "foo" {
		t.Errorf("Required invalid on %+v, expected foo, got %+v", uattr, uattr.Validation.Required)
	}
}

func TestRequiredInAttributeDSLDoesNotMutateSharedUserType(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var Person = Type("Person", func() {
			Attribute("name", String)
			Attribute("email", String)
		})
		Type("Trip", func() {
			Attribute("driver", Person, func() {
				Required("name")
			})
			Attribute("passenger", Person)
		})
	})

	person := root.UserType("Person")
	if person.Attribute().IsRequired("name") {
		t.Fatalf("attribute-level Required mutated shared Person type")
	}

	trip := root.UserType("Trip")
	driver := trip.Attribute().Find("driver")
	if driver == nil {
		t.Fatalf("missing driver attribute")
	}
	if !driver.IsRequired("name") {
		t.Fatalf("driver attribute did not keep local requiredness")
	}
	passenger := trip.Attribute().Find("passenger")
	if passenger == nil {
		t.Fatalf("missing passenger attribute")
	}
	if passenger.IsRequired("name") {
		t.Fatalf("passenger unexpectedly inherited driver requiredness")
	}
}

func TestRequiredInArrayElemDSLDoesNotMutateSharedUserType(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var Person = Type("Person", func() {
			Attribute("name", String)
			Attribute("email", String)
		})
		Type("Trip", func() {
			Attribute("drivers", ArrayOf(Person), func() {
				Elem(func() {
					Required("name")
				})
			})
			Attribute("passenger", Person)
		})
	})

	person := root.UserType("Person")
	if person.Attribute().IsRequired("name") {
		t.Fatalf("array element Required mutated shared Person type")
	}

	trip := root.UserType("Trip")
	drivers := trip.Attribute().Find("drivers")
	if drivers == nil {
		t.Fatalf("missing drivers attribute")
	}
	array := expr.AsArray(drivers.Type)
	if array == nil {
		t.Fatalf("drivers attribute is not an array")
	}
	if !array.ElemType.IsRequired("name") {
		t.Fatalf("drivers element did not keep local requiredness")
	}
	passenger := trip.Attribute().Find("passenger")
	if passenger == nil {
		t.Fatalf("missing passenger attribute")
	}
	if passenger.IsRequired("name") {
		t.Fatalf("passenger unexpectedly inherited drivers element requiredness")
	}
}

func TestRequiredInMapElemDSLDoesNotMutateSharedUserType(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var Person = Type("Person", func() {
			Attribute("name", String)
			Attribute("email", String)
		})
		Type("Trip", func() {
			Attribute("drivers", MapOf(String, Person), func() {
				Elem(func() {
					Required("name")
				})
			})
			Attribute("passenger", Person)
		})
	})

	person := root.UserType("Person")
	if person.Attribute().IsRequired("name") {
		t.Fatalf("map element Required mutated shared Person type")
	}

	trip := root.UserType("Trip")
	drivers := trip.Attribute().Find("drivers")
	if drivers == nil {
		t.Fatalf("missing drivers attribute")
	}
	mapping := expr.AsMap(drivers.Type)
	if mapping == nil {
		t.Fatalf("drivers attribute is not a map")
	}
	if !mapping.ElemType.IsRequired("name") {
		t.Fatalf("drivers element did not keep local requiredness")
	}
	passenger := trip.Attribute().Find("passenger")
	if passenger == nil {
		t.Fatalf("missing passenger attribute")
	}
	if passenger.IsRequired("name") {
		t.Fatalf("passenger unexpectedly inherited drivers element requiredness")
	}
}

func TestRequiredInNestedCollectionElemDSLDoesNotMutateSharedUserType(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var Person = Type("Person", func() {
			Attribute("name", String)
			Attribute("email", String)
		})
		Type("Trip", func() {
			Attribute("driverGroups", ArrayOf(MapOf(String, Person)), func() {
				Elem(func() {
					Elem(func() {
						Required("name")
					})
				})
			})
			Attribute("passenger", Person)
		})
	})

	person := root.UserType("Person")
	if person.Attribute().IsRequired("name") {
		t.Fatalf("nested collection Required mutated shared Person type")
	}

	trip := root.UserType("Trip")
	driverGroups := trip.Attribute().Find("driverGroups")
	if driverGroups == nil {
		t.Fatalf("missing driverGroups attribute")
	}
	array := expr.AsArray(driverGroups.Type)
	if array == nil {
		t.Fatalf("driverGroups attribute is not an array")
	}
	mapping := expr.AsMap(array.ElemType.Type)
	if mapping == nil {
		t.Fatalf("driverGroups element is not a map")
	}
	if !mapping.ElemType.IsRequired("name") {
		t.Fatalf("nested driver element did not keep local requiredness")
	}
	passenger := trip.Attribute().Find("passenger")
	if passenger == nil {
		t.Fatalf("missing passenger attribute")
	}
	if passenger.IsRequired("name") {
		t.Fatalf("passenger unexpectedly inherited driverGroups element requiredness")
	}
}

func TestRequiredInArrayConstructorDSLDoesNotMutateSharedUserType(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var Person = Type("Person", func() {
			Attribute("name", String)
			Attribute("email", String)
		})
		Type("Trip", func() {
			Attribute("drivers", ArrayOf(Person, func() {
				Required("name")
			}))
			Attribute("passenger", Person)
		})
	})

	person := root.UserType("Person")
	if person.Attribute().IsRequired("name") {
		t.Fatalf("array constructor Required mutated shared Person type")
	}

	trip := root.UserType("Trip")
	drivers := trip.Attribute().Find("drivers")
	if drivers == nil {
		t.Fatalf("missing drivers attribute")
	}
	array := expr.AsArray(drivers.Type)
	if array == nil {
		t.Fatalf("drivers attribute is not an array")
	}
	if !array.ElemType.IsRequired("name") {
		t.Fatalf("drivers element did not keep constructor requiredness")
	}
	passenger := trip.Attribute().Find("passenger")
	if passenger == nil {
		t.Fatalf("missing passenger attribute")
	}
	if passenger.IsRequired("name") {
		t.Fatalf("passenger unexpectedly inherited drivers element requiredness")
	}
}

func TestRequiredInMapConstructorDSLDoesNotMutateSharedUserTypes(t *testing.T) {
	root := expr.RunDSL(t, func() {
		var Person = Type("Person", func() {
			Attribute("name", String)
			Attribute("email", String)
		})
		var Region = Type("Region", func() {
			Attribute("code", String)
		})
		Type("Trip", func() {
			Attribute("drivers", MapOf(Region, Person, func() {
				Key(func() {
					Required("code")
				})
				Elem(func() {
					Required("name")
				})
			}))
			Attribute("passenger", Person)
			Attribute("origin", Region)
		})
	})

	person := root.UserType("Person")
	if person.Attribute().IsRequired("name") {
		t.Fatalf("map constructor Elem Required mutated shared Person type")
	}
	region := root.UserType("Region")
	if region.Attribute().IsRequired("code") {
		t.Fatalf("map constructor Key Required mutated shared Region type")
	}

	trip := root.UserType("Trip")
	drivers := trip.Attribute().Find("drivers")
	if drivers == nil {
		t.Fatalf("missing drivers attribute")
	}
	mapping := expr.AsMap(drivers.Type)
	if mapping == nil {
		t.Fatalf("drivers attribute is not a map")
	}
	if !mapping.KeyType.IsRequired("code") {
		t.Fatalf("drivers key did not keep constructor requiredness")
	}
	if !mapping.ElemType.IsRequired("name") {
		t.Fatalf("drivers element did not keep constructor requiredness")
	}
	passenger := trip.Attribute().Find("passenger")
	if passenger == nil {
		t.Fatalf("missing passenger attribute")
	}
	if passenger.IsRequired("name") {
		t.Fatalf("passenger unexpectedly inherited drivers element requiredness")
	}
	origin := trip.Attribute().Find("origin")
	if origin == nil {
		t.Fatalf("missing origin attribute")
	}
	if origin.IsRequired("code") {
		t.Fatalf("origin unexpectedly inherited drivers key requiredness")
	}
}

func TestPatternInvalidRegexReportsAttributedError(t *testing.T) {
	eval.SetupTestContext(t)
	att := &expr.AttributeExpr{Type: expr.String}
	eval.Execute(func() { Pattern("[invalid(") }, att)

	if eval.Context.Errors == nil {
		t.Fatalf("expected DSL error for invalid regex pattern, got none")
	}
	if got := eval.Context.Error(); !strings.Contains(got, "invalid pattern") {
		t.Errorf("expected error to mention 'invalid pattern', got %q", got)
	}
	if att.Validation != nil && att.Validation.Pattern != "" {
		t.Errorf("invalid pattern should not be stored on attribute, got %q", att.Validation.Pattern)
	}
}
