package testdata

import (
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// ViewedStreamResultDSL exercises scalar and collection views over JSON-RPC
// WebSocket and SSE, with required fields omitted by the tiny view.
func ViewedStreamResultDSL() {
	dsl.API("viewed", func() {
		dsl.JSONRPC(func() {})
	})
	note := dsl.ResultType("application/vnd.note", func() {
		dsl.TypeName("Note")
		dsl.Attributes(func() {
			dsl.Attribute("id", dsl.String)
			dsl.Attribute("title", dsl.String)
			dsl.Attribute("body", dsl.String)
			dsl.Required("id")
		})
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("title")
			dsl.Attribute("body")
			dsl.Required("title")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	dsl.Service("files", func() {
		viewedStreamFiles(note)
	})
	dsl.Service("feed", func() {
		viewedStreamFeed(note)
	})
}

// ViewedStreamResponseIDDSL keeps response IDs distinct from projected body fields.
func ViewedStreamResponseIDDSL() {
	dsl.API("idviews", func() {
		dsl.JSONRPC(func() {})
	})
	input := dsl.Type("Input", func() {
		dsl.ID("id", dsl.String)
		dsl.Required("id")
	})
	note := dsl.ResultType("application/vnd.idnote", func() {
		dsl.TypeName("Note")
		dsl.Attributes(func() {
			dsl.ID("id", dsl.String)
			dsl.Attribute("title", dsl.String)
			dsl.Required("id", "title")
		})
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("title")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	dsl.Service("files", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		dsl.Method("talk", func() {
			dsl.StreamingPayload(input)
			dsl.StreamingResult(note)
			dsl.JSONRPC(func() {})
		})
	})
}

func viewedStreamFiles(note *expr.ResultTypeExpr) {
	tiny := func() {
		dsl.View("tiny")
	}
	dsl.JSONRPC(func() {
		dsl.GET("/ws")
	})
	dsl.Method("upload", func() {
		dsl.StreamingPayload(dsl.String)
		dsl.Result(note)
		dsl.JSONRPC(func() {})
	})
	dsl.Method("uploadTiny", func() {
		dsl.StreamingPayload(dsl.String)
		dsl.Result(note, tiny)
		dsl.JSONRPC(func() {})
	})
	dsl.Method("uploadList", func() {
		dsl.StreamingPayload(dsl.String)
		dsl.Result(dsl.CollectionOf(note))
		dsl.JSONRPC(func() {})
	})
	dsl.Method("talk", func() {
		dsl.StreamingPayload(dsl.String)
		dsl.StreamingResult(note)
		dsl.JSONRPC(func() {})
	})
	dsl.Method("talkTiny", func() {
		dsl.StreamingPayload(dsl.String)
		dsl.StreamingResult(note, tiny)
		dsl.JSONRPC(func() {})
	})
	dsl.Method("talkList", func() {
		dsl.StreamingPayload(dsl.String)
		dsl.StreamingResult(dsl.CollectionOf(note))
		dsl.JSONRPC(func() {})
	})
	dsl.Method("watch", func() {
		dsl.Payload(dsl.String)
		dsl.StreamingResult(note)
		dsl.JSONRPC(func() {})
	})
}

func viewedStreamFeed(note *expr.ResultTypeExpr) {
	dsl.JSONRPC(func() {
		dsl.POST("/feed")
	})
	dsl.Method("follow", func() {
		dsl.Payload(dsl.String)
		dsl.StreamingResult(note)
		dsl.JSONRPC(func() {
			dsl.ServerSentEvents()
		})
	})
	dsl.Method("followTiny", func() {
		dsl.Payload(dsl.String)
		dsl.StreamingResult(note, func() {
			dsl.View("tiny")
		})
		dsl.JSONRPC(func() {
			dsl.ServerSentEvents()
		})
	})
	dsl.Method("followList", func() {
		dsl.Payload(dsl.String)
		dsl.StreamingResult(dsl.CollectionOf(note))
		dsl.JSONRPC(func() {
			dsl.ServerSentEvents()
		})
	})
}
