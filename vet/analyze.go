package vet

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type scalarRecommendation struct {
	semantic    string
	replacement string
}

var (
	describedRangePattern = regexp.MustCompile(`(?i)\bfrom\s+(-?\d+(?:\.\d+)?)\s+to\s+(-?\d+(?:\.\d+)?)\b`)
	normalizedWordPattern = regexp.MustCompile(`(?i)\bnormalized\b`)
	booleanAnyPattern     = regexp.MustCompile(`\b(boolean|true/false|true or false|whether)\b`)
	integerAnyPattern     = regexp.MustCompile(`\b(integer|whole number|number of)\b`)
	numberAnyPattern      = regexp.MustCompile(`\b(numeric value|decimal number|decimal value|floating-point|floating point|number value)\b`)
	timestampAnyPattern   = regexp.MustCompile(`\b(timestamp|date-time|datetime|rfc3339|rfc 3339)\b`)
	uuidAnyPattern        = regexp.MustCompile(`\b(uuid|universally unique identifier)\b`)
	uriAnyPattern         = regexp.MustCompile(`\b(uri|url)\b`)
	emailAnyPattern       = regexp.MustCompile(`\bemail (address|string|value)\b`)
	ipAnyPattern          = regexp.MustCompile(`\b(ip|ipv4|ipv6) address\b`)
)

// Analyze inspects an evaluated design and the Go module rooted at moduleDir.
func Analyze(root *expr.RootExpr, moduleDir string) (Report, error) {
	var report Report
	analyzeHTTPErrorSemantics(root, &report)
	analyzeAttributeSemantics(root, &report)

	sourceDiagnostics, err := analyzeModuleWithDesign(moduleDir, root)
	if err != nil {
		return Report{}, err
	}
	report.Diagnostics = append(report.Diagnostics, sourceDiagnostics...)
	report.Sort()
	return report, nil
}

func analyzeHTTPErrorSemantics(root *expr.RootExpr, report *Report) {
	if root == nil || root.API == nil || root.API.HTTP == nil {
		return
	}
	owners := designErrorLocations(root)
	type diagnosticKey struct {
		errorExpr *expr.ErrorExpr
		status    int
		rule      string
	}
	seen := make(map[diagnosticKey]struct{})
	appendDiagnostic := func(httpError *expr.HTTPErrorExpr, status int, rule, message string) {
		key := diagnosticKey{errorExpr: httpError.ErrorExpr, status: status, rule: rule}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		location, exists := owners[httpError.ErrorExpr]
		if !exists {
			location = Location{Path: "error." + httpError.Name}
		}
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Rule:     rule,
			Severity: SeverityError,
			Message:  message,
			Location: location,
		})
	}
	for _, service := range root.API.HTTP.Services {
		if service == nil {
			continue
		}
		for _, endpoint := range service.HTTPEndpoints {
			if endpoint == nil || endpoint.MethodExpr == nil {
				continue
			}
			for _, httpError := range endpoint.HTTPErrors {
				if httpError == nil || httpError.ErrorExpr == nil || httpError.Response == nil {
					continue
				}
				status := httpError.Response.StatusCode
				if status >= 500 && status <= 599 && !hasMeta(httpError.Meta, "loom:error:fault") && !suppressed(httpError.Meta, RuleServerErrorFault) {
					appendDiagnostic(
						httpError,
						status,
						RuleServerErrorFault,
						fmt.Sprintf("HTTP %d error %q must declare Fault()", status, httpError.Name),
					)
				}
				if isRetryableStatus(status) && !hasRetryMetadata(httpError.ErrorExpr) && !suppressed(httpError.Meta, RuleRetryableError) {
					appendDiagnostic(
						httpError,
						status,
						RuleRetryableError,
						fmt.Sprintf(
							"HTTP %d error %q must declare Temporary() or Remedy(func() {\n    RetryHint(\"...\")\n})",
							status,
							httpError.Name,
						),
					)
				}
			}
		}
	}
}

func designErrorLocations(root *expr.RootExpr) map[*expr.ErrorExpr]Location {
	locations := make(map[*expr.ErrorExpr]Location)
	add := func(errorExpr *expr.ErrorExpr, path string) {
		if errorExpr == nil {
			return
		}
		if _, exists := locations[errorExpr]; !exists {
			locations[errorExpr] = Location{Path: path}
		}
	}
	for _, errorExpr := range root.Errors {
		if errorExpr == nil {
			continue
		}
		add(errorExpr, "api.error."+errorExpr.Name)
	}
	for _, service := range root.Services {
		if service == nil {
			continue
		}
		for _, errorExpr := range service.Errors {
			if errorExpr == nil {
				continue
			}
			add(errorExpr, fmt.Sprintf("service.%s.error.%s", service.Name, errorExpr.Name))
		}
		for _, method := range service.Methods {
			if method == nil {
				continue
			}
			for _, errorExpr := range method.Errors {
				if errorExpr == nil {
					continue
				}
				add(errorExpr, fmt.Sprintf("service.%s.method.%s.error.%s", service.Name, method.Name, errorExpr.Name))
			}
		}
	}
	return locations
}

