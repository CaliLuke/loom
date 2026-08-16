package framework

import (
	"fmt"
	"strings"
)

func renderDesignSource(design *DesignData) string {
	var b strings.Builder
	b.WriteString("package design\n\n")
	b.WriteString("import . \"github.com/CaliLuke/loom/dsl\"\n\n")
	fmt.Fprintf(&b, "var _ = API(%q, func() {\n", design.APIName)
	fmt.Fprintf(&b, "\tTitle(%q)\n", design.APITitle)
	fmt.Fprintf(&b, "\tDescription(%q)\n", design.APIDescription)
	b.WriteString("})\n")
	for _, svc := range design.Services {
		b.WriteString("\n")
		fmt.Fprintf(&b, "var _ = Service(%q, func() {\n", svc.Name)
		fmt.Fprintf(&b, "\tDescription(%q)\n\n", svc.Description)
		b.WriteString("\t// Enable JSON-RPC\n")
		b.WriteString("\tJSONRPC(func() {\n")
		fmt.Fprintf(&b, "\t\t%s(%q)\n", svc.JSONRPCMethod, svc.JSONRPCPath)
		b.WriteString("\t})\n")
		for _, m := range svc.Methods {
			if m.IsNotification && !m.IsStreaming {
				continue
			}
			b.WriteString("\n")
			b.WriteString(indentBlock(renderMethodDSL(m), 1))
			b.WriteString("\n")
		}
		b.WriteString("})\n")
	}
	return b.String()
}

