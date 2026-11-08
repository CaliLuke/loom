# Description resolver identity (#957)

These models are design evidence for the description resolver in
[`roadmap/issue-957-external-references-design.md`](../../../../roadmap/issue-957-external-references-design.md).
The models check the algorithm, not the Go implementation. Concrete correspondence
is recorded below.

## Models

`ResolutionIdentity.tla` runs the proposed worklist:

1. Load each referenced URI that explicit policy can open.
2. Register every identity of the loaded source: its retrieval URI and its
   absolute `$self`.
3. Resolve occurrences to `(document key, decoded pointer)` target keys, after
   alias resolution. The document key is an internal per-source token. It is
   separate from the source's optional URI base.
4. Fail an occurrence only when no further load is possible.

The entry is either retrieved from a file or byte-only (`entryRetrieved`). It
may have no `$self`, an absolute `$self` (`X` or `Hb`), or a relative `$self`
(`RelSelf`).

- **Byte-only entry without an absolute `$self`.** It has no URI base and
  registers no URI identity. Its fragment-only references (`Here`) still
  resolve to its own document key. Its relative reference (`Rel`, that is
  `b.yaml#…`) fails. File confinement does not exist.
- **Relative `$self` on a retrieved entry.** It resolves against the retrieval
  URI to `Fs`.
- **Absolute `$self`.** It re-bases `Rel` onto the https mapping (`Hb`).

The run is compared with a declarative oracle, a least fixed point over the
same inputs. The oracle computes the reachable sources, their identities, the
expected fatal conflicts and the expected outcome of each occurrence.

`FragmentPointer.tla` checks two things over fragment spellings built from
literal and percent-encoded units:

- **Key computation.** Following RFC 6901 §6, the fragment is percent-decoded,
  then split on `/`, then unescaped (`~1`, `~0`).
- **Canonical emission.** Each token escapes `~` as `~0` and `/` as `~1` in a
  single pass.

## Decisions informed

- **Load before alias.** A referenced URI that a mapping or file confinement can
  open is always loaded, even when another document's `$self` already claims it.
  Otherwise a conflict goes undetected, depending on order. Control `aliasShortcut`
  shows this. This rule was added to the design because of the model.
- **Fail only at the fixed point.** An occurrence fails only once the registry is
  final, as OAS 3.2 §4.1.2.1 requires. See `earlyFail`.
- **Keys after aliases.** Target keys are built after alias resolution and
  normalization. See `spellingKeys`.
- **One base per selfless file.** A real path reached through two retrieval URIs
  without `$self` is fatal. See `noBaseCheck`.
- **Sources own identities.** Competing sources that claim one identity are fatal
  even if their bytes are identical. See `canonicalDedupe`.
- **No invented base.** A byte-only entry without an absolute `$self` gets no
  default base, no file confinement and no URI identity. See `inventBase`.
  Document keys are internal tokens, so its fragment-only references still have
  stable keys.
- **Decode, then split.** The fragment is percent-decoded before it is split. See
  `splitFirst`.
- **Single-pass escaping.** Canonical escaping is one pass. See `escapeOrder`.

## Commands

Toolchain:

- Amazon Corretto 25.0.5: `/Library/Java/JavaVirtualMachines/amazon-corretto-25.jdk/Contents/Home/bin/java`
- `tla2tools.jar` v1.7.4: SHA-256
  `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`

