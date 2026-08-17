module github.com/CaliLuke/loom/http/integration_tests

go 1.27rc2

require (
	github.com/CaliLuke/loom v1.7.1
	github.com/stretchr/testify v1.12.0
	github.com/tmaxmax/go-sse v0.11.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace github.com/CaliLuke/loom => ../..
