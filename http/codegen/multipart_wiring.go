package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

func renderMultipartRequestEncode(request *RequestData) string {
	return "{\nencodedMultipart, partError := func() (*loomhttp.MultipartForm, error) {\n" + renderMultipartEncode(request.ClientBody, request.multipart) + "}()\nif partError != nil {\nreturn partError\n}\nif partError := loomhttp.SetMultipartRequest(req, encodedMultipart); partError != nil {\nreturn partError\n}\n}\n"
}

func renderMultipartRequestDecode(request *RequestData) string {
	var out strings.Builder
	form := "multipartForm"
	if !multipartBodyValueUsed(request.multipart) && request.multipart.ObjectOpenness != expr.ObjectClosed {
		form = "_"
	}
	fmt.Fprintf(&out, "%s, multipartErr := loomhttp.ReadMultipartForm(r)\nif multipartErr != nil {\nif errors.Is(multipartErr, loomhttp.ErrMultipartBodyAbsent) {\n", form)
	if request.MustHaveBody {
		out.WriteString("return payload, loom.MissingPayloadError()\n")
	} else {
		if request.OptionalObjectBody {
			out.WriteString("body = nil\n")
		}
		out.WriteString("multipartErr = nil\n")
	}
	out.WriteString("} else {\nvar serviceError *loom.ServiceError\nif errors.As(multipartErr, &serviceError) {\nreturn payload, serviceError\n}\nreturn payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(multipartErr))\n}\n} else {\n")
	if len(request.multipart.Branches) > 0 {
		out.WriteString(renderMultipartMatcher(request.ServerBody, request.multipart))
		out.WriteString("matched, _, partError := matchMultipart()\nif partError != nil {\nreturn payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(partError))\n}\nbody = matched\n")
	} else {
		if request.OptionalObjectBody || strings.HasPrefix(request.ServerBody.ValueRef, "*") {
			fmt.Fprintf(&out, "if body == nil {\nbody = new(%s)\n}\n", request.ServerBody.VarName)
		}
		out.WriteString("if partError := func() error {\n")
		out.WriteString(renderMultipartLeafDecode(request.ServerBody, request.multipart, "body"))
		out.WriteString(renderMultipartClosedKeys(request.multipart))
		out.WriteString("return nil\n}(); partError != nil {\nvar serviceError *loom.ServiceError\nif errors.As(partError, &serviceError) {\nreturn payload, serviceError\n}\nreturn payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(partError))\n}\n")
	}
	out.WriteString("}\n")
	return out.String()
}

func renderMultipartClosedKeys(contract *expr.HTTPMultipartEncoding) string {
	if contract.ObjectOpenness != expr.ObjectClosed {
		return ""
	}
	var out strings.Builder
	out.WriteString("projection, _, projectionError := loomhttp.MultipartObjectJSON(multipartForm, []loomhttp.FormField{\n")
	for _, field := range *expr.AsObject(contract.Body.Type) {
		captured := contract.Fields[field.Name]
		if !captured.Omitted {
			fmt.Fprintf(&out, "{Name: %q, Keys: %s, View: func() (jsontext.Value, bool, error) { return nil, true, nil }},\n", captured.Name, formKeySetLiteral(captured.Codec.Keys))
		}
	}
	out.WriteString("})\nif projectionError != nil {\nreturn projectionError\n}\naccepted, projectionError := projection.Schema(func(jsontext.Value) (bool, error) { return true, nil }, true)\nif projectionError != nil {\nreturn projectionError\n}\nif !accepted {\nreturn fmt.Errorf(\"undeclared multipart part\")\n}\n")
	return out.String()
}

// multipartBodyValueUsed reports whether native part emission reads the body.
// Empty objects still have a multipart wire envelope, but no value projection.
func multipartBodyValueUsed(contract *expr.HTTPMultipartEncoding) bool {
	if len(contract.Branches) != 0 {
		return true
	}
	for _, field := range contract.Fields {
		if !field.Omitted {
			return true
		}
	}
	return false
}

func clientBodyValueUsed(request *RequestData) bool {
	return !request.MultipartGenerated || multipartBodyValueUsed(request.multipart)
}
