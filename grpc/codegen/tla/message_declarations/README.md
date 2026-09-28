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