func isRetryableStatus(status int) bool {
	switch status {
	case 429, 502, 503:
		return true
	default:
		return false
	}
}

func hasRetryMetadata(errorExpr *expr.ErrorExpr) bool {
	if hasMeta(errorExpr.Meta, "loom:error:temporary") {
		return true
	}
	return errorExpr.Remedy != nil && strings.TrimSpace(errorExpr.Remedy.RetryHint) != ""
}

func analyzeAttributeSemantics(root *expr.RootExpr, report *Report) {
	if root == nil {
		return
	}
	seenTypes := make(map[string]struct{}, len(root.Types)+len(root.ResultTypes))
	for _, userType := range root.Types {
		analyzeUserType(userType, seenTypes, report)
	}
	for _, resultType := range root.ResultTypes {
		analyzeUserType(resultType, seenTypes, report)
	}
	seenErrors := make(map[*expr.ErrorExpr]struct{})
	for _, errorExpr := range root.Errors {
		analyzeErrorAttribute("api", errorExpr, seenErrors, report)
	}
	for _, service := range root.Services {
		if service == nil {
			continue
		}
		for _, errorExpr := range service.Errors {
			analyzeErrorAttribute("service."+service.Name, errorExpr, seenErrors, report)
		}
		for _, method := range service.Methods {
			if method == nil {
				continue
			}
			for _, errorExpr := range method.Errors {
				analyzeErrorAttribute(
					fmt.Sprintf("service.%s.method.%s", service.Name, method.Name),
					errorExpr,
					seenErrors,
					report,
				)
			}
			prefix := fmt.Sprintf("service.%s.method.%s", service.Name, method.Name)
			analyzeInlineAttribute(prefix+".payload", "payload", method.Payload, report, make(map[string]struct{}))
			analyzeInlineAttribute(prefix+".result", "result", method.Result, report, make(map[string]struct{}))
			if method.StreamingPayload != method.Payload {
				analyzeInlineAttribute(prefix+".streaming_payload", "streaming_payload", method.StreamingPayload, report, make(map[string]struct{}))
			}
			if method.StreamingResult != method.Result {
				analyzeInlineAttribute(prefix+".streaming_result", "streaming_result", method.StreamingResult, report, make(map[string]struct{}))
			}
		}
	}
}

func analyzeErrorAttribute(prefix string, errorExpr *expr.ErrorExpr, seen map[*expr.ErrorExpr]struct{}, report *Report) {
	if errorExpr == nil {
		return
	}
	if _, exists := seen[errorExpr]; exists {
		return
	}
	seen[errorExpr] = struct{}{}
	analyzeInlineAttribute(
		prefix+".error."+errorExpr.Name,
		errorExpr.Name,
		errorExpr.AttributeExpr,
		report,
		make(map[string]struct{}),
	)
}

func analyzeUserType(userType expr.UserType, seen map[string]struct{}, report *Report) {
	if userType == nil {
		return
	}
	if _, ok := seen[userType.ID()]; ok {
		return
	}
	seen[userType.ID()] = struct{}{}
	analyzeInlineAttribute("type."+userType.Name(), userType.Name(), userType.Attribute(), report, make(map[string]struct{}))
}

func analyzeInlineAttribute(path, name string, attribute *expr.AttributeExpr, report *Report, stack map[string]struct{}) {
	if attribute == nil || attribute.Type == nil {
		return
	}
	analyzeAttributeDescription(path, name, attribute, report)

	if userType, ok := attribute.Type.(expr.UserType); ok {
		if _, recursive := stack[userType.ID()]; recursive {
			return
		}
		stack[userType.ID()] = struct{}{}
		defer delete(stack, userType.ID())
		return
	}

	switch actual := attribute.Type.(type) {
	case *expr.Object:
		for _, named := range *actual {
			if named == nil {
				continue
			}
			analyzeInlineAttribute(path+"."+named.Name, named.Name, named.Attribute, report, stack)
		}
	case *expr.Array:
		analyzeInlineAttribute(path+"[]", name, actual.ElemType, report, stack)
	case *expr.Map:
		analyzeInlineAttribute(path+".key", name, actual.KeyType, report, stack)
		analyzeInlineAttribute(path+".value", name, actual.ElemType, report, stack)
	case *expr.Union:
		for _, variant := range actual.Values {
			if variant == nil {
				continue
			}
			analyzeInlineAttribute(path+"."+variant.Name, variant.Name, variant.Attribute, report, stack)
		}
	}
}

