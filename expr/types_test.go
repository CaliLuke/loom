package expr

import "testing"

func TestAsObject(t *testing.T) {
	var (
		object = &Object{
			&NamedAttributeExpr{
				Name: "foo",
				Attribute: &AttributeExpr{
					Type: String,
				},
			},
		}
		objectUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: object,
			},
		}
		notObjectUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: Boolean,
			},
		}
		objectResultType = &ResultTypeExpr{
			UserTypeExpr: objectUserType,
		}
		notObjectResultType = &ResultTypeExpr{
			UserTypeExpr: notObjectUserType,
		}
	)
	cases := map[string]struct {
		dt       DataType
		expected *Object
	}{
		"object user type": {
			dt:       objectUserType,
			expected: object,
		},
		"not object user type": {
			dt:       notObjectUserType,
			expected: nil,
		},
		"object result type": {
			dt:       objectResultType,
			expected: object,
		},
		"not object result type": {
			dt:       notObjectResultType,
			expected: nil,
		},
		"object": {
			dt:       object,
			expected: object,
		},
		"not object": {
			dt:       Boolean,
			expected: nil,
		},
	}

	for k, tc := range cases {
		if actual := AsObject(tc.dt); actual != tc.expected {
			if Equal(actual, tc.expected) != true {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestAsArray(t *testing.T) {
	var (
		array = &Array{
			ElemType: &AttributeExpr{
				Type: String,
			},
		}
		arrayUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: array,
			},
		}
		notArrayUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: Boolean,
			},
		}
		arrayResultType = &ResultTypeExpr{
			UserTypeExpr: arrayUserType,
		}
		notArrayResultType = &ResultTypeExpr{
			UserTypeExpr: notArrayUserType,
		}
	)
	cases := map[string]struct {
		dt       DataType
		expected *Array
	}{
		"array user type": {
			dt:       arrayUserType,
			expected: array,
		},
		"not array user type": {
			dt:       notArrayUserType,
			expected: nil,
		},
		"array result type": {
			dt:       arrayResultType,
			expected: array,
		},
		"not array result type": {
			dt:       notArrayResultType,
			expected: nil,
		},
		"array": {
			dt:       array,
			expected: array,
		},
		"not array": {
			dt:       Boolean,
			expected: nil,
		},
	}

	for k, tc := range cases {
		if actual := AsArray(tc.dt); actual != tc.expected {
			if Equal(actual, tc.expected) != true {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestAsMap(t *testing.T) {
	var (
		mapIntString = &Map{
			KeyType: &AttributeExpr{
				Type: Int,
			},
			ElemType: &AttributeExpr{
				Type: String,
			},
		}
		mapUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: mapIntString,
			},
		}
		notMapUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: Boolean,
			},
		}
		mapResultType = &ResultTypeExpr{
			UserTypeExpr: mapUserType,
		}
		notMapResultType = &ResultTypeExpr{
			UserTypeExpr: notMapUserType,
		}
	)
	cases := map[string]struct {
		dt       DataType
		expected *Map
	}{
		"map user type": {
			dt:       mapUserType,
			expected: mapIntString,
		},
		"not map user type": {
			dt:       notMapUserType,
			expected: nil,
		},
		"map result type": {
			dt:       mapResultType,
			expected: mapIntString,
		},
		"not map result type": {
			dt:       notMapResultType,
			expected: nil,
		},
		"map": {
			dt:       mapIntString,
			expected: mapIntString,
		},
		"not map": {
			dt:       Boolean,
			expected: nil,
		},
	}

	for k, tc := range cases {
		if actual := AsMap(tc.dt); actual != tc.expected {
			if Equal(actual, tc.expected) != true {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestIsObject(t *testing.T) {
	var (
		object = &Object{
			&NamedAttributeExpr{
				Name: "foo",
				Attribute: &AttributeExpr{
					Type: String,
				},
			},
		}
		objectUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: object,
			},
		}
		notObjectUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: Boolean,
			},
		}
		objectResultType = &ResultTypeExpr{
			UserTypeExpr: objectUserType,
		}
		notObjectResultType = &ResultTypeExpr{
			UserTypeExpr: notObjectUserType,
		}
	)
	cases := map[string]struct {
		dt       DataType
		expected bool
	}{
		"object user type": {
			dt:       objectUserType,
			expected: true,
		},
		"not object user type": {
			dt:       notObjectUserType,
			expected: false,
		},
		"object result type": {
			dt:       objectResultType,
			expected: true,
		},
		"not object result type": {
			dt:       notObjectResultType,
			expected: false,
		},
		"object": {
			dt:       object,
			expected: true,
		},
		"not object": {
			dt:       Boolean,
			expected: false,
		},
	}

	for k, tc := range cases {
		if actual := IsObject(tc.dt); actual != tc.expected {
			if actual != tc.expected {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestIsArray(t *testing.T) {
	var (
		array = &Array{
			ElemType: &AttributeExpr{
				Type: String,
			},
		}
		arrayUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: array,
			},
		}
		notArrayUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: Boolean,
			},
		}
		arrayResultType = &ResultTypeExpr{
			UserTypeExpr: arrayUserType,
		}
		notArrayResultType = &ResultTypeExpr{
			UserTypeExpr: notArrayUserType,
		}
	)
	cases := map[string]struct {
		dt       DataType
		expected bool
	}{
		"array user type": {
			dt:       arrayUserType,
			expected: true,
		},
		"not array user type": {
			dt:       notArrayUserType,
			expected: false,
		},
		"array result type": {
			dt:       arrayResultType,
			expected: true,
		},
		"not array result type": {
			dt:       notArrayResultType,
			expected: false,
		},
		"array": {
			dt:       array,
			expected: true,
		},
		"not array": {
			dt:       Boolean,
			expected: false,
		},
	}

	for k, tc := range cases {
		if actual := IsArray(tc.dt); actual != tc.expected {
			if actual != tc.expected {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestIsMap(t *testing.T) {
	var (
		mapIntString = &Map{
			KeyType: &AttributeExpr{
				Type: Int,
			},
			ElemType: &AttributeExpr{
				Type: String,
			},
		}
		mapUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: mapIntString,
			},
		}
		notMapUserType = &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type: Boolean,
			},
		}
		mapResultType = &ResultTypeExpr{
			UserTypeExpr: mapUserType,
		}
		notMapResultType = &ResultTypeExpr{
			UserTypeExpr: notMapUserType,
		}
	)
	cases := map[string]struct {
		dt       DataType
		expected bool
	}{
		"map user type": {
			dt:       mapUserType,
			expected: true,
		},
		"not map user type": {
			dt:       notMapUserType,
			expected: false,
		},
		"map result type": {
			dt:       mapResultType,
			expected: true,
		},
		"not map result type": {
			dt:       notMapResultType,
			expected: false,
		},
		"map": {
			dt:       mapIntString,
			expected: true,
		},
		"not map": {
			dt:       Boolean,
			expected: false,
		},
	}

	for k, tc := range cases {
		if actual := IsMap(tc.dt); actual != tc.expected {
			if actual != tc.expected {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}
