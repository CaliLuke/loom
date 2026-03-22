module github.com/CaliLuke/loom/http/integration_tests

go 1.26.0

require (
	github.com/CaliLuke/loom v1.0.0
	github.com/stretchr/testify v1.11.1
	github.com/tmaxmax/go-sse v0.11.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/CaliLuke/loom => ../..
