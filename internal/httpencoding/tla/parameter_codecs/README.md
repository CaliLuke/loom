# Parameter binding codecs (#968)

Design evidence for `roadmap/issue-968-parameter-serialization-design.md`.
These bounded models informed the `internal/httpencoding` contract:
`Resolve`, `ResolveUnion` and `KeySet`. The non-union codecs are implemented;
text-union admission and generated branch adapters remain owned by #1014.
See [validation evidence](../../parameter_codecs_validation.md) for implementation
coverage. Bytes use raw text in text bindings and base64 in JSON values. The
codec model covers token spelling and representability; it does not model Go
byte slices or JSON base64 conversion. Runtime and generated tests cover those
adapters.

The existing `../serialization` model and its uniform-JSON history are
unchanged. These modules add:

- the path, header, cookie and explicitly styled form-field locations;
- styled scalar-valued maps (`DynamicObject`);
- key namespaces;
- the per-branch union contract that #1014 consumes.

## Modules

| Module | What the transitions do | Independent check |
| --- | --- | --- |
| `ParameterResolution.tla` | Authored metadata, including `allowReserved` and an explicit content type for query parameters and form fields → design `Resolve` (the effective style is resolved once per binding) → generated client/server → export (3.2.0 or 3.1.1) → import with the OpenAPI defaults → re-resolve. | Encoded as data: the OAS 3.2 §4.12.3 "in" table, the non-`n/a` cells of §4.12.6, and Appendix D.1. Also encoded: §4.12.2 (Parameter `content` excludes the schema-only style fields) and §4.15.1.2 (an Encoding Object's explicit style fields, by presence, override its content type). The 3.1.1 cookie projection requires both the key and the value domain to be unreserved-only. |
| `CodecRoundTrip.tla` | Encoder admission and spelling, including protection of a complete `.`/`..` path segment → transport → decoder. The transport covers: a raw path capture, optionally spelled with lowercase hex and optionally put through RFC 3986 `%2E` normalization inside a segment; the raw query carrier; RFC 9110 header list framing; and cookie `; ` parsing by either a net/http peer or a strict RFC 6265 §5.2-style peer. | Each value must round trip to its stated domain state. `DomainMaximal` re-encodes every rejected value, with complete dot segments protected but without admission, and requires that spelling to fail under the specified decoders. |
| `KeyNamespaces.tla` | Endpoint admission of literal, prefix (`name[...]`) and dynamic query namespaces, including one union binding whose two alternatives are dynamic maps → encoder key admission → decoder ownership. | Every binding decodes exactly what it sent. A dynamic binding also receives unclaimed foreign keys. |
| `BranchMatch.tla` | Endpoint admission of a literal sibling beside the union's residual claim → key ownership removes the sibling's key from the union universe → every branch's `Schema` and `Decode` is evaluated independently, without stopping early → commit only when the one schema member is also the one decoder member. Fatal callback errors propagate. | Source `oneOf` semantics, computed separately, over the union's owned key universe. |

## Reproduce

Toolchain: Amazon Corretto 25.0.5 and `tla2tools.jar` release 1.7.4 (TLC 2.19,
rev `5a47802`), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`. Run from
this directory, with `JAVA` and `TLA_JAR` set and `META` set to a new
task-owned directory outside the repository:

```sh
for m in resolution:ParameterResolution codec:CodecRoundTrip keys:KeyNamespaces branch:BranchMatch; do
  prefix=${m%%:*}; module=${m##*:}
  for cfg in "$prefix"-*.cfg; do
    "$JAVA" -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
      -config "$cfg" -metadir "$META/${cfg%.cfg}" "$module.tla"
  done
done
```

Expected outcomes:

- every `*-checked.cfg` exits 0;
- every other configuration is a negative control and exits 12 on its named invariant;
- exit 150 (parse error) or 75 (TLC exception) means the model is broken, not that the expected counterexample was found.

## Recorded results (2026-11-05, source `f0c7a2b8c`, `-workers 1`)

The design-blocker correction (DB-1 protected dot segments, DB-3
location-specific defaults) changed `CodecRoundTrip` and `ParameterResolution`.
All of their configurations were rerun. `KeyNamespaces` and `BranchMatch` did not
change, and their earlier results are reused. Control counts are at the first
violation.

| Configuration | Initial | Generated / distinct | Result and witness |
| --- | --- | --- | --- |
| `resolution-checked` | 29,160 | 151,402 / 122,242 | pass: NoUndefinedAdmitted, FullScope, UnionOneStyle, UnionKeysCover, DefaultEquivalence, ClientServerAgree, ExportPreservesWire, EncodingStyleFirst, ParameterContentExclusive, QueryReservedNeutral |
| `resolution-legacy` | 29,160 | 116,641 / 116,641 | ExportPreservesWire: the current exporter copies an unvalidated `style: matrix` onto a raw cookie |
| `resolution-reduced` | 29,160 | 29,465 / 29,465 | FullScope: the first-draft V1 rejected a D.1-endorsed `form`, `explode: false` cookie array |
| `resolution-deep-arrays` | 29,160 | 35,217 / 35,217 | NoUndefinedAdmitted: `deepObject` on an array (`n/a`) |
| `resolution-form31` | 29,160 | 116,809 / 116,665 | ExportPreservesWire: 3.1.1 describes a raw string cookie as `form` |
| `resolution-form31-values-only` | 29,160 | 117,261 / 116,755 | ExportPreservesWire: an unexploded `cookie` object with unreserved values but reserved keys is described as `form` |
| `resolution-maps-json-only` | 29,160 | 37,377 / 37,377 | FullScope: explicit `style: form` on a scalar-valued query map is rejected |
| `resolution-union-name-only` | 29,160 | 30,841 / 30,841 | FullScope: a `cookie` union with an exploded object branch is rejected instead of composing KeySets |
| `resolution-per-branch-default` | 29,160 | 44,577 / 44,577 | UnionOneStyle: in a metadata-free query union of a scalar and a map, the map branch applies the JSON map default while the scalar uses `form` |
| `resolution-reserved-switches-query` | 29,160 | 38,113 / 38,113 | QueryReservedNeutral: `allowReserved: true` alone switches a metadata-free query map from JSON to `form` |
| `resolution-encoding-on-parameters` | 29,160 | 33,481 / 33,481 | ParameterContentExclusive: a query scalar with `content: application/json` plus `style: form` resolves a style instead of being rejected |
| `resolution-form-content-first` | 29,160 | 49,249 / 49,249 | EncodingStyleFirst: a form-field object with explicit `contentType: application/json` and an explicit style keeps JSON |
| `resolution-media-style-reject` | 29,160 | 47,089 / 47,089 | FullScope: the old blanket rejection refuses a form-field scalar with explicit `contentType` plus `style: form` |
| `resolution-form-explicit-false-ignored` | 29,160 | 50,433 / 50,433 | EncodingStyleFirst: an explicit `allowReserved: false` on a form-field object is treated as absent, so the content type wins |
| `resolution-form-flag-dropped` | 29,160 | 137,161 / 120,139 | ExportPreservesWire: a form-field object with explicit `style: form` exports without a style, and reimport selects the JSON content-type default |
| `codec-checked` | 59,080 | 295,400 / 236,320 | pass: RoundTrip, RequiredPresent, DomainMaximal |
| `codec-label-pct-dot` | 59,080 | 199,245 / 199,245 | RoundTrip: label array `["."]` spelled `.%2E` is normalized into the dot segment `..` |
| `codec-no-delim-check` | 59,080 | 215,729 / 215,729 | RoundTrip: a `spaceDelimited` element containing a space |
| `codec-single-pass-comma` | 59,080 | 210,217 / 210,217 | RoundTrip: a `form`, `explode: false` element `,` decoded before splitting (url.Values) |
| `codec-decode-first` | 59,080 | 177,337 / 177,337 | RoundTrip: a path element `,` unescaped before splitting |
| `codec-go-quote` | 59,080 | 228,186 / 228,186 | RoundTrip: net/http `AddCookie` quotes `n=,`, and a strict peer keeps the quotes |
| `codec-cookie-nonascii` | 59,080 | 230,889 / 230,889 | RoundTrip: a `cookie`-style element with a non-ASCII character; both peers drop the pair |
| `codec-no-rule-a` | 59,080 | 212,925 / 212,925 | RequiredPresent: a required empty exploded query array is sent as absence |
| `codec-map-key-unchecked` | 59,080 | 227,449 / 227,449 | RoundTrip: an exploded header map key `=` |
| `codec-dot-reject` | 59,080 | 177,289 / 177,289 | DomainMaximal: the superseded candidate rejects simple path array `["."]`, whose protected spelling `%2E` (here sent as `%2e`) round-trips |
| `codec-unprotect-after-prefix` | 59,080 | 199,193 / 199,193 | RoundTrip: label empty scalar `%2E` has its prefix stripped before unprotection, so the prefix is missing |
| `codec-dot-case-sensitive` | 59,080 | 199,193 / 199,193 | RoundTrip: label empty scalar sent as lowercase `%2e` is not recognized as the protected spelling |
| `codec-unprotect-late` | 59,080 | 177,417 / 177,417 | RoundTrip: simple path element with the literal text `%2E` (wire `%252E`) is decoded twice into `.` |
| `keys-checked` | 15,756 | 50,284 / 34,528 | reused: pass: NoStealing, UniqueOwner, DisjointAdmitted, PrefixComplete |
| `keys-greedy` | 15,756 | 47,401 / 33,649 | reused: NoStealing: the dynamic map also decodes sibling `a` |
| `keys-no-encode-check` | 15,756 | 47,387 / 33,643 | reused: NoStealing: map entry `a` is decoded as sibling `a` |
| `keys-two-residual` | 15,756 | 47,890 / 39,202 | reused: NoStealing: two dynamic endpoint bindings cannot both own foreign `z` |
| `keys-prefix-literal` | 15,756 | 16,153 / 16,153 | reused: PrefixComplete: literal `m[x]` shadows the `m[...]` map namespace |
| `keys-maps-rejected` | 15,756 | 15,765 / 15,765 | reused: DisjointAdmitted: blanket rejection of a lone `deepObject` map |
| `keys-alternatives-conflict` | 15,756 | 15,809 / 15,809 | reused: DisjointAdmitted: a union whose two alternatives are dynamic maps is rejected as two claims |
| `branch-checked` | 87 | 609 / 522 | reused: pass: LoomRoundTrip, CommittedValueOwned, ExactOneBothDomains, UniqueCommitted, FatalPropagated, including a literal sibling `p` |
| `branch-decoder-only` | 87 | 446 / 446 | reused: ExactOneBothDomains: foreign `n=9` matches the int and map schemas; only the int64 carrier decodes and is committed |
| `branch-schema-only` | 87 | 464 / 464 | reused: ExactOneBothDomains: map `x=9` is the only schema member but overflows its int32 carrier |
| `branch-first-match` | 87 | 265 / 265 | reused: ExactOneBothDomains: `n=1` is committed to the first branch without visiting the rest |
| `branch-fatal-as-mismatch` | 87 | 438 / 438 | reused: FatalPropagated: a fatal int callback is counted as a mismatch, so the map branch wins |
| `branch-shared-scratch` | 87 | 436 / 436 | reused: CommittedValueOwned: a failed map candidate's partial write leaks into the string value |
| `branch-alternatives-as-siblings` | 93 | 466 / 466 | reused: ExactOneBothDomains: the map alternative is excluded from key `n`, so `n=1` is accepted as int although the map also matches |
| `branch-greedy-ownership` | 87 | 467 / 467 | reused: LoomRoundTrip: the map reads the sibling's literal key, so a Loom map `{x:1}` beside sibling `p=a` is rejected |

`DomainMaximal` changed the design four times:

1. Exploded header objects admit empty values and values with a leading space.
2. The `cookie`-style whitespace rules apply to the ends of each serialized value.
3. After the raw-carrier correction, query `form`/`explode: false` admits commas in data.
4. Protection of complete dot segments replaced the earlier rejection of `.`/`..` path segments (`dot-reject` control).

## Assumptions and limits

- **Codec bounds.** The alphabet is `a , . ; = SP | e P`. `e` stands for a non-ASCII character (one UTF-8 escape token). `P` stands for the literal text `%2E` (one `%252E` token). Strings have length at most one, arrays at most two elements, fixed objects range over `p`/`q`, and maps have at most one entry. Escape tokens are atomic. The model spells the encoder's `%2E` in uppercase. The transport can lowercase it, and only that spelling's case is modeled. The case-insensitive decoding of every other escape is an implementation regression obligation.
- **Simple path codecs.** For simple path scalars and sequences, the model escapes `=` and protects only a complete dot segment. Today's `http.EscapePathSegment` also escapes a whole-element `.` or `..` inside a sequence and does not escape `=`. Both spellings decode identically under the specified decoder. The difference between them is not modeled.
- **Protected dot segments.** The normalizing intermediary percent-decodes `%2E` inside a segment (RFC 3986 §6.2.2.2), but it preserves a complete protected segment. This is the same raw-path assumption that today's `EscapePathSegment` relies on. The model makes no claim about intermediaries that then remove dot segments (RFC 3986 §5.2.4). A literal or empty dot segment is modeled as never reaching the route handler.
- **Empty collections.** A path list or associative array with zero members expands to nothing (RFC 6570 §3.2.1), not even the label prefix. A defined empty string keeps the prefix (§3.2.5). So label `""` and `[""]` are representable, and empty collections are not representable under `simple` or `label`.
- **Scope of `DomainMaximal`.** It proves a rejection is necessary only relative to the decoders in this model. Those decoders follow the specification. Path, cookie and query `form`/`explode: false` values are split before percent-decoding (Appendix C.2). `spaceDelimited` and `pipeDelimited` are decoded in a single pass, which Appendix E.6 states is the specified decoding: a `+` convention would be a non-interoperable second pass, and a data `|` and the delimiter are both `%7C`. The `,` rejection for `cookie` is exempted as spec-driven (`SpecForbidden`).
- **Cookie peers.** Two peers are modeled. net/http strips surrounding quotes, keeps a leading space after `=`, and drops non-token names and values with bytes outside 0x20-0x7E minus `"` `;` `\`. The strict peer does the same, except that it keeps quotes and trims value whitespace. `"` and `\` are not in the alphabet, so their rejection is a direct regression obligation.
- **Resolution bounds.** `ParameterResolution` uses two-branch unions and one abstract "plain" flag each for values and keys. `allowReserved` and an explicit content type are varied only for query parameters and form fields. The exported document records `style` and `explode`, not `allowReserved`. Form-field flags survive export because a styled form-field object, map or union always names its style. `UnionKeysCover` is structural. Multipart parts are not modeled.
- **`BranchMatch` bounds.** It abstracts carriers to an int64-like integer, an int32-like map value and a letters-only string. It has three branches, one sibling key, and fatal callbacks on two branches. In this model `Decode` implies `Schema` for every branch, so a unique schema member and a different unique decoder member is unreachable. The model has no independent identity-mismatch witness and makes no claim about that check, which #1014's `UntaggedText` model owns.
- **`KeyNamespaces` bounds.** It covers query keys only. Cookie and form-field namespaces follow the same `CheckNamespaces` rules by argument, not by model.
- **Not established.** No liveness, concurrency, Go, generated-code or serializer correctness.

## Implementation correspondence

| Model claim | Owner and regression |
| --- | --- |
| Resolution, export round trip, 3.1.1 projection | `internal/httpencoding` table tests over every location/shape/style/explode cell. Location-specific default tests: query `allowReserved` alone (the retained JSON codec stays and the content rejection applies), form-field explicit `false` flags, and an explicit content type plus a flag. The OpenAPI IR parameter and Encoding Object builders, including an explicit style for styled composite form fields. The v3 3.1 filter. Importer analysis. An exporter-symmetry fixture with reserved map keys and property names. |
| Codec round trip and domain | `http` helpers for path segments (protected dot segments, case-insensitive `%2E`/`%2e`, the literal `%252E` decoded once, label `""` and `[""]` representable, empty collections rejected), raw query values, delimited lists, deepObject keys, header objects/maps and verbatim cookie pairs. Generated client/server wire tests over the alphabet plus `"`, `\` and multi-byte UTF-8, with both cookie peers and a `%2E`-normalizing proxy. The existing simple-path and catch-all escaping regressions stay unchanged. |
| Key namespaces | One `CheckNamespaces`, called by expr validation and the importer planner. Mounted tests with a dynamic map beside literal, deepObject and credential siblings, with mounted dynamic union alternatives remaining a future #1014 obligation. |
| Branch contract | `ResolveUnion` tests and the current hand-written branch seam controls; future #1014 `MatchUntaggedText` tests for overflow, identity mismatch, fatal callbacks and shared universes. |
| Shape adapter agreement | `expr.HTTPValueShape` and the importer's `urlShape` produce equal `BranchShape` values, including the `Domain` of scalars, sequence elements and map values. |

Task-owned TLC metadirs are kept outside the repository and removed after
review. The metadir for this correction is `/tmp/loom-968-fix.nCGRm6`.

Concrete checks include `internal/httpencoding/resolve_test.go` and `keys_test.go`,
`expr/http_parameter_serialization_test.go`, `http/param_*_test.go`, generated
`parameter_styles_generated_test.go`, `parameter_native_carriers_test.go` and
`parameter_root_collections_test.go`, plus the importer parameter-style,
namespace and adapter-agreement suites. `TestGeneratedStyledMediaContracts`
checks exact form bytes, required empty maps before commitment, malformed fields,
replay and callback lifetime across request reuse. The exporter uses the same
resolved codec and checks 3.1 raw-cookie equivalence before version filtering.
These tests cover implementation behavior outside the finite model as well;
no proof command is part of a normal build, test or release gate.

## Binding identity correction

`BindingIdentity.tla` separates a binding's identity from its diagnostic label.
The previous `map[string]KeySet` representation overwrote an authored `MapParams`
field with the synthetic whole-query claim. The resulting single claim passed
validation even though the two authored bindings overlapped. The `labels`
configuration reproduces this loss of ownership: `Sound` fails with the field
and map selected (13 generated/distinct states, depth 2). The `checked`
configuration preserves both bindings and passes `Sound` and `Complete` (24
generated / 16 distinct states, depth 2).

Run the same pinned TLC tool explicitly, from this directory:

```sh
"$JAVA" -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir "$META/identity-labels" \
  -config identity-labels.cfg BindingIdentity.tla
"$JAVA" -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir "$META/identity-checked" \
  -config identity-checked.cfg BindingIdentity.tla
```

The first command must exit 12 on `Sound`; the second must exit 0. These are
opt-in design checks, never ordinary build, test, commit, CI or release gates.
The model covers three query claims and one label collision, not general wire
encoding. Existing `KeyNamespaces` evidence still covers key-set semantics.

`httpencoding.Binding` now retains labels as data, and `CheckNamespaces` checks
every pair without collapsing equal labels. Expr and importer admission use
that representation. Generated parameter namespace collection excludes the
actual `transportir.Parameter` pointer; form fields use their object position.
Neither path uses a display name as identity. Direct regressions are
`TestParameterBindingLabelsDoNotHideClaims`,
`TestCheckNamespacesPreservesBindingIdentity` and
`TestParameterNamespaceUsesBindingIdentity`. The first failed before the fix.

Validation compared all 25 maintained determinism specimens against parent
`f022c3c5f`: every generated raw/rendered section is byte-identical. No
checked-in generated fixture changes. The corrected rejection concerns an
invalid design, so it has no generated output. Mounted parameter-style,
property-name and tagged-form controls exercise accepted designs, credentials,
residual maps and field namespaces. Scratch evidence is under
`/tmp/loom-fresh-audit/binding-identity`; temporary checker state is removed
after independent review.
