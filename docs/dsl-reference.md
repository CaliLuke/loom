---
title: DSL Reference
weight: 2
description: "Complete reference for Loom's design language - data modeling, services, methods, HTTP/gRPC mapping, and security."
llm_optimized: true
aliases:
---

Loom's Domain Specific Language (DSL) is the cornerstone of the design-first approach. This reference covers all aspects of the DSL, from basic type definitions to complex transport mappings and security schemes.

## Data Modeling

Loom provides a powerful type system for modeling your domain with precision. From simple primitives to complex nested structures, the DSL offers a natural way to express data relationships, constraints, and validation rules.

### Primitive Types

Loom provides these built-in primitive types:

```go
Boolean  // JSON boolean
Int      // Signed integer
Int32    // Signed 32-bit integer 
Int64    // Signed 64-bit integer
UInt     // Unsigned integer
UInt32   // Unsigned 32-bit integer
UInt64   // Unsigned 64-bit integer
Float32  // 32-bit floating number
Float64  // 64-bit floating number
String   // JSON string
Bytes    // Binary data
Any      // Arbitrary JSON value
```

### Type Definition

The `Type` DSL function is the primary way to define structured data types:

```go
var Person = Type("Person", func() {
    Title("Person Record")
    Description("A person")
    
    // Basic attribute
    Attribute("name", String)
    
    // Attribute with validation
    Attribute("age", Int32, func() {
        Minimum(0)
        Maximum(120)
    })
    
    // Required fields
    Required("name", "age")
})
```

Use `Title(...)` inside a `Type` or `Attribute` block to set that JSON Schema
node's OpenAPI `title`. The title is independent of the Loom type or field name
and is preserved on component schemas, properties, array items, and request or
response body schemas.

Design files conventionally dot-import `github.com/CaliLuke/loom/dsl`.
Top-level Go identifiers cannot reuse exported DSL names. Examples include
`Fault`, `Error`, `Result`, and `Type`. Go reports a redeclaration through the
dot-import. Give the Go variable an application-specific name. The declaration
string can still use the public contract name.

Each object needs distinct generated Go field names. For example, `foo_bar`
and `fooBar` both become `FooBar`, so declaring both in one object is rejected.
Rename an attribute or set `Meta("struct:field:name", "OtherFooBar")` on one
of them. The override changes its Go field name without changing its design
name or transport mapping. This check includes fields inherited with `Extend`.
Error types also reserve `Error` and `LoomErrorName` for generated methods,
and `LoomErrorRemedy` when the error declares a `Remedy`. Use the same field
metadata override to keep a wire field such as `error` under a different Go name.

### Presence and Nullability

Requiredness and nullability are separate parts of an attribute contract.
`Required` controls whether a property must be present. `Nullable` controls
whether a present property may contain JSON `null`:

```go
var Patch = Type("Patch", func() {
    Attribute("nickname", String) // absent or a string; null is rejected
    Attribute("bio", String, func() {
        Nullable() // absent, null, or a string
        Example("clear", Null())
    })
    Attribute("status", String, func() {
        Nullable() // null or a string; absence is rejected by Required
    })
    Required("status")
})
```

Generated service types represent nullable attributes as `loom.Nullable[T]`.
Its zero value is absent. Construct the other states with
`loom.NullValue[T]()` and `loom.NullableValue(value)`, and inspect them with
`Present`, `IsNull`, and `Value`. A required nullable attribute still uses the
wrapper so that generated validation can reject absence while accepting null.

`Nullable()` also admits null when combined with `Enum`: the enum restricts
concrete values, such as `"open"` and `"closed"`. Ordinary record validation and
untagged union matching use this same rule, including inherited nullability.
Upgrade Loom and regenerate services and transports to adopt the matching fix.

| Contract | Absent JSON property | JSON `null` | Concrete JSON value | Generated service form |
|---|---|---|---|---|
| Required, non-null | Rejected | Rejected | Accepted | Native `T` |
| Required, nullable | Rejected | Accepted | Accepted | `loom.Nullable[T]` |
| Optional, non-null | Accepted | Rejected | Accepted | Loom-selected pointer, slice, or map form |
| Optional, nullable | Accepted | Accepted | Accepted | `loom.Nullable[T]` |

Generated JSON transport types use `loom.Optional[T]` internally when an
optional non-null property needs an explicit absent-or-value decoding state.
Service implementations do not use that transport-only wrapper.

`Example(Null())` authors an explicit null example. `Null()` is not a default:
defaults must be concrete values and apply only when an input property is
absent. An explicit null never activates a default.

`Nullable` also works on array elements and map values. Map keys cannot be
nullable. JSON HTTP bodies and JSON-RPC preserve all four combinations of
requiredness and nullability. String-encoded HTTP parameters, headers, cookies,
forms, multipart bodies, and gRPC messages reject nullable shapes during design
validation because those transports cannot preserve the same JSON presence
contract. An unconstrained `Any` value remains the gRPC exception.

An HTTP WebSocket `StreamingPayload` cannot have a nullable root: a JSON `null`
frame signals end of client input. Put nullable values in fields of a non-null
payload object instead. Nullable fields, array elements, and map values inside
a stream message remain supported. This restriction does not apply to JSON-RPC
envelopes or HTTP WebSocket responses, which use different stream framing.

OpenAPI 3.1 and 3.2 render nullable schemas as a value-or-null union. The
OpenAPI importer maps those unions, and OpenAPI 3.0 `nullable: true`, back to
the same `Nullable()` DSL contract.

### Complex Types

#### OneOf Unions

`OneOf` defines tagged union values. It can be used as a named attribute inside
an object DSL or as a data type for payloads, results, and attributes.

A primitive branch with `Meta("struct:field:type", "pkg.Type", "import/path")`
uses that exact Go type, preserving its methods. Regenerate after adopting this
correction: synthetic branch types previously emitted for these fields are
removed, and constructors accept the authored type directly. Distinct branch
tags may use the same custom type.

Within one union, branch names and effective discriminator values must be unique.
Mapped name suffixes do not create distinct branch identities. Loom reports
conflicts during design validation, before generation, for both tagged and
untagged unions.

Different objects can reuse the same block and branch names. Each branch keeps
its declared type; Loom adds stable suffixes to generated Go type names when
needed. These suffixes do not change the JSON discriminator values.

Named union branches may refer to types declared later. Recursion must pass
through an object field. A union branch cycle without an object boundary, such
as `Type("Tree", OneOf(Leaf, "Tree"))`, is rejected during design validation,
even if it passes through an array or map. To represent a recursive tree, define
a node object with a union field that refers to the node or a leaf.

```go
var Created = Type("Created", func() {
    Field(1, "id", String)
    Required("id")
})

var Deleted = Type("Deleted", func() {
    Field(1, "id", String)
    Field(2, "reason", String)
    Required("id")
})

var Event = Type("Event", func() {
    OneOf("payload", func() {
        Attribute("created", Created)
        Attribute("deleted", Deleted)
    })
})

Method("next", func() {
    Result(OneOf(Created, Deleted))
})
```

Loom encodes unions as objects with discriminator and value fields. The default
field names are stable, and you can override them with metadata:

```go
Attribute("event", OneOf(Created, Deleted), func() {
    Meta("oneof:typename", "RealtimeEvent")
    Meta("oneof:type:field", "type")
    Meta("oneof:value:field", "payload")
})
```

When a union is used as an optional object attribute, generated Go types expose
it as a pointer. Leaving the pointer nil omits the JSON property; setting it
still requires a valid discriminator and matching branch value. Required union
attributes retain their value-type API. Canonical JSON decoding always requires
the `value` property. An explicit `"value": null` is accepted only when the
selected branch allows null through `Nullable()` (including a named type's
inherited declaration) or `Any`; other branches reject it. This also applies
to custom discriminator and value property names. Regenerate service and
transport packages to receive this decoding behavior.

Generated conversions preserve nullable branch presence across service and
transport types, including named object branches. Both inherited `Nullable()`
and `Nullable()` on the branch occurrence use this contract. Regenerate code
to receive the corrected nullable-object conversion helpers.

Nullable named union branches also retain their presence contract in projected
result types. Generated declarations and validation helpers use concrete Go
identifiers; fields and branches retain their `loom.Nullable[T]` storage.
Regenerate service, views and transport packages to receive the corrected
nullable declaration and helper names.

Generated validators reject a selected nil object or nested-union pointer with
a validation error. A selected nil slice or map is an empty collection: length
and element constraints still apply. For a constructed nullable branch,
`loom.NullValue[T]()` passes presence validation, while an absent nullable carrier
does not. Regenerate transport and service code to receive these checks.

For OpenAPI 3.1 and 3.2, Loom renders supported union schemas as `oneOf` references to
generated branch-envelope components and includes an OpenAPI discriminator with
stable tag-to-component mappings. Custom `oneof:type:field` and
`oneof:value:field` metadata changes the discriminator field and branch payload
field used in that OpenAPI contract.

For gRPC, named maps and arrays can be union branches, as can another union.
Loom generates a message wrapper around the map, repeated field, or nested
`oneof`. Declare a map branch with `Type("Index", MapOf(String, Int))`, or use
`Field` inside a `OneOf` block to name an inline map automatically. A selected
map branch remains selected even when its map is nil or empty; protobuf does
not distinguish nil maps from empty maps. Map key and value restrictions still
apply: keys must be booleans, strings, or integers, and a union cannot be a map
value or array element unless it is wrapped in an object with a `Field`.

Use `Untagged()` when a JSON contract encodes a named object or array
branch directly:

```go
var Items = Type("Items", ArrayOf(Item))
var Page = Type("Page", func() {
    Attribute("items", ArrayOf(Item))
    Attribute("total", Int)
    Required("total")
})

Method("lookup", func() {
    Result(OneOf(Items, Page), func() {
        Untagged()
    })
})
```

Generated Go keeps the same sum-type constructors, `Kind`, and `As...`
accessors. JSON contains the selected array or object without an envelope.
Both encoding and decoding require exactly one schema match and one valid Go
candidate, with the same branch identity. Overlapping schemas are rejected even
if numeric widths let Go decode only one candidate. Encoding also rejects values
whose JSON would be ambiguous; constructors select a branch but do not guarantee
that its value is valid on the wire. Failed decoding leaves the destination unchanged.

For viewed results, generated branch validators use the projected field storage:
required and defaulted scalar fields are checked for absence before their values
are validated. Regenerate affected clients and servers to pick up this correction.
Selected-view validation remains separate from full union-branch matching.

Branches must be concrete named objects or arrays. Their recursive contents may
be primitives, named objects, and arrays of these shapes. Inline object leaves,
maps, nested unions, opaque custom codecs, and schema encoding overrides are
rejected throughout the branch. Matching preserves exact JSON names, required
membership before defaults, nullability, closed-object policy, and effective
value constraints. A selected nil slice encodes as `[]`; a nil object branch is
invalid. Multiple array branches that accept `[]` make that value ambiguous.
Declare nullability on the enclosing union occurrence, not on a branch.

OpenAPI renders branch component references directly under `oneOf` and omits
the discriminator. This encoding is for JSON bodies; string-encoded parameters,
headers, cookies, forms, and multipart bodies do not support it. Protobuf keeps
its existing explicitly numbered `oneof` alternatives and collection wrappers.
Regenerate consumers to adopt these checks; previously emitted ambiguous objects
and nested-map branches are no longer accepted.

#### Result View Requiredness

Result views inherit canonical field requiredness by default. A view may
override included fields in either direction with `ViewRequired` and
`ViewOptional`:

```go
var Report = ResultType("application/vnd.report", func() {
    Attributes(func() {
        Attribute("data", String)
        Attribute("warning", String)
        Required("data")
    })
    View("partial", func() {
        Attribute("data")
        Attribute("warning")
        ViewOptional("data")
        ViewRequired("warning")
    })
})
```

Overrides may reference only fields selected by that view. Declaring the same
field both required and optional is a design error. Named views selected for
nested result fields carry their own overrides recursively.

Method-specific customization such as `Result(Report, func() { Required("warning") })`
also applies to the fields included in each view. Explicit `ViewRequired` and
`ViewOptional` overrides still take precedence. The original result type and
other methods using it retain their own requirements.

Selecting a view also preserves a result type's intrinsic `Nullable()` contract,
including nested results and collection elements. Declare intrinsic result
nullability inside `Attributes`; field-local `Nullable()` remains local to that
field. Method-local `Nullable()` inside `Result` applies only to that method's
result, including a result collection. View selection does not make an ordinary
shared result type nullable.
Generated views preserve absent, explicit null, and concrete nullable values.
Required nullable fields reject absence and accept explicit null; only concrete
values run the selected child view. Excluded fields are not validated. Nullable
collection elements accept explicit null but reject constructed absent wrappers.
Regenerate service, views, and transport packages to obtain these carrier checks.