Run each configuration from this directory with a fresh temporary metadir:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 4 -metadir "$(mktemp -d)" -config checked.cfg ResolutionIdentity.tla
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 4 -metadir "$(mktemp -d)" -config fragment-checked.cfg FragmentPointer.tla
```

For the negative controls, substitute the configuration names in the table
below. Do not commit metadirs or logs.

## Observed results (2026-11-05, extended no-base revision)

`noBaseCheck` names the rejected alternative that keeps per-retrieval
documents. It is not the absent-base case. That case is checked by
`checked.cfg` and controlled by `invent-base.cfg`.

The `ResolutionIdentity` runs used `-workers 8`.

| Configuration | Expected | Observed |
|---|---|---|
| `checked.cfg` | All of these hold: `TypeOK`, `FatalMatches`, `OutcomeMatches`, `OneKeyPerTarget`, `TransplantComplete`, `NoInventedIdentity` and `Termination` | Pass: 6,315,952 generated, 3,835,998 distinct, depth 9, exit 0 |
| `spelling-keys.cfg` | `OneKeyPerTarget` fails | Fails, exit 12. A byte-only entry spells `Hb#p` and `Hb#pe`. The two spellings produce two keys for one target. |
| `early-fail.cfg` | `OutcomeMatches` fails | Fails, exit 12. A byte-only entry references `X` and `Hb`, and `b` declares `$self: X`. `X` fails before `b` loads. |
| `no-base-check.cfg` | `OneKeyPerTarget` fails | Fails, exit 12. Selfless `b` is keyed once per retrieval URI (`Hb`) and once by its own `#p`. One target ends up with two keys. |
| `canonical-dedupe.cfg` | `FatalMatches` fails | Fails, exit 12. A byte-only entry declares `$self: Hb`, and the prefix mapping loads `b` at `Hb`. No conflict is reported. |
| `alias-shortcut.cfg` | `FatalMatches` fails | Fails, exit 12. Same input: `Hb` resolves to the entry without `b` being loaded. |
| `invent-base.cfg` | `OutcomeMatches` fails | Fails, exit 12. A byte-only entry with no `$self` references its would-be file URI `Fe`. An invented base makes it resolve. The oracle says it fails: nothing is supplied, and no base exists. |
| `fragment-checked.cfg` | pass | Pass: 2,701,446 generated, 1,350,723 distinct, exit 0 |
| `fragment-split-first.cfg` | fail `PercentInvariant` | Fail, exit 12: fragment `/` versus `%2F` |
| `fragment-escape-order.cfg` | fail `RoundTrip` | Fail, exit 12: token `/` emitted as `~01` |

## Assumptions and limits

**Bounds.**

- Three sources: the entry, `b` and `c`.
- Four loadable retrieval URIs. The `file:` ones load only when the entry was
  retrieved from a file.
- One unsupplied absolute URI, `X`.
- `$self` values:
  - entry: `{none, X, Hb, RelSelf}`
  - other documents: `{none, X, Hb}`
- At most two entry occurrences, and one occurrence in each other document.
- Only entry occurrences use the relative spelling `Rel`, so the oracle never
  depends on a non-entry base.
- Two pointer spellings.

**Abstracted.**

- Phase A, the eager loading of exact `--document` mappings, is not modeled.
  Those documents enlarge the initial source set: the operational start state
  and the oracle's `Lfp` seed would both include them. The fixed-point
  argument is unchanged, but this bounded run does not check it.
- `Normalize` and `RelTarget` are tables. They stand for RFC 3986 resolution
  and normalization, which are implementation obligation P1.
- Content, kinds and budgets are not modeled.
- Transplant naming is modeled separately in `../bundle_naming`.
- Fragment units are limited to `a / ~ 0 1 %2F %7E %7e %61`, at most four units
  long. Tokens are at most two tokens of two characters each.

**Not proved.**

- Go URI parsing.
- `os.Root` confinement.
- libopenapi behavior.
- Any arbitrary Go implementation.

## Implementation correspondence

`resolve.go`, `resolve_uri.go` and `resolve_load.go` implement the worklist,
source registry, normalization and confined loading. `TestResolverIdentity`,
`TestResolverNoBase`, `TestResolverFragmentPointer` and `TestResolverNormalization`
cover the model's controls. `TestResolverConfinement`, `TestResolverBudgets` and
`TestResolverNoNetwork` cover concrete boundaries outside its abstractions.