func analyzeAttributeDescription(path, name string, attribute *expr.AttributeExpr, report *Report) {
	description := strings.ToLower(attribute.Description)
	if recommendation, ok := untypedScalarRecommendation(name, description); isAny(attribute.Type) && ok && !attributeSuppressed(attribute, RuleUntypedSemanticAttribute) {
		appendWarning(
			report,
			RuleUntypedSemanticAttribute,
			path,
			fmt.Sprintf(
				"Any attribute %q has %s semantics; use %s instead of Any or suppress this warning",
				name,
				recommendation.semantic,
				recommendation.replacement,
			),
		)
	}
	normalizedMissing := isNumber(attribute.Type) &&
		normalizedWordPattern.MatchString(description) &&
		!hasRange(attribute, 0, 1) &&
		!attributeSuppressed(attribute, RuleNormalizedRange)
	if isNumber(attribute.Type) && strings.Contains(description, "1-based") && !hasLowerBound(attribute, 1, isInteger(attribute.Type)) && !attributeSuppressed(attribute, RuleDescriptionMinimum) {
		appendWarning(report, RuleDescriptionMinimum, path, "description says 1-based but the attribute does not enforce Minimum(1)")
	}
	if isNumber(attribute.Type) && strings.Contains(description, "zero-based") && !hasLowerBound(attribute, 0, isInteger(attribute.Type)) && !attributeSuppressed(attribute, RuleDescriptionMinimum) {
		appendWarning(report, RuleDescriptionMinimum, path, "description says zero-based but the attribute does not enforce Minimum(0)")
	}
	if match := describedRangePattern.FindStringSubmatch(description); isNumber(attribute.Type) && len(match) == 3 && !attributeSuppressed(attribute, RuleDescriptionRange) {
		minimum, minimumErr := strconv.ParseFloat(match[1], 64)
		maximum, maximumErr := strconv.ParseFloat(match[2], 64)
		if minimumErr == nil && maximumErr == nil && !hasRange(attribute, minimum, maximum) {
			if !normalizedMissing || minimum != 0 || maximum != 1 {
				appendWarning(report, RuleDescriptionRange, path, fmt.Sprintf("description says from %s to %s but the attribute does not enforce both bounds", match[1], match[2]))
			}
		}
	}
	if kind := semanticStringKind(name, description); isString(attribute.Type) && kind != "" && !hasStringValidation(attribute) && !attributeSuppressed(attribute, RuleStringFormat) {
		appendWarning(report, RuleStringFormat, path, fmt.Sprintf("%s string has no Format or Pattern validation", kind))
	}
	if normalizedMissing {
		appendWarning(report, RuleNormalizedRange, path, "normalized number does not enforce the range from 0 to 1")
	}
}

func appendWarning(report *Report, rule, path, message string) {
	report.Diagnostics = append(report.Diagnostics, Diagnostic{
		Rule:     rule,
		Severity: SeverityWarning,
		Message:  message,
		Location: Location{Path: path},
	})
}

func hasMeta(meta expr.MetaExpr, key string) bool {
	_, ok := meta[key]
	return ok
}

func suppressed(meta expr.MetaExpr, rule string) bool {
	for _, value := range meta[SuppressionMeta] {
		if value == rule || value == "all" {
			return true
		}
	}
	return false
}

func attributeSuppressed(attribute *expr.AttributeExpr, rule string) bool {
	return anyAttributeLayer(attribute, func(layer *expr.AttributeExpr) bool {
		return suppressed(layer.Meta, rule)
	})
}

func hasLowerBound(attribute *expr.AttributeExpr, minimum float64, integer bool) bool {
	return anyAttributeLayer(attribute, func(layer *expr.AttributeExpr) bool {
		validation := layer.Validation
		if validation == nil {
			return false
		}
		if validation.Minimum != nil && *validation.Minimum >= minimum {
			return true
		}
		if validation.ExclusiveMinimum == nil {
			return false
		}
		if integer {
			return *validation.ExclusiveMinimum >= minimum-1
		}
		return *validation.ExclusiveMinimum >= minimum
	})
}

func hasRange(attribute *expr.AttributeExpr, minimum, maximum float64) bool {
	var lower, upper bool
	anyAttributeLayer(attribute, func(layer *expr.AttributeExpr) bool {
		validation := layer.Validation
		if validation == nil {
			return false
		}
		if validation.Minimum != nil && *validation.Minimum >= minimum {
			lower = true
		}
		if validation.ExclusiveMinimum != nil && *validation.ExclusiveMinimum >= minimum {
			lower = true
		}
		if validation.Maximum != nil && *validation.Maximum <= maximum {
			upper = true
		}
		if validation.ExclusiveMaximum != nil && *validation.ExclusiveMaximum <= maximum {
			upper = true
		}
		return lower && upper
	})
	return lower && upper
}