func renderMethodDSL(m *MethodData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Method(%q, func() {\n", m.Name)
	fmt.Fprintf(&b, "\tDescription(%q)\n", m.Description)
	if m.Payload != nil {
		fmt.Fprintf(&b, "\tPayload(%s)\n", renderInlineType(m.Payload, true))
	}
	if m.StreamingPayload != nil {
		fmt.Fprintf(&b, "\tStreamingPayload(%s)\n", renderInlineType(m.StreamingPayload, true))
	}
	if m.Result != nil {
		if m.IsStreaming && (m.StreamKind == "result" || m.StreamKind == "bidirectional") {
			fmt.Fprintf(&b, "\tStreamingResult(%s)\n", renderInlineType(m.Result, true))
		} else if !m.IsNotification {
			fmt.Fprintf(&b, "\tResult(%s)\n", renderInlineType(m.Result, true))
		}
	}
	if m.ReturnsError {
		b.WriteString("\t// Methods with error modifier return ServiceError\n")
	}
	b.WriteString("\tJSONRPC(func() {\n")
	if m.IsSSE() {
		fmt.Fprintf(&b, "\t\tServerSentEvents(func() { SSENotificationMethod(%q) })\n", m.Name)
	}
	b.WriteString("\t})\n")
	if shouldGenerateGRPC(m) {
		b.WriteString("\tGRPC(func() {})\n")
	}
	b.WriteString("})")
	return b.String()
}

func renderInlineType(spec *TypeSpec, methodContext bool) string {
	if spec == nil {
		return "Any"
	}
	switch spec.Kind {
	case "primitive":
		return spec.Primitive
	case "array":
		if spec.ArrayElem != nil && spec.ArrayElem.Kind == "primitive" {
			return fmt.Sprintf("ArrayOf(%s)", spec.ArrayElem.Primitive)
		}
		return "func() {\n\tField(1, \"items\", ArrayOf(" + renderInlineType(spec.ArrayElem, methodContext) + "))\n\tRequired(\"items\")\n}"
	case "object":
		var b strings.Builder
		b.WriteString("func() {\n")
		for _, f := range spec.Fields {
			if f.IsID {
				fmt.Fprintf(&b, "\tID(%q, %s", f.Name, renderInlineType(f.Type, methodContext))
				if f.Description != "" {
					fmt.Fprintf(&b, ", %q", f.Description)
				}
				b.WriteString(")\n")
				continue
			}
			fmt.Fprintf(&b, "\tField(%d, %q, %s", f.Position, f.Name, renderInlineType(f.Type, methodContext))
			if f.Description != "" {
				fmt.Fprintf(&b, ", %q", f.Description)
			}
			if validation := renderFieldValidations(f); validation != "" {
				fmt.Fprintf(&b, ", func() {\n%s\t}", indentBlock(validation, 2))
			}
			b.WriteString(")\n")
		}
		req := collectRequired(spec.Fields)
		if len(req) > 0 {
			b.WriteString("\tRequired(")
			for i, f := range req {
				if i > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "%q", f)
			}
			b.WriteString(")\n")
		}
		if spec.NeedsID {
			if methodContext {
				b.WriteString("\t// Accept JSON-RPC ID in payload for WS, optional; transport-level ID is handled separately\n")
				b.WriteString("\tField(99, \"id\", String)\n")
			} else {
				b.WriteString("\tID(\"id\")\n")
			}
		}
		b.WriteString("}")
		return b.String()
	case "map":
		return "func() {\n\tField(1, \"data\", MapOf(" + renderInlineType(spec.MapKey, methodContext) + ", " + renderInlineType(spec.MapValue, methodContext) + "))\n\tRequired(\"data\")\n}"
	default:
		return "Any"
	}
}

func renderFieldValidations(field FieldSpec) string {
	var b strings.Builder
	if field.MinLength > 0 {
		fmt.Fprintf(&b, "MinLength(%d)\n", field.MinLength)
	}
	if field.Minimum != nil {
		fmt.Fprintf(&b, "Minimum(%d)\n", *field.Minimum)
	}
	return b.String()
}

func renderImplementationSource(service *ServiceImplData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// %ssrvc implements the %s service.\n", service.ServicePackage, service.ServicePackage)
	fmt.Fprintf(&b, "type %ssrvc struct {\n", service.ServicePackage)
	b.WriteString("\tlogger *log.Logger\n")
	for _, m := range service.Methods {
		if m.Info.Action == ActionCollect && m.Info.Type == TypeArray && m.Transport == "ws" {
			fmt.Fprintf(&b, "\t// State for accumulating items in %s\n", m.Name)
			b.WriteString("\tcollectedItems []string\n")
		}
	}
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "// New%s returns the %s service implementation.\n", service.Title, service.ServicePackage)
	fmt.Fprintf(&b, "func New%s() %s.Service {\n", service.Title, service.ServicePackage)
	fmt.Fprintf(&b, "\treturn &%ssrvc{}\n", service.ServicePackage)
	b.WriteString("}\n")
	if service.Name == "testws" {
		b.WriteString(`

// HandleStream handles the JSON-RPC WebSocket streaming connection
func (s *` + service.ServicePackage + `srvc) HandleStream(ctx context.Context, stream ` + service.ServicePackage + `.Stream) error {
	// For testing purposes, we only send broadcasts when explicitly called through the broadcast method
	// In a real application, you might send broadcasts based on external events or timers

	// Ensure the stream is closed on exit
	defer func() {
		if err := stream.Close(); err != nil {
			log.Printf("HandleStream Close error: %v", err)
		}
	}()

	// Loop to handle incoming requests
	for {
		// Recv reads and dispatches the next request
		if err := stream.Recv(ctx); err != nil {
			// Log the error type and value for diagnostics
			log.Printf("HandleStream Recv error: %T %v", err, err)
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
`)
	}
	for _, m := range service.Methods {
		if m.IsNotification && !m.IsStreaming {
			continue
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "// %s implements %s.\n", m.GoName, m.Name)
		b.WriteString(renderMethodSignature(m))
		b.WriteString(" {\n")
		fmt.Fprintf(&b, "\tlog.Printf(%q)\n", m.GoName+" called")
		body := renderMethodImplementation(service, m)
		b.WriteString(indentBlock(body, 1))
		if !strings.HasSuffix(body, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("}\n")
	}
	return b.String()
}

func renderMethodSignature(m *MethodImplData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "func (s *%ssrvc) %s(ctx context.Context", m.ServicePackage, m.GoName)
	if m.HasPayload {
		fmt.Fprintf(&b, ", p %s", m.PayloadRef)
	}
	if m.IsStreaming {
		fmt.Fprintf(&b, ", stream %s.%s", m.ServicePackage, m.StreamInterface)
	}
	b.WriteString(") ")
	if m.IsStreaming {
		b.WriteString("error")
	} else if m.HasResult {
		fmt.Fprintf(&b, "(%s, error)", m.ResultRef)
	} else {
		b.WriteString("error")
	}
	return b.String()
}

