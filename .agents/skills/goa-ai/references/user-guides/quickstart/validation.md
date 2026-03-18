# Goa-AI Quickstart — Step 4: Validation and Retry Hints

Update tool args in `design/design.go`:

```go
Args(func() {
    Attribute("city", String, "City name", func() {
        MinLength(2)
        MaxLength(100)
    })
    Attribute("units", String, "Temperature units", func() {
        Enum("celsius", "fahrenheit")
    })
    Required("city")
})
```

Regenerate:

```bash
goa gen quickstart/design
```

Runtime rejects invalid payloads before executor. A failure returns `RetryHint`:

```go
&planner.ToolResult{
    Name: "weather.get_weather",
    RetryHint: &planner.RetryHint{
        Message: `validation failed: city length must be >= 2; units must be one of ["celsius", "fahrenheit"]`,
    },
}
```

The planner can use this hint to self-correct.
