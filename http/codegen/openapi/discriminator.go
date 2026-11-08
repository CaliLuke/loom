package openapi

type discriminator Discriminator

// MarshalJSON encodes the discriminator and its extension properties.
func (d Discriminator) MarshalJSON() ([]byte, error) {
	return MarshalJSON(discriminator(d), d.Extensions)
}

// MarshalYAML encodes the discriminator and its extension properties.
func (d Discriminator) MarshalYAML() (any, error) {
	return MarshalYAML(discriminator(d), d.Extensions)
}
