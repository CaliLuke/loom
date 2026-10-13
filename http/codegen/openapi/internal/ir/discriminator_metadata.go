package ir

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type (
	discriminatorMetadata struct {
		optional bool
		fallback string
	}
	discriminatorReference struct {
		name    string
		context string
	}
)

func decodeDiscriminatorMetadata(meta expr.MetaExpr, hasDiscriminator bool) (*discriminatorMetadata, error) {
	optional, hasOptional := meta.Last("openapi:discriminator:optional")
	fallback, hasFallback := meta.Last("openapi:discriminator:defaultMapping")
	if !hasOptional && !hasFallback {
		return nil, nil
	}
	if !hasDiscriminator {
		return nil, fmt.Errorf("OpenAPI discriminator metadata requires a union discriminator")
	}
	decoded := &discriminatorMetadata{}
	if hasOptional {
		switch strings.ToLower(strings.TrimSpace(optional)) {
		case "", "true":
			decoded.optional = true
		case "false":
		default:
			return nil, fmt.Errorf("OpenAPI discriminator optional must be true or false")
		}
	}
	if hasFallback {
		var err error
		decoded.fallback, err = discriminatorFallback(fallback)
		if err != nil {
			return nil, err
		}
	}
	if decoded.optional && decoded.fallback == "" {
		return nil, fmt.Errorf("OpenAPI optional discriminator requires an explicit defaultMapping")
	}
	return decoded, nil
}

func discriminatorFallback(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("OpenAPI discriminator defaultMapping: %w", err)
	}
	if !validDiscriminatorURI(value, parsed.Scheme) {
		return "", fmt.Errorf("OpenAPI discriminator defaultMapping must be a nonempty schema name or valid URI reference")
	}
	if strings.HasPrefix(value, "#") {
		name, ok := schemaComponentName(value)
		if !ok || !componentMappingName(name) {
			return "", fmt.Errorf("OpenAPI discriminator defaultMapping local reference must name a component schema")
		}
		return toRef(name), nil
	}
	if parsed.Scheme == "" && parsed.Host == "" && !strings.ContainsAny(value, "/#?") {
		if !componentMappingName(value) {
			return "", fmt.Errorf("OpenAPI discriminator defaultMapping has an invalid component schema name")
		}
		return toRef(value), nil
	}
	return value, nil
}

// validDiscriminatorURI enforces RFC 3986's ASCII character vocabulary. net/url
// also parses browser-style paths; those must not escape this contract boundary.
func validDiscriminatorURI(value, scheme string) bool {
	if value == "" || strings.Count(value, "#") > 1 {
		return false
	}
	remainder := value
	if scheme != "" {
		remainder = value[len(scheme)+1:]
	}
	authorityEnd := 0
	if strings.HasPrefix(remainder, "//") {
		authorityEnd = strings.IndexAny(remainder[2:], "/?#")
		if authorityEnd < 0 {
			authorityEnd = len(remainder)
		} else {
			authorityEnd += 2
		}
	}
	for i := 0; i < len(remainder); i++ {
		ch := remainder[i]
		if ch == '%' {
			if i+2 >= len(remainder) || !uriHex(remainder[i+1]) || !uriHex(remainder[i+2]) {
				return false
			}
			i += 2
			continue
		}
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.ContainsRune("-._~:/?#@!$&'()*+,;=", rune(ch)) {
			continue
		}
		if i < authorityEnd && (ch == '[' || ch == ']') {
			continue
		}
		return false
	}
	return true
}

func uriHex(ch byte) bool {
	return ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F'
}

func componentMappingName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func (a *Analyzer) applyDiscriminatorMetadata(s *Schema, meta expr.MetaExpr, context string) {
	decoded, err := decodeDiscriminatorMetadata(meta, s.Discriminator != nil)
	if err != nil {
		if a.err == nil {
			a.err = fmt.Errorf("%s: %w", context, err)
		}
		return
	}
	if decoded == nil {
		return
	}
	s.Discriminator.DefaultMapping = decoded.fallback
	s.Discriminator.Optional = decoded.optional
	if decoded.optional {
		s.Required = slices.DeleteFunc(s.Required, func(name string) bool {
			return name == s.Discriminator.PropertyName
		})
	}
	if name, local := schemaComponentName(decoded.fallback); local {
		a.discriminatorReferences = append(a.discriminatorReferences, discriminatorReference{name, context})
	}
}
