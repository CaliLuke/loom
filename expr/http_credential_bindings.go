package expr

type (
	// HTTPCredentialBinding describes one effective wire credential operation.
	// It is a detached projection of the binding plan used for request validation
	// and body inference, including all Basic username/password source roles.
	HTTPCredentialBinding struct {
		// Location is the resolved header, query, cookie, path or body location.
		Location string
		// Name is the physical credential slot at that location.
		Name string
		// Sources lists logical payload members that supply this operation.
		// Explicit bodies may duplicate these sources outside the primary slot.
		Sources []string
	}
)

// CredentialBindings returns a detached snapshot of effective credential wire
// operations, including explicit duplicate locations for their source members.
// Unrelated ordinary fields and disabled security do not produce entries.
// Call it after endpoint preparation, as with other evaluated transport metadata.
func (e *HTTPEndpointExpr) CredentialBindings() []HTTPCredentialBinding {
	plan := e.ensureWireBindings()
	credentials := make(map[string]bool)
	for _, binding := range plan.bindings {
		if len(binding.schemes) == 0 {
			continue
		}
		for _, source := range binding.sources {
			if source != "" {
				credentials[binding.sourceKind+":"+source] = true
			}
		}
	}
	var result []HTTPCredentialBinding
	for _, binding := range plan.bindings {
		credential := len(binding.schemes) > 0
		for _, source := range binding.sources {
			credential = credential || credentials[binding.sourceKind+":"+source]
		}
		if !credential {
			continue
		}
		result = append(result, HTTPCredentialBinding{
			Location: binding.location,
			Name:     binding.name,
			Sources:  append([]string(nil), binding.sources...),
		})
	}
	return result
}
