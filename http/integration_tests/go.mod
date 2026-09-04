module github.com/CaliLuke/loom/http/integration_tests

go 1.27.0

require (
	github.com/CaliLuke/loom v1.7.1
	github.com/stretchr/testify v1.12.1
	github.com/tmaxmax/go-sse v0.11.0
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace github.com/CaliLuke/loom => ../..
