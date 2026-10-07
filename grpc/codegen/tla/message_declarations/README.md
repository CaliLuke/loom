# Protocol buffer message declarations

The model covers allocation and collection as separate steps. Two generated
collection wrappers compete with a design name. Two explicit declarations also
use that name: one has compatible fields and one has incompatible fields.
A third explicit declaration has compatible fields but a different protobuf
name that produces the same Go type name. Every allocation and collection order
is explored.

Run from this directory with a TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy_wire.cfg MessageDeclarations.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy_allocation.cfg MessageDeclarations.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy_go.cfg MessageDeclarations.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg MessageDeclarations.tla
```

The legacy wire model fails `WireCorrect`: collection keeps the first
message with a name but accepts a second, incompatible use (10 distinct states
before the counterexample). Checking shapes alone still leaves unsafe wrapper
allocation: `legacy_allocation` fails `UniqueGenerated` (3 distinct states).
Those failures were reproduced in the initial model before implementation.
The extended model also checks protobuf names separately from Go names:
`legacy_go` fails `NamesMatch` after 12 distinct states.

The checked model passes wire correctness, distinct generated names, acceptance
of generated wrappers, matching protobuf names, and completion: 1095 generated
states, 377 distinct states. Compatible declarations can share one message.
Explicit incompatible names are rejected, whichever declaration is encountered
second.

## Conditional argument and limits

Allocation starts with reserved design names. Each generated owner selects the
first free name and records its ownership; no later owner can select it. Equal
collection shapes use the same stable owner and may share a wrapper. The
existing `../message_names` model checks repeated allocation stability.

Collection records the actual protobuf name and shape under its generated Go
type name before visiting children. A repeated Go name is accepted only when
both protobuf name and shape agree. Inductively, every accepted reference has
the declaration it expects; incompatible explicit names cannot silently discard
one another. Recording before recursion terminates cyclic graph traversal.

The finite model abstracts names to integers and shapes to distinct symbols.
It assumes stable owner keys, correct shape comparison, and deterministic
protobuf-to-Go naming. It does not prove identifier normalization, hashing,
recursive shape computation, protoc output, or conversion code. Tests cover
both collection orders, compatible sharing, explicit and normalized name
collisions, Go-name collisions, generated compilation and payload round trips.
The model does not claim every rejected explicit-name design can be allocated
without changing the author's requested protobuf name.

From the repository root:

```sh
LOOM_DIR="$PWD" go test ./grpc/codegen -run 'TestNested(Collection|Explicit)MessageDeclarations' -count=1
```

## Recursive normalization

Naming an inline object duplicates its graph. Stopping at a previously visited
canonical type ID can therefore leave a recursive reference pointing to an
unnormalized copy. `RecursiveAliases` models two physical copies of one type
and an unrelated type, including references reached while normalization is
still active.

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config recursive_legacy.cfg RecursiveAliases.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config recursive_canonical.cfg RecursiveAliases.tla
```

The legacy model fails `NormalizedAliases` after 26 distinct states. Rebinding
repeated IDs to the first message passes normalization, canonical identity,
and termination: 51 generated states, 27 distinct states. Both runs preceded
the normalization fix. This argument assumes that equal canonical IDs denote
the same type and that the first visit completes normalization. It does not
prove that the DSL assigns IDs correctly. Direct pointer-identity assertions,
recursive inline-object compilation, and recursive union compilation cover
the corresponding generator paths.

## Explicit names at every message position (#414)

`ExplicitNames` models normalization of six occurrences of one authored named
message: root, field, element, map value, union branch and wrapper. The old
root-only propagation violates `AuthoredIntent` after 3 generated/distinct
states. Shared normalization preserves authored intent, reference/declaration
agreement, one identity and completion in all orders: 194 generated states,
64 distinct states. Run from this directory:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config explicit_legacy.cfg ExplicitNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config explicit_shared.cfg ExplicitNames.tla
```

The name is preserved while `wrapperAttribute` and `collectionMessageAttribute`
lower values into messages, then `makeProtoBufMessageR` resolves it on each
normalized occurrence. References, collection and response contracts consume
that occurrence. The existing `MessageDeclarations` model supplies the separate
conflict-rejection obligation. No name substitution is permitted to reconcile
conflicting explicit declarations. Compatible copies may share one declaration.

The finite model assumes correctly classified message positions and shared
shape comparison. It does not prove actual graph traversal, metadata copy
ownership, naming conversion or generated code. `TestProtoMessageNameNormalization`
checks root wrappers and detached metadata. `TestProtoMessageShapeIncludesReferencedNames`
reproduces a comparison that loses a differently named repeated child reference;
`writeProtoFieldShapes` must include every named edge before its recursion guard.
`TestNestedStructMetaNames` covers rejected metadata projections, requiredness
changes, explicit/implicit collisions, generated-message collisions and Go-name
collisions, plus accepted shared copies. `ProtoStructMetaDSL` and
`TestGeneratedStructMetaRoundTrip` cover nested fields, arrays, maps, map aliases,
unions, recursion, unary calls, errors and both streaming directions through
compiled protobuf clients and servers. The generator package's existing
`TestProtoStructMetaGeneratedCodeCompiles` also covers the aggregate CLI.

The chosen contract accepts one explicit name wherever its declaration agrees;
it rejects conflicting declarations instead of silently ignoring names. Former
root-only naming is not retained as an alternative mode. See the protocol buffer
naming section of `docs/grpc-guide.md` for migration consequences.

### Bounded generated-output validation

Compared base `0d7bf3d9` with the #414 implementation using Go 1.27.1,
protoc 35.1, protoc-gen-go v1.36.12 and protoc-gen-go-grpc 1.6.2. Three designs
used identical inputs at both revisions and in an independent repeat:
the expanded `ProtoStructMetaDSL`, `PayloadShapesDSL`, and the checked-in gRPC
quality fixture design. Each generated module built and vetted. All 57 generated
files matched byte for byte between independent candidate runs. Nine of 25
explicit-name specimen files changed versus the base; all 32 control files were
unchanged. No checked-in generated fixture uses the affected naming metadata.

Intended changes are: nested and wrapper protobuf declarations use their explicit
names; repeated root/nested declarations collapse; protobuf descriptors and
service signatures reference those names; converters, validators, client CLI
builders and response-contract messages follow the resulting Go/protobuf names.
The service types, examples, field numbers, default behavior and unannotated
control output remain unchanged for identical design inputs. The map and alias
fixture extension is supplied to both revisions to isolate generator effects.
Rejected designs are checked by direct generation tests, not included as
successful compile specimens. No full exported-design audit was run.

Independent review found that the legacy metadata validator accepted zero or
multiple explicit names, allowing unrelated endpoint metadata to select the
last name at the root while nested occurrences selected the first. The model's
single-authored-name assumption is now enforced by `expr.validateStructMeta`.
`TestProtoMessageNameRequiresSingleValue` rejects missing, repeated and distinct
multiple names on types and fields, including a root message with unrelated
metadata. All six cases failed before the validation repair. Public DSL GoDoc
and usage docs state this cardinality; no first/last precedence is introduced.

After that repair, both process-isolated candidate generations reproduced all
57 previously compiled/vetted files byte for byte, so the existing compilation
evidence remains valid. The accepted derived-type case in
`TestNestedStructMetaNames` gives a stricter `Type("RequiredA", A, ...)` its own
explicit name and compiles it alongside the original nested `A` declaration.
