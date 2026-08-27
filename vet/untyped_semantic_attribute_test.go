package vet

import (
	"bytes"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeUntypedSemanticAttributes(t *testing.T) {
	object := expr.Object{
		&expr.NamedAttributeExpr{Name: "created_at", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		&expr.NamedAttributeExpr{Name: "deleted_at", Attribute: &expr.AttributeExpr{Type: expr.Any, Nullable: true}},
		&expr.NamedAttributeExpr{Name: "actor_id", Attribute: &expr.AttributeExpr{Type: expr.Any, Description: "User UUID."}},
		&expr.NamedAttributeExpr{Name: "enabled", Attribute: &expr.AttributeExpr{Type: expr.Any, Description: "Boolean value controlling delivery."}},
		&expr.NamedAttributeExpr{Name: "retry_count", Attribute: &expr.AttributeExpr{Type: expr.Any, Description: "Integer value for retry attempts."}},
		&expr.NamedAttributeExpr{Name: "ratio", Attribute: &expr.AttributeExpr{Type: expr.Any, Description: "Numeric value for the ratio."}},
		&expr.NamedAttributeExpr{Name: "callback", Attribute: &expr.AttributeExpr{Type: expr.Any, Description: "Callback URL."}},
		&expr.NamedAttributeExpr{Name: "contact", Attribute: &expr.AttributeExpr{Type: expr.Any, Description: "Email address for the owner."}},
		&expr.NamedAttributeExpr{Name: "origin", Attribute: &expr.AttributeExpr{Type: expr.Any, Description: "IP address of the caller."}},
		&expr.NamedAttributeExpr{Name: "metadata", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		&expr.NamedAttributeExpr{Name: "nullable_metadata", Attribute: &expr.AttributeExpr{Type: expr.Any, Nullable: true}},
		&expr.NamedAttributeExpr{Name: "config", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		&expr.NamedAttributeExpr{Name: "preferences", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		&expr.NamedAttributeExpr{Name: "document_body", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		&expr.NamedAttributeExpr{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.Any, Description: "Provider-specific identifier."}},
		&expr.NamedAttributeExpr{Name: "boolean_or_object", Attribute: &expr.AttributeExpr{Type: expr.Any, Description: "Boolean or object value."}},
		&expr.NamedAttributeExpr{Name: "ignored_at", Attribute: &expr.AttributeExpr{
			Type: expr.Any,
			Meta: expr.MetaExpr{SuppressionMeta: {
				RuleUntypedSemanticAttribute,
			}},
		}},
	}
	root := &expr.RootExpr{Types: []expr.UserType{&expr.UserTypeExpr{
		TypeName:      "Records",
		AttributeExpr: &expr.AttributeExpr{Type: &object},
	}}}

	var report Report
	analyzeAttributeSemantics(root, &report)

	diagnostics := diagnosticsForRule(report.Diagnostics, RuleUntypedSemanticAttribute)
	require.ElementsMatch(t, []string{
		RuleUntypedSemanticAttribute + ":type.Records.created_at",
		RuleUntypedSemanticAttribute + ":type.Records.deleted_at",
		RuleUntypedSemanticAttribute + ":type.Records.actor_id",
		RuleUntypedSemanticAttribute + ":type.Records.enabled",
		RuleUntypedSemanticAttribute + ":type.Records.retry_count",
		RuleUntypedSemanticAttribute + ":type.Records.ratio",
		RuleUntypedSemanticAttribute + ":type.Records.callback",
		RuleUntypedSemanticAttribute + ":type.Records.contact",
		RuleUntypedSemanticAttribute + ":type.Records.origin",
	}, diagnosticKeys(diagnostics))
	for _, diagnostic := range diagnostics {
		require.Equal(t, SeverityWarning, diagnostic.Severity)
		require.Contains(t, diagnostic.Message, "instead of Any")
	}

	for _, format := range []Format{FormatText, FormatJSON, FormatSARIF} {
		var output bytes.Buffer
		require.NoError(t, WriteReport(&output, Report{Diagnostics: diagnostics}, format))
		require.Contains(t, output.String(), RuleUntypedSemanticAttribute)
		require.Contains(t, output.String(), string(SeverityWarning))
	}
}
