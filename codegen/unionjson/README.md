# Untagged union matching and generation

`codegen/unionjson` projects evaluated union contracts into generated typed JSON
adapters. Service, HTTP and JSON-RPC generators use the same analysis and runtime
matching rules. Text, form and multipart adapters use their captured location
codecs and the same independent schema/decoder selection core.

This contract extends [#350](https://github.com/CaliLuke/loom/issues/350) with the
branch grammar and location support in
[#1014](https://github.com/CaliLuke/loom/issues/1014). Parameter serialization is
owned by [#968](https://github.com/CaliLuke/loom/issues/968); discriminator import
is separate work in [#1005](https://github.com/CaliLuke/loom/issues/1005).

## Authoring and admission

An untagged branch may be a primitive, `Any`, scalar alias, named object, array,
string-keyed map or another untagged union. Nullable alternatives retain an
explicit null-capable representation. Nested fields and collection members use
primitives, named objects, untagged unions, arrays and string-keyed maps of these
shapes. Typed additional object members use the same recursive grammar where the
location codec supports them.

Use named types or declare branch-local constraints with the constructor DSL:

```go
Payload(OneOf(func() {
    Untagged()
    Attribute("text", String, func() {
        MinLength(1)
    })
    Attribute("count", Int)
}))
```

The constructor retains branch identities and union naming/envelope/closure
metadata. Whole-union nullability, annotations and ordinary constraints belong
in the enclosing `Type`, `Attribute`, `Payload` or `Result` DSL. Unsupported
constructor directives fail rather than disappearing with its temporary attribute.
Generated constructors, `Kind`, setters and `As...` accessors retain typed values;
constructing a selection does not establish that its encoded value is unique.
Without `Untagged`, JSON uses the existing discriminator/value envelope.

Overlap is legal. For example, two string alternatives and one integer
alternative accept a uniquely matched integer while rejecting an ambiguous
string. `OneOf(String, Any)` accepts non-string values that match only `Any`.
Go integer widths and custom decoding accidents cannot disambiguate a schema.
Non-string map keys, opaque replacement codecs, unsupported schema overrides,
tagged alternatives inside untagged branches and cycles that revisit the same
wire value without consuming a member or element remain rejected.

## Ownership

- `expr/untagged_validation.go` owns the recursive semantic grammar.
- Effective constraints, occurrence policies and `expr.ValuePlan.JSONShape`
  own schema predicates. Field visibility, spelling, closure and additional
  members are captured before lowering.
- `unionjson.Plan` projects the finalized service or transport representation;
  `Analyze` and the renderers supply typed branch operations. A transport uses
  its own body/view layout, not a service predicate with different wire names.
- `internal/unionmatch.Select` owns independent schema and decoder counts.
  `pkg.MatchUntaggedJSON` and `http.MatchUntaggedText` adapt their wire domains.
- `internal/httpencoding.Resolve` and expression-level form/multipart capture
  own each location's grammar, physical key claims and source-aware encoding.
  Transport adapters consume that contract instead of re-inferring it from a
  Go type or media label.

Schema membership and Go representability are separate. Runtime validation
checks typed values, while schema matching retains the public domain. A value
that matches two schemas and only one Go decoder is still ambiguous.

## Decode and encode

Every branch observes the same original input and decodes into private storage.
The matcher visits every candidate, even after ambiguity or a successful branch,
so declaration order cannot suppress a fatal failure.

```text
S = branches whose schema accepts the original wire value
D = branches whose typed decoder and validation accept it
require exactly one member in S
require exactly one member in D
require the same branch identity in S and D
apply selected-schema evaluated-member closure, if authored
publish the selected private candidate
```

Zero or multiple matches report the count and the schema or decoder domain.
Different unique matches report an identity mismatch. Ordinary wrong shape,
missing members, disallowed null, constraint failures and numeric overflow are
candidate mismatches. Invalid matching descriptions, missing callbacks and
unsupported predicate execution are fatal errors. Malformed JSON follows the
strict JSON decoding contract. No failed candidate mutates the destination.

Encoding serializes the selected typed alternative, then runs the same matching
contract on the actual emitted value. The selected identity must survive; nested
text/form alternatives retain the entire selected path. Failure occurs before
request publication or response commitment. The encoder never tries another
alternative to rescue invalid or ambiguous output.

## Presence and closure

Null is an ordinary value for branch counting. A nullable branch, `Any`, or a
nested union with a uniquely null-admitting alternative can accept it. Multiple
null matches are ambiguous. An explicit outer `Nullable` keeps its own wrapper;
an implicitly null-admitting union retains its native union selection so an
absent union and a selected null branch remain different states.

Requiredness is checked before defaults. Defaults apply only to absence;
supplied scalar zeros and nonnil empty collections remain supplied. Selected
empty collections still undergo constraints and exact-one matching. Nil nested
union conversion preserves the outer selection without dereferencing nil, so
normal validation can reject the invalid selected payload.

`openapi:unevaluatedProperties=false` closes a union occurrence after unique
selection. Closing each candidate first would incorrectly disambiguate an
ambiguous value. Closure uses evaluated-member annotations from the selected
schema, including nested alternatives and additional-member policies, rather
than a list of declared property names. The physical form/multipart source
retains unknown keys even when a field's schema projection fails, so closure
cannot lose evidence of an undeclared member.

## Transport boundaries

| Surface | Contract |
| --- | --- |
| JSON service and HTTP bodies | Supported branches use strict JSON matching, typed validation and outgoing identity replay. HTTP body/view policies remain occurrence-specific. |
| JSON-RPC values | The same JSON codecs operate inside existing protocol envelopes and stream framing. |
| Path, query, header and cookie bindings | Every branch requires an admitted location codec. Raw tokenization and escaping precede schema/typed matching. Nullable text and unsupported nested style shapes stay rejected. |
| Forms | Supported object/map roots and recursive untagged root alternatives use captured field codecs. Explicit style selects text serialization; JSON fields retain JSON presence. |
| Multipart | Generated roots are objects or recursive untagged object alternatives. Ordered physical parts and literal MIME names/values preserve matching and closure evidence. Existing explicit custom carrier hooks retain their separate contract. |
| OpenAPI | Supported branches export without inventing a discriminator. Import retains source encoding facts through normalization and names inline objects where concrete declarations are required. |
| gRPC | Existing numbered protobuf alternatives and collection wrappers retain explicit identity. JSON ambiguity matching does not run on protobuf messages; its own presence and admission restrictions remain. |

## Verification

The owner-level regressions exercise count disagreement, order reversal, fatal
callbacks, recursive evaluated members, null multiplicity and occurrence
isolation. Maintained `UntaggedRootsDSL`, `UntaggedLocationsDSL`, `UntaggedFormDSL`
and `UntaggedMultipartDSL` fixtures compile generated service/client/server/CLI
code and exercise normal, invalid, ambiguous and outgoing-replay paths. Importer
behavior-table, normalization, compiled round-trip and independent regeneration
checks cover the same admitted boundaries.

Compare bounded representative output byte for byte with its recorded baseline
and across separate generation processes. Compile the compared artifacts and
validate rendered OpenAPI with libopenapi and pinned Vacuum. A full exported-design
corpus is an occasional separate audit, not the default ticket gate.

Formal models remain opt-in design tools. Their assumptions, counterexamples and
implementation correspondence live beside their owners, including
[`pkg/tla/untagged_json`](../../pkg/tla/untagged_json/README.md),
[`codegen/tla/union_branch_carrier`](../tla/union_branch_carrier/README.md) and
[`http/codegen/tla/form_body_identity`](../../http/codegen/tla/form_body_identity/README.md).
Models do not prove generated Go typing or real transport behavior; the compiled
and mounted regressions check those boundaries.