#### Arrays

Arrays define ordered collections with optional validation:

```go
var Names = ArrayOf(String, func() {
    MinLength(1)
    MaxLength(10)
})

var Team = Type("Team", func() {
    Attribute("members", ArrayOf(Person))
})
```

Typed array elements and map values are non-null by default; unconstrained
`Any` retains its JSON domain, including null. JSON `null` in a
non-null element is rejected during decoding, before result conversion or view
validation. A known field in the decoded wire type must satisfy this rule even
if the selected view excludes it, just as it must have the right JSON kind.
Fields absent from that wire type follow the decoder's unknown-field policy.
Missing fields are still checked against the selected view.

HTTP clients return `decoding_error` with the underlying `json.SemanticError`
and JSON pointer (for example, `/members/2`). WebSocket receivers preserve the
decoder error. HTTP requests return 400 `decode_payload` with the safe detail
`invalid request body`, and do not invoke the service. This replaces the earlier
`invalid_field_type` collection validation error; successful wire values are
unchanged. Use `Nullable()` when null is part of the element contract:

```go
var OptionalNames = ArrayOf(String, func() {
    Nullable()
})
```

Array requiredness is separate: `Required("members")` controls whether the
array field must be present, while element nullability controls its members.

#### Maps

Maps provide key-value associations with type safety:

```go
var Config = MapOf(String, Int32, func() {
    Key(func() {
        Pattern("^[a-z]+$")
    })
    Elem(func() {
        Minimum(0)
    })
})
```

Map keys must have comparable generated Go storage. Native `Bytes`, arrays and
maps, including named aliases, are rejected during design validation. Scalar
keys, named scalar keys, object-pointer keys and `Any` keys retain their existing
representations. An authored custom Go key type must itself support comparison;
Loom does not inspect external type definitions. Transport codecs may restrict
keys further, including gRPC's scalar-key rules. Correct an invalid key design
and regenerate; do not edit the generated map declaration.

### Type Composition

#### Reference

Use `Reference` to inherit attribute definitions from another type:

```go
var Employee = Type("Employee", func() {
    Reference(Person)
    Attribute("name")  // Inherits from Person
    Attribute("age")   // Inherits from Person
    
    Attribute("employeeID", String, func() {
        Format(FormatUUID)
    })
})
```

`Reference` copies the selected object field templates. It does not make the
referencing object a derived named type. The copied fields retain their own
declared constraints.

#### Extend

`Extend` creates a new type that automatically inherits all attributes:

```go
var Manager = Type("Manager", func() {
    Extend(Employee)
    Attribute("reports", ArrayOf(Employee))
})
```

Using `Extend` inside a named `Payload`, `Result`, `StreamingPayload`, or
`StreamingResult` customization creates a method-specific type. It inherits the
base fields and their requiredness. Other uses of the original named type keep
their original fields and requirements.
Automatic result views include the inherited fields. Explicit views keep their
authored field selection.

```go
Payload(Employee, func() {
    Extend(AuditFields)
})
```

Redeclaring an inherited field refines its contract. The order of `Extend` and
an explicitly typed `Attribute` or `Field` does not change the result. All
inherited constraints remain in force: minimum 0 plus a local maximum 100
accepts 0 through 100, and an inherited required field stays required. Patterns,
formats, collection-element constraints and enum predicates also conjoin.
Incompatible types, conflicting field numbers or wire/codec identities, and
statically detectable empty numeric, length or enum intersections are design
errors. General regular-expression satisfiability is not checked.

A named type can refine an inherited named type only through explicit ancestry.
Two unrelated named types are incompatible even when both represent strings.
Omitting `Nullable()` inherits presence. Adding it cannot widen an inherited
non-null contract. A nullable named descendant of a non-null type is likewise
rejected because its named representation cannot express the non-null
intersection. Ordinary nullable inheritance remains supported.

With multiple bases, equal defaults coalesce. Different inherited defaults need
an explicit local `Default`, including when the chosen value equals one base's
default. The chosen default must satisfy every inherited and local constraint.
Local descriptions, documentation and examples take precedence; conflicting peer
documentation without a local choice is diagnosed. Metadata that controls field
identity, codecs or security roles must preserve inherited guarantees. Custom
metadata retains its ordered append/removal operations; ambiguous inherited
custom values require an explicit removal and replacement.

JSON tag compatibility uses the complete effective tag, including omission
options, using the canonical service field storage after inherited defaults and
requiredness are resolved.
For an optional ordinary string, a full `struct:tag:json` value of `"one"`
conflicts with `struct:tag:json:name` of `"one"`: the latter renders
`"one,omitzero"`. A full `"one,omitzero"` and name-only `"one"` are
compatible. If the field is required, including through an inherited
`Required`, or has a materialized scalar default, full `"one"` and name-only
`"one"` are compatible instead; authored `omitzero` would discard native zeros.
Nullable fields retain Loom's normal `omitzero` normalization. The same check
applies between bases and between a local redeclaration and its inherited field.
Previously accepted conflicting tags must be made consistent before regeneration.

Inheritance diamonds are supported. Recursive object members are supported, but
cycles in `Extend` ancestry are rejected. Composition owns inherited fields, so
method customizations cannot mutate a shared base or another method. Automatic
views and transport validation use the complete composed shape.

Regenerate code after upgrading. Designs that relied on the former last-base
field overwrite must express compatible refinements or resolve their conflicts.
The existing named-alias policy for empty domains is unchanged; the structural
`Extend` checks above are a separate admission boundary.

### Validation Rules

Generated validation follows every present occurrence in a finite recursive
value. Optional fields are validated when present; a nullable field's concrete
value is validated while an allowed null remains valid. These rules also apply
through named types, arrays and maps. Field declaration order does not limit
validation depth.

Nested named validators retain the caller's diagnostic path. For example, an
invalid value two nodes down a request tree reports `body.next.next.value`;
array and map paths retain the existing `[*]` and `[key]` notation. Transport
error classification and safe public error messages are unchanged. Regenerate
service and transport code after upgrading to adopt complete recursive
validation and corrected nested diagnostic paths.

Length-validation diagnostics report the field, actual length and allowed
bound without including the rejected value. Regenerate service and transport
code after upgrading. Handwritten calls to `loom.InvalidLengthError` now pass
`(name, actualLength, bound, minimum)`; the rejected-value argument is removed.


#### String Validations
- `Pattern(regex)` - Validates against a regular expression
- `MinLength(n)` - Minimum string length
- `MaxLength(n)` - Maximum string length
- `Format(format)` - Validates against predefined formats

`Pattern` uses Go regular expressions at runtime. OpenAPI export accepts a
portable subset for ECMA-262 matching without flags: BMP literals and positive
classes excluding surrogate ranges, ASCII `\d` and `\w`, word boundaries,
grouping, alternatives, and quantifiers. It translates `\A`/`\z` to `^`/`$`,
`\s` to Go's ASCII whitespace set, and `\a` to Unicode-escaped BEL.
Export renders the parsed pattern; equivalent canonical spelling can differ.

Export fails with a diagnostic for wildcards, non-BMP or surrogate character
sets (including ordinary negated classes and `\S`, `\D`, `\W`), inline flags,
named captures, POSIX classes, Unicode properties, quoted literals, octal and
braced hex escapes. For example, `Pattern("^[a-z]+$")` and
``Pattern(`\Afoo\z`)`` export faithfully; `Pattern("^.$")` is rejected
because Go and ECMA engines differ on newline and Unicode handling. Regenerate
OpenAPI after upgrading. Change a rejected pattern only when the replacement
expresses the intended application contract; runtime validation is unchanged.

#### Numeric Validations
- `Minimum(n)` - Minimum value (inclusive)
- `Maximum(n)` - Maximum value (inclusive)
- `ExclusiveMinimum(n)` - Minimum value (exclusive)
- `ExclusiveMaximum(n)` - Maximum value (exclusive)

Numeric bounds preserve exact values, including builtin Go `uint` arguments.
Integers retain all their digits; strings
such as `Minimum("9007199254740993")` and `Maximum("0.1")` express exact
mathematical integers and decimals. Finite hexadecimal strings such as
`Minimum("0x1.4p0")` express the same language-neutral number as `Minimum("1.25")`.
OpenAPI emits numeric tokens, never strings containing numbers. Integer attributes
require whole-number bounds. Nonfinite bounds and strings that overflow Float64
admission are rejected; underflowing strings remain exact nonzero bounds.

Use decimal strings for decimal contracts shared with other languages. A Go
floating-point argument retains its actual binary value: `Minimum(0.1)` is
slightly greater than `Minimum("0.1")`. Float32 and Float64 also have different
sets of representable values. Generated validators compare the decoded native
value against the exact bound. JSON Schema compares the raw JSON number;
decoding can round it, so those judgments are distinct. For example, the Float64
value `0.1` satisfies `Minimum(0.1)`, but its shortest JSON encoding `0.1` falls
below that exact binary bound. Loom rejects a default or enum member that cannot be projected into its required
generated schema artifact; it does not widen the schema to make that encoding
fit. A default must satisfy the final effective contract. Inherited enum clauses
remain predicates, so their constants are checked at their declaring occurrence
and can remain in the schema after a later refinement excludes them. Opaque
custom encoders retain their external codec boundary; known builtin values are
still checked. Optional examples retain their existing omission and partial-value
contracts.

This also applies to object enums in HTTP bodies. If a base object declares
`Enum(map[string]any{"n": 1}, map[string]any{"n": 2})` and a derived field adds
`Minimum(2)`, only `n=2` is accepted. Implicit bodies, explicit bodies and named
aliases preserve the original enum declaration. A field mapping such as
`"n:wire_n"` changes its wire spelling, not its membership. Regenerate transports
and OpenAPI after upgrading to obtain the corrected prepared-body contract.

Authored numeric values must fit their declared type. For example, `Int32`
rejects `Default(uint32(2147483648))`, and unsigned types reject `Enum(-1, 2)`.
Defaults and enum members are checked recursively through aliases and collections.
Float values use the declared destination precision: Float32 permits rounding
and underflow to zero but rejects overflow and nonfinite inputs. The concrete
Go source-type rules still apply; equal numeric values do not make named Go
scalars or otherwise unsupported source types eligible. Primitive example
compatibility uses the same checks; optional object examples retain their partial
shape contract. A redundant bound such as `Maximum(1e12)` on Int32 stays valid,
because a bound describes a predicate rather than a stored Int32 value.

Regenerate clients, servers and OpenAPI after upgrading: large bounds no longer
round through Float64, and exclusive bounds use the nearest accepted value of
the declared native type. Service implementations still own values returned at
runtime, including their chosen numeric carrier and serialization precision.

#### Defaults During Conversion

Defaults fill absence, not an explicitly supplied zero. Native Go scalar fields
have no absence bit: `false`, `0`, and `""` are supplied even when a Go struct
literal does not spell the field. Generated HTTP, JSON-RPC and gRPC conversions
preserve them. Optional pointers and presence wrappers retain absence; nil bytes,
slices and maps can represent absence, while nonnil empty values are supplied.
Explicit nullable null remains null. Required input is checked before defaults;
a default does not satisfy a missing required field.

Regenerate clients and servers to adopt this correction. Applications that relied
on zero substitution must set their intended native value explicitly. This does
not change transport omission tags or the presence supported by collection wire
formats; those boundaries must also retain presence for a full round trip.

#### Object and Collection Defaults

Object defaults are checked against the complete declared contract and generated
in the service's native representation, including objects inside arrays and maps.
Within an authored default, missing nested members use their own defaults when
the target applies defaults; supplied members retain their values. Every authored
member needs declared native storage. An unknown default member fails generation
with its member path, even when incoming wire objects permit unknown members.
Regenerate affected clients and servers after upgrading. Existing rules for when
a default applies are unchanged by this representation correction.

#### Collection Validations
- `MinLength(n)` - Minimum number of elements
- `MaxLength(n)` - Maximum number of elements

Optional arrays and maps may be omitted even when they declare `MinLength`.
In native Go values, a nil optional collection is absent; a nonnil empty
collection is supplied and must satisfy its bounds. Required fields still need
to be present and satisfy their bounds. Regenerate services and transports to
adopt the corrected optional collection validation.
Protobuf repeated/map fields retain their wire format's absence/empty collapse;
see the collection mapping rules in the [gRPC guide](grpc-guide.md).

