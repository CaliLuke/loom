package expr

import (
	"fmt"
	"mime"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

const (
	// DefaultView is the name of the default result type view.
	DefaultView = "default"

	// ViewMetaKey is the key used to store the view name in the attribute meta.
	ViewMetaKey = "view"
)

type (
	// ResultTypeExpr is a user type which describes views used to
	// render responses.
	ResultTypeExpr struct {
		// A result type is a user type
		*UserTypeExpr
		// Identifier is the RFC 6838 result type media type identifier.
		Identifier string
		// Views list the supported views indexed by name.
		Views []*ViewExpr
	}

	// ViewExpr defines which fields to render when building a response. The view
	// is an object whose field names must match the names of the parent result
	// type field names. The field definitions are inherited from the parent
	// result type but may be overridden.
	ViewExpr struct {
		// Set of properties included in view
		*AttributeExpr
		// Name of view
		Name string
		// Parent result Type
		Parent *ResultTypeExpr
		// RequiredOverrides lists selected fields explicitly required by the view.
		RequiredOverrides []string
		// OptionalOverrides lists selected fields explicitly optional in the view.
		OptionalOverrides []string
	}
)

var (
	// ProblemResultIdentifier is the standards-first result type identifier used
	// for RFC 9457-style problem responses.
	ProblemResultIdentifier = "application/problem+json"

	// ErrorResultIdentifier is the result type identifier used for default HTTP
	// error responses. It intentionally matches ProblemResultIdentifier so the
	// framework defaults to RFC 9457 problem documents.
	ErrorResultIdentifier = ProblemResultIdentifier

	// ErrorResult is the built-in result type for error responses.
	ErrorResult = &ResultTypeExpr{
		UserTypeExpr: &UserTypeExpr{
			AttributeExpr: &AttributeExpr{
				Type:        problemResultType,
				Description: "Problem response result type",
				Validation:  &ValidationExpr{Required: []string{"type", "title", "status", "detail", "instance", "code"}},
			},
			TypeName: "problem",
		},
		Identifier: ErrorResultIdentifier,
		Views:      []*ViewExpr{problemResultView},
	}

	// ProblemResult is the built-in result type for problem-document error
	// responses.
	ProblemResult = ErrorResult

	problemResultType = &Object{
		{"type", &AttributeExpr{
			Type:        String,
			Description: "Type identifies the problem category using a URI reference.",
			UserExamples: []*ExampleExpr{{
				Value: "https://github.com/CaliLuke/loom/problems/bad-request",
			}},
		}},
		{"title", &AttributeExpr{
			Type:        String,
			Description: "Title is a short, human-readable summary of the problem type.",
			UserExamples: []*ExampleExpr{{
				Value: "Bad Request",
			}},
		}},
		{"status", &AttributeExpr{
			Type:        Int,
			Description: "Status is the HTTP status code generated for this occurrence of the problem.",
			UserExamples: []*ExampleExpr{{
				Value: 400,
			}},
		}},
		{"detail", &AttributeExpr{
			Type:        String,
			Description: "Detail is a human-readable explanation specific to this occurrence of the problem.",
			UserExamples: []*ExampleExpr{{
				Value: "parameter 'p' must be an integer",
			}},
		}},
		{"instance", &AttributeExpr{
			Type:        String,
			Description: "Instance identifies this specific occurrence of the problem.",
			UserExamples: []*ExampleExpr{{
				Value: "urn:loom:error:123abc",
			}},
		}},
		{"code", &AttributeExpr{
			Type:         String,
			Description:  "Code is the stable machine-readable problem code.",
			Meta:         MetaExpr{"struct:error:name": nil},
			UserExamples: []*ExampleExpr{{Value: "bad_request"}},
		}},
		{"retry_hint", &AttributeExpr{
			Type:        String,
			Description: "Retry hint is concise guidance on how to correct the request or retry the operation.",
			UserExamples: []*ExampleExpr{{
				Value: "Correct the payload and retry.",
			}},
		}},
	}

	problemResultView = &ViewExpr{
		AttributeExpr: &AttributeExpr{Type: problemResultType},
		Name:          DefaultView,
	}
)

// IsDefaultErrorResult returns true when dt is one of the built-in problem
// document result types used for default HTTP error contracts.
func IsDefaultErrorResult(dt DataType) bool {
	return dt == ErrorResult || dt == ProblemResult
}

// NewResultTypeExpr creates a result type definition but does not
// execute the DSL.
func NewResultTypeExpr(name, identifier string, fn func()) *ResultTypeExpr {
	return &ResultTypeExpr{
		UserTypeExpr: &UserTypeExpr{
			AttributeExpr: &AttributeExpr{Type: &Object{}, DSLFunc: fn},
			TypeName:      name,
			UID:           identifier,
		},
		Identifier: identifier,
	}
}

// CanonicalIdentifier returns the result type identifier sans suffix
// which is what the DSL uses to store and lookup result types.
func CanonicalIdentifier(identifier string) string {
	base, params, err := mime.ParseMediaType(identifier)
	if err != nil {
		return identifier
	}
	id := base
	if i := strings.Index(id, "+"); i != -1 {
		id = id[:i]
	}
	return mime.FormatMediaType(id, params)
}

// Kind implements DataKind.
func (*ResultTypeExpr) Kind() Kind { return ResultTypeKind }

// Dup creates a copy of the result type given a deep copy of its attribute.
// Views keep independent metadata, validation and requiredness overrides while
// sharing field types whose DSL may still be awaiting evaluation.
func (rt *ResultTypeExpr) Dup(att *AttributeExpr) UserType {
	dup := &ResultTypeExpr{
		UserTypeExpr: rt.UserTypeExpr.Dup(att).(*UserTypeExpr),
		Identifier:   rt.Identifier,
	}
	if rt.Views != nil {
		dup.Views = make([]*ViewExpr, len(rt.Views))
	}
	for i, view := range rt.Views {
		copy := *view
		attribute := *view.AttributeExpr
		if attribute.Validation != nil {
			attribute.Validation = attribute.Validation.Dup()
		}
		if attribute.Meta != nil {
			attribute.Meta = attribute.Meta.Dup()
		}
		copy.AttributeExpr = &attribute
		copy.Parent = dup
		copy.RequiredOverrides = append([]string(nil), view.RequiredOverrides...)
		copy.OptionalOverrides = append([]string(nil), view.OptionalOverrides...)
		dup.Views[i] = &copy
	}
	return dup
}

// ID returns the identifier of the result type.
func (rt *ResultTypeExpr) ID() string {
	return rt.Identifier
}

// Name returns the result type name.
func (rt *ResultTypeExpr) Name() string { return rt.TypeName }

// View returns the view with the given name.
func (rt *ResultTypeExpr) View(name string) *ViewExpr {
	for _, v := range rt.Views {
		if v.Name == name {
			return v
		}
	}
	return nil
}

// HasMultipleViews returns true if the result type has more than one view.
func (rt *ResultTypeExpr) HasMultipleViews() bool {
	return len(rt.Views) > 1
}

// ViewHasAttribute returns true if the result type view has the given
// attribute.
func (rt *ResultTypeExpr) ViewHasAttribute(view, attr string) bool {
	v := rt.View(view)
	if v == nil {
		return false
	}
	_, att := v.FindAttribute(attr)
	return att != nil
}

func (rt *ResultTypeExpr) validateExplicitViewMeta() error {
	view, ok := rt.Meta.Last(ViewMetaKey)
	if !ok || view == DefaultView {
		return nil
	}
	if _, err := Project(rt, view); err != nil {
		return err
	}
	return nil
}

// Finalize builds the default view if not explicitly defined and finalizes
// the underlying UserTypeExpr.
func (rt *ResultTypeExpr) Finalize() {
	rt.finalizeViews()
	rt.useExplicitView()
	rt.UserTypeExpr.Finalize()
	// Implicit views must snapshot the complete shape, including inherited
	// fields. Explicit views retain their authored field selection.
	rt.ensureDefaultView()
	rt.finalizeViews()
	seen := make(map[string]struct{})
	walkAttribute(rt.AttributeExpr, func(_ string, att *AttributeExpr) error { // nolint: errcheck
		if rt, ok := att.Type.(*ResultTypeExpr); ok {
			if _, ok := seen[rt.Identifier]; !ok {
				seen[rt.Identifier] = struct{}{}
				rt.finalizeViews()
				rt.useExplicitView()
				rt.ensureDefaultView()
			}
		}
		return nil
	})
}

// useExplicitView projects the result type using the view explicitly set on the
// attribute if any.
func (rt *ResultTypeExpr) useExplicitView() {
	if view, ok := rt.Meta.Last(ViewMetaKey); ok {
		if view == DefaultView {
			return
		}
		p, err := Project(rt, view)
		if err != nil {
			eval.ReportError(err.Error())
			return
		}
		*rt = *p
	}
}

// ensureDefaultView builds the default view if not explicitly defined.
func (rt *ResultTypeExpr) ensureDefaultView() {
	if rt.View(DefaultView) == nil {
		att := DupAtt(rt.AttributeExpr)
		if arr := AsArray(att.Type); arr != nil {
			att.Type = AsObject(arr.ElemType.Type)
		}
		v := &ViewExpr{
			AttributeExpr: att,
			Name:          DefaultView,
			Parent:        rt,
		}
		rt.Views = append(rt.Views, v)
	}
}

// Project creates a ResultTypeExpr containing the fields defined in the view
// expression of m named after the view argument.
//
// The resulting result type defines a default view. The result type identifier is
// computed by adding a parameter called "view" to the original identifier. The
// value of the "view" parameter is the name of the view.
//
// Project returns an error if the view does not exist for the given result type
// or any result type that makes up its attributes recursively. Note that
// individual attributes may use a different view. In this case Project uses
// that view and returns an error if it isn't defined on the attribute type.
func Project(rt *ResultTypeExpr, view string) (*ResultTypeExpr, error) {
	return newValueViewPlan().project(rt, view)
}

func projectedResultValidation(rt *ResultTypeExpr, view *ViewExpr) *ValidationExpr {
	copy := *view
	attribute := *view.AttributeExpr
	copy.AttributeExpr = &attribute
	copy.Parent = rt
	copy.FinalizeRequiredness()
	return copy.Validation
}

func projectedResultDescription(rt *ResultTypeExpr, view string) string {
	desc := rt.Description
	if desc == "" {
		desc = rt.TypeName + " result type"
	}
	return desc + " (" + view + " view)"
}

func projectedResultTypeName(rt *ResultTypeExpr, view string) string {
	typeName := rt.TypeName
	if view != DefaultView {
		typeName += Title(view)
	}
	return typeName
}

// projectIdentifier computes the projected result type identifier by adding the
// "view" param. We need the projected result type identifier to be different so
// that looking up projected result types from ProjectedResultTypes works
// correctly. It's also good for clients.
func (rt *ResultTypeExpr) projectIdentifier(view string) string {
	base, params, err := mime.ParseMediaType(rt.Identifier)
	if err != nil {
		base = rt.Identifier
	}
	if params == nil {
		params = make(map[string]string)
	}
	params["view"] = view
	return mime.FormatMediaType(base, params)
}

// EvalName returns the generic definition name used in error messages.
func (v *ViewExpr) EvalName() string {
	var prefix, suffix string
	if v.Name != "" {
		prefix = fmt.Sprintf("view %#v", v.Name)
	} else {
		prefix = "unnamed view"
	}
	if v.Parent != nil {
		suffix = fmt.Sprintf(" of %s", v.Parent.EvalName())
	}
	return prefix + suffix
}