func renderMethodImplementation(service *ServiceImplData, m *MethodImplData) string {
	if m.IsStreaming {
		if m.IsSSE() {
			return renderSSEImplementation(m)
		}
		if m.IsWebSocket() {
			return renderWebSocketImplementation(service, m)
		}
	}
	if m.ReturnsError {
		if m.HasResult {
			return "\treturn nil, &loom.ServiceError{Name: \"test_error\", Message: \"Invalid params\"}"
		}
		return "\treturn &loom.ServiceError{Name: \"test_error\", Message: \"Invalid params\"}"
	}
	switch m.Info.Action {
	case ActionEcho:
		return renderNonStreamingEcho(m)
	case ActionTransform:
		return renderNonStreamingTransform(m)
	case ActionGenerate:
		return renderNonStreamingGenerate(m)
	default:
		if m.HasResult {
			return "\treturn nil, fmt.Errorf(\"not implemented\")"
		}
		return "\treturn fmt.Errorf(\"not implemented\")"
	}
}

func renderNonStreamingEcho(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString, TypeInt, TypeBool:
		if m.Info.Modifier == ModifierValidate || m.Info.Modifier == ModifierIDMap {
			return "\treturn p.Value, nil"
		}
		return "\treturn p, nil"
	case TypeArray:
		return fmt.Sprintf("\treturn &%s.%sResult{\n\t\tItems: p.Items,\n\t}, nil", m.ServicePackage, m.GoName)
	case TypeObject:
		return fmt.Sprintf("\treturn &%s.%sResult{\n\t\tField1: p.Field1,\n\t\tField2: p.Field2,\n\t\tField3: p.Field3,\n\t}, nil", m.ServicePackage, m.GoName)
	case TypeMap:
		return fmt.Sprintf("\treturn &%s.%sResult{\n\t\tData: p.Data,\n\t}, nil", m.ServicePackage, m.GoName)
	default:
		if m.Info.Modifier == ModifierIDMap {
			return "\treturn p.Value, nil"
		}
		return "\treturn p, nil"
	}
}

func renderNonStreamingTransform(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return "\t// Transform to uppercase\n\treturn strings.ToUpper(p), nil"
	case TypeArray:
		return fmt.Sprintf("\t// Reverse the array\n\treversed := make([]string, len(p.Items))\n\tfor i, item := range p.Items {\n\t\treversed[len(p.Items)-1-i] = item\n\t}\n\treturn &%s.%sResult{\n\t\tItems: reversed,\n\t}, nil", m.ServicePackage, m.GoName)
	case TypeObject:
		return fmt.Sprintf("\t// Transform: uppercase field1, double field2, negate field3\n\treturn &%s.%sResult{\n\t\tField1: strings.ToUpper(p.Field1),\n\t\tField2: p.Field2 * 2,\n\t\tField3: !p.Field3,\n\t}, nil", m.ServicePackage, m.GoName)
	case TypeMap:
		return fmt.Sprintf("\t// Transform: prefix all keys with \"transformed_\"\n\tresult := make(map[string]any)\n\tfor k, v := range p.Data {\n\t\tresult[\"transformed_\"+k] = v\n\t}\n\treturn &%s.%sResult{\n\t\tData: result,\n\t}, nil", m.ServicePackage, m.GoName)
	default:
		return "\t// Default transform: return as-is\n\treturn p, nil"
	}
}

