package codegen

// scalarDecodeData binds the shared scalar decoder to a transport location.
// Carrier lookup supplies raw contents and presence before rendering this plan.
type scalarDecodeData struct {
	*Element
	// Location names the transport source in missing-field diagnostics.
	Location string
}

func scalarElementData(element *Element, location string) *scalarDecodeData {
	return &scalarDecodeData{Element: element, Location: location}
}
