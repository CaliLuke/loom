package openapiimport

func isUnconstrainedSchema(schema *Schema) bool {
	return schema != nil && schema.Ref == "" && schema.Type == "" && schema.Format == "" && len(schema.OneOf) == 0 && len(schema.Bases) == 0 &&
		len(schema.Properties) == 0 && len(schema.Required) == 0 && schema.Items == nil &&
		schema.AdditionalProperties == nil && len(schema.Enum) == 0 && schema.Pattern == "" &&
		schema.Minimum == nil && schema.Maximum == nil && schema.ExclusiveMinimum == nil &&
		schema.ExclusiveMaximum == nil && schema.MinLength == nil && schema.MaxLength == nil &&
		schema.MinItems == nil && schema.MaxItems == nil && !schema.unsupportedComposition
}
