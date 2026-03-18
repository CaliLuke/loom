# Goa-AI Quickstart — Step 1: Project Setup and DSL

## Step 1: Project Setup

```bash
mkdir quickstart && cd quickstart
go mod init quickstart
go get goa.design/goa/v3@latest goa.design/goa-ai@latest
```

Create `design/design.go`.

```go
package design

import (
    . "goa.design/goa/v3/dsl"
    . "goa.design/goa-ai/dsl"
)

var _ = Service("demo", func() {
    // Agent defines an AI agent with a name and description
    Agent("assistant", "A helpful assistant", func() {
        // Use declares a toolset the agent can access
        Use("weather", func() {
            // Tool defines a capability the LLM can invoke
            Tool("get_weather", "Get current weather", func() {
                // Args defines the input schema
                Args(func() {
                    Attribute("city", String, "City name")
                    Required("city")
                })
                // Return defines the output schema
                Return(func() {
                    Attribute("temperature", Int, "Temperature in Celsius")
                    Attribute("conditions", String, "Weather conditions")
                    Required("temperature", "conditions")
                })
            })
        })
    })
})
```

## Generate code

```bash
goa gen quickstart/design
```

This creates Goa-AI generated registration and toolset packages:

- `gen/<service>/agents/<agent>`
- `gen/<service>/toolsets/<toolset>`
- Tool schemas, codecs, and specs used by planners and runtime

Never edit `gen/` files directly.
