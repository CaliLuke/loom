package codegen

import (
	"fmt"
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
	var b strings.Builder
	fmt.Fprintf(&b, "// processEvent converts a parsed SSE event into the expected type\nfunc (s *%s) processEvent(parsed loomhttp.SSEEvent) (event %s, err error) {\n", implName, ed.SSE.EventTypeRef)
	if len(ed.SSE.Projections) > 0 {
		renderSSEProjectionClientDecode(&b, ed)
		b.WriteString("\treturn\n")
		b.WriteString("}\n")
		return b.String()
	}
	if ed.SSE.EventIsStruct {
		fmt.Fprintf(&b, "\tevent = new(%s)\n", strings.TrimPrefix(ed.SSE.EventTypeRef, "*"))
	}
	renderSSEClientEventFields(&b, ed)
	b.WriteString("\tdataContent := parsed.Data\n")
	switch {
	case ed.SSE.DataField != "":
		b.WriteString(renderSSEParseAssignment("event."+ed.SSE.DataField, ed.SSE.DataFieldTypeRef, ed.SSE.DataPointer))
	case ed.SSE.EventIsStruct:
		b.WriteString("\t// Decode JSON into the struct pointer directly\n")
		b.WriteString("\trespBody := &http.Response{\n")
		b.WriteString("\t\tStatusCode: http.StatusOK,\n")
		b.WriteString("\t\tBody:       io.NopCloser(bytes.NewReader([]byte(dataContent))),\n")
		b.WriteString("\t}\n")
		b.WriteString("\terr = s.decoder(respBody).Decode(event)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn\n")
		b.WriteString("\t}\n")
	default:
		// Whole-event primitive and collection data is JSON: strings are JSON
		// strings and bytes are base64 JSON strings.
		b.WriteString("\trespBody := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dataContent))}\n")
		b.WriteString("\tif err = s.decoder(respBody).Decode(&event); err != nil {\n")
		b.WriteString("\t\treturn\n")
		b.WriteString("\t}\n")
	}
	b.WriteString("\treturn\n")
	b.WriteString("}\n")
	return b.String()
}

func renderSSEProjectionClientDecode(b *strings.Builder, ed *EndpointData) {
	b.WriteString("\tvar view string\n")
	b.WriteString("\tswitch parsed.Type {\n")
	for _, projection := range ed.SSE.Projections {
		fmt.Fprintf(b, "\tcase %q:\n\t\tview = %q\n", projection.EventType, projection.View)
	}
	b.WriteString("\tdefault:\n\t\treturn event, fmt.Errorf(\"invalid SSE projection discriminator %q\", parsed.Type)\n\t}\n")
	fmt.Fprintf(b, "\tprojected := new(%s)\n", strings.TrimPrefix(ed.SSE.ProjectedTypeRef, "*"))
	b.WriteString("\trespBody := &http.Response{\n")
	b.WriteString("\t\tStatusCode: http.StatusOK,\n")
	b.WriteString("\t\tBody:       io.NopCloser(strings.NewReader(parsed.Data)),\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif err = s.decoder(respBody).Decode(projected); err != nil {\n\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\tvres := &%s{Projected: projected, View: view}\n", strings.TrimPrefix(ed.SSE.ViewedResultRef, "*"))
	fmt.Fprintf(b, "\tif err = %s(vres); err != nil {\n\t\treturn\n\t}\n", ed.SSE.ViewedValidateRef)
	fmt.Fprintf(b, "\tevent, err = %s(vres)\n", ed.SSE.ResultInitRef)
	b.WriteString("\tif err != nil {\n\t\treturn\n\t}\n")
	renderSSEClientEventFields(b, ed)
}

// renderSSEClientEventFields assigns the parsed SSE id and event type to the
// mapped event fields. The ID is the library's persistent last-event-ID buffer.
func renderSSEClientEventFields(b *strings.Builder, ed *EndpointData) {
	if ed.SSE.IDField != "" {
		renderSSEClientStringField(b, "event."+ed.SSE.IDField, "parsed.ID", "id", ed.SSE.IDPointer, ed.SSE.IDDefault)
	}
	if ed.SSE.EventField != "" {
		renderSSEClientStringField(b, "event."+ed.SSE.EventField, "parsed.Type", "eventType", ed.SSE.EventPointer, ed.SSE.EventDefault)
	}
}

// renderSSEClientStringField assigns the parsed SSE string field source to
// target. When source is empty, a pointer target stays nil and a target with
// a default value receives the default literal def instead.
func renderSSEClientStringField(b *strings.Builder, target, source, local string, pointer bool, def string) {
	switch {
	case pointer:
		fmt.Fprintf(b, "\tif %s := %s; %s != \"\" {\n\t\t%s = &%s\n\t}\n", local, source, local, target, local)
	case def != "":
		fmt.Fprintf(b, "\t%s = %s\n\tif %s == \"\" {\n\t\t%s = %s\n\t}\n", target, source, target, target, def)
	default:
		fmt.Fprintf(b, "\t%s = %s\n", target, source)
	}
}

// renderSSEParseAssignment returns the statements that decode the event data
// into the target field of type typeRef. A pointer target stays nil when the
// data of an optional String field is empty; other pointer targets decode JSON
// literals and stay nil for null.
func renderSSEParseAssignment(target, typeRef string, pointer bool) string {
	var b strings.Builder
	switch {
	case pointer && typeRef == "string":
		fmt.Fprintf(&b, "\tif dataContent != \"\" {\n\t\t%s = &dataContent\n\t}\n", target)
	case pointer:
		renderSSEDecodeAssignment(&b, target)
	case typeRef == "string":
		fmt.Fprintf(&b, "\t%s = dataContent\n", target)
	case typeRef == "[]byte":
		fmt.Fprintf(&b, "\t%s = []byte(dataContent)\n", target)
	case typeRef == "int":
		fmt.Fprintf(&b, "\tv, parseErr := strconv.Atoi(dataContent)\n")
		b.WriteString("\tif parseErr != nil {\n")
		b.WriteString("\t\terr = parseErr\n")
		b.WriteString("\t\treturn\n")
		b.WriteString("\t}\n")
		fmt.Fprintf(&b, "\t%s = v\n", target)
	default:
		renderSSEDecodeAssignment(&b, target)
	}
	return b.String()
}

// renderSSEDecodeAssignment writes the statements that decode the JSON event
// data into target with the decoder. A null decodes to a nil pointer target.
func renderSSEDecodeAssignment(b *strings.Builder, target string) {
	b.WriteString("\trespBody := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dataContent))}\n")
	fmt.Fprintf(b, "\tif err = s.decoder(respBody).Decode(&%s); err != nil {\n", target)
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
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
