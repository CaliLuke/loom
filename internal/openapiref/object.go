package openapiref

// Object identifies a reusable OpenAPI object and its occurrence annotations.
// Nil annotations inherit; a pointer to an empty string explicitly clears one.
type Object struct {
	// Ref is a local reference to a request body, response, or header component.
	Ref string `json:"$ref"`
	// Description overrides the target description at this occurrence.
	Description *string `json:"description,omitzero"`
	// Summary overrides the target summary when that object supports summaries.
	Summary *string `json:"summary,omitzero"`
}
