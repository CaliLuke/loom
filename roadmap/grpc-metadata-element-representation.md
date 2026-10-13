# Named gRPC metadata element representation — #877

Implementation parent `71a0758f6`. #705 binary key readiness is
frozen separately in this checkout; its files must remain unchanged.

1. [x] Preserve the explicit-bin named Bytes element compiler failure and four
   passing scalar/outer-array controls. Ticket877 records the separate owner.
2. [x] Inspect metadata extraction, value encoding/decoding, constructor inputs,
   defaults and effective validation; establish one shared physical representation.
3. [x] Add direct analysis and bounded aggregate named-element compile controls.
   Model current representation disagreement and rejected partial repairs first.
4. [x] Freeze owner/model/acceptance for independent readiness review.
5. [x] Implement after approval; compare bounded parent/candidate/repeat
   artifacts, compile/vet generated modules, and check mounted behavior.
6. [ ] Fresh exact final review before root integration, atomic commit and closure.

## Existing witness

`/tmp/loom-877-named-metadata-element_test.go` is the original five-shape probe.
Native Bytes, named Bytes, ArrayOf(Bytes) and named ArrayOf(Bytes) compile.
ArrayOf(named Bytes), mapped to explicit `x-raw-bin` in request metadata and
response headers, produces undefined Raw and undeclared valueStr in both
transport packages. This is independent of705's key derivation. The original
output remains under `/tmp/loom-705-explicit-shapes/named-element`.

## Original ownership defect

The parent `extractMetadata` unwraps only the outer occurrence. That leaves named
array elements in transport Type/TypeName/TypeRef while codec switches require
builtin primitive names. Request constructors, response constructors and request
CLI argument records all consume this same MetadataData.Type; repair its physical
representation before rendering. Preserve FieldType as the actual service type
so existing InitStructFields can convert detached primitive storage back into
qualified service declarations. Service aliases and their original constraints
must remain unmodified.

Encoding traverses the service value, whereas decoding produces transport storage.
A named string/bool/int64 element is not directly assignable to builtin codec
arguments; selecting the builtin codec alone is insufficient. One analyzed
metadata representation must specify both storage and the source-to-storage
conversion consumed before formatting, alongside storage-to-service construction.
Default literals and validation must target the same physical carrier while
retaining all inherited predicates/default selection. Do not add independent
parser/formatter alias exceptions or change the shared generic unalias helper.

## Bounded acceptance

Use the preserved bytes witness plus one aggregate module with named byte/string/
bool/uint32/float elements, an outer named array, inherited element constraints,
optional defaults and required fields. Direct analysis checks primitive physical
storage, original service FieldType, constructor arguments, qualified casts and
constraint/default agreement across request/header/trailer. Mounted checks cover
binary and empty elements, malformed numeric strings, inherited constraint
failure, optional absence/default/explicit supplied values, and required absence.
Retain ordinary scalar/native array controls using the existing five-shape
receipt. Compile/vet complete modules, compare exact generated parent/candidate/
fresh-repeat bytes and enumerate intended differences. No full fixture corpus.

Unsupported nested metadata collections, custom Go codecs and error transport
metadata lifecycle are separate contracts; do not silently add or repair them.

## Selected owner and implementation sequence

Add one private metadata representation query in `grpc/codegen`, called once per
mapping from `extractMetadata`. It returns a detached physical attribute and
recorded source-to-physical element conversion. Unwrap nominal primitive
and outer array declarations; lower one-level array elements to builtin storage.
Use the effective constraint owner's Lowered predicates at each lowered occurrence
so inherited validations are not discarded with declaration edges. Defaults are
selected from the original effective occurrence and rendered for that same
physical attribute. Preserve original FieldType and authored source attributes.

Derive Type, TypeName, TypeRef, slice flags, validation and default literals from
this representation. Existing request/response constructors and CLI argument
assembly already consume MetadataData.Type; keep that route intact. The direct
constructor probe confirms InitStructFields restores qualified service-element
identities from primitive storage for all five kinds. Do not add constructor-local
casts or change generic unalias globally.

The same query records an encoding cast when the source element is nominal.
Resolve its primitive reference with NameScope, not handwritten type spelling.
A shared metadata value encoder consumes that conversion before existing builtin
formatting. Consolidate the request/response formatting decision rather than
adding alias switches to both renderers. A direct string-slice fast path is valid
only if source elements already have primitive string storage; otherwise iterate
with the recorded conversion. Decoder parsing and validation target the physical
variable, while service construction targets the preserved nominal field.