func renderNonStreamingGenerate(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return "\treturn \"generated-string\", nil"
	case TypeInt:
		return "\treturn 42, nil"
	case TypeBool:
		return "\treturn true, nil"
	case TypeArray:
		return fmt.Sprintf("\treturn &%s.%sResult{\n\t\tItems: []string{\"item1\", \"item2\", \"item3\"},\n\t}, nil", m.ServicePackage, m.GoName)
	case TypeObject:
		return fmt.Sprintf("\treturn &%s.%sResult{\n\t\tField1: \"generated-value1\",\n\t\tField2: 42,\n\t\tField3: true,\n\t}, nil", m.ServicePackage, m.GoName)
	case TypeMap:
		return fmt.Sprintf("\treturn &%s.%sResult{\n\t\tData: map[string]any{\n\t\t\t\"generated\": true,\n\t\t\t\"count\": 3,\n\t\t\t\"status\": \"ok\",\n\t\t},\n\t}, nil", m.ServicePackage, m.GoName)
	default:
		return "\treturn nil, nil"
	}
}

func renderSSEImplementation(m *MethodImplData) string {
	switch m.Info.Action {
	case ActionEcho:
		return renderSSEEcho(m)
	case ActionTransform:
		return renderSSETransform(m)
	case ActionGenerate:
		return renderSSEGenerate(m)
	case ActionStream:
		return renderSSEStream(m)
	default:
		return renderSSEDefault(m)
	}
}

func renderSSEFinalize(m *MethodImplData) string {
	switch m.Info.Modifier {
	case ModifierFinal:
		switch m.Info.Type {
		case TypeString:
			return fmt.Sprintf("\tfinalResult := &%s.%sResult{\n\t\tValue: \"Final response\",\n\t}\n\treturn stream.SendAndClose(ctx, finalResult)", m.ServicePackage, m.GoName)
		case TypeArray:
			return fmt.Sprintf("\tfinalResult := &%s.%sResult{\n\t\tItems: []string{\"completed\"},\n\t}\n\treturn stream.SendAndClose(ctx, finalResult)", m.ServicePackage, m.GoName)
		case TypeObject:
			return fmt.Sprintf("\tfinalResult := &%s.%sResult{\n\t\tField1: \"completed\",\n\t\tField2: 100,\n\t\tField3: true,\n\t}\n\treturn stream.SendAndClose(ctx, finalResult)", m.ServicePackage, m.GoName)
		case TypeMap:
			return fmt.Sprintf("\tfinalResult := &%s.%sResult{\n\t\tData: map[string]any{\"status\": \"completed\", \"final\": true},\n\t}\n\treturn stream.SendAndClose(ctx, finalResult)", m.ServicePackage, m.GoName)
		}
	case ModifierError:
		return "\treturn &loom.ServiceError{Message: \"Streaming error occurred\"}"
	}
	return "\treturn nil"
}

