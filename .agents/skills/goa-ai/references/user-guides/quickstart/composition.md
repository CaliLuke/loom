# Goa-AI Quickstart — Step 6: Agent Composition

Agents can invoke other agents as tools.

```go
var _ = Service("weather", func() {
    Agent("forecaster", "Weather specialist", func() {
        Use("weather_tools", func() {
            Tool("get_forecast", "Get forecast", func() {
                Args(func() { Attribute("city", String, "City") Required("city") })
                Return(func() { Attribute("forecast", String, "Forecast") Required("forecast") })
            })
        })

        Export("ask_weather", func() {
            Tool("ask", "Ask weather specialist", func() {
                Args(func() { Attribute("question", String, "Question") Required("question") })
                Return(func() { Attribute("answer", String, "Answer") Required("answer") })
            })
        })
    })
})

var _ = Service("demo", func() {
    Agent("assistant", "A helpful assistant", func() {
        UseAgentToolset("weather", "forecaster", "ask_weather")
    })
})
```

Regenerate:

```bash
goa gen quickstart/design
```

Runtime handles nested agent runs with child `RunLink` and linked streaming events.
