package testdata

var CreateExternalNameCode = `// CreateFromExternalNameT initializes t from the fields of v
func (t *ExternalNameType) CreateFromExternalNameT(v *testdata.ExternalNameT) {
	temp := &ExternalNameType{
		String: &v.String,
	}
	*t = *temp
}
`

var CreateExternalNameRequiredCode = `// CreateFromExternalNameT initializes t from the fields of v
func (t *ExternalNameType) CreateFromExternalNameT(v *testdata.ExternalNameT) {
	temp := &ExternalNameType{
		String: v.String,
	}
	*t = *temp
}
`

var CreateExternalNameWithInitialismCode = `// CreateFromApiNameT initializes t from the fields of v
func (t *ExternalNameWithInitialismType) CreateFromApiNameT(v *testdata.ApiNameT) {
	temp := &ExternalNameWithInitialismType{
		String: &v.String,
	}
	*t = *temp
}
`
