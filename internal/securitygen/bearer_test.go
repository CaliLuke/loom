package securitygen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
)

func TestBearerMapping(t *testing.T) {
	for _, tc := range []struct {
		name, attribute, kind string
		want                  bool
	}{
		{"Authorization", "token", "JWT", true},
		{"authorization", "token", "OAuth2", true},
		{"Authorization", "token", "APIKey", false},
		{"X-Token", "token", "JWT", false},
		{"Authorization", "unrelated", "JWT", false},
	} {
		t.Run(tc.name+tc.kind+tc.attribute, func(t *testing.T) {
			require.Equal(t, tc.want, PlanBearer(tc.name, tc.attribute, false, service.SchemesData{
				&service.SchemeData{KeyAttr: "token", Type: tc.kind},
			}, nil).Framed)
		})
	}
}

func TestBearerAlternativePlan(t *testing.T) {
	jwt := &service.SchemeData{SchemeName: "jwt", Type: "JWT", KeyAttr: "token"}
	key := &service.SchemeData{SchemeName: "key", Type: "APIKey", KeyAttr: "key"}
	basic := &service.SchemeData{SchemeName: "basic", Type: "Basic", UsernameAttr: "user", PasswordAttr: "pass"}
	same := &service.SchemeData{SchemeName: "other", Type: "OAuth2", KeyAttr: "token"}
	cookie := &service.SchemeData{SchemeName: "cookie", Type: "APIKey", TransportOwned: true}
	req := func(schemes ...*service.SchemeData) *service.RequirementData {
		return &service.RequirementData{Schemes: schemes}
	}
	for _, tc := range []struct {
		name         string
		required     bool
		requirements service.RequirementsData
		allow        bool
	}{
		{"singleton", false, service.RequirementsData{req(jwt)}, false},
		{"and", false, service.RequirementsData{req(jwt, key)}, false},
		{"basic-or", false, service.RequirementsData{req(basic), req(jwt)}, true},
		{"reverse-or", false, service.RequirementsData{req(jwt), req(basic)}, true},
		{"key-or", false, service.RequirementsData{req(jwt), req(key)}, true},
		{"shared-field", false, service.RequirementsData{req(jwt), req(same)}, false},
		{"shared-across-and", false, service.RequirementsData{req(jwt, basic), req(jwt, key)}, false},
		{"anonymous", false, service.RequirementsData{req(jwt), req()}, true},
		{"required", true, service.RequirementsData{req(jwt), req(key)}, false},
		{"transport-cookie", false, service.RequirementsData{req(jwt), req(cookie)}, true},
		{"no-requirements", false, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := PlanBearer("Authorization", "token", tc.required, service.SchemesData{jwt}, tc.requirements)
			require.True(t, plan.Framed)
			require.Equal(t, tc.allow, plan.AllowOtherScheme)
		})
	}
}
