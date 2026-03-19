name: goa-light
description: Build and maintain `goa-light` services in Go. Use this skill when a user mentions Goa, `goa-light`, Goa DSL, `goa gen`, generated `gen/` transport code, OpenAPI/proto generation, service implementation after DSL changes, or refactoring a project with a `design` package.
---
# Goa Light

Use this skill for `goa-light` framework work only. It does not cover Goa-AI.

## Non-Negotiables

- Treat `design/*.go` as the source of truth.
- Regenerate after every design change with `goa gen <module-import-path>/design`.
- Never hand-edit generated `gen/` files.
- Implement business logic in non-generated files.
- Use Go import paths for Goa commands, not filesystem paths.
- Commit generated code; do not rely on CI to regenerate it.

## Runtime Gotchas

- SSE server streams do not expose a generated `Open()` hook. Goa emits SSE headers on the first `Send`, so long-idle streams that must return `200` before the first domain event need a non-generated transport/runtime flush strategy or an explicit initial event designed into the contract.
- Do not "fix" SSE by hand-editing generated stream files. Keep the fix in `design/*.go` or non-generated transport/runtime code.
- Do not map multi-cookie responses through ad hoc `Header("set_cookies:Set-Cookie")` bags and then patch generated encoders. Prefer idiomatic Goa cookies in the DSL when feasible. If the response shape still depends on raw cookie header values, emit them from non-generated transport code on the live `http.ResponseWriter` instead of editing generated files.

## Goa-Light Contract Rules

- `goa-light` emits OpenAPI 3.1 / JSON Schema 2020-12 only. The canonical artifacts are `gen/http/openapi.json` and `gen/http/openapi.yaml`.
- Treat OpenAPI output shape as framework contract. Stable schema names, canonical `operationId`, and `libopenapi` validation are intentional behavior, not incidental formatting.
- Structurally identical generated OpenAPI components are deduplicated and reused by `$ref`; explicit `openapi:typename` declarations still keep their own named components.
- Wrapper-style unions now emit OpenAPI discriminators with:
  - `discriminator.propertyName`
  - `discriminator.mapping`
  - `oneOf` refs to generated `...Envelope` component schemas
- `OneOf(...)` works both as a named union declaration and as a type constructor.
- Explicit union discriminator tags control the wire value even when schema/type names are renamed for OpenAPI purposes.
- Use `FormRequest()` on HTTP endpoints when the request body contract is `application/x-www-form-urlencoded`.
- `FormRequest()` is for typed object payloads and constructor unions only; incompatible body/param mixes are rejected during design validation instead of silently falling back to app-local parsing.
- Form-encoded unions use the same canonical wrapper shape as JSON: discriminator under `type`, branch payload under `value`.
- `MultipartRequest()` now generates server-side decoding for supported object payloads, including common file-plus-fields uploads, instead of requiring a handwritten decoder hook.
- Generated multipart decoding is intentionally narrower than form decoding: unsupported multipart payload shapes still use the legacy custom encoder/decoder seam instead of partial magic.
- For supported multipart object payloads with a single top-level file field, sibling body attributes named `filename` and `content_type` are auto-populated from the uploaded part when those fields are present and not explicitly supplied.
- Use `OptionalRequestBody()` when an HTTP endpoint may omit a JSON request body entirely.
- `OptionalRequestBody()` is intentionally narrow: JSON only, object request bodies only, no raw body streaming, no multipart, no form bodies, and no required body-mapped payload attribute.
- OpenAPI request bodies generated from `OptionalRequestBody()` render with `required: false`.
- Session auth is first-class. Prefer the built-in DSL instead of hand-rolling bearer-or-cookie glue:
  - `SessionAuth(name, fn)`
  - `BearerTransport(scheme, fieldName, fn...)`
  - `CookieTransport(scheme, fieldName, fn...)`
  - `CookieName(name)`
  - `SessionSecurity(contract)`
- Use `AuthErrorResponses()` for standard HTTP auth failures instead of duplicating 401/403 mappings.
- Prefer modeled response cookies over raw `Set-Cookie` header bags. `SessionCookie(...)` is the secure-default helper for common session issuance.
- Structured remediation metadata is part of the contract surface. Use:
  - `Remedy(fn)`
  - `RemedyCode(code)`
  - `SafeMessage(message)`
  - `RetryHint(hint)`
- JSON-RPC is a first-class transport in this repo. Do not assume HTTP or gRPC semantics automatically carry over.

## Practical Checks

- If a design hand-models bearer-or-cookie auth, duplicated auth responses, or raw `Set-Cookie` headers, check whether the newer session and cookie DSL should replace that glue first.
- If a consumer compares OpenAPI outputs, verify it reads the OpenAPI 3.1 artifacts before changing framework code.
- If a union-related change looks wrong, inspect both `OneOf(...)` usage and explicit discriminator tags before changing codegen.
- If the task touches generated transport errors, confirm whether remediation metadata should flow through the contract before adding ad hoc fields.

## Default Workflow

1. Detect the Goa surface: `go.mod`, `design/`, DSL imports, or `gen/` folders.
2. Edit the DSL in `design/`.
3. Run `goa gen <module>/design`.
4. Run `goa example <module>/design` only when scaffolding a new service or new starter files are explicitly wanted.
5. Implement logic outside `gen/`.
6. Verify with `go mod tidy` and project tests.

## Command Reminders

```bash
go install goa.design/goa/v3/cmd/goa@latest
goa version
goa gen <module-import-path>/design
goa example <module-import-path>/design
```

- Correct: `goa gen example.com/myapi/design`
- Incorrect: `goa gen ./design`

## References

- Framework/source map: `references/repo-map.md`
- Use only the original full guide pages under `references/user-guides/*.md`.
- For framework/runtime internals, inspect the Goa source tree described in `references/repo-map.md`.

## Original Guide Pages

- `references/user-guides/quickstart.md`
- `references/user-guides/dsl-reference.md`
- `references/user-guides/code-generation.md`
- `references/user-guides/http-guide.md`
- `references/user-guides/grpc-guide.md`
- `references/user-guides/error-handling.md`
- `references/user-guides/interceptors.md`
- `references/user-guides/production.md`

## Selection Rules

- Start with the one full guide page that best matches the immediate task.
- For repo-specific behavior differences from upstream Goa, use the `Goa-Light Contract Rules` section in this skill before inspecting the wider source tree.
- Load additional full guide pages only if the first one is insufficient.
- Prefer `references/repo-map.md` and the Goa source tree for framework internals or runtime behavior.
