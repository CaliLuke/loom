package codegen

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func sseClientNeedsDecoder(ed *EndpointData) bool {
	if ed.SSE.DataField != "" {
		return sseParseAssignmentNeedsDecoder(ed.SSE.DataFieldTypeRef)
	}
	return ed.SSE.EventIsStruct || sseParseAssignmentNeedsDecoder(ed.SSE.EventTypeRef)
}

func sseParseAssignmentNeedsDecoder(typeRef string) bool {
	switch typeRef {
	case "string", "[]byte", "int":
		return false
	default:
		return true
	}
}

func renderSSEClientProcessEvent(implName string, ed *EndpointData) string {
	var b sourceBuilder
	b.Addf("// processEvent processes a raw SSE event into the expected type\nfunc (s *%s) processEvent(eventData []byte) (event %s, err error) {\n", implName, ed.SSE.EventTypeRef)
	b.Add("\tparsed, err := loomhttp.ParseSSEEvent(eventData)\n")
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn event, err\n")
	b.Add("\t}\n")
	if ed.SSE.EventIsStruct {
		b.Addf("\tevent = new(%s)\n", strings.TrimPrefix(ed.SSE.EventTypeRef, "*"))
	}
	if ed.SSE.IDField != "" {
		b.Addf("\tevent.%s = parsed.ID\n", ed.SSE.IDField)
	}
	if ed.SSE.EventField != "" {
		b.Addf("\tevent.%s = parsed.Type\n", ed.SSE.EventField)
	}
	b.Add("\tdataContent := parsed.Data\n")
	switch {
	case ed.SSE.DataField != "":
		b.Add(renderSSEParseAssignment("event."+ed.SSE.DataField, ed.SSE.DataFieldTypeRef))
	case ed.SSE.EventIsStruct:
		b.Add("\t// Decode JSON into the struct pointer directly\n")
		b.Add("\trespBody := &http.Response{\n")
		b.Add("\t\tStatusCode: http.StatusOK,\n")
		b.Add("\t\tBody:       io.NopCloser(bytes.NewReader([]byte(dataContent))),\n")
		b.Add("\t}\n")
		b.Add("\terr = s.decoder(respBody).Decode(event)\n")
		b.Add("\tif err != nil {\n")
		b.Add("\t\treturn\n")
		b.Add("\t}\n")
	default:
		b.Add(renderSSEParseAssignment("event", ed.SSE.EventTypeRef))
	}
	b.Add("\treturn\n")
	b.Add("}\n")
	return b.String()
}

func renderSSEParseAssignment(target, typeRef string) string {
	var b sourceBuilder
	switch typeRef {
	case "string":
		b.Addf("\t%s = dataContent\n", target)
	case "[]byte":
		b.Addf("\t%s = []byte(dataContent)\n", target)
	case "int":
		b.Addf("\tv, parseErr := strconv.Atoi(dataContent)\n")
		b.Add("\tif parseErr != nil {\n")
		b.Add("\t\terr = parseErr\n")
		b.Add("\t\treturn\n")
		b.Add("\t}\n")
		b.Addf("\t%s = v\n", target)
	default:
		b.Addf("\trespBody := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dataContent))}\n")
		b.Addf("\tif err = s.decoder(respBody).Decode(&%s); err != nil {\n", target)
		b.Add("\t\treturn\n")
		b.Add("\t}\n")
	}
	return b.String()
}

func addSSEClientSection(stmt *jen.Statement, ed *EndpointData) {
	streamName := ed.Method.VarName + "ClientStream"
	implName := ed.Method.VarName + "StreamImpl"
	addSSEClientInterface(stmt, ed, streamName)
	addSSEClientImplStruct(stmt, ed, streamName, implName)
	addSSEClientConstructor(stmt, ed, streamName, implName)
	stmt.Line()
	codegen.Doc(stmt, "Recv reads and returns the next event from the SSE stream, respecting context cancellation.")
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(implName)).
		Id("Recv").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(jen.Id("event").Add(codegen.TypeRef(ed.SSE.EventTypeRef)), jen.Id("err").Error()).
		BlockFunc(func(group *jen.Group) {
			addRawWebSocketGroup(group, renderSSEClientRecvBody())
		})
	stmt.Line()
	codegen.Doc(stmt, "Close closes the SSE stream and releases any associated resources.")
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(implName)).
		Id("Close").
		Params().
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawWebSocketGroup(group, renderSSEClientCloseBody())
		})
	stmt.Line()
	stmt.Add(codegen.Expr(strings.TrimSpace(renderSSEClientProcessEvent(implName, ed))))
	stmt.Line()
}