`checkReferenceDepth` in `resolve_depth.go` constructs target containment edges
by walking each reference occurrence's pointer ancestors. Iterative Kosaraju
finds strongly connected components; a topological pass computes their longest
path. The graph passes take O(V+E) time and space, with no recursive Go calls or
simple-path fallback. `TestResolverComponentDepth` covers the 64/65-group
boundary, a 65-node cycle, 64 linked recursive groups and the reviewer's dense
12-node core with 53 bidirectional leaves. `TestReferenceDepthContainment` checks
pointer boundaries, document identity and a referenced document root.
`TestReferenceGraphDepthExhaustive` compares all 512 three-vertex directed graphs
against an independent transitive-closure/simple-path oracle, then reverses node
numbering and duplicates edges. `TestResolverSharedReferenceGraph` retains the
shared 30-layer DAG regression.

### Reference-depth policy model

`ReferenceDepth.tla` models the selected group-depth contract independently of
Kosaraju. Mutual reachability defines groups, an enumerated path oracle defines
the expected depth, and finite relaxation computes it. It covers every directed
graph on three vertices, including self-loops and disconnected graphs. The
`enumerate` control first models the old algorithm's simple-path work on the
complete four-node graph: 64 distinct simple paths already exceed V+E = 20.
This witnesses the old complexity failure before the implementation correction;
it is a finite counterexample, not an asymptotic proof. The old Go code also
fails the new 65-node-cycle admission control before the fix.

