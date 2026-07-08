package codegen

type (
	// CORSData contains a service's generated CORS policy.
	CORSData struct {
		Origins []*CORSOriginData
	}

	// CORSOriginData contains a generated CORS origin policy.
	CORSOriginData struct {
		Pattern     string
		Regex       bool
		Methods     []string
		Headers     []string
		Expose      []string
		MaxAge      int
		Credentials bool
	}
)