Implementation order: direct lowered representation/constraint/default checks;
shared extraction owner; shared source conversion for encoding; constructor and
CLI consumer verification; bounded module generation and mounted controls. No
wire-key mutation is part of877. #705's frozen expression packet is separate and
will compose by supplying canonical mapped names to the same extraction entry.

## Readiness results

The aggregate module exercises named Bytes, String, Boolean, UInt32 and Float64
array elements in three methods, one per request/header/trailer placement. All
five direct physical-storage checks fail on parent. The named module fails both
client and server compilation; the equivalent native module builds/vets (1.73s).
The original scalar/outer-array controls remain valid. Existing shared constructor
projection passes all five named targets under race (1.813s).

The current/codec-only models reproduce unresolvable nominal transport storage;
the storage-only alternative fails preserved agreement. The complete owner passes
80 distinct states. At readiness, constraint/default runtime behavior remained an implementation
gate; the completed checks below establish that correspondence.

1. [x] Owner inventory and candidate design established.
2. [x] Current direct/full-module failures and native/constructor controls recorded.
3. [x] Models compare current, codec-only, storage-only and complete owner.
4. [x] Independent readiness review, then implementation authorization.

Constraint/default cases were added to the same aggregate specimen, keeping
the bounded matrix to native and named aggregate modules plus the preserved
original named-Bytes witness. The aggregate includes an external qualified service type location. It does not
add custom-codec support. Inspect any generated-name
collision as a separate identity defect instead of changing this storage owner.

## Implementation and validation

Root approved readiness in `/tmp/loom-877-independent-readiness-verdict.json`.
The promised outer-alias, inherited-bound and default cases were added and run
against the parent before production changes. The final parent overlay also
includes the external type package and CLI source: native module passes, named
module fails in both generated transport packages.

`metadataStorage` makes one detached primitive attribute per mapped occurrence
and records the source element cast. Effective constraints are lowered before
nominal edges are removed. `extractMetadata` publishes that physical type,
validation and default literal while retaining the service FieldType. Both
metadata encoders consume the recorded conversion; shared constructors and CLI
assembly consume the same physical type. No key, service type, protobuf or
transport lifecycle policy changes.

Completed bounded checks:

- Two final native/named aggregate modules build, vet and pass mounted race tests
  (9.000s total). Each covers request metadata, success headers and trailers;
  binary bytes, empty elements, inherited min/max, malformed numeric text,
  required absence and optional absent/default/supplied values. Native peer RPCs
  retain InvalidArgument; direct decoding preserves the field diagnostic.
- Parent/candidate/fresh-process repeat contain 29 generated files each. All
  repeat bytes match. Exactly five named-module files change: client CLI,
  client/server codecs and client/server constructor types. Their intended changes
  are primitive variables/defaults/validation, supported parsers/formatters and
  qualified service casts. The named Bytes carrier now also receives the existing
  primitive null-element admission. All native, service and protobuf outputs
  remain identical. No files are added or removed.
- The original explicit-bin five-shape probe builds all five modules (4.320s).
  Its 60 artifacts differ from the parent only in four named-element transport
  codec/constructor files. The other four shapes remain byte-identical.
- Existing affected codec, metadata defaults, credentials and analysis controls
  pass under race (8.085s), as do existing CLI goldens (2.116s). No checked-in
  fixture or golden changes are needed.
- The final owner extraction also plans the existing optional default literal,
  keeping the enlarged analysis within the package maintainability gate. Scoped
  golangci-lint reports zero issues; focused representation/default race tests
  pass (6.090s). Final-owner generation is byte-identical to all 29 already built
  candidate artifacts; the earlier independent-process repeat is identical too.
  Harness formatting is gofmt-equivalent to the executed input.
- Current/codec-only model invariant failures and storage-only agreement failure
  remain the approved design evidence; owned passes 120 generated/80 distinct
  states. Implementation matches the modeled owner, so documentation updates do
  not invalidate that checker result.

Evidence and exact input/artifact hashes are collected in the external final
manifest. Fresh exact final review and root delivery remain outstanding. #705
normalization, designed-error metadata delivery, nested collections and opaque
custom codecs remain outside this atomic repair.
