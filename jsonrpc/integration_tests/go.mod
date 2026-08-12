module github.com/CaliLuke/loom/jsonrpc/integration_tests

go 1.27rc2

require (
	github.com/CaliLuke/loom v1.3.2
	github.com/gorilla/websocket v1.5.3
	github.com/stretchr/testify v1.11.1
	github.com/tmaxmax/go-sse v0.11.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/dave/jennifer v1.7.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dimfeld/httppath v0.0.0-20170720192232-ee938bf73598 // indirect
	github.com/go-chi/chi/v5 v5.3.1 // indirect
	github.com/manveru/faker v0.0.0-20171103152722-9fbc68a78c4d // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

replace github.com/CaliLuke/loom => ../..
