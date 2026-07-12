package expr

import (
	"regexp"

	"github.com/CaliLuke/loom/eval"
)

type (
	// HTTPCORSExpr describes the CORS policy for an API or HTTP service.
	HTTPCORSExpr struct {
		Origins []*HTTPCORSOriginExpr
		Runtime bool
	}

	// HTTPCORSOriginExpr describes one allowed CORS origin and its response
	// policy. Pattern is either an exact origin, "*", or a regular expression.
	HTTPCORSOriginExpr struct {
		Pattern     string
		Regex       bool
		Methods     []string
		Headers     []string
		Expose      []string
		MaxAge      int
		Credentials bool
	}
)

// EvalName returns the expression name used in validation errors.
func (*HTTPCORSExpr) EvalName() string {
	return "HTTP CORS"
}

// EvalName returns the expression name used in validation errors.
func (o *HTTPCORSOriginExpr) EvalName() string {
	return "HTTP CORS origin " + o.Pattern
}

// Validate validates the CORS policy.
func (c *HTTPCORSExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if c == nil {
		return verr
	}
	if c.Runtime {
		if len(c.Origins) > 0 {
			verr.Add(c, "runtime CORS cannot define design-time origins")
		}
		return verr
	}
	if len(c.Origins) == 0 {
		verr.Add(c, "CORS must define at least one origin")
		return verr
	}
	for _, origin := range c.Origins {
		if origin.Pattern == "" {
			verr.Add(origin, "CORS origin cannot be empty")
		}
		if origin.Credentials && origin.Pattern == "*" {
			verr.Add(origin, "CORS credentials are incompatible with wildcard origin")
		}
		if origin.Regex {
			// Validate only that the pattern compiles. The runtime matcher
			// anchors patterns to the full origin string (\A(?:...)\z), so
			// unanchored patterns cannot cause cross-origin bypass; no
			// anchoring is required or enforced at design time.
			if _, err := regexp.Compile(origin.Pattern); err != nil {
				verr.Add(origin, "CORS origin regex %q is invalid: %s", origin.Pattern, err)
			}
		}
		if origin.MaxAge < 0 {
			verr.Add(origin, "CORS max age cannot be negative")
		}
	}
	return verr
}

// Dup returns a deep copy of c.
func (c *HTTPCORSExpr) Dup() *HTTPCORSExpr {
	if c == nil {
		return nil
	}
	out := &HTTPCORSExpr{Origins: make([]*HTTPCORSOriginExpr, len(c.Origins)), Runtime: c.Runtime}
	for i, origin := range c.Origins {
		cp := *origin
		cp.Methods = append([]string(nil), origin.Methods...)
		cp.Headers = append([]string(nil), origin.Headers...)
		cp.Expose = append([]string(nil), origin.Expose...)
		out.Origins[i] = &cp
	}
	return out
}
