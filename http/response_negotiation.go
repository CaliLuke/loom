package http

import (
	"encoding/json/v2"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// ResponseNegotiationPolicy is an immutable list of response media types
	// supported by an HTTP handler.
	ResponseNegotiationPolicy struct {
		mediaTypes []string
	}

	mediaRange struct {
		typeName    string
		subtype     string
		quality     float64
		specificity int
	}
)

// NewResponseNegotiationPolicy validates mediaTypes and returns a strict
// response negotiation policy. Supported values must be concrete media types.
func NewResponseNegotiationPolicy(mediaTypes ...string) (ResponseNegotiationPolicy, error) {
	if len(mediaTypes) == 0 {
		return ResponseNegotiationPolicy{}, fmt.Errorf("response negotiation policy must define at least one media type")
	}
	seen := make(map[string]struct{}, len(mediaTypes))
	normalized := make([]string, 0, len(mediaTypes))
	for _, value := range mediaTypes {
		mediaType, _, err := mime.ParseMediaType(value)
		if err != nil || mediaType == "" || strings.Contains(mediaType, "*") {
			return ResponseNegotiationPolicy{}, fmt.Errorf("response media type %q must be concrete", value)
		}
		if _, ok := seen[mediaType]; ok {
			continue
		}
		seen[mediaType] = struct{}{}
		normalized = append(normalized, mediaType)
	}
	return ResponseNegotiationPolicy{mediaTypes: normalized}, nil
}

// MediaTypes returns a copy of the response media types accepted by the
// policy.
func (p ResponseNegotiationPolicy) MediaTypes() []string {
	return append([]string(nil), p.mediaTypes...)
}

// Handler rejects requests whose Accept header excludes every configured
// response media type. Rejections use Loom's default RFC 9457 problem body.
// Handler panics when p was not constructed by NewResponseNegotiationPolicy.
func (p ResponseNegotiationPolicy) Handler(next http.Handler) http.Handler {
	if len(p.mediaTypes) == 0 {
		panic("loom: use NewResponseNegotiationPolicy to construct a response negotiation policy")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeVary(w.Header(), "Accept")
		accept := requestAcceptHeader(r)
		if acceptsAnyMediaType(accept, p.mediaTypes) {
			next.ServeHTTP(w, r)
			return
		}
		writeNegotiationProblem(w, r, loom.NotAcceptableError(accept))
	})
}

func acceptsAnyMediaType(accept string, supported []string) bool {
	if strings.TrimSpace(accept) == "" {
		return true
	}
	ranges := parseMediaRanges(accept)
	for _, mediaType := range supported {
		parts := strings.SplitN(mediaType, "/", 2)
		bestSpecificity := -1
		bestQuality := float64(0)
		for _, candidate := range ranges {
			if candidate.typeName != "*" && candidate.typeName != parts[0] {
				continue
			}
			if candidate.subtype != "*" && candidate.subtype != parts[1] {
				continue
			}
			if candidate.specificity > bestSpecificity ||
				(candidate.specificity == bestSpecificity && candidate.quality > bestQuality) {
				bestSpecificity = candidate.specificity
				bestQuality = candidate.quality
			}
		}
		if bestSpecificity >= 0 && bestQuality > 0 {
			return true
		}
	}
	return false
}

func parseMediaRanges(accept string) []mediaRange {
	values := splitHTTPList(accept)
	ranges := make([]mediaRange, 0, len(values))
	for _, value := range values {
		mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		parts := strings.SplitN(mediaType, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		quality := float64(1)
		if rawQuality, ok := params["q"]; ok {
			parsed, parseErr := strconv.ParseFloat(rawQuality, 64)
			if parseErr != nil || parsed < 0 || parsed > 1 {
				continue
			}
			quality = parsed
		}
		specificity := 2
		if parts[0] == "*" {
			specificity = 0
		} else if parts[1] == "*" {
			specificity = 1
		}
		ranges = append(ranges, mediaRange{
			typeName:    parts[0],
			subtype:     parts[1],
			quality:     quality,
			specificity: specificity,
		})
	}
	return ranges
}

func splitHTTPList(value string) []string {
	values := make([]string, 0, strings.Count(value, ",")+1)
	start := 0
	quoted := false
	escaped := false
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '\\':
			if quoted && !escaped {
				escaped = true
				continue
			}
		case '"':
			if !escaped {
				quoted = !quoted
			}
		case ',':
			if !quoted {
				values = append(values, value[start:i])
				start = i + 1
			}
		}
		escaped = false
	}
	return append(values, value[start:])
}

func writeNegotiationProblem(w http.ResponseWriter, r *http.Request, err error) {
	problem := NewErrorResponse(r.Context(), err)
	body, marshalErr := json.Marshal(problem)
	if marshalErr != nil {
		panic(fmt.Errorf("marshal Loom response negotiation problem: %w", marshalErr))
	}
	w.Header().Set("Content-Type", ProblemJSONContentType)
	w.WriteHeader(problem.StatusCode())
	if _, writeErr := w.Write(append(body, '\n')); writeErr != nil {
		return
	}
}
