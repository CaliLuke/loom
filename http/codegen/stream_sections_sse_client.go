package codegen

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func sseClientNeedsDecoder(ed *EndpointData) bool {
	if ed.SSE.DataField != "" {
		return sseParseAssignmentNeedsDecoder(ed.SSE.DataFieldTypeRef, ed.SSE.DataPointer)
	}
	// Whole-event data is always JSON and decodes through the decoder.
	return true
}

// sseParseAssignmentNeedsDecoder reports whether the client decodes the data
// of a field of type typeRef with the decoder. Optional primitives other than
// String, pointer data fields, encode as JSON literals or null.
func sseParseAssignmentNeedsDecoder(typeRef string, pointer bool) bool {
	if pointer && typeRef != "string" {
		return true
	}
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
	if len(ed.SSE.Projections) > 0 {
		renderSSEProjectionClientDecode(&b, ed)
		b.Add("\treturn\n")
		b.Add("}\n")
		return b.String()
	}
	if ed.SSE.EventIsStruct {
		b.Addf("\tevent = new(%s)\n", strings.TrimPrefix(ed.SSE.EventTypeRef, "*"))
	}
	renderSSEClientEventFields(&b, ed)
	b.Add("\tdataContent := parsed.Data\n")
	switch {
	case ed.SSE.DataField != "":
		b.Add(renderSSEParseAssignment("event."+ed.SSE.DataField, ed.SSE.DataFieldTypeRef, ed.SSE.DataPointer))
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
		// Whole-event primitive and collection data is JSON: strings are JSON
		// strings and bytes are base64 JSON strings.
		b.Add("\trespBody := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dataContent))}\n")
		b.Add("\tif err = s.decoder(respBody).Decode(&event); err != nil {\n")
		b.Add("\t\treturn\n")
		b.Add("\t}\n")
	}
	b.Add("\treturn\n")
	b.Add("}\n")
	return b.String()
}

func renderSSEProjectionClientDecode(b *sourceBuilder, ed *EndpointData) {
	b.Add("\tvar view string\n")
	b.Add("\tswitch parsed.Type {\n")
	for _, projection := range ed.SSE.Projections {
		b.Addf("\tcase %q:\n\t\tview = %q\n", projection.EventType, projection.View)
	}
	b.Add("\tdefault:\n\t\treturn event, fmt.Errorf(\"invalid SSE projection discriminator %q\", parsed.Type)\n\t}\n")
	b.Addf("\tprojected := new(%s)\n", strings.TrimPrefix(ed.SSE.ProjectedTypeRef, "*"))
	b.Add("\trespBody := &http.Response{\n")
	b.Add("\t\tStatusCode: http.StatusOK,\n")
	b.Add("\t\tBody:       io.NopCloser(strings.NewReader(parsed.Data)),\n")
	b.Add("\t}\n")
	b.Add("\tif err = s.decoder(respBody).Decode(projected); err != nil {\n\t\treturn\n\t}\n")
	b.Addf("\tvres := &%s{Projected: projected, View: view}\n", strings.TrimPrefix(ed.SSE.ViewedResultRef, "*"))
	b.Addf("\tif err = %s(vres); err != nil {\n\t\treturn\n\t}\n", ed.SSE.ViewedValidateRef)
	b.Addf("\tevent, err = %s(vres)\n", ed.SSE.ResultInitRef)
	b.Add("\tif err != nil {\n\t\treturn\n\t}\n")
	renderSSEClientEventFields(b, ed)
}

// renderSSEClientEventFields assigns the parsed SSE id and event type to the
// mapped event fields.
func renderSSEClientEventFields(b *sourceBuilder, ed *EndpointData) {
	if ed.SSE.IDField != "" {
		renderSSEClientStringField(b, "event."+ed.SSE.IDField, "parsed.ID", "id", ed.SSE.IDPointer, ed.SSE.IDDefault)
	}
	if ed.SSE.EventField != "" {
		renderSSEClientStringField(b, "event."+ed.SSE.EventField, "parsed.Type", "eventType", ed.SSE.EventPointer, ed.SSE.EventDefault)
	}
}

// renderSSEClientStringField assigns the parsed SSE string field source to
// target. A pointer target stays nil when the event omits the field. A target
// with a default value receives the default literal def instead.
func renderSSEClientStringField(b *sourceBuilder, target, source, local string, pointer bool, def string) {
	switch {
	case pointer:
		b.Addf("\tif %s := %s; %s != \"\" {\n\t\t%s = &%s\n\t}\n", local, source, local, target, local)
	case def != "":
		b.Addf("\t%s = %s\n\tif %s == \"\" {\n\t\t%s = %s\n\t}\n", target, source, target, target, def)
	default:
		b.Addf("\t%s = %s\n", target, source)
	}
}

// renderSSEParseAssignment returns the statements that decode the event data
// into the target field of type typeRef. A pointer target stays nil when the
// data of an optional String field is empty; other pointer targets decode JSON
// literals and stay nil for null.
func renderSSEParseAssignment(target, typeRef string, pointer bool) string {
	var b sourceBuilder
	switch {
	case pointer && typeRef == "string":
		b.Addf("\tif dataContent != \"\" {\n\t\t%s = &dataContent\n\t}\n", target)
	case pointer:
		renderSSEDecodeAssignment(&b, target)
	case typeRef == "string":
		b.Addf("\t%s = dataContent\n", target)
	case typeRef == "[]byte":
		b.Addf("\t%s = []byte(dataContent)\n", target)
	case typeRef == "int":
		b.Addf("\tv, parseErr := strconv.Atoi(dataContent)\n")
		b.Add("\tif parseErr != nil {\n")
		b.Add("\t\terr = parseErr\n")
		b.Add("\t\treturn\n")
		b.Add("\t}\n")
		b.Addf("\t%s = v\n", target)
	default:
		renderSSEDecodeAssignment(&b, target)
	}
	return b.String()
}

// renderSSEDecodeAssignment writes the statements that decode the JSON event
// data into target with the decoder. A null decodes to a nil pointer target.
func renderSSEDecodeAssignment(b *sourceBuilder, target string) {
	b.Add("\trespBody := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dataContent))}\n")
	b.Addf("\tif err = s.decoder(respBody).Decode(&%s); err != nil {\n", target)
	b.Add("\t\treturn\n")
	b.Add("\t}\n")
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
