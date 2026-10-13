package codegen

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/observability/diagnostics"
)

// DiagnosticFiles returns immutable-policy construction metadata for credential-
// bearing services. The artifact is shared by application client/server wiring;
// transport names follow the existing HTTP or JSON-RPC generated directory.
func DiagnosticFiles(services *ServicesData, transportName string) []*codegen.File {
	var files []*codegen.File
	for _, service := range services.Expressions.Services {
		data := services.Get(service.Name())
		fields := serviceDiagnosticFields(data)
		if emptyDiagnosticFields(fields) {
			continue
		}
		files = append(files, &codegen.File{
			Path: filepath.Join(codegen.Gendir, transportName, data.Service.PathName, "diagnostics.go"),
			Sections: []codegen.Section{
				codegen.Header("Credential locations for "+service.Name(), "diagnostics", []*codegen.ImportSpec{
					codegen.LoomNamedImport("observability/diagnostics", "loomdiagnostics"),
				}),
				codegen.NewRenderSection("diagnostic-fields", func() string {
					return "// Fields returns the service's credential wire locations for development diagnostics.\n" +
						"// Each call owns its slices; diagnostic constructors copy them into immutable policy.\n" +
						"func Fields() loomdiagnostics.Fields {\nreturn " + diagnosticFieldsLiteral(fields) + "\n}\n"
				}),
			},
		})
	}
	return files
}

func serviceDiagnosticFields(data *ServiceData) diagnostics.Fields {
	var fields diagnostics.Fields
	for _, endpoint := range data.Endpoints {
		request := endpoint.valueTransport.Request
		for _, binding := range endpoint.valueTransport.Security.CredentialBindings {
			switch binding.Location {
			case "header":
				fields.Headers = append(fields.Headers, binding.Name)
			case "cookie":
				fields.Headers = append(fields.Headers, "Cookie", "Set-Cookie")
			case "query":
				fields.Query = append(fields.Query, binding.Name)
			case "path":
				fields.Path = true
			}
			for _, source := range binding.Sources {
				projectBodyCredential(&fields, request, request.Body, source)
				projectBodyCredential(&fields, request, request.StreamingBody, source)
			}
		}
	}
	for _, names := range []*[]string{&fields.Headers, &fields.Query, &fields.JSON, &fields.Form} {
		slices.Sort(*names)
		*names = slices.Compact(*names)
	}
	return fields
}

func projectBodyCredential(
	fields *diagnostics.Fields, request *transportir.Request, body *expr.AttributeExpr, source string,
) {
	if body == nil || body.Type == expr.Empty {
		return
	}
	if request.BodyOriginKey == source || request.BodyOrigin == source {
		fields.Body = true
		return
	}
	name, attribute := body.FindAttribute(source)
	if attribute == nil {
		return
	}
	jsonName, formName, err := representation.FieldWireNames(attribute, name)
	if err != nil {
		panic(codegen.NewError(nil, attribute, fmt.Errorf("project credential diagnostic name: %w", err)))
	}
	if jsonName != "-" {
		fields.JSON = append(fields.JSON, jsonName)
	}
	if formName != "" && formName != "-" {
		fields.Form = append(fields.Form, formName)
	}
}

func emptyDiagnosticFields(fields diagnostics.Fields) bool {
	return len(fields.Headers)+len(fields.Query)+len(fields.JSON)+len(fields.Form) == 0 && !fields.Body && !fields.Path
}

func diagnosticFieldsLiteral(fields diagnostics.Fields) string {
	var result strings.Builder
	result.WriteString("loomdiagnostics.Fields{\n")
	for _, item := range []struct {
		name   string
		values []string
	}{
		{"Headers", fields.Headers}, {"Query", fields.Query}, {"JSON", fields.JSON}, {"Form", fields.Form},
	} {
		if len(item.values) == 0 {
			continue
		}
		result.WriteString(item.name + ": []string{")
		for _, value := range item.values {
			result.WriteString(strconv.Quote(value) + ",")
		}
		result.WriteString("},\n")
	}
	if fields.Body {
		result.WriteString("Body: true,\n")
	}
	if fields.Path {
		result.WriteString("Path: true,\n")
	}
	result.WriteString("}")
	return result.String()
}
