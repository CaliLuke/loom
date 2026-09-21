---
title: Application Authorization
description: "Declare typed application access requirements and enforce coverage across generated endpoints."
weight: 8
---

Loom connects typed method inputs to application-owned authorization decisions.
Declare the required action in the design; implement roles, memberships,
ownership, resource relationships, grants, and policy-engine calls in Go.

## Declare and bind a requirement

```go
var DocumentRef = Type("DocumentRef", func() {
    Attribute("id", String, func() {
        MinLength(1)
    })
    Required("id")
})

var EditDocument = Authorization("document.edit", DocumentRef)

var _ = Service("documents", func() {
    StrictAuthorization()

    Method("update", func() {
        Payload(func() {
            Attribute("document_id", String)
            Required("document_id")
        })
        Authorize(EditDocument, func() {
            Bind("id", "document_id")
        })
        Result(String)
    })
})
```

`Authorization(name, input)` is a top-level declaration. Its stable name is
independent of method names. Input is a named object type, or `Empty` for a
context-only decision. A typed input can distinguish source and destination
resources, include proposed values, or identify a parent for creation.

`Bind(inputField, payloadPath)` maps every top-level input field to a payload
value. Paths use dot-separated object fields. Missing fields and incompatible
types fail design validation. Named types must have the same identity, including
inside collections. Anonymous object bindings must have matching field presence
and Go representations. Nullable
access paths and nullable or unconstrained input fields are currently rejected;
they are not silently flattened. Collection indexing and wildcard extraction
are not supported. To check a collection, bind the collection as one typed input
and implement its all-or-nothing or partial-success policy in the evaluator.

Optional scalar sources may feed required inputs: a missing value denies
invocation before calling the evaluator. Input constraints are validated before
evaluation. Required strings do not imply nonempty strings; use `MinLength(1)`
when an empty identifier is invalid.

## Implement the generated contract

The service package contains an interface such as:

```go
type AccessAuthorizer interface {
    AuthorizeDocumentEdit(context.Context, *DocumentRef) error
}
```

Pass its implementation to `documents.NewEndpoints(service, access)`.
For services with interceptors, the interceptor implementation follows access:
`documents.NewEndpoints(service, access, interceptors)`.

Each evaluator returns nil to allow execution. Every error prevents execution;
return declared Loom errors to use the application's transport mappings. For
example, declare `Error("forbidden")` and HTTP `Response("forbidden",
StatusForbidden)`, or use `AuthErrorResponses()`. gRPC and JSON-RPC retain their
own error response mappings. Keep policy-backend failures distinguishable from
access denials. Neither failure may fall through to the service.

Missing evaluators, including typed nil implementations, panic when endpoints
are constructed. Generated examples leave an explicit evaluator placeholder
that must be supplied before startup.

Evaluators must be read-only and safe to repeat within a request. Resolve the
principal from the context populated by the existing security hooks. Resolve
resource ownership and parent relationships from trusted current facts; a
payload's tenant or parent identifier is a claim, not proof of membership.

## Authentication and coverage

Credential security is independent. Use `Security` or `SessionSecurity` to
authenticate, and `Authorize` for additional application access checks. Token
scopes continue to restrict credential authority. An authorization declaration
alone does not authenticate a caller; it can also implement access for anonymous
callers where that is the intended policy.

`StrictAuthorization()` at API or service scope requires every method to declare
`Authorize`, `AuthorizeBy`, or `NoAccessCheck`. Existing designs remain unchanged
until they opt in. A declared requirement is mandatory even without strict mode.
Multiple `Authorize` calls are conjunctive and evaluated in declaration order.
Alternative policy logic belongs in an evaluator.

`NoAccessCheck("reason")` records an intentional exemption and requires a
nonempty explanation. It preserves inherited authentication. For an anonymous
health probe, declare both `NoSecurity()` and `NoAccessCheck("Public health
probe")`. An exemption cannot be combined with access requirements or cases.

## Exhaustive variants

Use an existing string enum or union to select access requirements:

```go
AuthorizeBy("action", func() {
    AuthorizationCase("read", func() {
        NoAccessCheck("Returns the caller's own profile")
    })
    AuthorizationCase("write", func() {
        Authorize(EditDocument, func() {
            Bind("id", "document_id")
        })
    })
})
```

Every declared enum value or union branch must have exactly one case. Added,
missing, duplicate, and unknown cases fail validation. Unknown or missing
runtime selector values fail before dispatch. Cases cannot nest or mix with
method-level requirements.

For `AuthorizeBy("value", ...)` over a union, case names are the authored branch
names, independent of custom wire tags. Within that case, `Bind("id",
"value.document_id")` reads the active branch's field. An empty selector path
selects a root union. An empty binding path selects the whole payload, or its
selected root branch.

## Invocation boundary

Generated endpoints validate typed payloads, execute existing security hooks,
and evaluate access before invoking application interceptors or services.
`Endpoints.Use` preserves this ordering. Protected per-method constructors also
accept optional endpoint middleware after their mandatory dependencies.

A middleware short circuit still requires access. When middleware continues,
the inner protected endpoint checks again before service execution. This catches
target changes, invalid payload mutations, and changed permissions. It also means
authentication and authorization may run more than once. Checks are synchronous
and cancellation of the original request prevents subsequent service execution.
Middleware must preserve the supplied context or derive a child from it; dropping
that context fails closed. Middleware factories are constructed once.

HTTP, gRPC, and JSON-RPC servers use the shared generated endpoints. In-process
adapters must call those endpoints as well. Loom-MCP and agent-tool adapters
that call raw service implementations must be explicitly migrated to the
protected endpoints; this feature does not automatically wrap those calls.
Direct Go service calls, replacing exported endpoint fields, and middleware
installed outside the protected boundary remain application-owned escape hatches.

Access checks apply to the initial invocation, including the initial payload of
a streaming operation. They do not authorize individual stream messages or
promise continuing access after a grant is revoked. Applications own per-message
checks, subscription cancellation, authorized list queries and pagination,
response redaction, and transaction-time checks. An invocation check does not
make a database write atomic with a permission lookup.

## Static manifest

`gen/<service>/authorization.json` records classified methods, requirement names,
input type names, bindings, cases, exemptions, and the invocation phase. Its
versioned output is deterministic and contains no caller identities, dynamic
capabilities, credentials, or policy decisions. The design fingerprint includes
authorization declarations, so changed requirements require regeneration.

Use the manifest for documentation and coverage inspection. A client must not
treat a static requirement as evidence of the current caller's access.
