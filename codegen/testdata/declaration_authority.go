package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// DeclarationAuthorityDSL exercises authored declaration identity through
// controlled request, response and streaming wrappers. Scalar object fields
// keep its generated output independent of byte-schema vocabulary changes.
func DeclarationAuthorityDSL() {
	declarationAuthorityDSL(false)
}

// DeclarationAuthorityBytesDSL adds an optional byte field to the same authored
// alias graph, preserving independent example contexts across schema projections.
func DeclarationAuthorityBytesDSL() {
	declarationAuthorityDSL(true)
}

func declarationAuthorityDSL(bytes bool) {
	API("declaration-authority", func() {
	})
	base := Type("AuthorityBase", func() {
		Attribute("count", Int)
		if bytes {
			Attribute("blob", Bytes)
		}
		Required("count")
	})
	peer := Type("AuthorityPeer", func() {
		Attribute("count", Int)
		if bytes {
			Attribute("blob", Bytes)
		}
		Required("count")
		Example(map[string]any{"count": 17})
	})
	first := Type("AuthorityAliasOne", base)
	second := Type("AuthorityAliasTwo", first)
	envelope := Type("AuthorityEnvelope", func() {
		Attribute("base", base)
		Attribute("peer", peer)
		Attribute("items", ArrayOf(second))
		Attribute("entries", MapOf(String, first))
		Attribute("choice", OneOf(base, peer))
		Attribute("next", "AuthorityEnvelope")
	})
	Service("declaration-authority", func() {
		Method("ordinary", func() {
			Payload(envelope)
			Result(envelope)
			HTTP(func() {
				POST("/ordinary")
			})
		})
		declarationAuthorityStream("base", base)
		declarationAuthorityStream("one", first)
		declarationAuthorityStream("two", second)
		declarationAuthorityStream("positions", envelope)
	})
}

func declarationAuthorityStream(name string, typ expr.UserType) {
	Method(name, func() {
		StreamingPayload(typ)
		StreamingResult(typ)
		HTTP(func() {
			GET("/" + name)
		})
	})
}
