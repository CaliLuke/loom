---
name: go-code-review
description: Quick-reference checklist for Go code review based on the Go Wiki CodeReviewComments. Use when reviewing Go code or checking code against community style standards.
---

# Go Code Review Checklist

> Based on [Go Wiki CodeReviewComments](https://github.com/golang/wiki/blob/master/CodeReviewComments.md).
> Apply the repository’s AGENTS.md rules when they are more specific.

---

## Formatting

- [ ] **gofmt**: Code is formatted with `gofmt` or `goimports`

---

## Documentation

- [ ] **Comment sentences**: Comments are full sentences starting with the name being described, ending with a period
- [ ] **Doc comments**: All exported names have doc comments; non-trivial unexported declarations too
- [ ] **Package comments**: Package comment appears adjacent to package clause with no blank line
- [ ] **Named result parameters**: Only used when they clarify meaning (e.g., multiple same-type returns), not just to enable naked returns

---

## Error Handling

- [ ] **Handle errors**: No discarded errors with `_`; handle, return, or (exceptionally) panic
- [ ] **Error strings**: Lowercase, no punctuation (unless starting with proper noun/acronym)
- [ ] **In-band errors**: No magic values (-1, "", nil); use multiple returns with error or ok bool
- [ ] **Indent error flow**: Handle errors first and return; keep normal path at minimal indentation

---

## Naming

- [ ] **MixedCaps**: Use `MixedCaps` or `mixedCaps`, never underscores; unexported is `maxLength` not `MAX_LENGTH`
- [ ] **Initialisms**: Keep consistent case: `URL`/`url`, `ID`/`id`, `HTTP`/`http` (e.g., `ServeHTTP`, `xmlHTTPRequest`)
- [ ] **Variable names**: Short names for limited scope (`i`, `r`, `c`); longer names for wider scope
- [ ] **Receiver names**: One or two letter abbreviation of type (`c` for `Client`); no `this`, `self`, `me`; consistent across methods
- [ ] **Package names**: No stuttering (use `chubby.File` not `chubby.ChubbyFile`); avoid `util`, `common`, `misc`

---

## Concurrency

- [ ] **Goroutine lifetimes**: Clear when/whether goroutines exit; document if not obvious
- [ ] **Synchronous functions**: Prefer sync over async; let callers add concurrency if needed
- [ ] **Contexts**: First parameter; not in structs; no custom Context types; pass even if you think you don't need to

---

## Interfaces

- [ ] **Interface location**: Define in consumer package, not implementor; return concrete types from producers
- [ ] **No premature interfaces**: Don't define before used; don't define "for mocking" on implementor side
- [ ] **Receiver type**: Use pointer if mutating, has sync fields, or is large; value for small immutable types; don't mix

---

## Data Structures

- [ ] **Empty slices**: Preserve the contract’s distinction between nil absence and supplied empty collections.
- [ ] **Copying**: Be careful copying structs with pointer/slice fields; don't copy `*T` methods' receivers by value

---

## Security

- [ ] **Crypto rand**: Use `crypto/rand` for keys, not `math/rand`
- [ ] **Don't panic**: Use error returns for normal error handling; panic only for truly exceptional cases

---

## Style

- [ ] **Line length**: No rigid limit, but avoid uncomfortably long lines; break by semantics, not arbitrary length
- [ ] **Naked returns**: Only in short functions; explicit returns in medium/large functions
- [ ] **Pass values**: Don't use pointers just to save bytes; pass `string` not `*string` for small fixed-size types

---

## Imports

- [ ] **Import groups**: Standard library first, then blank line, then external packages
- [ ] **Import renaming**: Avoid unless collision; rename local/project-specific import on collision
- [ ] **Import blank**: `import _ "pkg"` only in main package or tests
- [ ] **Import dot**: Only for circular dependency workarounds in tests

---

## Testing

- [ ] **Examples**: Include runnable `Example` functions or tests demonstrating usage
- [ ] **Useful test failures**: Messages include what was wrong, inputs, got, and want; order is `got != want`

---

## Quick Review Commands

From the Loom repository root:

```bash
make fmt
make lint
```
