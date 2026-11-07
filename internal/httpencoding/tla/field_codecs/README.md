# Named content encoding and source application

The owner is `internal/httpencoding`. Capture source facts before normalization,
then resolve one immutable field contract for expression validation, transport
analysis, generation, and OpenAPI output.

[OpenAPI 3.2 §4.14.5.1 and §4.15.1.1–2](https://spec.openapis.org/oas/v3.2.0.html#encoding-object)
settle two distinctions that the earlier design missed:

- Named properties with source `type: array` apply content encoding separately
  to each item. The array row in the default-content table describes an array
  item, such as an inner array. It does not make an outer array one JSON field.
- An explicit `style`, `explode`, or `allowReserved` suppresses `contentType`,
  including an authored label, in both form and multipart media. Multipart
  style serialization does not percent-encode names or values.

A typeless union with explicit JSON remains one complete JSON value even if all
its alternatives are arrays. An explicitly array-valued union instead retains
item application and each alternative's item grammar. One field-wide fallback
label cannot describe string-array and object-array alternatives correctly.

## Captured contract

`BranchShape.SourceType` retains an effective source type, with `absent`
preserving omission and the empty string selecting native inference. `Items`
retains the array edge; `Binary` follows byte identity through aliases and item
edges. The importer persists necessary facts as attribute-local
`http:encoding:sourceType` metadata. The superseded `DefaultMediaType` and
body-field `defaultContentType` metadata are removed.

`Codec.Application` and `MultipartCodec.Application` select `Value` or `Items`.
Content application happens once; inner arrays are complete item values.
`Codec.Branches` and multipart `Branches` retain heterogeneous source-array
alternatives. Explicit style framing is captured separately. Multipart `Keys`
and form field keys come from the same namespace resolver, including exploded
object members. Code generation must consume these facts without re-resolving
or inferring them from a content label or normalized Go type.

`AnyValue` distinguishes a JSON-compatible unconstrained value from unsupported
Go shapes. Text content cannot carry null; JSON item nullability is independent
of the outer array. A nullable outer array has no distinct spelling under item
application and is rejected. Raw binary multipart values retain their bytes;
content labels do not turn them into JSON documents.

## Models and assumptions

`SourceApplication.tla` models content application and media precedence before
value codec selection. It includes outer arrays and array items independently.
It assumes valid source types and a supported style grammar, and does not model
raw bytes, URL escaping, JSON syntax, namespace overlaps, or typed decoding.

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| whole-array | ApplicationAgreement fails | 449 / 449 |
| media-conflict | MediaAgreement fails | 268 / 268 |
| source-checked | Both invariants pass | 768 / 512 |

`FieldCodecs.tla` starts after application has selected a complete value or one
item. Its `array` is an encoded array value, never an outer named array. It
checks text/JSON admission, null rejection on text, and preservation through
normalization. Supported style scenarios include fixed scalar objects and
scalar collections. It does not establish the complete style matrix.

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| checked | All three invariants pass | 1536 / 1152 |
| text-rejected | Agreement fails | 961 / 961 |
| shape-only | Agreement fails | 771 / 771 |
| normalize-first | Agreement fails | 775 / 775 |

The former materialized-default control depended on the incorrect multipart
media-conflict rule and is superseded by the two source-application controls.
Previous checks of that old policy are not evidence for the corrected contract.

## Explicit checks and implementation evidence

Use TLA+ tools 1.7.4 with Java 25. These checks are opt-in design aids, outside
ordinary tests, hooks, CI and release eligibility:

```sh
for policy in whole-array media-conflict source-checked; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -metadir "/tmp/loom-field-$policy" \
    -config "internal/httpencoding/tla/field_codecs/$policy.cfg" \
    internal/httpencoding/tla/field_codecs/SourceApplication.tla
done
for policy in checked text-rejected shape-only normalize-first; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -metadir "/tmp/loom-field-$policy" \
    -config "internal/httpencoding/tla/field_codecs/$policy.cfg" \
    internal/httpencoding/tla/field_codecs/FieldCodecs.tla
done
```

`source_application_test.go`, `field_content_test.go`, and `multipart_test.go`
check the owner APIs. Expression and importer tests cover source normalization,
alias context, nested items, binary items, and namespace collisions. Runtime
`http/json_field_items_test.go` checks exact repeated JSON documents, one-level
array application, malformed later items before typed publication, absence,
empty arrays, null items and deterministic nested maps. Generated compilation,
wire exchange, selected-path replay, and exported-schema tests must separately
prove each consumer uses the captured contract; these models and direct tests
do not establish that integration. Remove checker states after independent
review, retaining sources, configurations, reproducible inputs and findings.
