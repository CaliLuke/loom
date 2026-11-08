package openapiimport

import "strconv"

func (r *renderer) headerMetadata(header Header) ([]renderedMetadata, error) {
	metadata, err := reusableMetadata("header", header.referenceChain)
	if err != nil {
		return nil, err
	}
	planner := documentPlanner{schemas: r.schemas}
	parameter := Parameter{In: "header", Schema: header.Schema, Serialization: header.Serialization, AllowReserved: header.AllowReserved}
	codec, _, err := planner.parameterCodec(parameter)
	if err != nil {
		return nil, err
	}
	metadata = append(metadata, parameterCodecMetadata("header", codec)...)
	if header.AllowReserved {
		metadata = append(metadata, renderedMetadata{name: "openapi:allowReserved", value: "true"})
	}
	return metadata, nil
}

func (r *renderer) responseHeaderAttributes(headers []renderedHeader, path string) error {
	for _, header := range headers {
		metadata, err := r.headerMetadata(header.header)
		if err != nil {
			return err
		}
		if err := r.attribute(
			header.field,
			header.header.Schema,
			&header.header.Description,
			path+"/"+escapeJSONPointer(header.name),
			metadata...,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) responseMapping(response renderedResponse, failure bool, path string) error {
	prefix := responseMappingPrefix(response, failure)
	if !responseMappingNeedsBlock(response, failure) {
		r.line("%s)", prefix)
		return nil
	}
	r.open("%s, func()", prefix)
	if response.statusField != "" {
		r.line("StatusField(%q)", response.statusField)
	}
	if err := r.responseMappingMetadata(response, failure); err != nil {
		return err
	}
	if err := r.responseMappingRawBody(response, path); err != nil {
		return err
	}
	r.responseMappingHeaders(response.headers)
	if err := r.responseMappingBody(response, failure, path); err != nil {
		return err
	}
	r.close()
	return nil
}

func responseMappingPrefix(response renderedResponse, failure bool) string {
	prefix := "Response("
	if failure {
		prefix += strconv.Quote(response.errorName) + ", "
	}
	prefix += selectorDSL(response.selector)
	if response.handshake != "" {
		prefix += ", " + response.handshake
	}
	return prefix
}

func responseMappingNeedsBlock(response renderedResponse, failure bool) bool {
	return response.statusField != "" || response.branch != "" || response.response.Schema != nil && len(response.response.Schema.MediaTypes) > 0 || len(response.response.referenceChain) > 0 || failure || response.response.Summary != "" || response.response.Description != "" ||
		response.response.ContentType != "" || len(response.headers) > 0 || len(response.response.Extensions) > 0
}

func (r *renderer) responseMappingMetadata(response renderedResponse, failure bool) error {
	metadata, err := reusableMetadata("response", response.response.referenceChain)
	if err != nil {
		return err
	}
	for _, item := range metadata {
		r.line("Meta(%q, %q)", item.name, item.value)
	}

	if response.response.Summary != "" {
		r.line("Meta(%q, %q)", "openapi:summary", response.response.Summary)
	}
	if response.response.Description != "" {
		r.line("Description(%q)", response.response.Description)
	}
	if failure {
		r.line("Meta(%q, %q)", "openapi:description:errorName", "false")
	}
	if err := r.emitExtensions("response", response.response.Extensions); err != nil {
		return err
	}
	if response.response.Schema != nil && len(response.response.Schema.MediaTypes) > 0 {
		if err := r.mediaTypeMappings(response.response.Schema); err != nil {
			return err
		}
	}
	if response.response.ContentType != "" {
		r.line("ContentType(%q)", response.response.ContentType)
	}
	if failure && response.response.Schema == nil {
		r.line("Body(Empty)")
	}
	return nil
}

func (r *renderer) responseMappingRawBody(response renderedResponse, path string) error {
	if response.rawBody && response.response.Schema != nil {
		if err := r.responseSchemaBody("OpenAPIBody", schemaWithExamples(response.response.Schema, response.response.Examples), path+"/content/schema"); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) responseMappingBody(response renderedResponse, failure bool, path string) error {
	if response.response.stream != nil {
		return nil
	}
	if response.statusField != "" || response.branch != "" {
		// Raw transports carry bytes outside the native value. OpenAPIBody
		// already describes them; even Body(Empty) is an authored body there.
		if response.rawBody {
			return nil
		}
		if response.response.Schema == nil {
			if !failure {
				r.line("Body(Empty)")
			}
		} else {
			r.line("Body(%q)", response.body)
		}
		return nil
	}
	wrapUnconstrainedError := failure && r.effectiveUnconstrained(response.response.Schema)
	wrapMediaError := failure && r.unionSchema(response.response.Schema)
	if response.body != "" && (len(response.headers) > 0 || wrapUnconstrainedError || wrapMediaError) {
		r.line("Body(%q)", response.body)
	}
	if len(response.headers) == 0 && !response.rawBody && response.response.Schema != nil {
		schema := response.response.Schema
		// Reference admission has already rejected mandatory initializer cycles.
		for schema.Ref != "" {
			resolved, err := r.resolveSchemaReference(schema, path+"/content/schema")
			if err != nil {
				return err
			}
			schema = resolved
		}
		if representation := classifyObject(schema); representation == objectStructured || representation == objectTypedRecord {
			if err := r.responseSchemaBody("Body", schemaWithExamples(response.response.Schema, response.response.Examples), path+"/content/schema"); err != nil {
				return err
			}
		}
	}
	if wrapUnconstrainedError || response.response.Schema != nil && response.response.Schema.Ref != "" {
		if err := r.responseSchemaBody("OpenAPIBody", schemaWithExamples(response.response.Schema, response.response.Examples), path+"/content/schema"); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) responseMappingHeaders(headers []renderedHeader) {
	for _, header := range headers {
		name := header.field
		if name != header.name {
			name += ":" + header.name
		}
		r.line("Header(%q)", name)
	}
}

func schemaWithExamples(schema *Schema, examples []Example) *Schema {
	if schema == nil || len(examples) == 0 {
		return schema
	}
	combined := *schema
	combined.Examples = append(append([]Example(nil), schema.Examples...), examples...)
	return &combined
}

func (r *renderer) result(response renderedResponse, path string) error {
	return r.responseType("Result", "", response, path)
}

func (r *renderer) errorDefinition(response renderedResponse, path string) error {
	return r.responseType("Error", response.errorName, response, path)
}

func (r *renderer) errorTypeClones() error {
	for operationIndex := range r.operations {
		for responseIndex := range r.operations[operationIndex].failures {
			response := r.operations[operationIndex].failures[responseIndex]
			if !response.cloneErrorType {
				continue
			}
			path := errorCloneSchemaPath(operationIndex, response.status)
			r.declaration = errorCloneDeclaration(operationIndex, response.status)
			schema := schemaWithExamples(response.response.Schema, response.response.Examples)
			expression, _, err := r.schemaExpression(schema, path)
			if err != nil {
				return err
			}
			r.open("var %s = Type(%q, func()", response.cloneErrorGoName, response.cloneErrorTypeName)
			r.line("Extend(%s)", expression)
			if err := r.emitNullableGoType(schema, path); err != nil {
				return err
			}
			if err := r.schemaBlock(schema, path, true, nil); err != nil {
				return err
			}
			r.close()
			r.line("")
			r.declaration = ""
		}
	}
	return nil
}

func (r *renderer) responseType(call, name string, response renderedResponse, path string) error {
	responseSchema := schemaWithExamples(response.response.Schema, response.response.Examples)
	wrapUnconstrainedError := call == "Error" && r.effectiveUnconstrained(responseSchema)
	wrapMediaError := call == "Error" && r.unionSchema(responseSchema)
	prefix := call + "("
	if name != "" {
		prefix += strconv.Quote(name) + ", "
	}
	if call == "Error" && response.cloneErrorType {
		r.line("%s%s)", prefix, response.cloneErrorGoName)
		return nil
	}
	if len(response.headers) == 0 && response.statusField == "" && !wrapUnconstrainedError && !wrapMediaError {
		if call == "Error" && response.response.Schema == nil {
			r.open("Error(%q, func()", name)
			r.line("Attribute(%q, String)", "message")
			r.close()
			return nil
		}
		if call == "Error" && response.rawBody {
			r.line("Error(%q)", name)
			return nil
		}
		return r.responseTypeWithoutHeaders(call, prefix, response, responseSchema, path+"/content/schema")
	}
	return r.responseCarrier(prefix, response, responseSchema, wrapUnconstrainedError, path)
}

func (r *renderer) responseCarrier(prefix string, response renderedResponse, responseSchema *Schema, wrapUnconstrainedError bool, path string) error {
	r.open("%sfunc()", prefix)
	if response.statusField != "" {
		r.line("Attribute(%q, Int)", response.statusField)
	}
	if err := r.responseHeaderAttributes(response.headers, path+"/headers"); err != nil {
		return err
	}
	if response.response.Schema != nil && !response.rawBody {
		bodySchema := responseSchema
		if wrapUnconstrainedError && response.statusField == "" {
			bodySchema = &Schema{Unconstrained: true}
		}
		if err := r.attribute(response.body, bodySchema, nil, path+"/content/schema"); err != nil {
			return err
		}
	}
	var required []string
	if response.statusField != "" {
		required = append(required, response.statusField)
	}
	for _, header := range response.headers {
		if header.header.Required {
			required = append(required, header.field)
		}
	}
	if response.response.Schema != nil && !response.rawBody {
		required = append(required, response.body)
	}
	if len(required) > 0 {
		r.quotedCall("Required", required)
	}
	r.close()
	return nil
}

func (r *renderer) responseTypeWithoutHeaders(call, prefix string, response renderedResponse, schema *Schema, path string) error {
	if response.response.Schema == nil || response.rawBody {
		r.line("%sEmpty)", prefix)
		return nil
	}
	expression, object, err := r.schemaExpression(schema, path)
	if err != nil {
		return err
	}
	if object {
		r.open("%sfunc()", prefix)
		if err := r.emitNullableGoType(schema, path); err != nil {
			return err
		}
		if err := r.schemaBlock(schema, path, call == "Error", nil); err != nil {
			return err
		}
		r.close()
		return nil
	}
	if r.hasSchemaBlock(schema) || r.effectiveNullable(schema) {
		r.open("%s%s, func()", prefix, expression)
		if err := r.emitNullableGoType(schema, path); err != nil {
			return err
		}
		if err := r.schemaAttributeBlock(schema, path, nil); err != nil {
			return err
		}
		r.close()
		return nil
	}
	r.line("%s%s)", prefix, expression)
	return nil
}
