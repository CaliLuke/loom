# JSON query carrier publication

`QueryPublication.tla` explores the declaration and conversion boundaries for a
JSON query value. It compares the body-only registry with the proposed shared
server JSON root, service-owned native destination, and detached CLI JSON input.
It is an opt-in design aid, not a build, test, hook, CI or release requirement.

## Concrete bound and decisions

The graph is `Choice -> Cat, Dog`, optionally with the recursive edge
`Cat -> Choice`. A separate prepared body occurrence may use the same semantic
graph. Server wire, CLI wire and service-native declarations have separate
package membership. The model explores all worklist collection orders and
tracks the conversion-helper graph separately from the type graph. A `Choice`
helper node represents the captured plan root even when its code is rendered
inline; its branch nodes represent dependent helper bodies. The model does not
require an actual Go function named after every abstract node. Runtime
steps explore both valid and invalid required members for server and CLI input.

`declaring` also ranges over both values. A non-declaring carrier, such as a
JSON map of scalars, has root `Value`: a physical carrier with no named
declarations, so `NeededDeclarations` excludes it while the same wire carrier,
validation and conversion obligations remain. Invalid input there stands for a
null map member that the physical carrier rejects.

The current counterexample publishes service declarations and captured server
conversion helpers, but no query wire declarations. Adding only that server root
still leaves two physical layouts assigned to the same server-local type name.
The proposed design converts the JSON carrier directly to the service-owned
native value, so payload construction copies that value instead of inventing
another transport-local native declaration and conversion. CLI input receives
its own detached JSON carrier and the complete captured conversion plan.

This supports four implementation decisions:

1. Register the exact prepared JSON query graph with its physical layout. Reuse
   the existing declaration/union/validator owner; a second naming-only registry
   does not establish declaration closure.
2. Share allocated identity where appropriate, but retain package-specific
   membership and visited sets. Visiting the server graph cannot suppress CLI
   declarations with the same semantic identity.
3. Retain and publish the whole captured conversion plan. Direct conversion to
   the service value removes the extra constructor conversion; it does not make
   the actual wire-to-service helpers optional.
4. Validate required wire fields before conversion dereferences their storage.
   This ordering applies independently to server query decoding and CLI flags.
5. Decode every non-custom carrier, with or without declarations, through its
   physical layout before conversion, on the server and in the CLI. Declaring
   types decides only declaration publication. The `native-cli` control
   reproduces the R2 CLI, which decoded a non-declaring flag directly into
   native storage and so accepted input the server rejects.

## Explicit checks