func hasStringValidation(attribute *expr.AttributeExpr) bool {
	return anyAttributeLayer(attribute, func(layer *expr.AttributeExpr) bool {
		validation := layer.Validation
		return validation != nil && (validation.Format != "" || validation.Pattern != "")
	})
}

func anyAttributeLayer(attribute *expr.AttributeExpr, predicate func(*expr.AttributeExpr) bool) bool {
	seen := make(map[*expr.AttributeExpr]struct{})
	for attribute != nil {
		if _, exists := seen[attribute]; exists {
			return false
		}
		seen[attribute] = struct{}{}
		if predicate(attribute) {
			return true
		}
		userType, ok := attribute.Type.(expr.UserType)
		if !ok {
			return false
		}
		attribute = userType.Attribute()
	}
	return false
}

func semanticStringKind(name, description string) string {
	lowerName := strings.ToLower(name)
	switch {
	case lowerName == "email" || strings.HasSuffix(lowerName, "_email") || strings.Contains(description, "email address"):
		return "email"
	case strings.Contains(lowerName, "uuid") || mentionsUUIDContract(description):
		return "UUID"
	case lowerName == "url" || lowerName == "uri" || strings.HasSuffix(lowerName, "_url") || strings.HasSuffix(lowerName, "_uri") || strings.Contains(description, " url") || strings.Contains(description, " uri"):
		return "URL"
	default:
		return ""
	}
}

func mentionsUUIDContract(description string) bool {
	description = strings.TrimSpace(description)
	if strings.Contains(description, " or ") {
		return false
	}
	return strings.HasPrefix(description, "uuid ") ||
		strings.HasPrefix(description, "a uuid ") ||
		strings.Contains(description, "must be a uuid")
}

func untypedScalarRecommendation(name, description string) (scalarRecommendation, bool) {
	description = strings.TrimSpace(description)
	if strings.Contains(description, " or ") && !strings.Contains(description, "true or false") {
		return scalarRecommendation{}, false
	}
	lowerName := strings.ToLower(name)
	switch {
	case lowerName == "timestamp" || strings.HasSuffix(lowerName, "_at") || strings.HasSuffix(lowerName, "_timestamp"):
		return scalarRecommendation{"timestamp", "String with Format(FormatDateTime)"}, true
	case uuidAnyPattern.MatchString(description):
		return scalarRecommendation{"UUID", "String with Format(FormatUUID)"}, true
	case timestampAnyPattern.MatchString(description):
		return scalarRecommendation{"timestamp", "String with Format(FormatDateTime)"}, true
	case booleanAnyPattern.MatchString(description):
		return scalarRecommendation{"boolean", "Boolean"}, true
	case integerAnyPattern.MatchString(description):
		return scalarRecommendation{"integer", "Int"}, true
	case numberAnyPattern.MatchString(description):
		return scalarRecommendation{"number", "Float64"}, true
	case uriAnyPattern.MatchString(description):
		return scalarRecommendation{"URI", "String with Format(FormatURI)"}, true
	case emailAnyPattern.MatchString(description):
		return scalarRecommendation{"email", "String with Format(FormatEmail)"}, true
	case ipAnyPattern.MatchString(description):
		return scalarRecommendation{"IP address", "String with Format(FormatIP)"}, true
	default:
		return scalarRecommendation{}, false
	}
}

func isAny(dataType expr.DataType) bool {
	seen := make(map[string]struct{})
	for dataType != nil {
		if dataType == expr.Any {
			return true
		}
		userType, ok := dataType.(expr.UserType)
		if !ok || userType.Attribute() == nil {
			return false
		}
		if _, exists := seen[userType.ID()]; exists {
			return false
		}
		seen[userType.ID()] = struct{}{}
		dataType = userType.Attribute().Type
	}
	return false
}

func isString(dataType expr.DataType) bool {
	return underlyingKind(dataType) == expr.StringKind
}

func isInteger(dataType expr.DataType) bool {
	switch underlyingKind(dataType) {
	case expr.IntKind, expr.Int32Kind, expr.Int64Kind, expr.UIntKind, expr.UInt32Kind, expr.UInt64Kind:
		return true
	default:
		return false
	}
}

func isNumber(dataType expr.DataType) bool {
	switch underlyingKind(dataType) {
	case expr.IntKind, expr.Int32Kind, expr.Int64Kind, expr.UIntKind, expr.UInt32Kind, expr.UInt64Kind, expr.Float32Kind, expr.Float64Kind:
		return true
	default:
		return false
	}
}

func underlyingKind(dataType expr.DataType) expr.Kind {
	for {
		userType, ok := dataType.(expr.UserType)
		if !ok || userType.Attribute() == nil || userType.Attribute().Type == nil {
			return dataType.Kind()
		}
		dataType = userType.Attribute().Type
	}
}
