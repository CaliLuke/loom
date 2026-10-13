# Collection wrapper contents (#684)

A present collection message carries an empty or populated value. Protobuf
normalizes empty repeated/map contents to nil; that does not remove the wrapper
message. The current nested-only Required(contents) invents an admission rule.
The candidate derives contents requiredness from the wrapped kind: scalar and
union values need presence, collection contents do not. Authored collection
minimum length and required containing-message presence remain separate.

Run from this directory with Corretto Java 25.0.4.1 and TLC 1.7.4 (TLC2 2.19),
JAR SHA256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
export TLA_JAR="$HOME/.cache/loom-tools/tla2tools-1.7.4.jar"
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-684-current -config Current.cfg CollectionContents.tla
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-684-typed -config Typed.cfg CollectionContents.tla
```

Current must fail Admission on a present empty nested collection. Typed must
pass admission, empty preservation and required-message rejection. Source kind,
position, presence, minimum length and empty/populated contents vary independently.

Correspondence: `wrapperAttribute` constructs the contents contract;
`wrapNestedCollection` must allocate using that same contract;
`collectionMessageAttribute` and endpoint shape analysis must agree. Generated
validators enforce wrapper and length constraints. `transformArrayInitBuffer`
and `transformMapInit` allocate present collections even when decoded child
storage is nil. This bounded model assumes those conversions and native
protobuf presence semantics; generated binary encode/decode tests establish
that correspondence. It does not prove schema identity, recursion, naming,
nullable positional admission, custom codecs or ordinary unwrapped fields.
The message-declarations model owns sharing/canonical recursion separately.

The DropAll.cfg alternative removes every contents presence requirement; it
fails Admission for a present scalar wrapper with missing value (345/345
states). Current fails on an empty nested collection (329/329). Typed passes
576/576 states. The model independently varies scalar/union value presence;
it does not confuse an explicit scalar zero with an absent scalar field.

## Shared wire shape versus contents contract

The implementation probe exposed a separate identity assumption: generated
anonymous collection wrappers currently share a name/UID solely by wire shape,
but their complete named validators own contents predicates. Differing bounds
therefore compete for one helper. The parent requiredness check masks missing
helper emission but does not retain the later bound.

That prerequisite was repaired independently in #831. The shared
[wrapper contract model](../wrapper_contracts/README.md) owns its counterexamples,
complete-contract identity and equal-contract sharing invariants. This directory
keeps only the collection-contents model; it does not duplicate the delivered
identity model. Direct #831 tests and representative mounted order cases remain
part of this repair's bounded acceptance.

`masked_contract_probe.go.txt` is a parent-source generated repro: copy it as
`grpc/codegen/collection_wrapper_readiness_test.go` in an isolated parent checkout
and run `LOOM_DIR="$PWD" go test ./grpc/codegen -run
'^TestPresenceNestedCollectionReadiness$' -count=1 -v`. The expected length error
is absent for a one-element collection with MinLength2. This probe is kept
outside ordinary Go test discovery as historical prerequisite evidence; #831 now supplies the correct identity.

The implemented `wrapperAttribute` has no caller-selected requiredness flag.
Anonymous registry shape allocation uses this same constructor. The generated
Rich request/result and mounted tests cover empty contents, authored MinLength,
required-wrapper absence, native nil element rejection and decoded service state.
The three-specimen comparison confirms unchanged protobuf/service declarations
and conversions, with only validator/helper reachability changes. Sources and
configurations are unchanged from scoped readiness, so the recorded checker
results remain applicable.
