# Distinct HTTP media contracts

Use `MediaType(media, branch)` when one HTTP body has different representations.
The service value is an ordinary typed union. HTTP chooses the branch from the
request or response `Content-Type` and encodes only that branch's value. It does
not send the union's JSON discriminator envelope.

```go
var Message = Type("Message", func() {
    OneOf("body", func() {
        Attribute("document", Int)
        Attribute("text", String)
    })
    Required("body")
})

var _ = Service("messages", func() {
    Method("exchange", func() {
        Payload(Message)
        Result(Message)
        HTTP(func() {
            POST("/messages")
            Body("body")
            MediaType("application/json", "document")
            MediaType("text/plain", "text")
            Response(StatusOK, func() {
                Body("body")
                MediaType("application/json", "document")
                MediaType("text/plain", "text")
            })
        })
    })
})
```

Regenerate the service and transports. Callers and service implementations select
a generated union constructor, such as `NewBodyText("hello")`. The generated
client publishes `text/plain`; the server decodes the text branch, validates it,
and passes the typed union to the service. A returned text branch produces a
text response. `Accept` does not select or change the returned branch.

Each branch must have exactly one media mapping. Equal schemas can have separate
branches with separate examples. Examples on an enclosing payload or result are
projected to their selected branch; examples authored directly on a branch take
precedence. Ordinary union JSON serialization and single-media endpoints keep
their existing behavior.

## Ranges and parameters

Incoming selection prefers an exact type, then `type/*`, then `*/*`. Within one
type specificity, more matching authored parameters take precedence. Additional
incoming parameters are allowed. Equally specific overlapping declarations are
rejected. A parameter authored with an empty value still requires that parameter.
Media type names and charset values are case insensitive.

A mapped `Content-Type` must be a non-nullable string. A range requires this
field on the payload or result envelope:

```go
Header("content_type:Content-Type")
MediaType("application/*", "binary")
```

The caller or service must supply a concrete content type for that branch. It
must select the same branch under the incoming matching rules. An exact mapping
supplies its own content type when no mapped value is supplied. A mapped value
that would select another branch fails before encoding. OpenAPI describes this
selection through its content map, without an ignored `Content-Type` parameter.

Missing, malformed, wildcard, and unmatched incoming content types are rejected.
An optional absent request body needs no content type. Empty text, binary, and form
bodies remain supplied values when their content type is present. An empty form
selects an object with all optional fields absent. JSON empty
input represents absence; JSON null follows the selected branch's null contract.

## Codecs and encoding

The generated transports call the configured encoder or decoder factory after
selection and pass only the selected typed branch. Built-in factories support
JSON (including `+json`), text, octet streams, URL-encoded forms, and multipart.
XML, GOB, and other concrete media types require an application codec factory;
they do not silently fall back to JSON. A range itself does not select a codec.

A custom error formatter replaces the declared error body and uses the configured
encoder directly, as it does for single-media errors.

Text requires a non-nullable string or byte branch. Octet streams require a
non-nullable byte branch. Form variants use non-nullable object or string-map
roots; multipart variants require non-nullable objects. Recursive untagged
alternatives must satisfy the same root and captured-field codec rules.
Named array fields apply content encoding to each item: scalar items default to
text, while object and nested-array items default to JSON. A typeless union of
array alternatives with explicit JSON content encodes the whole union value.
Imported designs retain that distinction through normalization and regeneration.
Any explicit `style`, `explode`, or `allowReserved` setting, including `false`,
selects parameter-style serialization and suppresses `contentType`. Multipart
records each part's concrete content type. Per-part media ranges and lists are rejected because
there is no per-part concrete-type selector. Use the existing `openapi:encoding:<field>:contentType`,
`style`, `explode`, and `allowReserved` metadata on that branch to describe an
equivalent encoding; incompatible declarations fail validation. Custom part
headers and incompatible field serialization remain outside the import subset.

OpenAPI export preserves field names, explicit MIME labels, and defaults that
belong to individual root alternatives. OpenAPI has one Encoding Object per
physical field name. If alternatives require incompatible authored encodings for
the same name, export reports that field rather than silently choosing a branch.


Do not combine `MediaType` with static `ContentType`, documentation-only body
mappings, raw-body flags, endpoint-wide `FormRequest` or `MultipartRequest`, or
streaming/JSON-RPC framing. Forms belong to their individual media branches.
The enclosing union cannot be nullable; individual JSON branches may be nullable.

## OpenAPI import

The importer preserves each request and response content entry as a distinct
named branch, including entries with equal schemas. Media examples and field
encoding stay on their own branch. Ranges add a mapped content-type field so
application code can select a concrete representation. Re-import and regenerate
to replace the former shared-schema raw request body with typed variants.

Form media branches accept object and typed map roots, including nested untagged
alternatives. Each map entry uses its value schema's default form-field codec;
adding another media representation or a `oneOf` wrapper does not change this
admission rule. Dynamic keys retain their exact decoded names, including the
empty name; arrays use their captured item codec. Entries that have no wire
spelling fail encoding instead of disappearing. Values without a default field
codec remain invalid.

Multipart media branches and untagged multipart roots require structured objects.
An ordinary single-media multipart map retains its existing custom whole-body
codec. A singleton `oneOf` that normalizes to that same map has the same contract;
a genuine untagged alternative cannot use the custom codec.
