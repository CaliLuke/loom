# Negotiated response cache identity

`http.negotiatedResponseEncoder` selects a representation using `Accept`.
The previous implementation varied the content type and bytes without publishing
`Vary: Accept`. A fresh shared cache entry could therefore replace another
request's JSON response with HTML. With attacker-controlled string data and a
cacheable endpoint, that creates a potential script-execution path when a
browser receives the cached HTML. The missing cache dimension is a framework
defect; exploitability also depends on application data and cache deployment.

[RFC 9110 section 12.5.5](https://www.rfc-editor.org/rfc/rfc9110.html#section-12.5.5)
defines response variance, and
[RFC 9111 section 4.1](https://www.rfc-editor.org/rfc/rfc9111.html#section-4.1)
requires caches to match the nominated request fields before reuse.

## Model and decision

The finite model contains two sequential requests for the same cacheable GET
resource. Each has an HTML, JSON or default/fallback Accept value. The origin
either negotiates or has a fixed JSON representation. The first response is
fresh and stored; the cache respects Vary, and an incompatible variant triggers
an origin request. There is no concurrent mutation or cache revalidation.

`legacy.cfg` reproduces the missing-header implementation and must fail
`NoHTMLSubstitution`: an HTML entry is reused for a JSON/default request.
`checked.cfg` publishes Accept variance for every negotiated response and passes
both `CorrectRepresentation` and `NoHTMLSubstitution` over 54 distinct states.
The fixed-JSON cases retain cache reuse without an unnecessary Accept dimension.

Chosen repair: publish the dimension at the shared negotiator, even for missing
or unsupported Accept. Adding it only for HTML leaves other representations with
incorrect cache keys. Disabling caching globally would discard valid same-variant
reuse; removing supported media would change the content-negotiation contract.

The model assumes a compliant cache and abstracts MIME parsing, header token
merging, application authorization, expiry, browser behavior, and body encoding.
It does not prove that arbitrary cache deployments or HTML applications are safe.

## Implementation evidence

Before the runtime change at `91dd76227`,
`TestResponseEncoderVariesNegotiatedRepresentation` failed nine missing-Vary
cases. The existing generated transport-policy specimen also failed all four
new `TestDefaultGeneratedNegotiationVariesAccept` cases. Both exercise response
encoding and headers, including JSON fallback for a browser's multi-value Accept.
The direct matrix retains explicit-type and existing-Vary controls; the generated
specimen preserves actual response status, content type and body bytes.

The fix calls the existing `writeVary` helper before encoder selection. The
helper preserves other dimensions and recognizes case-insensitive Accept tokens
across field lines and comma-separated lists. Explicit response ContentType and
service-selected media variants retain their existing non-negotiating paths.
Applications need the updated runtime, not regenerated source.

## Run explicitly

Use TLA+ tools 1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
These checks are opt-in design tools, never build, hook or CI prerequisites.
From this directory, use separate temporary state directories:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/loom-vary-legacy -config legacy.cfg ResponseNegotiation.tla
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/loom-vary-checked -config checked.cfg ResponseNegotiation.tla
```

Legacy must exit 12 with the named invariant violation; checked must exit 0.
Remove task-owned checker output when no active validation or review needs it.
From the repository root, implementation checks are:

```sh
go test ./http -run TestResponseEncoderVariesNegotiatedRepresentation
LOOM_DIR="$PWD" go test ./http/codegen -run TestGeneratedHTTPTransportPoliciesComposeAtRuntime
```