Run explicitly from this directory using the same pinned Java/TLC versions:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -config depth-checked.cfg -metadir "$META/depth-checked" ReferenceDepth.tla
```

Run `depth-enumerate.cfg`, `depth-vertices.cfg` and `depth-collapseAll.cfg`
separately as negative controls. Results on 2026-11-06 with Corretto 25.0.5 and
tla2tools 1.7.4 (TLC 2.19): checked passes with 2,048 generated/distinct states (512 initial,
depth 4); enumerate fails BoundedWork in its initial state; vertices fails
ExactDepth after 1,556 states; collapseAll fails ExactDepth after 1,539 states.
Negative controls exit 12; checked exits 0. Vertices overcounts recursion;
collapseAll hides dependency chains. TypeOK holds in the checked model.

The model checks the depth policy, not the SCC implementation, pointer grammar,
resource loading, stack/memory bounds or downstream schema processing. The Go
oracle and concrete regression tests cover those implementation seams within
their stated bounds. The original identity model still does not model budgets.

These models are explicit design tools. They are not part of any build, hook,
CI or release gate.

### Public projection and full occurrence identity

`PublicProjection.tla` separates exact internal identity from safe public display,
and document identity from occurrence identity. Sixteen initial combinations
cover secret-bearing/ordinary keys, moved/local/external occurrences and analysis
versus planner diagnostics. The raw-display, redacted-lookup and document-only
origin controls each still fail their intended invariant (exit 12). The added
`public-planner-early.cfg` reproduces attaching origins only before planning:
`MovedOriginRetained` fails after 42 distinct states, depth 3. Final publication
passes all invariants: 64 generated, 48 distinct states, depth 3 (exit 0).

Run the existing TLC command with `PublicProjection.tla` and respectively
`public-display-legacy.cfg`, `public-lookup-redacted.cfg`,
`public-origin-legacy.cfg`, `public-planner-early.cfg` or `public-checked.cfg`. This is bounded design evidence;
it assumes a correct text redactor and does not prove arbitrary URI parsing,
regular expressions or error formatting. No proof execution enters normal gates.

`diagnostic_display.go` projects detached diagnostics and locations only after
internal selection and provenance lookups. `RedactSourceError` also covers flag
wrappers quoting rejected inputs and preserves errors.Is classification without
exposing a raw secret-bearing unwrap. `attachOrigins` compares both document and
pointer and preserves descendant suffixes. Tests `TestSourceDisplayProjection`,
`TestPublicDiagnosticPrivacy`, `TestOpenAPIImportPrivacy` and
`TestLocalPathItemDiagnosticOrigins` cover actual strings, structured fields,
strict/report/partial surfaces, malformed URI and duplicate-key errors. Internal
source keys stay exact; the model explicitly rejects using redacted display keys
for graph matching. Ordinary nonsecret text remains byte-identical.

Final publication owns origin attachment for every diagnostic family: selected
analysis appends planner diagnostics before `publicDiagnostics`; partial analysis
uses the retained bundle for blocked, warning, omission and skipped-operation
lists. `TestPlannerDiagnosticOrigins` covers local/external, GET/PURGE,
selected/unfiltered/partial and ordinary unmoved paths. The redactor assumption
remains explicit: Go tests `TestPublicDisplayBoundaries` and
`TestPublicCredentialFieldContexts` check tokenization outside this finite model.
Public JSON Pointer fields decode and project each token separately; typed URI
values and prose/quoted text use their own boundaries before the same userinfo
primitive. Tests retain valid/malformed credential refusal and ordinary trailing
email/prose bytes. Internal keys are never redacted before lookup.

### Typed syntax publication before library formatting

The revision-6 extension adds a `syntax` family and `ParsePolicy`. Its finite
witness has two quoted values with a URI-looking prefix: actual credentials and
ordinary prose. `libraryFirst` loses that structural type and models the rejected
prefix heuristic by masking both. `public-library-first.cfg` violates
`NoOrdinaryLoss` after 65 distinct states (depth 3, exit 12). Typed publication
preserves ordinary text and removes credentials: `public-checked.cfg` passes
96 generated / 72 distinct states (24 initial, depth 3, exit 0). Run with the
same explicit TLC command. These replace the earlier family bounds for the
current model; prior negative controls remain configurations, not new run claims.

This model checks ordering/type availability, not Go string parsing. The original
raw-bypass model attempt was rejected as inaccurate correspondence: the actual
old heuristic hid credentials but erased ordinary values. The retained final
counterexample models that observed loss. No proof tools enter ordinary gates.

Implementation correspondence: the resolver validates entry syntax and supported
version before opening supplied roots, reuses its entry tree when binding the
retrieval identity, and precedes libopenapi. Alias validation precedes structural
classification of duplicate URI keys by the existing grammar walker. Security
requirements retain exact key nodes through aliases/merges; generic property
keys and pointer tokens stay token-bounded. Known authored URI metadata supplies
whole-value display context without changing internal lookup keys. Public source
mapping errors return safe reasons without echoing rejected composite arguments;
the flag owner retains useful context without the raw-value wrapper. `TestEntryAdmissionPrecedesLoading`, `TestDuplicateURIKeySourceRoles`,
`TestQuotedSourceProseBoundary` and `TestOpenAPIImportQuotedProse` pair those owners
with actual malformed/ordinary CLI, property, pointer and source-loading controls.

### Complete merge roles and rejected mapping arguments

Revision 7 adds a finite `boundary` (node, members, components, mapping) and
`RolePolicy`. `public-collections-raw.cfg` models roles lost when collection
traversal ignores inherited members; `NoCredentialDisclosure` fails after 237
distinct states. `public-mapping-split.cfg` models echoing a rejected composite
whose malformed URI contains the first separator; the same invariant fails after
285 distinct states. Both controls exit 12, depth 3. The corrected
`public-checked.cfg` passes 384 generated / 288 distinct states (96 initial,
depth 3, exit 0), including `MappingReasonOnly`. Run these configurations with the
same explicit TLC command; other configurations were updated for the new constant
but were not rerun. Earlier state counts above describe their historical model.

This abstraction assumes correct effective merge selection and safe reason text;
it does not prove YAML traversal, separator parsing, or string redaction. The
implementation uses `orderedYAMLMapping` for both grammar collections and
canonicalization, reusing `effectiveYAMLMapping` precedence and original value
nodes. `TestSourceURICollectionRoles` covers node/map/component/array boundaries
across direct, alias, single, sequence and nested merges.
`TestSourceURICollectionOverrideRoles` checks shallow explicit override and
first-source merge precedence. `TestCanonicalMappingMergeOrder` checks exact
canonical bytes before and after the change, including explicit order and sorted
inherited keys. `TestDocumentMappingErrorBoundary` and real CLI privacy tests
check malformed separator/userinfo refusal and successful literal local paths.
These are normal regression tests; model execution remains opt-in only.
