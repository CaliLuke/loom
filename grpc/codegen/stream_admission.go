package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

// StreamAdmissionData binds metadata availability to shared authentication.
type StreamAdmissionData struct {
	// CheckField holds the shared authentication entrypoint on the gRPC server.
	CheckField string
	// DecoderName decodes metadata into a partial service payload.
	DecoderName string
	// MetadataName owns metadata extraction for both decoding phases.
	MetadataName string
	// PayloadName is the resolved service payload type name.
	PayloadName string
}

func buildStreamAdmission(endpoint *EndpointData, data *ServiceData, payloadName string) *StreamAdmissionData {
	if endpoint.Request.StreamEnvelope == nil || endpoint.Method.Authentication == nil || len(endpoint.Method.Schemes) == 0 {
		return nil
	}
	metadata := make(map[string]bool, len(endpoint.Request.Metadata))
	for _, field := range endpoint.Request.Metadata {
		metadata[field.AttributeName] = true
	}
	for _, requirement := range endpoint.Method.Requirements {
		for _, scheme := range requirement.Schemes {
			if scheme.TransportOwned {
				return nil
			}
			if scheme.Type == "Basic" {
				if !metadata[scheme.UsernameAttr] || !metadata[scheme.PasswordAttr] {
					return nil
				}
			} else if !metadata[scheme.KeyAttr] {
				return nil
			}
		}
	}
	return &StreamAdmissionData{
		CheckField:   data.Scope.Unique("authenticate" + endpoint.Method.VarName),
		DecoderName:  data.Scope.Unique("decode" + endpoint.Method.VarName + "Admission"),
		MetadataName: data.Scope.Unique("decode" + endpoint.Method.VarName + "Metadata"),
		PayloadName:  payloadName,
	}
}

func grpcAdmissionDecoderSection(endpoint *EndpointData) codegen.Section {
	var b strings.Builder
	admission := endpoint.Request.Admission
	fmt.Fprintf(&b, "func %s(md metadata.MD) (", admission.MetadataName)
	for _, field := range endpoint.Request.Metadata {
		fmt.Fprintf(&b, "%s %s,", field.VarName, field.TypeRef)
	}
	b.WriteString("err error) {\n")
	for _, field := range endpoint.Request.Metadata {
		b.WriteString(renderGRPCMetadataDecode(field, "md", true))
		b.WriteString(field.Validate)
		b.WriteByte('\n')
	}
	b.WriteString("\nreturn\n}\n")
	fmt.Fprintf(&b, "func %s(ctx context.Context, _ any, md metadata.MD) (any,error) {\n", admission.DecoderName)
	writeAdmissionMetadataCall(&b, endpoint)
	init := endpoint.Request.ServerConvert.Init
	fmt.Fprintf(&b, "%s := new(%s)\n", init.ReturnVarName, admission.PayloadName)
	b.WriteString(init.FieldCode)
	fmt.Fprintf(&b, "\nreturn %s,nil\n}\n", init.ReturnVarName)
	return codegen.NewJenniferSection("stream-admission-decoder", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(b.String()))
	})
}

func writeAdmissionMetadataCall(b *strings.Builder, endpoint *EndpointData) {
	for _, field := range endpoint.Request.Metadata {
		fmt.Fprintf(b, "%s,", field.VarName)
	}
	fmt.Fprintf(b, "err := %s(md)\nif err != nil {\nreturn nil,err\n}\n", endpoint.Request.Admission.MetadataName)
}
