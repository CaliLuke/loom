package expr

import "github.com/CaliLuke/loom/eval"

type grpcPresenceCheck struct {
	attribute *AttributeExpr
	source    *AttributeExpr
	mapping   bool
	location  string
	metadata  bool
}

func (e *GRPCEndpointExpr) validateNullableTransport(verr *eval.ValidationErrors) {
	if e == nil {
		return
	}
	var checks []grpcPresenceCheck
	var payload, streamingPayload, result *AttributeExpr
	if e.MethodExpr != nil {
		payload, streamingPayload, result = e.MethodExpr.Payload, e.MethodExpr.StreamingPayload, e.MethodExpr.Result
		// Transport placeholders are incomplete until Finalize. The service
		// occurrences already own all inherited presence intent at this phase.
		checks = append(checks,
			grpcPresenceCheck{attribute: payload, location: "request payloads"},
			grpcPresenceCheck{attribute: e.MethodExpr.StreamingPayload, location: "streaming request payloads"},
			grpcPresenceCheck{attribute: result, location: "response results"},
			grpcPresenceCheck{attribute: e.MethodExpr.StreamingResult, location: "streaming response results"},
		)
	}
	checks = appendGRPCMessagePresenceChecks(checks, e.Request, payload, "request messages")
	checks = appendGRPCMessagePresenceChecks(checks, e.StreamingRequest, streamingPayload, "streaming request messages")
	checks = appendGRPCMetadataPresenceChecks(checks, e.Metadata, payload, "request metadata")
	checks = appendGRPCResponsePresenceChecks(checks, e.Response, result, "response")
	for _, mapped := range e.GRPCErrors {
		if mapped == nil || mapped.Response == nil {
			continue
		}
		var source *AttributeExpr
		if err := mapped.sourceError(); err != nil {
			source = err.AttributeExpr
			checks = append(checks, grpcPresenceCheck{attribute: source, location: "error response results"})
		}
		checks = appendGRPCResponsePresenceChecks(checks, mapped.Response, source, "error response")
	}
	seen := make(map[string]bool)
	for _, check := range checks {
		unsupported := containsUnsupportedGRPCPresence(check.attribute)
		if check.mapping {
			unsupported = unsupportedGRPCMappingPresence(check.attribute, check.source, check.metadata, true, true, make(map[grpcPresencePair]bool))
		}
		if unsupported && !seen[check.location] {
			verr.Add(e, "gRPC %s do not support nullable attributes; use Any in messages when null must cross gRPC", check.location)
			seen[check.location] = true
		}
	}
}

func appendGRPCResponsePresenceChecks(checks []grpcPresenceCheck, response *GRPCResponseExpr, source *AttributeExpr, prefix string) []grpcPresenceCheck {
	if response == nil {
		return checks
	}
	checks = appendGRPCMessagePresenceChecks(checks, response.Message, source, prefix+" messages")
	checks = appendGRPCMetadataPresenceChecks(checks, response.Headers, source, prefix+" header metadata")
	return appendGRPCMetadataPresenceChecks(checks, response.Trailers, source, prefix+" trailer metadata")
}

func appendGRPCMetadataPresenceChecks(checks []grpcPresenceCheck, mapped *MappedAttributeExpr, source *AttributeExpr, location string) []grpcPresenceCheck {
	if mapped == nil {
		return checks
	}
	return append(checks, grpcPresenceCheck{
		attribute: mapped.Attribute(), source: source, mapping: true, location: location, metadata: true,
	})
}

func appendGRPCMessagePresenceChecks(checks []grpcPresenceCheck, local, source *AttributeExpr, location string) []grpcPresenceCheck {
	return append(checks, grpcPresenceCheck{
		attribute: local, source: source, mapping: true, location: location,
	})
}