func renderSSEEcho(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tresult := &%s.%sResult{\n\t\tValue: p,\n\t}\n\tif err := stream.Send(ctx, result); err != nil {\n\t\treturn err\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeArray:
		return fmt.Sprintf("\tfor _, item := range p.Items {\n\t\tresult := &%s.%sResult{\n\t\t\tItems: []string{item},\n\t\t}\n\t\tif err := stream.Send(ctx, result); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeObject:
		return fmt.Sprintf("\tresult := &%s.%sResult{\n\t\tField1: p.Field1,\n\t\tField2: p.Field2,\n\t\tField3: p.Field3,\n\t}\n\tif err := stream.Send(ctx, result); err != nil {\n\t\treturn err\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeMap:
		return fmt.Sprintf("\tresult := &%s.%sResult{\n\t\tData: p.Data,\n\t}\n\tif err := stream.Send(ctx, result); err != nil {\n\t\treturn err\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	default:
		return renderSSEFinalize(m)
	}
}

func renderSSETransform(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tresult := &%s.%sResult{\n\t\tValue: strings.ToUpper(p),\n\t}\n\tif err := stream.Send(ctx, result); err != nil {\n\t\treturn err\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeArray:
		return fmt.Sprintf("\treversed := make([]string, len(p.Items))\n\tfor i, item := range p.Items {\n\t\treversed[len(p.Items)-1-i] = item\n\t}\n\tresult := &%s.%sResult{\n\t\tItems: reversed,\n\t}\n\tif err := stream.Send(ctx, result); err != nil {\n\t\treturn err\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeObject:
		return fmt.Sprintf("\tresult := &%s.%sResult{\n\t\tField1: strings.ToUpper(p.Field1),\n\t\tField2: p.Field2 * 2,\n\t\tField3: !p.Field3,\n\t}\n\tif err := stream.Send(ctx, result); err != nil {\n\t\treturn err\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeMap:
		return fmt.Sprintf("\ttransformed := make(map[string]any)\n\tfor k, v := range p.Data {\n\t\ttransformed[\"transformed_\"+k] = v\n\t}\n\tresult := &%s.%sResult{\n\t\tData: transformed,\n\t}\n\tif err := stream.Send(ctx, result); err != nil {\n\t\treturn err\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	default:
		return renderSSEFinalize(m)
	}
}

func renderSSEGenerate(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tfor i := 1; i <= 3; i++ {\n\t\tresult := &%s.%sResult{\n\t\t\tValue: fmt.Sprintf(\"generated-%%d\", i),\n\t\t}\n\t\tif err := stream.Send(ctx, result); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeArray:
		return fmt.Sprintf("\tfor i := 1; i <= 3; i++ {\n\t\tresult := &%s.%sResult{\n\t\t\tItems: []string{fmt.Sprintf(\"item-%%d\", i)},\n\t\t}\n\t\tif err := stream.Send(ctx, result); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeObject:
		return fmt.Sprintf("\tfor i := 1; i <= 3; i++ {\n\t\tresult := &%s.%sResult{\n\t\t\tField1: fmt.Sprintf(\"generated-%%d\", i),\n\t\t\tField2: i * 10,\n\t\t\tField3: i%%2 == 0,\n\t\t}\n\t\tif err := stream.Send(ctx, result); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeMap:
		return fmt.Sprintf("\tfor i := 1; i <= 3; i++ {\n\t\tresult := &%s.%sResult{\n\t\t\tData: map[string]any{\n\t\t\t\t\"iteration\": i,\n\t\t\t\t\"status\": fmt.Sprintf(\"step-%%d\", i),\n\t\t\t},\n\t\t}\n\t\tif err := stream.Send(ctx, result); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	default:
		return renderSSEFinalize(m)
	}
}

func renderSSEStream(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tcount := 3\n\tif p != \"\" {\n\t\tcount = len(p)\n\t\tif count > 10 {\n\t\t\tcount = 10\n\t\t}\n\t}\n\tfor i := 1; i <= count; i++ {\n\t\tresult := &%s.%sResult{\n\t\t\tValue: fmt.Sprintf(\"Stream %%d of %%d\", i, count),\n\t\t}\n\t\tif err := stream.Send(ctx, result); err != nil {\n\t\t\treturn err\n\t\t}\n\t\ttime.Sleep(10 * time.Millisecond)\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeArray:
		return fmt.Sprintf("\tif len(p.Items) == 0 {\n\t\tresult := &%s.%sResult{Items: []string{\"empty\"}}\n\t\tif err := stream.Send(ctx, result); err != nil {\n\t\t\treturn err\n\t\t}\n\t} else {\n\t\tfor i, item := range p.Items {\n\t\t\tresult := &%s.%sResult{Items: []string{fmt.Sprintf(\"Processing: %%s\", item)}}\n\t\t\tif err := stream.Send(ctx, result); err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tif i < len(p.Items)-1 { time.Sleep(10 * time.Millisecond) }\n\t\t}\n\t}\n%s", m.ServicePackage, m.GoName, m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeObject:
		return fmt.Sprintf("\tcount := p.Field2\n\tif count <= 0 { count = 3 }\n\tif count > 10 { count = 10 }\n\tfor i := 1; i <= count; i++ {\n\t\tresult := &%s.%sResult{\n\t\t\tField1: fmt.Sprintf(\"%%s-%%d\", p.Field1, i),\n\t\t\tField2: i,\n\t\t\tField3: i == count,\n\t\t}\n\t\tif err := stream.Send(ctx, result); err != nil {\n\t\t\treturn err\n\t\t}\n\t\ttime.Sleep(10 * time.Millisecond)\n\t}\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeMap:
		return fmt.Sprintf("\tif len(p.Data) == 0 {\n\t\tresult := &%s.%sResult{Data: map[string]any{\"status\": \"empty\"}}\n\t\tif err := stream.Send(ctx, result); err != nil { return err }\n\t} else {\n\t\tkeys := make([]string, 0, len(p.Data))\n\t\tfor k := range p.Data { keys = append(keys, k) }\n\t\tsort.Strings(keys)\n\t\tfor _, k := range keys {\n\t\t\tv := p.Data[k]\n\t\t\tresult := &%s.%sResult{Data: map[string]any{\"key\": k, \"value\": v}}\n\t\t\tif err := stream.Send(ctx, result); err != nil { return err }\n\t\t\ttime.Sleep(10 * time.Millisecond)\n\t\t}\n\t}\n%s", m.ServicePackage, m.GoName, m.ServicePackage, m.GoName, renderSSEFinalize(m))
	default:
		return renderSSEFinalize(m)
	}
}

func renderSSEDefault(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tresult := &%s.%sResult{Value: p}\n\tif err := stream.Send(ctx, result); err != nil { return err }\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeArray:
		return fmt.Sprintf("\tresult := &%s.%sResult{Items: p.Items}\n\tif err := stream.Send(ctx, result); err != nil { return err }\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeObject:
		return fmt.Sprintf("\tresult := &%s.%sResult{Field1: p.Field1, Field2: p.Field2, Field3: p.Field3}\n\tif err := stream.Send(ctx, result); err != nil { return err }\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	case TypeMap:
		return fmt.Sprintf("\tresult := &%s.%sResult{Data: p.Data}\n\tif err := stream.Send(ctx, result); err != nil { return err }\n%s", m.ServicePackage, m.GoName, renderSSEFinalize(m))
	default:
		return renderSSEFinalize(m)
	}
}

func renderWebSocketImplementation(service *ServiceImplData, m *MethodImplData) string {
	// Keep the current scenario semantics but in direct Go rendering.
	// This is intentionally straightforward rather than abstract.
	switch m.Info.Action {
	case ActionEcho:
		if m.Info.Modifier == ModifierError {
			return "\ttestErr := &loom.ServiceError{Name: \"test_error\", Message: \"Invalid params\"}\n\tif err := stream.SendError(ctx, testErr); err != nil {\n\t\treturn err\n\t}\n\treturn nil"
		}
		return renderWebSocketEcho(m)
	case ActionTransform:
		return renderWebSocketTransform(m)
	case ActionGenerate:
		return renderWebSocketGenerate(m)
	case ActionStream:
		return renderWebSocketStream(m)
	case ActionCollect:
		return fmt.Sprintf("\tif p != nil && p.Items != nil {\n\t\ts.collectedItems = append(s.collectedItems, p.Items...)\n\t}\n\tresult := &%s.%sResult{Items: s.collectedItems}\n\tif err := stream.SendResponse(ctx, result); err != nil {\n\t\treturn err\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case ActionBroadcast:
		return renderWebSocketBroadcast(m)
	default:
		return "\treturn nil"
	}
}

func renderWebSocketEcho(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{Value: p.Value}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeArray:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{Items: p.Items}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeObject:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{Field1: p.Field1, Field2: p.Field2, Field3: p.Field3}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeMap:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{Data: p.Data}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	default:
		return "\treturn nil"
	}
}

func renderWebSocketTransform(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{Value: strings.ToUpper(p.Value)}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeArray:
		return fmt.Sprintf("\tif p != nil {\n\t\treversed := make([]string, len(p.Items))\n\t\tfor i, item := range p.Items { reversed[len(p.Items)-1-i] = item }\n\t\tresult := &%s.%sResult{Items: reversed}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeObject:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{Field1: strings.ToUpper(p.Field1), Field2: p.Field2 * 2, Field3: !p.Field3}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeMap:
		return fmt.Sprintf("\tif p != nil {\n\t\traw, present := p.Data.Value()\n\t\tif !present { return nil }\n\t\tdata, ok := raw.(map[string]any)\n\t\tif !ok { return nil }\n\t\ttransformed := make(map[string]any)\n\t\tfor k, v := range data { transformed[\"transformed_\"+k] = v }\n\t\tresult := &%s.%sResult{Data: loom.NullableValue[any](transformed)}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	default:
		return "\treturn nil"
	}
}

func renderWebSocketGenerate(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{Value: \"generated-string\"}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeArray:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{Items: []string{\"item1\", \"item2\", \"item3\"}}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeObject:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{Field1: \"generated-value1\", Field2: 42, Field3: true}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeMap:
		return fmt.Sprintf("\tif p != nil {\n\t\tresult := &%s.%sResult{ID: p.ID, Data: loom.NullableValue[any](map[string]any{\"generated\": true, \"count\": 3, \"status\": \"ok\"})}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	default:
		return fmt.Sprintf("\tresult := &%s.%sResult{}\n\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\treturn nil", m.ServicePackage, m.GoName)
	}
}

func renderWebSocketStream(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tif p != nil {\n\t\tcount := 3\n\t\tif p.Value != \"\" { count = len(p.Value); if count > 10 { count = 10 } }\n\t\tstreamCount := count\n\t\tif %t && streamCount > 2 { streamCount = 2 }\n\t\tfor i := 1; i <= streamCount; i++ {\n\t\t\tnotification := &%s.%sResult{Value: fmt.Sprintf(\"Stream %%d of %%d\", i, count)}\n\t\t\tif err := stream.SendNotification(ctx, notification); err != nil { return err }\n\t\t}\n\t\tif %t { testErr := &loom.ServiceError{Name: \"test_error\", Message: \"Streaming error occurred\"}; if err := stream.SendError(ctx, testErr); err != nil { return err }; return nil }\n\t\tresult := &%s.%sResult{Value: \"completed\"}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.Info.Modifier == ModifierError, m.ServicePackage, m.GoName, m.Info.Modifier == ModifierError, m.ServicePackage, m.GoName)
	case TypeArray:
		return fmt.Sprintf("\tif p == nil {\n\t\treturn nil\n\t}\n\tif len(p.Items) == 0 {\n\t\tnotification := &%s.%sResult{Items: []string{\"empty\"}}\n\t\tif err := stream.SendNotification(ctx, notification); err != nil { return err }\n\t} else {\n\t\tfor _, item := range p.Items {\n\t\t\tnotification := &%s.%sResult{Items: []string{fmt.Sprintf(\"Processing: %%s\", item)}}\n\t\t\tif err := stream.SendNotification(ctx, notification); err != nil { return err }\n\t\t}\n\t}\n\tresult := &%s.%sResult{Items: []string{\"completed\"}}\n\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\treturn nil", m.ServicePackage, m.GoName, m.ServicePackage, m.GoName, m.ServicePackage, m.GoName)
	case TypeObject:
		return fmt.Sprintf("\tif p == nil {\n\t\treturn nil\n\t}\n\tcount := p.Field2\n\tif count <= 0 { count = 3 }\n\tif count > 10 { count = 10 }\n\tfor i := 1; i <= count; i++ {\n\t\tnotification := &%s.%sResult{\n\t\t\tField1: fmt.Sprintf(\"%%s-%%d\", p.Field1, i),\n\t\t\tField2: i,\n\t\t\tField3: i == count,\n\t\t}\n\t\tif err := stream.SendNotification(ctx, notification); err != nil { return err }\n\t}\n\tresult := &%s.%sResult{Field1: \"completed\", Field2: 100, Field3: true}\n\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\treturn nil", m.ServicePackage, m.GoName, m.ServicePackage, m.GoName)
	case TypeMap:
		return fmt.Sprintf("\tif p == nil {\n\t\treturn nil\n\t}\n\traw, _ := p.Data.Value()\n\tdata, _ := raw.(map[string]any)\n\tif len(data) == 0 {\n\t\tnotification := &%s.%sResult{Data: loom.NullableValue[any](map[string]any{\"status\": \"empty\"})}\n\t\tif err := stream.SendNotification(ctx, notification); err != nil { return err }\n\t} else {\n\t\tkeys := make([]string, 0, len(data))\n\t\tfor k := range data { keys = append(keys, k) }\n\t\tsort.Strings(keys)\n\t\tfor _, k := range keys {\n\t\t\tnotification := &%s.%sResult{Data: loom.NullableValue[any](map[string]any{\"key\": k, \"value\": data[k]})}\n\t\t\tif err := stream.SendNotification(ctx, notification); err != nil { return err }\n\t\t}\n\t}\n\tresult := &%s.%sResult{Data: loom.NullableValue[any](map[string]any{\"status\": \"completed\", \"final\": true})}\n\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\treturn nil", m.ServicePackage, m.GoName, m.ServicePackage, m.GoName, m.ServicePackage, m.GoName)
	default:
		return "\treturn nil"
	}
}

func renderWebSocketBroadcast(m *MethodImplData) string {
	switch m.Info.Type {
	case TypeString:
		return fmt.Sprintf("\tfor i := 1; i <= 2; i++ {\n\t\tresult := &%s.%sResult{Value: fmt.Sprintf(\"Server announcement %%d\", i)}\n\t\tif err := stream.SendNotification(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeArray:
		return fmt.Sprintf("\tfor i := 1; i <= 2; i++ {\n\t\tresult := &%s.%sResult{Items: []string{fmt.Sprintf(\"broadcast-%%d\", i)}}\n\t\tif err := stream.SendNotification(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeObject:
		return fmt.Sprintf("\tfor i := 1; i <= 2; i++ {\n\t\tresult := &%s.%sResult{Field1: fmt.Sprintf(\"broadcast-%%d\", i), Field2: i, Field3: i%%2 == 0}\n\t\tif err := stream.SendNotification(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	case TypeMap:
		return fmt.Sprintf("\tfor i := 1; i <= 2; i++ {\n\t\tresult := &%s.%sResult{Data: loom.NullableValue[any](map[string]any{\"broadcast\": i, \"timestamp\": time.Now().Unix()})}\n\t\tif err := stream.SendResponse(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	default:
		return fmt.Sprintf("\tfor i := 1; i <= 2; i++ {\n\t\tresult := &%s.%sResult{}\n\t\tif err := stream.SendNotification(ctx, result); err != nil { return err }\n\t}\n\treturn nil", m.ServicePackage, m.GoName)
	}
}

func collectRequired(fields []FieldSpec) []string {
	var required []string
	for _, f := range fields {
		if f.Required {
			required = append(required, f.Name)
		}
	}
	return required
}

func shouldGenerateGRPC(m *MethodData) bool {
	return m.Info.GRPC
}

func indentBlock(s string, tabs int) string {
	prefix := strings.Repeat("\t", tabs)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