For `Bytes`, these bounds count decoded bytes. Built-in JSON represents the
value as padded base64, and the generated schema projects the decoded bounds
into constraints on that string. See [Bytes in JSON bodies](http-guide.md#bytes-in-json-bodies).

#### Object Validations
- `Required("field1", "field2")` - Required fields

#### Generic Validations
- `Enum(value1, value2)` - Restricts to enumerated values

#### Constraints on Derived Named Types

Constraints declared on a named type refine the constraints of its named base.
Numeric and length bounds are combined, and required object fields accumulate
by field identity. Loom rejects contradictory bounds before generation, whether
they are declared together, inherited from a named type, or combined by `Extend`.
For example, a base `Minimum(10)` and derived `Maximum(5)` are invalid. Equal
inclusive endpoints are valid; equal endpoints with either side exclusive are
invalid. Length bounds must be nonnegative at every declaration, even if a
stronger derived bound would hide a negative ancestor bound. Zero lengths are
valid. Existing designs with contradictory inherited bounds must correct their
constraints before regeneration.

Admission checks endpoint order and length eligibility. It does not decide
whether a mathematically nonempty numeric interval contains a value representable
by a particular machine type, or whether combined patterns have a solution.

Pattern and format rules are also conjunctive. Each named layer adds a rule;
none replaces an ancestor rule. For example, a base `Pattern("^a")` and a
derived `Pattern("b$")` admit `"ab"` and reject both `"xb"` and `"ax"`.
Distinct formats must all accept the value. Repeating the same pattern or format
does not add a duplicate runtime or schema check.

A derived `Enum` must be equal to or narrower than the effective ancestor
enum. Every derived member must belong to that ancestor under the declared
type's value rules. Adding an outside member, including a partial overlap,
fails design validation at the attribute path.

If a derived type declares a local default, it replaces the inherited default
and must satisfy the complete refined contract or design validation fails. When
no local default is declared, the inherited default remains effective and must
satisfy every derived refinement.

```go
base := Type("Base", String, func() {
    Pattern("^a")
    Enum("ab", "ax")
})
derived := Type("Derived", base, func() {
    Pattern("b$")
})
```

This creates named ancestry. The inherited enum and derived pattern are
conjunctive, so `derived` accepts `"ab"` and rejects `"ax"`. If `derived`
explicitly declares an enum, every member must belong to the ancestor enum and
must satisfy the complete derived contract.

Combined example:

```go
var UserProfile = Type("UserProfile", func() {
    Attribute("username", String, func() {
        Pattern("^[a-z0-9]+$")
        MinLength(3)
        MaxLength(50)
    })
    
    Attribute("email", String, func() {
        Format(FormatEmail)
    })
    
    Attribute("age", Int32, func() {
        Minimum(18)
        ExclusiveMaximum(150)
    })
    
    Attribute("tags", ArrayOf(String, func() { 
        Enum("tag1", "tag2", "tag3") 
    }), func() {
        MinLength(1)
        MaxLength(10)
    })

    Required("username", "email", "age")
})
```

### Built-in Formats

Loom includes predefined formats for common data patterns:

| Format | Description |
|--------|-------------|
| `FormatBase64` | Standard padded Base64 text without whitespace; alternate pad bits allowed |
| `FormatDate` | RFC3339 date values |
| `FormatDateTime` | RFC3339 date time values |
| `FormatUUID` | RFC4122 UUID values |
| `FormatEmail` | RFC5322 email addresses |
| `FormatHostname` | ASCII Internet hostnames, with an optional terminal root dot |
| `FormatIPv4` | RFC2373 IPv4 address values |
| `FormatIPv6` | RFC2373 IPv6 address values |
| `FormatIP` | RFC2373 IPv4 or IPv6 address values |
| `FormatURI` | Absolute RFC3986 URI values with a scheme |
| `FormatURIReference` | RFC3986 URI-reference values, including relative paths |
| `FormatMAC` | IEEE 802 MAC-48/EUI-48/EUI-64 addresses |
| `FormatCIDR` | RFC4632/RFC4291 CIDR notation |
| `FormatRegexp` | RE2 regular expression syntax |
| `FormatJSON` | JSON text |
| `FormatRFC1123` | RFC1123 date time values |

`FormatHostname` validates ASCII hostname syntax without DNS lookup or IDNA
conversion. Each nonempty label contains at most 63 bytes: letters, digits and
interior hyphens. Text has at most 253 bytes excluding an optional terminal root
dot. These limits follow [RFC 1035](https://www.rfc-editor.org/rfc/rfc1035#section-3.1);
leading digits follow [RFC 1123](https://www.rfc-editor.org/rfc/rfc1123#section-2.1).

Both `example.com` and `example.com.` are accepted: the terminal dot explicitly
marks the root and does not create an empty host label. Rejecting all terminal
dots would exclude that standard spelling. `a`, `1`, `a-b.example` and ASCII
Punycode spellings are accepted; `.`, `a..b`, `a-.b`, `a_b`, raw Unicode and
labels or names exceeding the limits are rejected. Validation preserves the
original spelling and does not check IDNA meaning or whether a host exists.

### Attribute vs Field DSL

Use `Attribute` for HTTP-only types. Use `Field` when you need gRPC support (includes field number tag):

```go
// HTTP only
var Person = Type("Person", func() {
    Attribute("name", String)
    Attribute("age", Int32)
})

// With gRPC support
var Person = Type("Person", func() {
    Field(1, "name", String)
    Field(2, "age", Int32)
})
```

Use literal integer `Field` tags. For each standalone type, payload, or result,
start at `1` and increment by `1` inside that definition. When a definition uses
`Extend`, start fields introduced by the extending definition at `100` and
increment by `1`. Do not carry counters across unrelated definitions or hide
field tags behind variables/helper calls.

Fields selected into gRPC messages must use numbers from `1` through `536870911`,
excluding `19000` through `19999`. Numbers are unique within each message;
decimal spellings such as `"01"` and `1` identify the same number. Invalid numbers
fail design evaluation before code generation. A constructor `OneOf` derives
consecutive branch numbers from its field tag; explicit branch tags override
those slots. Only emitted branch numbers participate in validation and message
identity, so an unused container seed does not change an explicitly tagged union.
Regenerate after correcting an invalid design; Loom does not assign replacement
numbers automatically.

### Examples

Provide sample values for documentation:

```go
var User = Type("User", func() {
    Attribute("name", String, func() {
        Example("John Doe")
    })
    
    Attribute("email", String, func() {
        Example("work", "john@work.com")
        Example("personal", "john@gmail.com")
        Format(FormatEmail)
    })
})

var Address = Type("Address", func() {
    Attribute("street", String)
    Attribute("city", String)
    Required("street", "city")
    
    Example("Home Address", func() {
        Description("Example of a residential address")
        Value(Val{
            "street": "123 Main St",
            "city": "Boston",
        })
    })
})
```

HTTP and JSON-RPC CLI flags decoded as JSON use the JSON representation of the
selected example. For example, authored text `"hi"` for a `Bytes` member appears
as base64 `"aGk="` inside its JSON body; the selected union branch is preserved.
Plain text flags keep their existing text representation. Incomplete, ambiguous
or target-unrepresentable examples have no CLI hint; invalid authored values
still fail generation. A missing required-flag example also omits the sample
invocation, while the command remains available. An explicit JSON `null` example
is a value, not a missing hint.

When examples are omitted, Loom auto-generates them by default using a random seed.

To generate deterministic examples, use the `Randomizer` function to ensure that the examples are consistent and predictable:

```go
var _ = API("exampleAPI", func() {
    Randomizer(expr.NewDeterministicRandomizer())
})
```

### OpenAPI Metadata

Loom emits OpenAPI 3.2.0 by default. There is one DSL parser, one shared IR,
and one renderer. Set `Meta("openapi:version", "3.1")` at API scope when a
consumer requires OpenAPI 3.1.1; the renderer omits only members whose minimum
version is 3.2 and continues emitting the surrounding compatible contract. Both
targets use the canonical `gen/http/openapi.json` and
`gen/http/openapi.yaml` paths. Both files are emitted by default. Set
`Meta("openapi:output", "json")` or `Meta("openapi:output", "yaml")` at API
scope to emit only one format; generation removes a stale unselected sibling.
JSON output is deterministically ordered, uses two-space indentation, and ends
with a newline.

The most commonly used metadata keys are:

| Key | Scope | Purpose |
|-----|-------|---------|
| `openapi:generate` | API, server, host, service, method, file server, attribute | Disable OpenAPI emission for a scope with `"false"` |
| `openapi:example` | API, attribute | At API scope, omit synthesized examples with `"false"`; at attribute scope, suppress that attribute's examples |
| `openapi:version` | API | Select `"3.1"` compatibility output; 3.2 is the default |
| `openapi:output` | API | Select `"json"`, `"yaml"`, or `"both"`; both is the default |
| `openapi:self` | API | Set the OpenAPI 3.2 `$self` document URI |
| `openapi:operationId` | API, service, method | Customize generated operation IDs; `{routeIndex}` counts emitted method paths, including service base-path expansion, while static IDs remain literal and must be unique |
| `openapi:summary` | Method | Set operation summary |
| `openapi:tag:<name>` | API, HTTP service, or method | Define or assign a tag whose name has no colon |
| `openapi:tag-id:<base64url-name>` | API, HTTP service, or method | Define or assign a tag whose name contains a colon |
| `openapi:extension:<x-name>` | API, service, method, route, param, response, security | Add OpenAPI extension JSON. At API scope it lands on the `info` object |
| `openapi:document:extension:<x-name>` | API | Add OpenAPI extension JSON to the document root |
| `openapi:schema:extension:<x-name>` | Type, attribute | Add OpenAPI extension JSON to that schema |
| `openapi:parameter:extension:<x-name>` | Attribute | Add OpenAPI extension JSON to that parameter |
| `openapi:requestBody:extension:<x-name>` | Payload/body attribute | Add OpenAPI extension JSON to that request body |
| `openapi:mediaType:extension:<x-name>` | Body attribute, response | Add OpenAPI extension JSON to that media type |
| `openapi:response:extension:<x-name>` | Response | Add OpenAPI extension JSON to that response |
| `openapi:header:extension:<x-name>` | Response header attribute | Add OpenAPI extension JSON to that header |
| `openapi:closed-objects` | API | Close object schemas by default |
| `openapi:readOnly` / `openapi:writeOnly` | Attribute | Mark request/response-only fields |
| `openapi:deprecated` | Attribute | Mark a schema field deprecated |
| `openapi:contentEncoding` / `openapi:contentMediaType` | Attribute | Emit JSON Schema content metadata |
| `openapi:typename` | Type, payload, result | Override generated component schema names |
| `openapi:component:requestBody` | Payload/body types | Hoist and name reusable request bodies |
| `openapi:component:parameter` | Attribute | Hoist and name reusable parameters |
| `openapi:allowEmptyValue` | Query parameter attribute | Override the generated OpenAPI `allowEmptyValue` flag; import uses `false` to preserve an authored query parameter |
| `openapi:component:response` | HTTP response | Hoist and name reusable responses |
| `openapi:component:example` | Example | Hoist and name reusable examples |

Tag names without colons use `openapi:tag:<name>`. For a name containing a
colon, encode its UTF-8 bytes with unpadded URL-safe Base64 and use
`openapi:tag-id:<base64url-name>`. Annotation keys append fields such as
`:desc`, `:parent`, or `:extension:x-name`; identity keys take no value and
annotation keys take exactly one value.

### Vet Metadata

Applications can opt into missing HTTP service mount analysis with API metadata:

```go
API("inventory", func() {
    Meta("loom:vet:http-entrypoint", "./cmd/api", "./cmd/worker")
})
```

Each value is a Go package pattern relative to the design module. `loom vet`
checks consumer-owned files in those packages for typed generated server
`Mount` calls. Calls inside conditional branches count. Add each package that
owns mounts. Omit the metadata for library-only modules. A deliberately
unhosted service can use
`Meta("loom:vet:ignore", "service-not-mounted")` at service scope.

Synthesized examples are deterministic and isolated by stable schema and
occurrence identity. The same design and generation options therefore produce
byte-identical examples, and an unrelated schema change does not shift them.
To omit only synthesized examples, set `Meta("openapi:example", "false")` at
API scope. Explicit `Example(...)` values remain in both `openapi.json` and
`openapi.yaml`.

Schema examples use JSON representations, including base64 strings for `Bytes`,
and omit incomplete object examples. Untagged union branches preserve this
encoding for byte fields, including fields authored as text and nested
collections. This applies to body schemas, nested parameter schemas,
response-header schemas, and `x-loom-async` message schemas
in both output formats. Explicit null examples remain null for nullable schemas.
Synthesized tagged-union examples retain a discriminator/value envelope at
every nesting level, including unions that reuse the same branch shapes in
objects, arrays, and maps. Each envelope uses its configured field names and
branch tags. Authored `Example(...)` values keep their existing interpretation.

#### OpenAPI 3.2 features

All OpenAPI 3.2 additions use the same design and renderer:

| Feature | DSL or metadata |
|---------|-----------------|
| Tag summary, hierarchy, and kind | `openapi:tag:<name>:summary`, `:parent`, and `:kind` on the API or HTTP service |
| QUERY and extension methods | `QUERY("/path")` and `Route("purge", "/path")`; extension methods must be non-empty RFC 9110 tokens and are normalized to uppercase; CONNECT is emitted through `additionalOperations` |
| Whole query string | `MapParams(...)` emits `in: querystring` with form content in 3.2 |
| Sequential and streaming media | SSE, `application/jsonl`, and `application/json-seq` use `itemSchema` automatically; use `openapi:itemSchema` for other sequential media such as `multipart/mixed` |
| Reusable media types | `openapi:component:mediaType` on a body or result type |
| Multipart and nested encoding | `openapi:encoding:<property>:<field>`, `openapi:prefixEncoding:<index>:<field>`, and `openapi:itemEncoding:<field>` |
| Structured examples | `openapi:example:dataValue` and `openapi:example:serializedValue` inside `Example(...)` |
| OAuth device flow | `DeviceAuthorizationFlow(deviceURL, tokenURL, refreshURL)` |
| Security metadata | `openapi:oauth2MetadataUrl`, `openapi:deprecated`, and `openapi:security:uri` on a security scheme |
| Optional discriminator and fallback | `openapi:discriminator:optional` and `openapi:discriminator:defaultMapping` on a `OneOf` |
| XML node mapping | `openapi:xml:name`, `:namespace`, `:prefix`, and `:nodeType`; valid node types are `element`, `attribute`, `text`, `cdata`, and `none` |
| Response metadata | `openapi:summary`, `openapi:description:omit`, and `openapi:description:errorName` inside `Response(...)` |
| Reserved characters and cookie serialization | `openapi:allowReserved` on parameters or headers and `openapi:style` set to `cookie` on cookies |
| Server names | The DSL `Server(...)` name is emitted as the OpenAPI Server Object `name` |

Optional discriminator metadata requires an explicit `defaultMapping`. Export
rejects missing or blank fallbacks, malformed URI references, unknown local
components, non-boolean optional values, and discriminator metadata on non-unions.
A bare fallback names a schema component; `#/components/schemas/Name` is also
accepted. Use `./schemas.json#/$defs/Fallback` or an absolute URI for an external
schema; export preserves that reference without fetching it. In the 3.1 target,
Loom requires the discriminator property and omits `defaultMapping`. Metadata is
validated before selecting either target, so changing the target cannot hide an
invalid authored combination.

Encoding metadata fields may be `contentType`, `style`, `explode`, or
`allowReserved`. Nest another `encoding:<property>`,
`prefixEncoding:<index>`, or `itemEncoding` segment before the field to describe
nested content, for example
`openapi:itemEncoding:encoding:id:style`. At each media or nested encoding
object, choose either named `encoding` entries or positional `prefixEncoding`
and `itemEncoding` entries. Prefix and item encodings may coexist. Named and
positional entries in the same object cause a generation error for both OpenAPI
targets. Put independent encoding choices in separate child objects.

Loom prefixes error-response descriptions with the error name by default. Set
`Meta("openapi:description:errorName", "false")` inside an HTTP error
`Response(...)` only when preserving an externally authored response
description verbatim, such as a contract imported with `loom import openapi`.

```go
var DeviceAuth = OAuth2Security("device", func() {
    DeviceAuthorizationFlow(deviceURL, tokenURL, refreshURL)
    Meta("openapi:oauth2MetadataUrl", metadataURL)
})

Method("search", func() {
    HTTP(func() {
        QUERY("/search")
    })
})

Method("purge", func() {
    HTTP(func() {
        Route("purge", "/cache")
    })
})
```

OpenAPI 3.2 deprecates the XML Object's older `attribute: true` and
`wrapped: true` forms. Use `openapi:xml:nodeType` instead.

With the 3.1 compatibility target, QUERY and extension-method operations are
omitted. New fields such as `$self`, tag hierarchy, server names,
`itemSchema`, reusable media types, nested encodings, security additions,
response summaries, XML `nodeType`, and discriminator fallbacks are skipped.
Shared content remains: sequential media falls back to `schema`, reusable media
is inlined, structured examples fall back to `value`, and `MapParams(...)`
remains an ordinary query parameter.

Generated OpenAPI component schemas are deduplicated by structural identity.
When two generated request or response body shapes are equivalent, Loom emits a
single schema component and reuses it by `$ref`. Explicit names from
`openapi:typename` are treated as public component names: equivalent schemas may
reuse the name, but different schemas claiming the same explicit name fail
generation instead of receiving hash-suffixed public names. When the same
declaration has different complete schemas for actual transport encodings,
Loom allocates representation-specific references; an actual JSON representation
keeps the canonical public name. This does not permit different authored shapes
to claim one explicit name.

Required payload, result and error fields need a runtime reconstruction source.
When selecting a field or listing a subset with `Body`, map the other required
fields to appropriate transport bindings. Existing authentication, envelope ID
and response-tag reconstruction count; a default or documentation-only body
does not supply an otherwise unmapped field. Views retain their own projected
requiredness. A custom required `ErrorName` must appear in its body or an
explicit header/cookie mapping, or use `ErrorNameFromStatus()` inside its HTTP
error `Response`. The latter reconstructs the declared error name from a unique
status and omits the field from the implicit body and the automatic `Loom-Error`
header. It rejects shared statuses and competing explicit field mappings.
The ordinary `Loom-Error` dispatch header does not assign that field without a
mapping, and required mapped headers never fall back to the status.
Previously accepted designs that silently lost required values now
fail generation; add the missing mapping before regenerating.

A response `Body(func() { ... })` that lists attributes of the result or error
type declares a shape of its own. It inherits the type's name only as a
fallback, like the implicit body: an identical schema reuses the type's
component, and a different schema gets a hash-suffixed name. Set
`Meta("openapi:typename", "...")` inside that `Body` to give it a public name.
Hash-suffixed fallback names are generated collision identities, not a public
naming API. If downstream SDKs or tooling depend on a component name, set the
corresponding `openapi:typename` or `openapi:component:*` metadata explicitly.

Loom now uses its own versioned fingerprint for hash-suffixed fallback names.
This causes one intentional transition. Existing hash-suffixed schemas, request
bodies, responses, and SDK type names can change after regeneration. Regenerate
and review these artifacts once when you upgrade. Set explicit component-name
metadata before the upgrade when a public name must not change.

OpenAPI security requirement values are empty arrays for HTTP bearer, API key,
basic, and cookie security schemes. OAuth2 is the only scheme kind that
publishes required scopes in those arrays; JWT and bearer scopes remain part of
generated auth data without being advertised as OpenAPI OAuth-style scopes.

JWT credentials use an HTTP bearer scheme when read from the Authorization
header. A JWT mapped to another header, a query parameter, or a cookie uses an
API-key scheme with that exact location. Reusing one scheme at different
locations produces separate security components, and each operation references
its own binding. API-level defaults use the Authorization header or the declared
session-cookie name; endpoint mappings override those defaults. Component names
can change when a scheme acquires another credential location.

HTTP request and response mappings must identify one source per wire location.
Header names compare case-insensitively; query and cookie names compare exactly.
Mapping different attributes to the same name is a design error, including
inherited mappings and implicit credential headers. Repeated bindings of the same
source with the same encoding are allowed. Basic and Bearer may share
Authorization as separate security alternatives because their framing distinguishes
them; their payload requiredness and validation constraints still apply.

For example, `Security(JWT, KeyAuth)` with separate token and key fields needs a
distinct mapping such as `Header("key:X-API-Key")`; leaving both implicit, or using
them as unframed alternatives in Authorization, is rejected. Basic OR Bearer with
optional credential fields remains supported, including an explicit Authorization
mapping for the token. A narrower security declaration excludes only its selected
credentials from the request body: overridden inherited credentials remain ordinary
payload fields unless explicitly mapped. Correct colliding mappings and regenerate
clients, servers and OpenAPI when upgrading.

Credential constraints apply to the extracted credential value. HTTP headers and
gRPC metadata named `Authorization` (case-insensitive) carry JWT/OAuth values as
`Bearer <token>`; generated clients add that framing to the service value, and
servers remove it before validating the token. Scheme names are case-insensitive.
Raw tokens, other schemes and empty bearer credentials in Authorization are
rejected. Optional credentials may be omitted; a present malformed header is an
error. Custom headers, query parameters, cookies and other metadata carry raw
values; API keys are opaque even in Authorization.

For example, `Pattern("^[a-z]+$")` accepts `Bearer valid` in Authorization and
`valid` in a custom token header. It rejects `Bearer UPPER` and a custom-header
value `Bearer valid`. Supply `valid`, not a prefixed value, to generated clients.
This replaces permissive prefix guessing with the documented mapping contract:
framing belongs to Authorization, while custom mappings preserve authored values.

OpenAPI generation rejects path credentials and OAuth2 credentials outside the
Authorization header. These mappings cannot be represented by standard security
schemes while preserving OAuth flows and scopes. The HTTP transport still supports
them. To document such an endpoint separately, exclude it from OpenAPI with
`Meta("openapi:generate", "false")` on the method or HTTP endpoint. OAuth2 schemes
using the Authorization header retain their flows and required scopes.

When `Meta("openapi:closed-objects", "true")` is set at API scope, normal object
schemas are emitted as closed JSON Schema objects with
`additionalProperties: false`. Explicit maps remain open by using schema-valued
`additionalProperties`, and union wrapper schemas use
`unevaluatedProperties: false` so `oneOf` branches and discriminators remain
valid OpenAPI 3.1 / JSON Schema 2020-12.

Use the `ReadOnly()` and `WriteOnly()` helpers instead of spelling the metadata
directly when marking attributes:

```go
var Account = Type("Account", func() {
    Attribute("id", String, func() {
        ReadOnly()
    })
    Attribute("password", String, func() {
        WriteOnly()
    })
})
```

Use `Deprecated()` inside an HTTP endpoint expression to mark an operation
deprecated. Attribute-level `Meta("openapi:deprecated", "true")` marks a
schema field deprecated.

`OpenAPIBody(...)` describes a documentation-only HTTP response body for
manual or raw response paths, including `SkipResponseBodyEncodeDecode()`. It
uses the same shape arguments as `Body(...)`, but it only affects the OpenAPI
contract and does not add runtime response encoding.
The generated raw response handler applies a declared `ContentType(...)`
before it streams the body. `OpenAPIBody(...)` does not select or change the
runtime content type.

`OpenAPIRequestBody(schema, contentType, required, fn...)` is the request-side
counterpart for endpoints using `SkipRequestBodyEncodeDecode()`. It publishes
the explicit schema, media type, description, examples, and
`requestBody.required` value without enabling generated request parsing,
buffering, or validation. The schema may be `Bytes`, `String`, a payload
attribute name, or a named type. It is incompatible with typed `Body(...)`,
form, multipart, optional JSON body, and non-raw request modes.

`OpenAPIRequestBodyTypes(schema, contentTypes, required, fn...)` publishes the
same request schema for multiple media types. Loom does not decode these
formats. The service must inspect the content type and decode the raw stream.

Documentation-only bodies accept scalar/object unions and branches containing
nested unions. For example, `OpenAPIBody(OneOf(String, Document), Untagged)`
publishes a `oneOf` without a discriminator envelope. The equivalent request
form is `OpenAPIRequestBody(OneOf(String, Document), "application/json", true, Untagged)`.
A modifier applies only to that body occurrence, even when its argument is a
shared named type. Nested unions retain their own authored tagged or untagged
representation. Required fields, constraints, and branch identities are still
validated. Generated runtime untagged codecs retain their narrower branch rules;
a named type declared with `Untagged()` must still satisfy those rules. Apply
`Untagged` on the documentation body to describe a broader raw wire contract.
Regenerate OpenAPI artifacts to adopt this support; raw stream handling remains
application-owned.

`Link(...)` inside an HTTP `Response(...)` publishes OpenAPI response links.
Use `LinkOperation(...)` or `LinkOperationRef(...)` to identify the target
operation, `LinkParam(...)` to map link parameters from runtime expressions,
and `LinkRequestBody(...)` when the linked operation body comes from the
source response.

SSE and WebSocket endpoints keep the HTTP handshake modeled as ordinary
OpenAPI operations. Loom also publishes framework-owned streaming contract
metadata under `x-loom-async` so downstream tools can discover stream message
schemas without pretending the OpenAPI response body is a finite JSON payload.
In 3.2 output, an SSE response additionally uses `itemSchema` to describe each
parsed event envelope. The envelope's `data` string uses `contentMediaType` and
`contentSchema` to identify the decoded JSON message schema; `x-loom-async`
remains available for Loom-specific handshake and field-mapping details.

---

## API Definition

The `API` function defines global properties for your service and serves as the root of your design.

### Basic Structure

```go
var _ = API("calculator", func() {
    Title("Calculator API")
    Description("A simple calculator service")
    Version("1.0.0")
})
```

### Complete Example

```go
var _ = API("bookstore", func() {
    Title("Bookstore API")
    Description("A modern bookstore management API")
    Version("2.0.0")
    TermsOfService("https://example.com/terms")
    
    Contact(func() {
        Name("API Support")
        Email("support@example.com")
        URL("https://example.com/support")
    })
    
    License(func() {
        Name("Apache 2.0")
        URL("https://www.apache.org/licenses/LICENSE-2.0.html")
    })
    
    Docs(func() {
        Description("Comprehensive API documentation")
        URL("https://example.com/docs")
    })
})
```

`Docs()` maps to OpenAPI `externalDocs`. API docs appear at the document root, service docs appear on the service tag, and method docs appear on the operation.

### Server Configuration

Define where your API can be accessed:

```go
var _ = API("bookstore", func() {
    Server("production", func() {
        Description("Production server")
        
        Host("production", func() {
            Description("Production API endpoint")
            URI("https://{version}.api.example.com")
            URI("grpcs://{version}.grpc.example.com")
            
            Variable("version", String, "API version", func() {
                Default("v2")
                Enum("v1", "v2")
            })
        })
    })
    
    Server("development", func() {
        Host("localhost", func() {
            URI("http://localhost:8000")
            URI("grpc://localhost:8080")
        })
    })
})
```

Each OpenAPI Server Object uses the description of its `Host`. If the host has no description, Loom uses the enclosing `Server` description.

Each `Variable` description becomes the description of the matching OpenAPI Server Variable Object.

Loom generates the example commands of a server in `cmd/<name>` and its CLI support packages in `gen/<transport>/cli/<name>`. The directory name is the snake_case form of the server name, with each non-ASCII rune escaped: `calc server` and `CalcServer` become `calc_server`, and `Café` becomes `cafu00e9`. Evaluation rejects a design in which two servers get the same directory name, or names that differ only in case, because their generated files would overwrite each other.

### API-Level Errors

Define reusable errors at the API level:

```go
var _ = API("bookstore", func() {
    Error("unauthorized", ErrorResult, "Authentication failed")
    
    HTTP(func() {
        Response("unauthorized", StatusUnauthorized)
    })
    
    GRPC(func() {
        Response("unauthorized", CodeUnauthenticated)
    })
})
```

Services can reference these errors by name:

```go
var _ = Service("billing", func() {
    Error("unauthorized")  // Inherits all properties
})
```

An error must have a concrete error type. Do not use `Error("name", Empty)`.
For a bodyless HTTP error, omit the type and use `Body(Empty)` in its response:

```go
Error("payload_too_large")

HTTP(func() {
    Response("payload_too_large", StatusRequestEntityTooLarge, func() {
        Body(Empty)
    })
})
```

---

## Services and Methods

Services group related methods that provide specific functionality.

### Service DSL

```go
var _ = Service("users", func() {
    Description("User management service")
    
    Docs(func() {
        Description("Detailed documentation for the user service")
        URL("https://example.com/docs/users")
    })

    // Service-level errors
    Error("unauthorized", String, "Authentication failed")
    Error("not_found", NotFound, "Resource not found")
    
    // Metadata
    Meta("openapi:tag:Users")
    
    // Security requirements
    Security(OAuth2, func() {
        Scope("read:users")
        Scope("write:users")
    })
    
    Method("create", func() {
        // ... method definition
    })
    
    Method("list", func() {
        // ... method definition
    })
})
```

The generated packages of a service are in `gen/<name>` and the transport directories such as `gen/http/<name>`. The directory name is derived from the service name as for servers. Evaluation rejects a design in which two services get the same directory name, or names that differ only in case.

The transport packages of a service whose directory name is `cli` are in `gen/<transport>/cli/server`, `gen/<transport>/cli/client` and, for gRPC, `gen/grpc/cli/pb`. These directories are in the directory of the client CLI packages, `gen/<transport>/cli/<server>`. Evaluation rejects such a service when a server that hosts a service of the same transport has the directory name `server`, `client` or, for gRPC, `pb`. It also rejects a `cli` service exposed over HTTP when `Meta("http:generate", "server")` is set, because that mode removes `gen/http/cli` as stale client output.

### Method DSL

```go
Method("add", func() {
    Description("Add two numbers together")
    
    Payload(func() {
        Field(1, "a", Int32, "First operand")
        Field(2, "b", Int32, "Second operand")
        Required("a", "b")
    })
    
    Result(Int32)
    
    Error("overflow")
})
```

### Payload Types

```go
// Simple payload
Method("getUser", func() {
    Payload(String, "User ID")
    Result(User)
})

// Structured payload
Method("createUser", func() {
    Payload(func() {
        Field(1, "name", String, "User's full name")
        Field(2, "email", String, "Email address", func() {
            Format(FormatEmail)
        })
        Field(3, "role", String, "User role", func() {
            Enum("admin", "user", "guest")
        })
        Required("name", "email", "role")
    })
    Result(User)
})

// Reference to predefined type
Method("updateUser", func() {
    Payload(UpdateUserPayload)
    Result(User)
})
```

### Result Types

Return a nonnil object when a method succeeds with a non-nullable viewed
`ResultType`. An intrinsically nullable result or a result with method-local
`Nullable()` uses `loom.Nullable[T]`: return
`loom.NullValue[T]()` for explicit null or `loom.NullableValue(value)` for a
concrete result. Its absent zero value is not a successful result. The generated
service-to-view constructor returns `loom.Fault` for a missing result before
projecting its fields. Ordinary nil and empty result collections remain valid
empty collections. A nullable collection result must be present: use explicit
null or a concrete collection, including an empty one. Regenerate service code
to obtain these checks.

```go
// Simple result
Method("count", func() {
    Result(Int64)
})

// Structured result
Method("search", func() {
    Result(func() {
        Field(1, "items", ArrayOf(User), "Matching users")
        Field(2, "total", Int64, "Total count")
        Required("items", "total")
    })
})
```

### Streaming Methods

```go
Method("streamNumbers", func() {
    Description("Stream a sequence of numbers")
    StreamingPayload(Int32)
    StreamingResult(Int32)
})

Method("processEvents", func() {
    StreamingPayload(func() {
        Field(1, "event_type", String)
        Field(2, "data", Any)
        Required("event_type", "data")
    })
    
    Result(func() {
        Field(1, "processed", Int64)
        Field(2, "errors", Int64)
        Required("processed", "errors")
    })
})
```

---

## Streaming

Streaming enables real-time, continuous data exchange between clients and servers. Loom's streaming DSL is transport-agnostic — the same design works for both HTTP (WebSocket, Server-Sent Events) and gRPC streaming.

### Streaming Patterns

Loom supports three streaming patterns:

| Pattern | DSL | Use Case |
|---------|-----|----------|
| Server-to-Client | `StreamingResult` | Live feeds, notifications, progress updates |
| Client-to-Server | `StreamingPayload` | File uploads, event ingestion |
| Bidirectional | Both | Chat, real-time collaboration |

### StreamingPayload

Use `StreamingPayload` when clients send a stream of messages to the server:

```go
// Client streams events, server returns summary
Method("ingestEvents", func() {
    Description("Ingest a stream of analytics events")
    
    StreamingPayload(func() {
        Field(1, "event_type", String, "Type of event")
        Field(2, "timestamp", String, "ISO 8601 timestamp")
        Field(3, "properties", MapOf(String, Any), "Event properties")
        Required("event_type", "timestamp")
    })
    
    Result(func() {
        Field(1, "events_processed", Int64, "Total events ingested")
        Field(2, "errors", Int64, "Events that failed validation")
        Required("events_processed", "errors")
    })
})
```

### StreamingResult

Use `StreamingResult` when the server sends a stream of messages to the client:

```go
// Client subscribes, server streams updates
Method("subscribe", func() {
    Description("Subscribe to real-time price updates")
    
    Payload(func() {
        Field(1, "symbols", ArrayOf(String), "Stock symbols to watch")
        Required("symbols")
    })
    
    StreamingResult(func() {
        Field(1, "symbol", String, "Stock symbol")
        Field(2, "price", Float64, "Current price")
        Field(3, "change", Float64, "Price change percentage")
        Field(4, "timestamp", String, "Update timestamp")
        Required("symbol", "price", "timestamp")
    })
})
```

### Bidirectional Streaming

Combine both for full-duplex communication:

```go
// Real-time chat with structured messages
Method("chat", func() {
    Description("Bidirectional chat stream")
    
    StreamingPayload(func() {
        Field(1, "message", String, "Chat message content")
        Field(2, "room_id", String, "Target chat room")
        Field(3, "reply_to", String, "Message ID being replied to")
        Required("message", "room_id")
    })
    
    StreamingResult(func() {
        Field(1, "id", String, "Message ID")
        Field(2, "sender", String, "Sender username")
        Field(3, "message", String, "Message content")
        Field(4, "room_id", String, "Chat room")
        Field(5, "timestamp", String, "When message was sent")
        Required("id", "sender", "message", "room_id", "timestamp")
    })
})
```

### Transport Mapping

The same streaming design maps to different transports:

```go
Method("watch", func() {
    StreamingResult(Event)
    
    // HTTP: WebSocket by default, or SSE with ServerSentEvents()
    HTTP(func() {
        GET("/events/watch")
        // ServerSentEvents()  // Uncomment for SSE instead of WebSocket
    })
    
    // gRPC: Server-side streaming RPC
    GRPC(func() {
        Response(CodeOK)
    })
})
```

**See also:**
- [HTTP Guide: WebSocket Integration](http-guide.md#websocket-integration) — HTTP-specific streaming implementation
- [HTTP Guide: Server-Sent Events](http-guide.md#server-sent-events) — SSE for one-way server streaming
- `SSEProjection(eventType, view)` maps an SSE event discriminator to a
  `StreamingResult` result view. It is opt-in, requires `SSEEventType`, and
  requires at least two unique mappings.
- [gRPC Guide: Streaming Patterns](grpc-guide.md#streaming-patterns) — gRPC streaming implementation

---

## Static Files

The `Files` DSL serves static content directly from the filesystem. This is an HTTP-only feature — gRPC does not support static file serving.

### Basic Usage

```go
var _ = Service("web", func() {
    // Serve a directory
    Files("/static/{*path}", "./public/static")
    
    // Serve a specific file
    Files("/favicon.ico", "./public/favicon.ico")
    
    // Serve index.html for root
    Files("/", "./public/index.html")
})
```

### Directory Serving

The `{*path}` wildcard captures the remaining URL path:

```go
// Request: GET /assets/css/style.css
// Serves: ./static/css/style.css
Files("/assets/{*path}", "./static")
```

### Single Page Applications

For SPAs with client-side routing, serve the same HTML for all routes:

```go
var _ = Service("spa", func() {
    // API endpoints
    Method("getData", func() {
        HTTP(func() { GET("/api/data") })
    })
    
    // Static assets
    Files("/assets/{*path}", "./dist/assets")
    
    // SPA catch-all — serves index.html for all other routes
    Files("/{*path}", "./dist/index.html")
})
```

**Note**: `Files` is HTTP-only. For detailed patterns including template integration, see [HTTP Guide: Static Content](http-guide.md#static-content).

---

## Error Handling (Design Level)

Errors in Loom are defined at the design level and automatically mapped to transport-specific responses. This section covers the DSL for defining errors; for transport mapping details, see [Error Handling Guide](error-handling.md).

### Error Scopes

Errors can be defined at three levels:

| Scope | Availability | Use Case |
|-------|--------------|----------|
| API-level | All services | Common errors (unauthorized, rate limited) |
| Service-level | All methods in service | Domain errors (not found, invalid state) |
| Method-level | Single method only | Operation-specific errors |

### API-Level Errors

Define once, use everywhere:

```go
var _ = API("myapi", func() {
    // Define common errors
    Error("unauthorized", ErrorResult, "Authentication required")
    Error("rate_limited", ErrorResult, "Too many requests", func() {
        Temporary()  // Client should retry
    })
    
    // Map to transports
    HTTP(func() {
        Response("unauthorized", StatusUnauthorized)
        Response("rate_limited", StatusTooManyRequests)
    })
    
    GRPC(func() {
        Response("unauthorized", CodeUnauthenticated)
        Response("rate_limited", CodeResourceExhausted)
    })
})
```

### Service-Level Errors

Available to all methods in the service:

```go
var _ = Service("users", func() {
    // Service-wide errors
    Error("not_found", ErrorResult, "User not found")
    Error("already_exists", ErrorResult, "User already exists")
    
    HTTP(func() {
        Response("not_found", StatusNotFound)
        Response("already_exists", StatusConflict)
    })
    
    Method("get", func() {
        // Can return not_found without declaring it
    })
    
    Method("create", func() {
        // Can return already_exists without declaring it
    })
})
```

### Method-Level Errors

Specific to a single operation:

```go
Method("transfer", func() {
    Description("Transfer funds between accounts")
    
    Payload(func() {
        Field(1, "from_account", String)
        Field(2, "to_account", String)
        Field(3, "amount", Float64)
        Required("from_account", "to_account", "amount")
    })
    
    Result(TransferResult)
    
    // Method-specific errors
    Error("insufficient_funds", ErrorResult, "Account has insufficient balance")
    Error("account_locked", ErrorResult, "Account is locked for transfers")
    
    HTTP(func() {
        POST("/transfer")
        Response("insufficient_funds", StatusUnprocessableEntity)
        Response("account_locked", StatusForbidden)
    })
})
```

### Application Authorization

Use `Authorization(name, InputType)` to declare a typed application access
requirement, `Authorize(requirement, func() { Bind("input", "payload.path") })`
to apply it, and `StrictAuthorization()` at API or service scope to require
complete classification. `NoAccessCheck("reason")` records an exemption while
preserving authentication. `AuthorizeBy` and `AuthorizationCase` classify every
string enum value or union branch. See [Application Authorization](authorization.md)
for evaluator wiring, mandatory execution, static manifests, and supported
binding shapes.

### Session Security

Use `SessionAuth` to model one logical authenticated session that can arrive
through more than one transport:

```go
var AccessToken = JWTSecurity("access_token")
var BrowserSession = APIKeySecurity("browser_session")

var AppSession = SessionAuth("app_session", func() {
    BearerTransport(AccessToken, "auth")
    CookieTransport(BrowserSession, "", func() {
        CookieName("app_session")
    })
})

var _ = Service("accounts", func() {
    SessionSecurity(AppSession)

    Method("profile", func() {
        Payload(func() {
            Attribute("account_id", String)
            Required("account_id")
        })
        HTTP(func() {
            GET("/profile/{account_id}")
            AuthErrorResponses()
        })
    })
})
```

`SessionSecurity` may appear at API, service, or method scope. The nearest scope
that declares `Security`, `SessionSecurity`, or method-level `NoSecurity` supplies
the complete authentication declaration. Outer declarations are replaced, even
when they use a different authentication DSL. For example, a method with
`Security(Basic)` inside a service with `SessionSecurity(AppSession)` accepts only
Basic credentials. A method with `SessionSecurity(AppSession)` inside a service
with `Security(Basic)` accepts only that session's transports. To accept both,
declare both at the same scope. Regenerate services after changing or upgrading
these declarations; previously generated adapters may retain inherited schemes.

The transports inside a `SessionAuth` are alternatives: generated OpenAPI security uses OR
semantics, and generated payload/binding code injects the required credential
fields when the design has not already modeled them. `CookieTransport(..., "")`
keeps browser-cookie auth transport-owned: Loom emits OpenAPI cookie security,
but does not synthesize a payload field, CLI flag, or HTTP decode local. The
generated auth wrapper still calls the API-key authorizer with an empty key.
Install `loomhttp.RequestMetadataMiddleware` with `Cookie` explicitly retained
when the authorizer must resolve the browser session from request-scoped
transport metadata; do not expect the empty field name to trigger implicit
cookie decoding.

Generated endpoints isolate the context used by each alternative security
requirement. If an authorizer returns a modified context together with an
error, Loom discards that candidate context before trying the next alternative.
Schemes declared together in one `Security(A, B)` requirement retain AND
semantics and chain the context returned by each successful authorizer. Only the
context from the requirement that ultimately succeeds reaches the service
method.

Within a requirement, `Scope("read")` applies to each scheme that declares that
name. Use `ScopeFor(oauth, "read")` to require it from only one OAuth scheme,
and `RoleFor(key, "admin")` to require an application role from one non-OAuth
scheme. These calls add to shared Scope declarations and accept a scheme name
or reference already listed in Security. Unknown OAuth scopes and references to
schemes outside that requirement are rejected; role names need no inventory.
Callbacks receive their own required names in `RequiredScopes` and validate
application-owned grants with `scheme.Validate(grantedNames)`. Roles are not
extra credentials exchanged on the wire. Regenerate transports and OpenAPI;
required non-OAuth names are now emitted as OpenAPI role arrays.

OAuth flow constructors accept an optional final block of `Scope(name, description)`
declarations. A flow without a block inherits the scheme-level scopes. A supplied
block replaces those defaults; `func() {}` declares an empty inventory. Different
flows may use disjoint names or different descriptions for the same name.
Requirements may reference names available in any effective flow. The generated
OAuth callback receives each flow's inventory in `Flows`; `Scopes` lists the
available names across flows. Neither inventory grants access: the application
validates the token, its issuer and flow, and its actual granted scopes.

Each separate `Security(...)` call adds an OR alternative. `Security()` with no
scheme adds an anonymous alternative, so callers may satisfy the endpoint
without credentials while the other alternatives remain available. This maps
to an empty OpenAPI Security Requirement Object (`{}`). Use `NoSecurity()` on a
method to replace inherited requirements with an explicit empty security list
(`security: []`). Combining `NoSecurity()` with authenticated `Security(...)` or
`SessionSecurity(...)` at the same method scope is a design error, regardless of
declaration order. Use `Security()` for an anonymous OR alternative instead.

Inherited API HTTP error mappings retain the custom type declared by the API,
including mappings reused by `AuthErrorResponses()`. Return that generated
custom error type to select the inherited response. Loom generates it in each
service that uses the mapping, or in its configured `struct:pkg:path` package.
Unused API error definitions do not add types to unrelated services. A local
HTTP response mapping takes precedence over the inherited mapping.

Use `AuthErrorResponses()` in HTTP scope to add standard 401/403 error response
mappings, and use `SessionCookie(...)` in HTTP response scope when setting a
session cookie with secure defaults: `Path("/")`, `Secure`, `HttpOnly`, and
`SameSite=Lax`. To clear that cookie on logout, return an empty value from the
result field mapped to the cookie and set `CookieMaxAge(-1)`:

```go
Method("logout", func() {
    Result(func() {
        Field(1, "session_token", String)
        Required("session_token")
    })
    HTTP(func() {
        POST("/logout")
        Response(StatusNoContent, func() {
            SessionCookie("session_token:app_session", String)
            CookieMaxAge(-1)
        })
    })
})
```

A negative max age instructs browsers to delete the cookie immediately. Zero
omits the `Max-Age` attribute, while a positive value is its lifetime in
seconds.

When deployment configuration supplies response-cookie attributes, keep the
cookie mapping in the design and apply a `loomhttp.ResponseCookiePolicy` to the
generated method handler.
Runtime policy can set `Domain`, `Secure`, and `Expires`. It cannot change the
designed name, result-derived value, path, lifetime, or browser security
attributes. See the HTTP guide for complete wiring.

> **Since: `v1.8.0`.**

For plain-HTTP local development, call `CookieInsecure()` immediately
after `SessionCookie(...)` to override the secure default. Do not use that
override in production or with `__Host-`/`__Secure-` cookie names or
`SameSite=None`, which require `Secure`.

### CORS

Use `CORS` in API or service `HTTP` scope to define browser cross-origin
policy. Service-level policy overrides API-level policy for that service.

```go
API("tasks", func() {
    HTTP(func() {
        CORS(func() {
            Origin("https://app.example.com", func() {
                Methods("GET", "POST")
                Headers("Authorization", "Content-Type")
                ExposeHeaders("X-Request-Id")
                MaxAge(600)
                Credentials()
            })
            OriginRegex(`^https://preview-[^.]+\.example\.com$`)
        })
    })
})
```

Generated HTTP servers mount preflight `OPTIONS` handlers and write actual
request CORS headers. `Origin("*")` is valid only without `Credentials()`.
OpenAPI path items include the effective policy as `x-loom-cors`.
When a design also defines an `OPTIONS` endpoint on the same path, the
generated route sends requests that contain `Access-Control-Request-Method`
to preflight handling and sends ordinary `OPTIONS` requests to the authored
endpoint.

When allowed origins come from startup or deployment configuration, declare
runtime mode instead of embedding those values in generated code:

```go
HTTP(func() {
    RuntimeCORS()
})
```

Construct a validated immutable snapshot with
`loomhttp.NewRuntimeCORSPolicy(configuredPolicy)` and pass it to the generated
server constructor. Runtime mode uses the same actual-request, preflight, and
streaming behavior as static `CORS`; invalid policies fail while the snapshot
is constructed. OpenAPI emits `x-loom-cors: {runtime: true}` without exposing
deployment-specific values.

`CORS` and `RuntimeCORS()` are mutually exclusive at the same API or service
scope. WebSocket handshake origin enforcement remains a separate upgrader
policy and is not granted by either CORS mode.

### Error Properties

Mark errors with semantic properties:

```go
Error("service_unavailable", ErrorResult, func() {
    Description("Backend service is temporarily unavailable")
    Temporary()  // Client should retry
})

Error("request_timeout", ErrorResult, func() {
    Description("Request exceeded time limit")
    Timeout()    // Deadline was exceeded
})

Error("internal_error", ErrorResult, func() {
    Description("Unexpected server error")
    Fault()      // Server-side issue
})
```

### Custom Error Types

For errors needing additional context:

```go
var ValidationError = Type("ValidationError", func() {
    Field(1, "name", String, "Error name", func() {
        Meta("struct:error:name")  // Required for custom error types
    })
    Field(2, "message", String, "Error message")
    Field(3, "field", String, "Field that failed validation")
    Field(4, "value", Any, "Invalid value provided")
    Required("name", "message", "field")
})

Method("create", func() {
    Error("validation_error", ValidationError, "Input validation failed")
})
```

**See also:**
- [Error Handling Guide](error-handling.md) — Complete error handling patterns
- [HTTP Guide: Error Responses](http-guide.md#best-practices) — HTTP status code mapping
- [gRPC Guide: Error Handling](grpc-guide.md#error-handling) — gRPC status code mapping

---

## HTTP Transport Mapping

The HTTP DSL defines how service methods map to HTTP endpoints.

An endpoint may share its name with a type nested in its body. For example,
`Method("other", ...)` can accept an object or union containing `Type("Other",
...)`. Loom assigns distinct transport type names, adding a numeric suffix
when needed. This allocation preserves the authored schema names and wire
format and also applies to WebSocket message bodies.

Loom generates HTTP servers, per-service clients, and the aggregate client CLI
by default. Set `Meta("http:generate", "server")` at API scope for a
server-only application. This keeps `gen/http/<service>/server/`, generated
service packages, and OpenAPI output while omitting
`gen/http/<service>/client/` and `gen/http/cli/`. Switching from the default
`all` mode removes those stale client directories during successful generation.

### HTTP Request Components

An HTTP request has four parts that can be mapped to payload attributes:

1. **URL Path Parameters** - e.g., `/bottle/{id}`
2. **Query String Parameters**
3. **HTTP Headers**
4. **Request Body**

Mapping expressions:
- `Param` - Loads from path or query string
- `Header` - Loads from HTTP headers
- `Body` - Loads from request body

Request body mode expressions:
- `FormRequest` - Use typed `application/x-www-form-urlencoded` request encoding
- `MultipartRequest` - Use `multipart/form-data` request encoding
- `OptionalRequestBody` - Allow an omitted JSON object body or typed form object/map body
- `SkipRequestBodyEncodeDecode` - Pass the raw request body reader to the service
- `OpenAPIRequestBody` - Document a raw request stream in OpenAPI without decoding it
- `OpenAPIRequestBodyTypes` - Document one raw request schema for multiple media types
- `SkipResponseBodyEncodeDecode` - Return a raw response body reader from the service
- `FileResponse` - Return seekable file content with `http.ServeContent` range,
  conditional request, GET, and explicit HEAD semantics

### Mapping Non-Object Payloads

For primitive, array, or map payloads, values load from the first defined element:

```go
// Path parameter
Method("show", func() {
    Payload(Int)
    HTTP(func() {
        GET("/{id}")
    })
})
// GET /1 → Show(1)

// Array in path (comma-separated)
Method("delete", func() {
    Payload(ArrayOf(String))
    HTTP(func() {
        DELETE("/{ids}")
    })
})
// DELETE /a,b → Delete([]string{"a", "b"})

// Array in query string
Method("list", func() {
    Payload(ArrayOf(String))
    HTTP(func() {
        GET("")
        Param("filter")
    })
})
// GET /?filter=a&filter=b → List([]string{"a", "b"})

// Header
Method("list", func() {
    Payload(Float32)
    HTTP(func() {
        GET("")
        Header("version")
    })
})
// GET / with header version=1.0 → List(1.0)

// Map in body
Method("create", func() {
    Payload(MapOf(String, Int))
    HTTP(func() {
        POST("")
    })
})
// POST / {"a": 1, "b": 2} → Create(map[string]int{"a": 1, "b": 2})
```

### Mapping Object Payloads

For object payloads, specify where each attribute comes from:

```go
Method("create", func() {
    Payload(func() {
        Attribute("id", Int)
        Attribute("name", String)
        Attribute("age", Int)
    })
    HTTP(func() {
        POST("/{id}")
        // id from path, name and age from body
    })
})
// POST /1 {"name": "a", "age": 2} → Create(&CreatePayload{ID: 1, Name: "a", Age: 2})
```

Use `Body` to specify a non-object body:

```go
Method("rate", func() {
    Payload(func() {
        Attribute("id", Int)
        Attribute("rates", MapOf(String, Float64))
    })
    HTTP(func() {
        PUT("/{id}")
        Body("rates")  // Body is the map directly
    })
})
// PUT /1 {"a": 0.5, "b": 1.0} → Rate(&RatePayload{ID: 1, Rates: map[string]float64{...}})
```

### Element Name Mapping

Map HTTP element names to attribute names:

```go
Header("version:X-Api-Version")  // version attribute from X-Api-Version header

GET("/items/{k}")
Param("key:k")                   // key attribute from the "k" path parameter

Body(func() {
    Attribute("name:n")  // name attribute from "n" field in JSON
    Attribute("age:a")   // age attribute from "a" field in JSON
})
```

An attribute of a `Body` defined with a function names the payload or result
attribute before the colon: `Attribute("name:n")` inherits the type,
description, validations and requiredness of the `name` attribute, as
`Attribute("name")` does, and the body names the field `"n"`. As in a type,
a `Required` list in the body names the attribute `"name:n"`, and a request
body may require it only when the payload requires `name`. Inline and named
payloads inherit requiredness the same way. Only attributes listed in the body
contribute required fields; fields mapped exclusively to parameters, headers,
or cookies do not become required body properties.

`Header`, `Param`, `Cookie`, `MapParams`, `Body` and route wildcards select a
payload, result or error attribute declared with a suffix by its attribute
name: with `Attribute("tok:t", String)` in the payload, `Header("tok")` reads
the `tok` header and `Header("tok:X-Token")` the `X-Token` header. The suffix
of a type or payload attribute names only the field of a body, never a header,
parameter or cookie. `Body("data")` selects the attribute declared as
`"data:d"`, and `Attribute("name")` in a `Body` function inherits the
attribute declared as `"name:nm"` and names the body field `"name"`.

gRPC `Message`, request `Metadata`, and response `Headers` and `Trailers`
select these fields by attribute name too. With `Attribute("tok:t", String)`
in the payload, `Metadata(func() { Attribute("tok:x-token") })` sends `tok`
as `x-token` metadata and leaves it out of the protobuf message. The payload's
`t` suffix does not name the gRPC metadata. Requiredness and defaults are
inherited from the selected service attribute.

A route wildcard names the parameter, not the attribute: `Param("key:k")`
maps the `{k}` wildcard of the route to the `key` attribute, and a route that
uses `{key}` for that attribute is rejected. A mapped `Param` that no route
wildcard names is a query string parameter named `k`.

The same suffix on an attribute of a type or payload, such as
`Attribute("name:n", String)`, names the field of the HTTP and JSON-RPC
bodies: the body uses `"n"`, and the Go field and `Required("name:n")` refer
to the `name` attribute. Each field of a body object needs its own JSON name:
a design is rejected when the suffix gives a body field the JSON name of
another field of the same object, as `"a:x"` and `"b:x"` or `"a:x"` and `"x"`
do, or an empty name, `"-"`, or a name that a JSON struct tag cannot hold. A
`struct:tag:json` name takes precedence over the suffix, and an attribute that
the endpoint maps to a header, param or cookie is not in the body. The OpenAPI properties, required lists and examples,
the client CLI `--body` examples and the fields that select the branch of an
untagged `OneOf` in those bodies also use `"n"`. gRPC ignores the suffix. `ConvertTo` and
`CreateFrom` select the external Go field with `struct:field:external` or
by the attribute name; the suffix does not select it.
The service package also uses the attribute name: the JSON tag of the Go
field, the kind of a union branch and the types of the branches of a `OneOf`
block use `name`. Validation errors, including those of HTTP and JSON-RPC
bodies, name the attribute and not the wire element, as they do for fields
renamed with `struct:tag:json`.

Automatic JSON tags omit only absent optional values. A nonnil pointer to an empty
string, a nonnil empty byte slice, array, map or object pointer stays on the wire.
Native scalar values, including `false`, `0` and `""` on fields with defaults,
are supplied values. Required empty collections are valid unless a length constraint
rejects them. Defaults apply only to absence. Application-defined replacement Go
types retain their codec contract; their `IsZero` methods do not define Loom absence.
Regenerate clients and servers after upgrading to apply these semantics.

Full `struct:tag:json` metadata preserves its authored options in service and
HTTP/JSON-RPC encoder types. For example, `"items,omitzero"` preserves a supplied
empty slice and omits nil, while `"items,omitempty"` omits both. A full tag with
no omission option emits the value even when empty; a nil pointer then encodes
as null and remains subject to the design's nullability rules. Explicit presence
carriers retain their normal `omitzero` normalization, and JSON-RPC IDs retain
protocol-owned omission. Regenerate code to apply corrected full-tag handling.

A `struct:tag:json` or `struct:tag:json:name` of `"-"` omits a field from
JSON. An HTTP or JSON-RPC body cannot hold such a field when it is optional,
that is not required or given a default value, because the generated decoders
track the presence of optional fields. The design is rejected; remove the tag,
or leave the attribute out of the body with an explicit `Body` or map it to a
header, param or cookie.

### Response Selection

An HTTP method has exactly one untagged application response, used when no
`Tag` matches the result. Loom supplies an implicit default if no responses are
authored. Each additional response must select a result value with `Tag`;
different status codes alone cannot select a response. Multiple untagged
responses are rejected during design validation. Replace any formerly ignored
response with an explicit tag or remove it, then regenerate.

Tagged responses are tested in authored order; the first matching tag wins.
The untagged default is always tested last, wherever its declaration appears.
Regenerate to correct servers that previously changed priority when the default
was declared before tagged alternatives.

Designed error responses and FileResponse's protocol-owned statuses are separate
from this application-response selection rule.

### Response Headers

Declare response headers inside a `Response` block. The mapped attribute comes
from the method result; result fields not mapped to headers or cookies remain
in the response body:

```go
Method("create", func() {
    Result(func() {
        Field(1, "href", String, "Canonical account URL")
        Field(2, "name", String, "Account name")
        Required("href", "name")
    })
    HTTP(func() {
        POST("/accounts")
        Response(StatusCreated, func() {
            Header("href:Location")
        })
    })
})
```

The mapping syntax is `"result_attribute:HTTP-Header-Name"`. Use
`Header("href")` when the result attribute name is already the desired HTTP
header name. Type, description, and validation are inherited from the result
attribute unless explicitly overridden.

### Form and Multipart Request Bodies

Use `FormRequest` when the endpoint accepts form-encoded request bodies:

```go
Method("token", func() {
    Payload(Grant)
    Result(Token)
    HTTP(func() {
        POST("/token")
        FormRequest()
        Response(StatusOK)
    })
})
```

`FormRequest` supports object payloads, map payloads, and constructor union
payloads. A root `MapOf(String, T)` maps dynamic keys directly to form field
names. To combine a map body with path, query, header, cookie, or security
attributes, put it in an object payload and select the attribute with
`Body("name")`. For constructor unions, scalar branches use the canonical
discriminator/value form shape, while object branches are flattened onto
top-level form fields so OAuth-style grant payloads do not need custom request
decoders.

Use `MultipartRequest` for multipart form uploads:

```go
Method("upload", func() {
    Payload(func() {
        Attribute("project_id", String)
        Attribute("file", Bytes)
        Attribute("filename", String)
        Attribute("content_type", String)
        Required("project_id", "file")
    })
    HTTP(func() {
        POST("/projects/{project_id}/uploads")
        Param("project_id")
        MultipartRequest()
    })
})
```

`MultipartRequest` requires a payload and does not support constructor union
payloads. Supported object payloads get generated multipart decoding. If the
object contains a single `Bytes` file field, `filename` and `content_type`
string attributes can be populated from the uploaded part metadata.
Unsupported multipart shapes keep the custom decoder hook instead of generating
partial framework-owned behavior.

Use `OptionalRequestBody` for endpoints that accept either no JSON request body
or an optional object body. It also supports object and map bodies with
`FormRequest`. It cannot be combined with `MultipartRequest` or
`SkipRequestBodyEncodeDecode`. Empty-body EOF is accepted only for opted-in JSON
endpoints, and empty forms are accepted only for opted-in form endpoints.
Required-body endpoints remain strict and malformed input still returns a
decode error.

Use `SkipRequestBodyEncodeDecode` and `SkipResponseBodyEncodeDecode` for raw
HTTP body streaming with `io.Reader` values. These flags are HTTP-only and are
incompatible with gRPC and method streaming.

Use `FileResponse` for HTTP-only seekable downloads. Every declared route must
be `GET` or `HEAD`; declare `HEAD` explicitly when required. The service method
returns `*loomhttp.FileResponse`, while generated clients retain the raw
`io.ReadCloser`. File responses require one untagged 200 application response;
Loom and `http.ServeContent` own the 206, 304, 412, and 416 protocol outcomes.
When response `ContentType` is omitted, OpenAPI uses binary `*/*` while the
runtime infers MIME type from `FileResponse.Name`. Every result metadata
attribute must map to a response header or cookie; FileResponse exclusively
owns the response body.

### Custom HTTP Field Types

Use `Meta("struct:field:type", "pkg.Type", "import/path")` when a generated
struct field needs a custom Go type. For string-backed path, query, header, and
cookie fields, generated HTTP request and response decoders call `UnmarshalText`
on that custom type. CLI payload builders use the same parser. CLI flags accept raw text, preserve omitted
optional values, and apply declared defaults. Generated clients and servers format these
values with `fmt.Sprint`; implement `fmt.Stringer` to return the wire text that
`UnmarshalText` accepts. This includes types such as `uuid.UUID`.
Response headers and cookies, including mapped errors and viewed results, use
the same custom text representation. Regenerate transports after adopting this
correction; optional nil values remain absent.

If the DSL field also uses `Format(...)`, Loom skips the duplicate generated
string format check and lets the custom type's parser own that validation.
Other string constraints, such as `MinLength`, validate the original text.

Custom String fields also participate in service view validation, defaults and
gRPC conversion. Loom uses `fmt.Stringer` on the value or its address to obtain
semantic text; native and defined string storage otherwise keep their string
value. For materialization, an authored `encoding.TextUnmarshaler` takes
precedence over assignment to native or defined string storage. Concrete pointer
chains preserve this precedence; nil values cannot supply semantic text. Opaque
types without the required capability return an error when that operation is
needed. Copying an unchanged custom representation needs neither capability.
JSON codecs do not substitute for these semantic operations.

Defaults still apply only to absence. A failed custom default or text conversion
returns an error and does not publish a partial result. Regenerate service and
transport code together: constructors such as `NewRecordFromRecordView`, and
`ConvertTo` / `CreateFrom` methods, now return an error when their selected
conversion can fail. Constructors for native values or views excluding every
fallible conversion keep their existing signatures. Existing outer viewed-result
constructors propagate these errors.

---

## gRPC Transport Mapping

The gRPC DSL defines how service methods map to gRPC procedures.

### gRPC Features

1. **Message Mapping** - Define request/response structures with field numbers
2. **Status Codes** - Map results to gRPC status codes
3. **Metadata** - Configure gRPC metadata handling
4. **Protocol Buffer Package** - Use `Package` in service-level `GRPC` to set the protobuf package name

### Mixed Protocol Support

Services can support both HTTP and gRPC:

```go
Method("create", func() {
    Payload(CreatePayload)
    Result(User)
    
    HTTP(func() {
        POST("/")
        Response(StatusCreated)
    })
    
    GRPC(func() {
        Response(CodeOK)
    })
})
```

RESTful resource mapping:

```go
Service("users", func() {
    HTTP(func() {
        Path("/users")
    })
    
    Method("list", func() {
        HTTP(func() { GET("/") })           // GET /users
        GRPC(func() { Response(CodeOK) })
    })
    
    Method("show", func() {
        HTTP(func() { GET("/{id}") })       // GET /users/{id}
        GRPC(func() { Response(CodeOK) })
    })
})
```

### Composing Transport Blocks

`HTTP`, `GRPC`, and `JSONRPC` may be declared more than once in the same API,
service, or method scope. Loom evaluates repeated blocks in declaration order
as though their contents appeared in one block. This allows routing, error
mapping, and other transport concerns to stay near the declarations they
describe:

```go
Service("users", func() {
    HTTP(func() {
        Path("/users")
    })

    Error("not_found")
    HTTP(func() {
        Response("not_found", StatusNotFound)
    })
})
```

Normal rules still apply to the combined contents: additive declarations are
appended, conflicting declarations report their usual design errors, and
single-value settings retain their documented replacement behavior.

---

## JSON-RPC Transport Mapping

The `JSONRPC` DSL exposes service methods through JSON-RPC 2.0. It can appear
at API, service, and method scope. Service-level `JSONRPC` defines the shared
endpoint path; method-level `JSONRPC` opts a method into JSON-RPC and maps IDs
and errors. URL captures retain their DSL types and may use `Param("field:wire")`
mappings alongside JSON-RPC body parameters. A method whose payload is entirely
path-bound omits `params` from its request. Regenerate after upgrading to pick
up corrected path/body decoder local allocation.

JSON-RPC WebSocket authentication uses credentials explicitly selected by `Body`
in each message. Implicit Authorization headers, Basic authentication, session
cookies, and other header/query/cookie credential mappings are rejected during
design validation. A server-streaming `Payload` keeps its credentials when Loom
normalizes it to message input. If you declare a separate `StreamingPayload`, it
must retain every selected credential's name and security tag with a supported
string type. For example, a JWT `Token("token", String)` selected in `Body` must
also be declared with `Token` in the separate streaming payload; an untagged
`Attribute("token", String)` is insufficient. This applies to every authenticated
alternative, including one beside an anonymous alternative. Authentication runs
per message; it does not authorize the shared connection handshake. Correct
previously admitted unsupported designs before regenerating.

`CORS` or `RuntimeCORS()` may be declared in API-level or service-level
`JSONRPC` scope. Without a policy, generated JSON-RPC servers retain Go's
secure cross-origin rejection default. With a policy, Loom generates preflight
handling and actual-response CORS headers for HTTP, SSE, mixed, and WebSocket
JSON-RPC servers. Runtime mode requires a validated
`loomhttp.RuntimeCORSPolicy` constructor argument. Use `Origin("*")` without
`Credentials()` for an explicit allow-all static policy.

The generated server's `ServeHTTP` method is the effective entry point for
actual requests: it includes the configured CORS or secure-default origin
policy and middleware installed with `Server.Use`. Custom mounts and transport
extensions should route actual requests through `ServeHTTP`; generated
`OPTIONS` routes terminate in the route-local preflight handler.
`Server.Use` may run before or after `Mount`, but all middleware configuration
must finish before serving requests. Concurrent reconfiguration is unsupported.
Regenerate to make already-mounted routes honor later setup-time `Use` calls.

For JSON-RPC SSE methods, use
`SSENotificationMethod("notifications/progress")` inside
`ServerSentEvents` when intermediate events must use a protocol-defined
notification. The default is `<service>/stream.event`, never the original
request method.

```go
var _ = Service("calc", func() {
    JSONRPC(func() {
        POST("/jsonrpc")
    })

    Method("add", func() {
        Payload(func() {
            ID("id", String)
            Field(1, "a", Int)
            Field(2, "b", Int)
            Required("a", "b")
        })
        Result(func() {
            ID("id", String)
            Field(1, "sum", Int)
            Required("sum")
        })
        JSONRPC(func() {})
    })
})
```

JSON-RPC supports regular HTTP request/response, batch requests, WebSocket
streaming, and SSE streaming. For SSE, use `ServerSentEvents` inside the
`JSONRPC` expression:

```go
Method("watch", func() {
    Payload(func() {
        ID("id", String)
        Field(1, "last_event_id", String)
    })
    StreamingResult(Event)
    JSONRPC(func() {
        ServerSentEvents(func() {
            SSERequestID("last_event_id")
            SSEEventID("id")
        })
    })
})
```

When a streamed result has views, dynamic WebSocket and SSE method streams
expose `SetView(name)`. The selection applies to subsequent sends. A method
with a fixed `View(...)` uses that view and has no setter. Connection-level
send methods use the fixed or default view of the method. A WebSocket method
that receives a streaming payload and returns one result also renders that
result with its fixed or default view.

Loom adds a `"loom_view"` string to each viewed streaming notification or
success response. This is a Loom protocol extension. The `params` or `result`
member still contains the projected body directly, including collections.
For example, a notification can contain:

```json
{"jsonrpc":"2.0","method":"feed/stream.event","params":{"id":"n1"},"loom_view":"tiny"}
```

Generated clients decode and validate each body using its own marker, then
return the canonical service type. They reject non-string markers (including `null`), unknown view names, missing
fields required by that view, and markers that conflict with a fixed view.
An absent or empty marker falls back to the fixed view or `default`, for older
servers; it cannot identify an older server's dynamically selected view.
Regenerate both peers to use dynamic view selection reliably. Non-viewed
messages and error responses omit the marker. Unary HTTP responses continue
to use the `loom-view` header.

Use JSON-RPC `Response` blocks to map service errors to JSON-RPC error codes:

```go
Error("rate_limited")
JSONRPC(func() {
    Response("rate_limited", func() {
        Code(-32029)
    })
})
```

See the [JSON-RPC Guide](../jsonrpc/README.md) for transport behavior and
streaming patterns.

---

## Security

Loom provides DSL constructs for authentication and authorization.

Credential attributes honor `Meta("struct:field:name", "AuthToken")` for their
generated Go fields and authentication selectors. This does not rename their
transport headers or parameters. Regenerate clients and servers after upgrading
to pick up corrected credential selectors and named-string conversions.
Use `String` or a named string type for credentials. Payload fields retain named
types, while authentication callbacks receive strings. Non-string, nullable and
custom Go type overrides on credentials are rejected at design validation.
Basic-auth CLI flags accept plain string values even for named credential types.
HTTP and JSON-RPC CLIs expose one direct flag per payload attribute, including a
credential explicitly mapped to query parameters, headers, cookies or a path.
That value supplies Basic authentication and every authored mapping of the same
attribute. Repeated mappings do not require duplicate flags. Body flags remain
separate inputs; direct field flags retain their existing precedence over body
values when constructing the payload. Regenerate the CLI after upgrading.

Basic-auth header presence is independent of its two components. The client
sends the header if either component is required or present; an absent optional
counterpart becomes an empty component. When both are optional, a missing or
invalid Basic header leaves their pointers nil. A valid header supplies both
components, including empty values. If either is required, a missing or invalid
header fails decoding. The server validates supplied username and password values
against their complete DSL constraints, including named-type and occurrence
constraints, before authentication runs. This applies to HTTP and JSON-RPC POST;
gRPC metadata uses the same attribute validation rules. A supplied empty component
is a value to validate, not an absent credential. Regenerate clients and servers
to adopt this behavior.

An optional Authorization bearer credential can coexist with other `Security`
alternatives. If another requirement does not use that credential, a header
using another scheme (such as `Basic`) leaves the bearer field absent instead
of failing bearer decoding. This applies to HTTP, JSON-RPC POST and gRPC
metadata. The authentication callbacks still decide whether an alternative
succeeds; failed alternatives do not pass their modified context to the next.
Explicit payload `Required` constraints remain unconditional. Malformed Bearer
framing and authored credential validation failures still reject the request,
even when another alternative is available. A bearer-only requirement retains
strict bearer decoding. Regenerate clients and servers to adopt this correction.



### Security Schemes

#### JWT (JSON Web Token)

```go
var JWTAuth = JWTSecurity("jwt", func() {
    Description("JWT-based authentication")
    Scope("api:read", "Read-only access")
    Scope("api:write", "Read and write access")
})
```

#### API Keys

```go
var APIKeyAuth = APIKeySecurity("api_key", func() {
    Description("API key-based authorization")
})
```

An HTTP endpoint reads the API key from a header, a query string parameter or
a cookie. A design that reads it from a path parameter, a `Param` that a route
wildcard names, is rejected, because an OpenAPI API key security scheme cannot
name a path parameter.

#### Basic Authentication

```go
var BasicAuth = BasicAuthSecurity("basic", func() {
    Description("Username/password authentication")
    Scope("api:read", "Read-only access")
})
```

#### OAuth2

```go
var OAuth2Auth = OAuth2Security("oauth2", func() {
    AuthorizationCodeFlow(
        "http://auth.example.com/authorize",
        "http://auth.example.com/token",
        "http://auth.example.com/refresh",
    )
    Scope("api:read", "Read-only access")
    Scope("api:write", "Read and write access")
})
```

### Applying Security

Credential declarations (`Token`, `AccessToken`, `APIKey`, `Username`,
`Password`, and their numbered `Field` forms) use attribute names without a
mapping suffix. For example, use `Token("token", String)` and map it with
`Header("token:Authorization")` in the HTTP DSL. A declaration such as
`Token("token:wire", String)` fails design validation. Ordinary attributes
can still use mapping suffixes.

#### Method Level

```go
Method("secure_endpoint", func() {
    Security(JWTAuth, func() {
        Scope("api:read")
    })
    
    Payload(func() {
        TokenField(1, "token", String)
        Required("token")
    })
    
    HTTP(func() {
        GET("/secure")
        Header("token:Authorization")
    })
})
```

#### Multiple Schemes

```go
Method("doubly_secure", func() {
    Security(JWTAuth, APIKeyAuth, func() {
        Scope("api:write")
    })
    
    Payload(func() {
        TokenField(1, "token", String)
        APIKeyField(2, "api_key", "key", String)
        Required("token", "key")
    })
    
    HTTP(func() {
        POST("/secure")
        Param("key:k")
    })
})
```

### Implementing Security

Loom generates an `Authorizer` interface that your service must implement:

```go
type Authorizer interface {
    BasicAuth(ctx context.Context, user, pass string, schema *security.BasicScheme) (context.Context, error)
    JWTAuth(ctx context.Context, token string, schema *security.JWTScheme) (context.Context, error)
    APIKeyAuth(ctx context.Context, key string, schema *security.APIKeyScheme) (context.Context, error)
    OAuth2Auth(ctx context.Context, token string, schema *security.OAuth2Scheme) (context.Context, error)
}
```

Example JWT implementation:

```go
func (s *svc) JWTAuth(ctx context.Context, token string, scheme *security.JWTScheme) (context.Context, error) {
    claims := make(jwt.MapClaims)
    
    _, err := jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) { 
        return Key, nil 
    })
    if err != nil {
        return ctx, ErrInvalidToken
    }

    // Validate required scopes
    scopes, ok := claims["scopes"].([]any)
    if !ok {
        return ctx, ErrInvalidTokenScopes
    }
    scopesInToken := make([]string, len(scopes))
    for _, scp := range scopes {
        scopesInToken = append(scopesInToken, scp.(string))
    }
    if err := scheme.Validate(scopesInToken); err != nil {
        return ctx, ErrInvalidScopes
    }

    return contextWithAuthInfo(ctx, authInfo{claims: claims}), nil
}
```

Example Basic Auth implementation:

```go
func (s *svc) BasicAuth(ctx context.Context, user, pass string, scheme *security.BasicScheme) (context.Context, error) {
    if user != "loom" || pass != "rocks" {
        return ctx, ErrUnauthorized
    }
    return contextWithAuthInfo(ctx, authInfo{user: user}), nil
}
```

---

## Best Practices

### Type Organization
- Group related types together
- Use meaningful field names and descriptions
- Follow consistent naming conventions
- Keep types focused and cohesive

### Validation Strategy
- Add appropriate constraints for each field
- Define required fields explicitly
- Use format validators for standard formats
- Consider domain-specific validation rules

### Service Design
- Group related functionality into services
- Keep service scope focused and cohesive
- Use clear, descriptive service names
- Document service purpose and usage

### Method Design
- Use clear, action-oriented method names
- Provide detailed descriptions
- Define appropriate error responses
- Consider validation requirements

### HTTP Design
- Use consistent URL patterns
- Follow RESTful conventions
- Choose appropriate status codes
- Handle errors consistently

### Security
- Use appropriate security schemes for your use case
- Implement proper token validation
- Define clear scope hierarchies
- Use HTTPS in production
