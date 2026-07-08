package codegen

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func renderSSEClientReadEvent(implName string) string {
	var b sourceBuilder
	b.Add("// readEvent reads a single SSE event from the stream, respecting context\n")
	b.Add("// cancellation.  It first checks the internal buffer for a complete event\n")
	b.Add("// (delimited by double newlines). If no complete event is found, it reads from\n")
	b.Add("// the HTTP response body until it either finds an event boundary, reaches EOF,\n")
	b.Add("// or encounters an error. Any data after the event boundary is saved in the\n")
	b.Add("// buffer for the next call.\n")
	b.Addf("func (s *%s) readEvent(ctx context.Context) ([]byte, error) {\n", implName)
	b.Add(renderSSEClientReadEventBody())
	return b.String()
}

func renderSSEClientReadEventBody() string {
	return `	const bufSize = 4096 // 4KB buffer size

	s.readLock.Lock()
	defer s.readLock.Unlock()

	// Check for event in existing buffer
	event, ok := s.checkBuffer()
	if ok {
		return event, nil
	}

	// Initialize with any data from buffer
	eventData := event
	wasNewline := len(eventData) > 0 && eventData[len(eventData)-1] == '\n'
	buf := make([]byte, bufSize)

	// Read data in chunks until we find an event or hit EOF
	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Continue processing
		}

		// Check if stream is closed
		s.lock.Lock()
		if s.closed {
			s.lock.Unlock()
			if len(eventData) > 0 {
				return eventData, nil
			}
			return nil, io.EOF
		}
		body := s.resp.Body
		s.lock.Unlock()

		// Read next chunk
		type readResult struct {
			n   int
			err error
		}
		readc := make(chan readResult, 1)
		go func() {
			n, err := body.Read(buf)
			readc <- readResult{n: n, err: err}
		}()

		var n int
		var err error
		select {
		case result := <-readc:
			n = result.n
			err = result.err
		case <-ctx.Done():
			select {
			case result := <-readc:
				n = result.n
				err = result.err
			default:
				_ = s.Close()
				return nil, ctx.Err()
			}
		}

		// Handle read errors
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Process data if we got any
		if n > 0 {
			// Look for event boundary in this chunk
			for i := 0; i < n; i++ {
				b := buf[i]
				eventData = append(eventData, b)

				// Check for double newlines (event boundary)
				if b == '\n' && wasNewline {
					// Save any remaining data for next read
					if i+1 < n {
						s.lock.Lock()
						s.buffer = append(s.buffer[:0], buf[i+1:n]...)
						s.lock.Unlock()
					}
					return eventData, nil
				}

				// Update newline tracking
				wasNewline = (b == '\n')
			}
		}

		// Return partial data at EOF
		if errors.Is(err, io.EOF) {
			if len(eventData) > 0 {
				return eventData, nil
			}
			return nil, io.EOF
		}
	}
}

`
}

func renderSSEClientCheckBuffer(implName string) string {
	var b sourceBuilder
	b.Add("// checkBuffer examines the internal buffer for a complete SSE event (delimited\n")
	b.Add("// by double newlines).  It returns two values: the event data (or all buffer\n")
	b.Add("// contents if no complete event is found), and a boolean indicating whether a\n")
	b.Add("// complete event was found. If a complete event is found, any remaining data\n")
	b.Add("// after the event is kept in the buffer for the next call.\n")
	b.Addf("func (s *%s) checkBuffer() ([]byte, bool) {\n", implName)
	b.Add("\ts.lock.Lock()\n")
	b.Add("\tdefer s.lock.Unlock()\n\n")
	b.Add("\t// Quick return if buffer is empty\n")
	b.Add("\tif len(s.buffer) == 0 {\n")
	b.Add("\t\treturn nil, false\n")
	b.Add("\t}\n\n")
	b.Add("\t// Look for double newline in buffer\n")
	b.Add("\tfor i := 0; i < len(s.buffer)-1; i++ {\n")
	b.Add("\t\tif s.buffer[i] == '\\n' && s.buffer[i+1] == '\\n' {\n")
	b.Add("\t\t\t// Found complete event\n")
	b.Add("\t\t\teventEnd := i + 2 // Include both newlines\n")
	b.Add("\t\t\teventData := s.buffer[:eventEnd]\n\n")
	b.Add("\t\t\t// Save remaining data for next time\n")
	b.Add("\t\t\tif eventEnd < len(s.buffer) {\n")
	b.Add("\t\t\t\ts.buffer = append(s.buffer[:0], s.buffer[eventEnd:]...)\n")
	b.Add("\t\t\t} else {\n")
	b.Add("\t\t\t\ts.buffer = s.buffer[:0]\n")
	b.Add("\t\t\t}\n\n")
	b.Add("\t\t\treturn eventData, true\n")
	b.Add("\t\t}\n")
	b.Add("\t}\n\n")
	b.Add("\t// No complete event found, return buffer contents\n")
	b.Add("\teventData := s.buffer\n")
	b.Add("\ts.buffer = s.buffer[:0] // Clear buffer but keep capacity\n")
	b.Add("\treturn eventData, false\n")
	b.Add("}\n\n")
	return b.String()
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
	addSSEClientImplStruct(stmt, streamName, implName)
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
	stmt.Add(codegen.Expr(strings.TrimSpace(renderSSEClientReadEvent(implName))))
	stmt.Line()
	stmt.Add(codegen.Expr(strings.TrimSpace(renderSSEClientCheckBuffer(implName))))
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
