package http

type (
	// ParamStyle selects an OpenAPI parameter wire grammar.
	ParamStyle uint8
	// ParamKind distinguishes scalar, sequential and associative raw values.
	ParamKind uint8
	// ParamCodec is a resolved style and explode setting.
	ParamCodec struct {
		// Style selects the wire grammar.
		Style ParamStyle
		// Explode selects the expanded spelling for collections.
		Explode bool
	}
	// ParamPair is one decoded associative entry.
	ParamPair struct {
		// Key is the decoded member name.
		Key string
		// Value is the decoded member text.
		Value string
	}
	// ParamValue preserves raw text and presence before typed conversion.
	ParamValue struct {
		// Kind selects Text, Items or Pairs.
		Kind ParamKind
		// Present distinguishes absent bindings from supplied empty values.
		Present bool
		// Text contains a scalar value.
		Text string
		// Items contains ordered sequence elements.
		Items []string
		// Pairs contains associative entries in received or authored order.
		Pairs []ParamPair
	}
	// ParamNamespace identifies sibling claims and an optional binding allowlist.
	ParamNamespace struct {
		// Literals contains exact decoded keys owned by siblings.
		Literals []string
		// Prefixes contains sibling deepObject names, each claiming name[key].
		Prefixes []string
		// OwnedLiterals limits a fixed binding to these decoded wire keys. Nil
		// permits dynamic ownership; a nonnil empty list owns no keys.
		OwnedLiterals []string
	}
)

const (
	// ParamSimple uses simple list or associative expansion.
	ParamSimple ParamStyle = iota
	// ParamMatrix uses named semicolon-prefixed path expansion.
	ParamMatrix
	// ParamLabel uses dot-prefixed path expansion.
	ParamLabel
	// ParamForm uses form-style URI expansion.
	ParamForm
	// ParamSpaceDelimited separates members with encoded spaces.
	ParamSpaceDelimited
	// ParamPipeDelimited separates members with encoded vertical bars.
	ParamPipeDelimited
	// ParamDeepObject carries entries in bracketed keys.
	ParamDeepObject
	// ParamCookie uses verbatim cookie pairs.
	ParamCookie
)

const (
	// ParamScalar carries one scalar text value.
	ParamScalar ParamKind = iota
	// ParamList carries an ordered scalar sequence.
	ParamList
	// ParamPairs carries an associative scalar collection.
	ParamPairs
)