From this directory, use Java 25 and the existing official TLA+ tools 1.7.4 jar
(TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-query-publication-current \
  -config current.cfg QueryPublication
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-query-publication-checked \
  -config checked.cfg QueryPublication
```

Run the other configurations with distinct temporary metadirs. A negative
control succeeds as evidence only when TLC reports the named invariant failure,
not a parse, evaluation or tool error. The current configuration was checked
before the corrected design and incomplete alternatives. With the `declaring`
dimension, `native-cli` was checked before the corrected non-declaring CLI.

| Configuration | Policy checked | Result / distinct states |
|---|---|---|
| `current.cfg` | Body-only registration, local-native target | `EveryReferenceDeclared` fails, exit 12 / 208 |
| `registry-only.cfg` | Add server wire root, retain local-native target | `OneLayoutPerSymbol` fails, exit 12 / 680 |
| `global-membership.cfg` | Share visited membership across packages | `EveryReferenceDeclared` fails, exit 12 / 2,211 |
| `dropped-helpers.cfg` | Correct types, drop CLI conversion helpers | `EveryHelperPublished` fails, exit 12 / 724 |
| `late-validation.cfg` | Correct publication, convert before validation | `ValidateBeforeConvert` fails, exit 12 / 2,162 |
| `invalid-conversion.cfg` | Same ordering, check invalid result publication | `CorrectResult` fails, exit 12 / 2,162 |
| `native-cli.cfg` | Non-declaring CLI flag decoded into native storage | `CorrectResult` fails, exit 12 / 2,163 |
| `checked.cfg` | Complete captured ownership | All six invariants pass, exit 0 / 3,914 |

The `native-cli` counterexample has `declaring = FALSE`, invalid CLI input and a
CLI that converts without validation, ending `done`. Checked design: 15,751
generated states, 3,914 distinct states, depth 32, exhausted queue.
No fairness or liveness theorem is claimed. Terminal states are allowed. The
results were obtained with Corretto 25.0.5; checker output and raw traces belong
in temporary evidence directories, never in this source directory.

## Source correspondence and regression obligations

The model was grounded in the immutable prospective source used for #1005's
query correction. The table maps its boundaries to the corrected source; the
checked policy corresponds to the corrected owners, while `current.cfg` and
`registry-only.cfg` describe the replaced local-native target.

| Model boundary | Implementation owner |
|---|---|
| `PreparedRoots`, `CollectType`, package membership | `json_field_carriers.go`: `jsonQueryCarrier` registers a carrier that declares types through `registerBodyRoot`; `service_data_body_registry.go`: `orderedBodyRoots`, `collectBodyUnionTypes`; `service_data_type_registry.go`: `collectBodyAttributeTypes`, `recordAttributeTypeLayouts` |
| `declaring`, `Declarable` | `declaresTypes` guards only that publication; validation, service binding and conversion apply to every non-custom carrier |
| `Wire("server")` and `Symbol`/`Layout` | the detached carrier drops `struct:pkg:path` and keeps JSON presence in the transport package; the native value keeps service placement |
| `PlanRoots` to `Native`, constructor copy | `jsonQueryCarrier` converts with `codegen.GoTransform` into the service declaration bound by `bindServiceCarrier` (`service_data_parameter_codecs.go`), so constructors copy the native value |
| `Wire("cli")` and its captured conversion | `service_data_json_input.go`: `prepareCLIQueryInput` creates a detached `newJSONBodyInput` for every non-custom carrier after every body input and retains its helpers; `client_cli.go`: `fieldLoadCode`; `codegen/cli`: `CarrierFieldLoadCode` |
| `NativeInput` (replaced) | the R2 `cli.FieldLoadCode` path for non-declaring JSON query flags, which decoded straight into the native map |
| `Validate`, `Convert` ordering | `template_sources_request_decoder_elements.go`: the JSONField branch runs `JSONValidateCode` and returns before `JSONDecodeCode`; `CarrierFieldLoadCode` decodes, validates, then converts |

`TestJSONQueryCarrierDeclarationClosure` checks package-local identifier closure
of the server and client packages for plain and hinted unions, a map of objects
and a body sharing the declarations. `TestJSONQueryCarrierGeneratedWire` compiles
a generated module and runs plain/hinted union, recursive union (`Branch` to
`Tree`), map-of-object, shared-body and `struct:pkg:path` round trips, mounted
HTTP calls, rejected missing, malformed, null and invalid (including nested)
members with the endpoint uncalled, and CLI input including null rejection.
`TestJSONQueryCarrierDefaults` covers the absent-key branch the model omits and
null root/member CLI rejection. `TestJSONQueryCarrierCLIParity` requires the
CLI carrier type of non-declaring maps to equal the server's and validation to
precede conversion on both sides.
`TestJSONBodyInputRole`, `TestCLICollectionInputKeepsServerInitialization` and
`TestDeepObjectReadinessGeneratedWire` provide adjacent existing controls.
Embedded tests in generated harness strings run inside those root-package tests;
they are not separate selectors. Generated compilation and exact output
comparisons remain necessary.

This model complements [union declarations](../union_declarations/README.md),
which checks allocation and package membership, and
[CLI JSON input](../cli_json_input/README.md), which checks presence, required
admission and defaults. It does not replace either model or its Go tests.

## Assumptions and limits

The fixed graph represents already admitted DSL and captured physical carriers.
Correct extraction of types, dependencies, physical layout and helper edges is
assumed. The finite worklists test closure and package ownership on this graph;
they do not prove arbitrary Go graph traversal or name allocation. The concrete
symbol function assumes detached body names and service qualification and
deliberately reproduces the old local-native/wire collision.

`inputValid` abstracts a required wire field before a pointer-dereferencing
conversion. It does not parse JSON, choose a union branch, model absence/null/
empty distinctions, implement defaults or determine custom codec policy. Those
remain captured value/runtime contracts and generated regression obligations.
`Decode` models only a present query value. An absent optional value with an
authored default bypasses validation and conversion: `jsonQueryDefault` renders
the literal through the shared `AttributeContext.DefaultValue` owner directly in
the service-owned native layout, which references only declarations already in
`NeededDeclarations`. It adds no wire declaration, helper or dereference, so the
model does not represent it. Server and CLI reject a null document, including
for an opaque custom codec; null map members are rejected by the physical
carrier, which the model abstracts as invalid input.
The late-validation controls are design counterexamples, not a claim that a
previously uncompilable native reproduction reached a runtime panic.

No check proves generated Go syntax, source-to-model refinement, exact byte
determinism, OpenAPI projection, ABI compatibility or unbounded correctness.
Passing this model approves no source diff or ticket. Keep its assumptions
aligned if the implementation chooses different carrier or helper ownership.
