package codegen

// Artifact describes a rendered file and its live-destination write policy.
// A staging directory changes Path but never changes SkipExist.
type Artifact struct {
	// Path is the rendered file path, absolute or relative to the process directory.
	Path string `json:"path"`
	// SkipExist preserves any entry already occupying the live destination.
	// Installation must create the destination exclusively when this is true.
	// Create-only artifacts are rejected inside the regenerated gen tree.
	SkipExist bool `json:"skip_exist"`
}
