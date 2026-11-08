# Design: import document identity and external reference graphs (#957)

Status: implemented by the atomic #957 change against parent
`b9e92c7c98b2bbc1f082a95bb76de363b55edf27`; bounded generation, runtime, parser, determinism and
independent source qualification are complete. Publication is pending the normal
batch push; this status does not assert a release. The external Security
Requirement URI lint gate passes with the official unreleased correctness pin
recorded in the [acceptance evidence](../internal/openapiimport/external_references_validation.md)
([upstream #1056](https://github.com/daveshanley/vacuum/issues/1056)).
The earlier v0.39.3 failure remains historical evidence. This file retains the
accepted resolver contract consumed by #1005 and #962.

Revision 3 extends the design to the full scope of the ticket:

- Path Item `$ref`, local and external, resolved by the same resolver
- typed discriminator mapping edges for #1005
- precise selection over unknown Path Item membership
- bundled regeneration with retained provenance, as decided policy

A focused post-review revision then settled three points:

- **Naming.** External targets get a total portable-key order. The complete
  entry design-name inventory is reserved before any external declaration is
  named. The inventory covers types, schema variables, security scheme
  variables and error clones (model M4).
- **Identity.** Internal document keys are separate from optional URI bases,
  so a byte-only entry has stable keys but no invented base (extended M1).
- **Security.** Aliases of one scheme within one security requirement object
  merge by scope union, in place of rejection (revised M3).

The required models have now been run (see Formal Models). The independent
full-scope design review and its focused re-check settled the contract. The
implementation follows this accepted design. No unresolved ticket-scope
decision remains.

## Summary

A Loom-owned **description resolver** runs before libopenapi. It builds the
complete OpenAPI Description (OAD) graph in five steps:

1. **Load.** It loads the entry document and every document reachable under
   explicit load policy. It never uses the network.
2. **Register identities.** It registers every identity of each loaded source:
   the retrieval URI and `$self`.
3. **Classify reference positions.** It walks the grammar to classify every
   reference position as a typed edge:
   - component `$ref`
   - Path Item `$ref`
   - Security Requirement key
   - discriminator `mapping` / `defaultMapping` value
4. **Resolve occurrences.** It resolves each occurrence to a canonical target
   key, after alias resolution and only at the registry's fixed point.
5. **Emit a bundled entry YAML tree.** In that tree:
   - Each distinct external component target appears exactly once.
   - Each Path Item occurrence is inlined at its path key.
   - Each unresolved occurrence becomes a per-occurrence *blocked* component or
     Path Item blocker.
   - Every bundled node keeps its source location.

The existing analyzer, selection pruner, planners and renderer then consume the
bundled tree. Root `$self` projects to the existing `Meta("openapi:self", ...)`.
External 3.2 security schemes project to the existing
`Meta("openapi:security:uri", ...)`, under the equivalence rules below.

## Problem Statement and Evidence

Today import stops in three cases:

- **Any non-fragment `$ref`.** Analysis stops before the model exists, and the
  report says that external references are unsupported.
- **`$self`.** It produces a `document-identity` blocker.
- **Path Item `$ref`, even a local one.** It produces `path-reference`.

Loom also cannot re-import two shapes its own exporter emits: a 3.2 document
with `$self`, and URI-keyed security requirements.

| Kind | Evidence |
|---|---|
| Repository evidence | The exporter/importer asymmetry (cited below) |
| Synthetic only | The issue probe; the rejection test at `internal/openapiimport/analyze_test.go:665` |
| Not in the repository | A real consumer contract with multi-file references. Checked-in YAML and JSON contain no `$ref` to `*.yaml` or `*.json`. |
| Missing consumer evidence | Ecosystem layouts such as `paths/*.yaml` and definitions-only files. Under the accepted rule, this gap does not block work on an inexpressible valid contract class. |

## Goals

- **Base URIs and identity.** Derive them as OAS 3.2 §4.1.1, §4.1.2.2 and
  Appendix F require, and resolve references per RFC 3986.
- **Reference coverage.** Import references to local files, supplied documents
  and relative URIs. This holds for every reference position the strict subset
  admits:
  - schemas
  - parameters
  - request bodies
  - responses
  - headers
  - examples
  - security schemes
  - Path Items

  Cycles are covered too. Canonical keys stay independent of spelling and
  alias.
- **Occurrence identity:**
  - Distinct targets are never merged.
  - Alias spellings never duplicate a declaration.
  - Each occurrence keeps its origin, including its Reference Object
    annotations.
- **Explicit policy.** The network and filesystem policy is explicit, confined
  and finite.
- **Shared resolution for other tickets.** Expose discriminator mapping
  resolution as typed edges, so #1005 needs no second loader. Expose Path Item
  edges, so #962 needs no second loader.
- **Byte-for-byte stability.** Output stays byte-for-byte unchanged for every
  document with none of these:
  - `$self`
  - external references
  - Path Item references
  - URI security keys

## Non-Goals (each owned elsewhere or explicit policy)

- **Network retrieval.** The resolver never fetches. This is explicit policy,
  not missing scope. It satisfies the ticket's "explicit network policy; do not
  fetch arbitrary URLs implicitly". Opt-in fetching would need a new ticket.
- **JSON Schema resource identity: `$id`, `$anchor`, `$dynamicAnchor`,
  `$dynamicRef`.** Owned by #1022.
  - The analyzer keeps rejecting these keywords (`analyze_schema_grammar.go:82-86`).
  - The resolver fails closed (§3) for targets whose base these keywords could
    change.
  - Its registry reserves resource identities as a key kind, so #1022 can
    register `$id` URIs in the same registry.
- **`jsonSchemaDialect`, `$schema` and `$vocabulary`.** Owned by #1021.
- **`externalValue`.** Owned by #984.
- **Link `operationRef`.** Owned by #961.
- **`webhooks`.** Owned by #959.
- **Callbacks.** Owned by #960.
- **Media Type components.** Owned by #967.
- **Entry `components.pathItems` declarations and reusable Path Item identity in
  regenerated output.** Owned by #962. #962 consumes this resolver's Path Item
  edges.
- **Discriminator admission.** Owned by #1005. #957 supplies only the resolved
  mapping edges (§5).
- **Runtime behavior.** Nothing changes in the HTTP, gRPC or JSON-RPC runtimes.

## Specification Basis (OAS 3.2.0, verified against the published text)

- **§4.1.1, `$self`.**
  - It supplies the document's self-assigned URI, which is also its base URI.
  - Its value must be a URI reference.
  - A fragment is allowed. RFC 3986 §5.1 strips the fragment before the value
    is used as a base.
- **§4.1.2, document roots.**
  - All documents MUST have an OpenAPI Object or a Schema Object at the root.
  - Other roots MAY be supported, with implementation-defined behavior.
- **§4.1.2.1, parsing.**
  - Each document MUST be fully parsed in order to locate reference targets.
  - Implementations "MUST NOT treat a reference as unresolvable before
    completely parsing all documents". This is the basis for the fixed point
    in §1.
- **§4.1.2.2.1–2, base URI and fragments.**
  - A relative `$self` resolves against the next base source. The retrieval URI
    is the common fallback.
  - Supplying documents with their intended retrieval URIs is recommended.
  - JSON and YAML fragments are interpreted as JSON Pointers (RFC 6901). Per
    RFC 6901 §6, the fragment is percent-decoded before the pointer is
    evaluated.
- **§4.1.2.3 and Appendix G.3, implicit connections.**
  - It is RECOMMENDED to resolve Components Object names and Tag names from the
    entry document. This covers Security Requirement names and discriminator
    mapping names.
  - "No aspect of implicit connection resolution changes how URIs are
    resolved."
  - The resolution process in a multi-document OAD is implementation-defined.
- **§4.9.1, Path Item `$ref`.**
  - The value MUST be a URI, and the referenced structure MUST be a Path Item
    Object.
  - If a field appears both in the referencing object and in the referenced
    object, the behavior is undefined.
- **§4.25.3, discriminator mapping values.**
  - A mapping value is either a schema name or a URI reference.
  - A value that is both is implementation-defined. It is RECOMMENDED to treat
    it as a name.
  - A `.` path-segment prefix (for example `./foo`) forces URI interpretation.
- **§4.30, Security Requirement keys.**
  - A key is a name or a URI.
  - A key identical to a component name MUST be treated as a component name.
  - Component names that look like URIs are NOT RECOMMENDED.
- **Appendix G.1 and G.2.**
  - Parsing a fragment in isolation is undefined in 3.1 and later.
  - Ignoring base-changing keywords is a security risk.
  - Interpreting one object in two contexts that require different types is
    implementation-defined, and may be an error.

## Current State (verified at `f0c7a2b8c`)

**CLI input:**

- The CLI passes only bytes (`cmd/loom/import.go:178-182`) and rejects stdin
  (`:84-86`).
- `--list-tags` fails whenever the analysis document is nil (`:254-266`). Today
  that includes every input with an external reference.

**Analyzer entry:**

- `libopenapi.NewDocument(source)` uses the default configuration
  (`analyze.go:45`).
- `externalReferenceDiagnostics` flags every `$ref` key whose value does not
  start with `#` (`:109-141`). It is grammar-blind, so it also flags `$ref` keys
  inside example or default data.
- When that check fires, analysis returns a nil document (`:54-60`).

**`$self` and `jsonSchemaDialect`:**

- Both produce `document-identity` (`analyze.go:164-166`).
- Partial import treats that diagnostic as a document-level omission
  (`partial.go:126-127`).

**References:**

- Model `Ref` fields hold local component references
  (`model.go:219,256,290,316,334`).
- `openapiref.Name` accepts fragment-only references with exactly one component
  token (`internal/openapiref/reference.go:15`).
- The coverage guide already treats canonical regeneration of local reference
  spellings as preserving the reference
  (`docs/openapi-import-coverage.md:258-262`).

**Path Items:**

- A Path Item `$ref` is rejected as `path-reference` (`selection.go:154-156`).
- Entry `components.pathItems` produce `component-kind` (`analyze.go:229-232`).
- #962's probe shows that libopenapi already resolves a local Path Item
  reference into operations, but identity is lost.

**Operation identities:**

- The importer emits `Meta("openapi:operationId", ...)` (`render.go:297`).
- The exporter rejects a duplicate operationId at generation time
  (`ValidateOperationIdentities`,
  `http/codegen/openapi/internal/ir/document_identity.go:35-80`).

**Selection:**

- `pruneComponents` builds a closure over five component kinds plus tags
  (`selection.go:211-240`).
- `filterSelectionDiagnostics` keeps a component diagnostic only when that
  component is in the closure (`:404-437`).
- Example and security-scheme diagnostics are not filtered by closure.
- Partial import filters per operation, by closure and by operation path
  (`partial.go:57-59,162-176`).
- Whether a diagnostic is fatal or lossy is decided by its code
  (`diagnostics.go:49-61`).

**Security:**

- Requirement keys must name entry components (`security.go:255-261`).
- `openapi:security:uri` is read per scheme (`builder.go:533`).
- The 3.2 export pass globally deletes that component and rekeys every
  requirement that names it (`openapi32.go:279-317`).

**Exporter:**

- `openapi:self` is documented (`dsl/meta.go:258-259`) and emitted
  (`openapi32.go:53-55`).
- 3.1 output strips it (`openapi31_filter.go:16`).

**libopenapi v0.42.3 (read-only reference):**

- Its index has no `$self` base logic.
- `BasePath` indexes every YAML or JSON file beneath it
  (`datamodel/document_config.go:49-55`).
- Remote lookup sits behind `AllowRemoteReferences` (`:99`).

## Proposed Design

**Owner.** The resolver in `internal/openapiimport` is the only code that:

- reads bytes from documents other than the entry
- interprets URIs
- registers identities
- classifies reference positions
- assigns targets

It replaces `externalReferenceDiagnostics`. libopenapi keeps its default
configuration and only ever sees local references.

The rolodex is rejected for five reasons:

- It ignores `$self`.
- It scans whole directory trees.
- It cannot provide per-occurrence provenance.
- It bypasses Loom's raw-source guards.
- It mixes network capability into resolution.

#1005, #962 and #1022 consume or extend this owner. None of them adds a loader.

**Grammar-aware positions.** The resolver walks each document using the
OpenAPI grammar. The table lists the positions it treats as reference edges.

| Edge kind | Position | Resolution |
|---|---|---|
| `component` | Reference Object `$ref`, and Schema `$ref`, in schema, parameter, request body, response, header, example and security scheme contexts | URI, resolved against the containing document's base |
| `pathItem` | Path Item `$ref` under `paths`. The walker classifies Path Items under `webhooks` and callbacks but does not admit them. | URI, resolved against the containing document's base |
| `securityKey` | Security Requirement property names: root and operation | Name, resolved from the entry's components (§4.30, G.3). Otherwise a URI, resolved against the containing document's base. |
| `discriminatorMapping` | Each `mapping` value and `defaultMapping` | Name, resolved from the entry's `components/schemas` (G.3). Otherwise a URI, resolved against the containing document's base. |

A `$ref` that appears inside data is never a reference. Data positions are:

- example, default, const and enum values
- `x-*` extensions

Inside a transplanted target, nested positions follow the grammar of that
target's kind.

**Representation.** The shapes are illustrative.

```go
type sourceID string              // internal document key: one per loaded source, never a URI.
                                  // The entry always has the private token "entry"; other sources
                                  // derive theirs from the symlink-resolved real path.
type docURI string                // optional normalized absolute URI without fragment
type document struct {
	key      sourceID
	base     *docURI              // nil when no base exists (byte-only entry without absolute $self)
	canonical *docURI             // absolute $self if any, else the retrieval URI; nil exactly when base is nil
	portable portableKey          // total-order and report key (§3)
}
type targetKey struct{ doc sourceID; pointer string } // pointer: percent-decode, then JSON Pointer tokens
type sourceLocation struct{ doc sourceID; portable portableKey; pointer string }
type edge struct {
	kind     edgeKind     // component(k) | pathItem | securityKey | discriminatorMapping
	at       sourceLocation; authored string
	target   *targetKey   // nil when failed or when the value names an entry component
	entry    string       // entry component name for name-form keys and mappings
	external string       // canonical absolute URI for an unsupplied non-file mapping target
	failure  *Diagnostic  // set when no other outcome applies
	bundled  string       // entry pointer of the occurrence in the bundled tree
}
type registry struct {
	alias     map[docURI]sourceID   // normalized identity URI -> its unique loaded source
	realPaths map[string]sourceID   // symlink-resolved path -> loaded source
	docs      map[sourceID]document // bases and canonical URIs assigned only after conflicts are checked
}
```

Target keys never depend on a URI. Two spellings or aliases of one document
reach the same `sourceID` through the alias registry. A document without a base
still has a key, so its fragment-only references stay stable. URIs are used only
for three things:

- resolving references
- registering aliases
- emitting regenerated spellings

`Diagnostic` gains `Origin *SourceLocation`:

- **`Path`.** It remains the JSON Pointer in the **bundled entry**. For an entry
  node, that is also its source pointer.
- **`Origin`.** It is non-nil exactly when the node came from another document,
  or from an inlined Path Item. It records the `portableKey` (§3), the source
  pointer and the authored spelling.
- **Unchanged keying.** Selection, partial ownership, `isDocumentOmission` and
  `uniqueDiagnostics` keep keying on `Path`.
- **Reports.** A report prints the origin and then the bundled path, for
  example: `models.yaml#/components/schemas/Pet (bundled as #/components/schemas/Pet_2)`.

## Detailed Behavior

### 1. Identity, aliasing and canonical keys

**Invariant.** A target key is `(sourceID, decoded pointer)`. It is built only
after alias resolution. Every spelling or alias of the same document and
pointer therefore yields one key and one declaration.

```text
normalize(u): RFC 3986 §6.2.2 syntax normalization (case, percent-encoding, dot segments)
              + §6.2.3 scheme normalization for http/https (default port, empty path "/"); drop fragment
pointer(f):   percent-decode f once, then split on "/", then unescape "~1" then "~0" per token (RFC 6901 §6)
emit(tokens): "/" + join(escape each token: "~"→"~0" and "/"→"~1" in one pass), percent-encoding
              only characters not allowed in an RFC 3986 fragment
selfURI(doc)    = normalize($self)                           if $self is absolute
                = normalize(resolve($self, retrievalURI))    if $self is relative and retrievalURI is known
                = none                                       otherwise (relative $self without a retrieval URI)
identities(doc) = { normalize(retrievalURI) if known } ∪ { selfURI(doc) if not none }
base(doc)       = selfURI(doc) if not none, else normalize(retrievalURI) if known, else none
canonical(doc)  = base(doc)
resolve(ref, doc):
    ref is fragment-only        -> target in doc's own sourceID (RFC 3986 §4.4; no base needed)
    ref is an absolute URI      -> that URI
    base(doc) is none           -> fail "the entry has no base URI to resolve <ref>"
    otherwise                   -> RFC 3986 §5.2 resolution against base(doc)
```

Only the entry can lack a base. Every other document is loaded through a
retrieval URI.

**Registration.**

1. Phase A runs eagerly. It loads the entry, then every exact-file `--document`
   mapping, in CLI order.
2. Phase B loads documents lazily, from a worklist, through prefix mappings and
   `file:` confinement.
3. **Source identity.**
   - Each loaded document registers all of its identities.
   - Its `sourceID` comes from the symlink-resolved source path, independent of
     `$self` and the retrieval URI.
   - A byte-only entry has its own source token.
   - Loading the same real path again reuses its source. Equal bytes at
     different paths do not share a source.
4. **Load before alias.** A referenced URI that an exact mapping, a prefix
   mapping or `file:` confinement can open is always loaded, even when an alias
   already names it. Without this rule, an entry with `$self: https://e.test/b.yaml`
   and a prefix mapping that loads `b.yaml` at that URI would resolve to the
   entry and never detect the conflict. Model control `aliasShortcut`
   demonstrates this.
5. **Conflicts.** These conflicts are fatal and order-independent:
   - **Identity owned by another source.** An identity already maps to a
     different `sourceID`. This applies even when both sources claim the same
     canonical URI. Check it before bases and canonical URIs are assigned and
     before target keys are deduplicated.
   - **Two bases for one file.** A real path already backs a document with a
     *different retrieval URI*, and either copy lacks `$self`.
6. **Aliases.** A real path reached again under a different retrieval URI, with
   the same `$self`, adds an alias to the existing document. It never creates a
   second document.

**Lookup is a fixed point.**

1. Load and resolve until the worklist is empty.
2. Re-check pending occurrences against the final alias registry.
3. Repeat while new documents load.

An occurrence fails only once the registry is final (§4.1.2.1). As a result, a
document that later declares `$self: X` satisfies an earlier reference to X.

**Same-document references.** A reference whose normalized document equals any
entry identity resolves to an entry component. For example, all of these
resolve to `Pet`:

- `openapi.json#/components/schemas/Pet`
- the absolute `$self` spelling of the same target
- `#/components/schemas/Pet`

A fragment-only reference inside an external document resolves against *that*
document.

**Byte-only `Analyze`** has no retrieval URI, no reference root and no supplied
documents. The entry keeps its private `sourceID` whether or not it has a base.
Loom never invents a default base. Appendix F.4 permits an application-defined
default; Loom does not adopt one.

| Entry `$self` | Identities and base | Fragment-only refs | Absolute refs | Relative non-fragment refs |
|---|---|---|---|---|
| absolute `X` | identity `X`; base `X` | resolve in the entry | back to `X`: resolve in the entry; others fail "not supplied" | resolve against `X`; then as for absolute refs |
| relative | none; no base | resolve in the entry | fail "not supplied" (nothing is loadable) | fail "the entry has no base URI to resolve <ref>" |
| absent | none; no base | resolve in the entry | fail "not supplied" | fail "the entry has no base URI to resolve <ref>" |

**Relative `$self`.**

- **Projection.** A relative `$self` is still projected verbatim (see the table
  below), and it is never registered as an identity.
- **No base.** Without a base, a reference spelled like the relative `$self`
  (for example `openapi.json#/...` with `$self: openapi.json`) is itself a
  relative reference. It fails. Loom does not guess that it means the entry.
- **Retrieved entry.** On the CLI the entry is retrieved from INPUT's real
  path, so a relative `$self` resolves against that retrieval URI and does
  register as an identity.
- **Byte stability.** Documents with no non-fragment references keep
  byte-identical output in every row.

Model M1 checks these rules (`NoInventedIdentity` and control `inventBase`).

**Projecting `$self`.**

| Input | Result |
|---|---|
| A 3.2 entry `$self` that is a valid URI reference: absolute or relative, with or without a fragment | Projected verbatim to `Meta("openapi:self", authored)`. Identity and base strip the fragment. 3.1 output omits it, as today. |
| `$self` that is not a valid URI reference | Fatal `document-identity` |
| `$self` in a 3.0 or 3.1 document | `versioned-field`, "$self requires OpenAPI 3.2". The document's base is its retrieval URI. |
| `jsonSchemaDialect` | New code `json-schema-dialect`, still in the document-omission class. `document-identity` is reserved for invalid `$self` and identity conflicts. |
| `$self` of a non-entry document | Used for identity and base only |

A relative `$self` is preserved verbatim. After regeneration, its meaning
depends on where the document is published, and Loom does not control that
location in either direction.

### 2. Load policy, confinement and budgets

```go
func load(abs URI) (bytes, retrievalURI, error) {
	if abs.User != nil { return failNoEcho() }               // never print userinfo
	if m, ok := exactMapping[abs]; ok { return readExplicit(m.path) }  // user-named file: not root-confined
	if m, ok := longestPrefixMapping(abs); ok {               // prefix ends in "/", same scheme+authority
		rel, err := uriSuffixToRelPath(abs, m.prefix)       // rules below
		return m.root.Open(rel)                              // os.Root per mapping directory
	}
	if abs.Scheme == "file" && entryRetrievedFromFile {
		rel, err := fileURIToRelPath(abs, referenceRootRealPath)
		return referenceRoot.Open(rel)                       // os.Root at --reference-root
	}
	return fail("target %s is not supplied; pass --document", portable(abs))
}
```

**CLI.**

- `--document URI=PATH` adds a mapping. If URI ends in `/`, it is a prefix
  mapping; otherwise it is an exact mapping.
- `--reference-root DIR` confines `file:` loads. The default is INPUT's real
  directory, and `--reference-root` must contain INPUT's real path.
- stdin stays rejected.
- Two mappings with the same URI are a CLI error.

**Mapping ownership.**

- An exact mapping beats any prefix mapping.
- The longest matching prefix wins.
- Each prefix mapping owns its own `os.Root`, opened at the symlink-resolved
  real path of its directory.

**Turning a URI into a relative path.**

- The URI is already normalized, so encoded and literal dot segments are already
  gone.
- A query is rejected.
- A non-empty `file:` host other than `localhost` is rejected.
- The remainder is split on `/`, and each segment is percent-decoded exactly
  once.
- A decoded segment is rejected if it:
  - is empty, `.` or `..`
  - contains `/`, `\`, NUL or a Windows volume separator
- For `file:` URIs, the path is then made relative to the root. A relative path
  that starts with `..` is rejected.
- `os.Root` rejects symlinks that leave its root.

**Real-path identity.**

- The real path, with symlinks resolved, feeds `registry.realPaths`.
- Two symlink spellings of one file therefore share one document.
- Hard links are not unified. This limitation is documented.

**Network policy.** The resolver performs no network access of any kind: no
fetch, no redirect following and no DNS. Remote documents must be supplied
through `--document`. This is the explicit policy the ticket requires.

**Budgets.** These are fixed defaults. Any breach is a fatal
`external-reference` error: "resolution budget exceeded".

| Limit | Value |
|---|---|
| Non-entry documents | 256 |
| Bytes per non-entry document | 16 MiB |
| Aggregate non-entry bytes | 64 MiB |
| Dependency depth between recursive reference groups | 64 groups |
| Path Item expansion chain | 64 Path Item objects |
| YAML alias expansion | libopenapi's existing limit |

A reference group is a strongly connected component of the resolved target graph.
A vertex is a `(document key, JSON Pointer)` target; its outgoing edges are the
resolved reference occurrences contained below that target. A group counts once
on a dependency path, including singleton groups. The bound is the longest path
in the acyclic graph of groups. Thus a chain of 64 targets is accepted and 65 is
rejected; one recursive group of 65 targets is accepted within the document/byte
budgets. The entry root contributes a vertex only if it is itself a reference
target. Disconnected graphs use the maximum depth. Path Item cycles remain
undefined and rejected; their finite expansion retains the separate 64-object
limit, including the initial occurrence.

The user delegated this policy choice to framework judgment. Exact longest
simple paths were rejected because checking dense cyclic input can require
factorial work. A separate work budget would make rejection depend on analysis
order and machine-independent but arbitrary bookkeeping. Recursive-group depth
expresses the intended dependency limit and permits iterative graph analysis in
O(V+E) time and space once containment edges are constructed. No recursive DFS or
exact-path fallback is retained. This limit does not claim to bound downstream
schema processing or the intentionally unbudgeted entry document's size.

The entry stays unbudgeted, as today. The visited set over `targetKey`
guarantees termination independently of these limits. Model check
`Termination` confirms this within its bounds.

**Document classes.**

| Root | Interpretation | Accepted targets |
|---|---|---|
| OpenAPI Object | Same minor line as the entry; otherwise the occurrence fails with "version mismatch" | Schema edges target actual Schema Object or boolean-schema positions identified by the description grammar, including inline alternatives and properties. Other kinds require `#/components/<kind>/<name>` with a matching kind. Path Item edges may also target `#/paths/<key>`, `#/components/pathItems/<name>` and `#/webhooks/<name>`. |
| Schema Object, or another mapping such as a definitions-only or Path Item root | Parsed as a whole document. The target is interpreted in its referencing context, under the entry's dialect (Appendix G.2; implementation-defined, adopted policy). | Any pointer, including `""` |

**Schema-role reconciliation with #1005.** The prior component-root-only table
is superseded for schemas so URI mappings and ordinary references use the same
admission owner. Accepted examples include `#/components/schemas/Pet/oneOf/0`,
`#/components/schemas/Pet/oneOf/0/properties/id` and response-content schemas.
The `properties` dictionary, `required` array, Operation Objects and example or
default data that resembles a schema remain rejected. Rejecting inline mappings
would drop accepted #1005 scope; a mapping-only resolver would duplicate source
policy; accepting arbitrary mapping nodes would erase object-kind boundaries.
The existing grammar identifies roles. This changes no version, dialect,
resource, source-loading or network policy and grants no discriminator admission
by itself. The source-contract model and role tests record these alternatives.

**Fail closed on schema resources.** A target occurrence is rejected with
`schema-resource` in either case:

- The target, or any ancestor of it, carries `$id`, `$anchor`,
  `$dynamicAnchor` or `$schema`.
- The target contains `$dynamicRef`.

Those keywords can change the base or the dialect. #1022 and #1021 own their
admission.

**Source contracts before transplant.** Retain an OpenAPI document's
`jsonSchemaDialect` at the document owner. A nonempty custom dialect blocks
schema-bearing referenced targets with `json-schema-dialect`, including schemas
nested in request bodies, responses, headers, parameters and Path Items. The
same typed grammar used to discover reference edges identifies those schemas;
example data and schema-free targets do not inherit a blocker. Blockers remain
occurrence-owned, so selection removes ones outside the selected dependency
closure. The entry's existing document-level dialect diagnostic is unchanged.
Loose dictionaries do not acquire OpenAPI semantics from a member named
`jsonSchemaDialect`.

Validate explicit YAML mapping-key uniqueness before canonicalization or merge
expansion. Reject repeated explicit keys (including repeated merge keys) rather
than choosing the last value. Legal merge-source overlap and explicit overrides
of inherited keys retain their existing shallow, first-source-wins semantics.
Validation operates on syntax nodes so exact numeric tokens remain intact.

**Context conflicts (Appendix G.2).** One target node may be reached from
contexts that require different Object types, for example as a Schema and as a
Path Item. In that case, every occurrence of that node fails with
`reference-context`. The outcome depends on the final occurrence set, not on
traversal order.

### 3. Transplant and naming

Each distinct resolved component `targetKey` outside the entry is copied once
into `#/components/<kind>/`.

- **Copy.** The copy is a verbatim YAML node copy. Before copying, the effective
  values of YAML aliases and merges are computed inside the source document,
  under the merge-ownership owner. The copy preserves exact scalar tokens, so
  `numericSources` stays accurate.
- **Nested references.** Nested references are rewritten to their resolved
  bundled pointers.
- **Source locations.** Every bundled node maps to a `sourceLocation`.

**Portable keys and the external total order.** Every non-entry source gets a
`portableKey`. The key never contains an absolute checkout path, and it never
depends on load or traversal order.

```text
portableKey(src) =
    (-1, "entry")                     if src is the entry (for introduced local targets)
    (0, "root:" + slash-joined path of canonical(src) relative to the reference root)
                                      if canonical(src) is a file: URI under the reference root
    (1, canonical(src))               otherwise (an absolute $self, or a URI named by a
                                      --document exact or prefix mapping)
order(target) = (portableKey(target.doc), pointerTokens(target.pointer), kind)
```

- **Comparisons.** `pointerTokens` compares decoded tokens lexicographically by
  byte. Tuples compare element by element. Class -1 precedes 0, then 1.
- **Mapping namespaces.** A prefix-mapped document is keyed by its own mapped
  URI, for example `https://a.test/models.yaml`. It is never keyed by the
  mapping directory's file path. Two mapping roots that both contain
  `models.yaml` therefore yield distinct keys:
  `(1, "https://a.test/models.yaml")` and `(1, "https://b.test/models.yaml")`.
- **Exact mappings.** An exact mapping is keyed by its mapping URI.
- **Totality.** Ties are impossible:
  - **Class 1.** Canonical URIs are unique per source, because a conflict is
    fatal (§1).
  - **Class 0.** Root-relative paths are unique per canonical `file:` URI.
  - **Pointers.** A pointer is unique within a document.
  - **Kinds.** A node reached in two kinds fails as `reference-context`, so
    `kind` only separates distinct namespaces.
- **Machine independence.** The order depends on the checkout location only if
  the author wrote an absolute `file:` URI into `$self` or a mapping. That is
  the author's input. In that case Loom emits the URI as authored, and P4
  documents this exception.

**Raw keys (owner: the resolver's transplant step, per component kind).**

1. Claim every entry key exactly as authored. Entry raw keys never change.
2. For each introduced target (external or local non-component Schema), in `order`:
   1. Compute the base raw key. Use the original component key. For a target
      that is not a component, use the last non-empty pointer token, or else
      the file stem.
   2. Sanitize the base to the component-key grammar.
   3. Claim the first free key among `base`, `base_2`, `base_3` and so on.
      Comparison is exact and case-sensitive, because component keys are
      case-sensitive.
3. Blocked keys (`x-loom-unresolved-<n>`) follow the same order, in their own
   numbering (§6).

Only authored entry component roots keep their existing pointers. Admitted local
nested Schema targets use the same canonical target-key promotion as external
targets. All ordinary references and URI mapping outcomes point to that one
allocated component; inline and promoted occurrences retain the same source
Document/Pointer. Local targets sort before external documents, then by decoded
pointer tokens. A rewritten local target makes the bundled tree the normalization
input. Keeping arbitrary entry pointers was rejected: downstream component
references cannot consume them. Promoting every entry component was also rejected:
it needlessly changes authored declarations and their stable names.

**Design names (owner: one allocator for the whole design-name inventory).**

The imported design has two shared namespaces:

- **T, design type names.**
  - schema Go names: `assignSchemaNames`, `names.go:20-28`
  - error-clone type names: the `NameScope` in `render_index.go:57-71`
- **V, design variables.**
  - schema variables, which are always `"Imported"+GoName` (`render.go:112`,
    `render_schema.go:147`, `plan.go:135`)
  - security scheme variables: `assignSecuritySchemeNames`, `names.go:30-42`
  - error-clone variables: `render_index.go:72`

Today `assignSchemaNames` sorts every schema by raw `Name`.
`assignSecuritySchemeNames` reserves the variable of every schema before
allocating any scheme. These allocators are case-insensitive `uniqueName`
passes over the whole namespace, so an external declaration processed first
can take an entry declaration's name. A schema's variable cannot be suffixed,
because the reservation in `assignSecuritySchemeNames` discards the value that
`uniqueName` returns.

Counterexamples:

- **Types.** External `PetStore` takes entry `pet_store`'s type name
  `PetStore`, and the entry becomes `PetStore2`. Likewise, external
  `FooSecurity` and entry `Foo_Security` push the entry to `FooSecurity2`.
- **Across namespaces.** Entry security scheme `Foo` gets `ImportedFooSecurity`
  when imported alone. External schema `FooSecurity` reserves the variable
  `ImportedFooSecurity` first, which renames the entry scheme to
  `ImportedFooSecurity2`. Per-namespace two-pass allocation, as in revision 4,
  still does this.

One function, `allocateDesignNames`, owns the whole inventory. It runs on the
final bundled document before rendering. It replaces the separate orderings in
`assignSchemaNames`, `assignSecuritySchemeNames` and `assignErrorCloneNames`.
Those become consumers of its result.

```text
allocateDesignNames(doc):
  T, V := {}, {}                                   // case-insensitive claims (uniqueName keys)
  // Phase E: the complete entry inventory, in today's order and with today's algorithms
  E1 for entry schemas sorted by raw Name:   s.GoName := uniqueName(Goify(s.raw), T)
  E2 for entry schemas sorted by raw Name:   claim "Imported"+s.GoName in V
  E3 for entry schemes sorted by raw Name:   k.GoName := uniqueName("Imported"+Goify(k.raw)+"Security", V)
  E4 for entry-origin error clones, in today's operation/response order:
         c.type := first free in T (NameScope rule); c.var := uniqueName("Imported"+c.type+"Error", V)
  // Phase X: introduced declarations (local promotions and external), in order(target)
  X1 for introduced schemas: s.GoName := first c in Goify(raw), Goify(raw)2, Goify(raw)3, ...
         with c free in T AND "Imported"+c free in V;   claim both
  X2 for introduced schemes: k.GoName := uniqueName("Imported"+Goify(k.raw)+"Security", V)
  X3 for introduced error clones: as E4, after X2
```

**Classification and claims.**

- **Declaration priority.** `declarationOrigin.introduced` records whether a
  declaration was introduced, independently of its source document. Local nested
  promotions and external declarations follow the complete authored inventory.
  A clone is introduced when its schema or a contributing response is introduced.
  Moved local Path Item operations keep their local operation classification.
  Exact SourceLocation identity remains unchanged. A promoted URI target does
  not gain an implicit discriminator component-name mapping.
- **Clone decisions.** The clone decision itself (`indexErrors`, `render_index.go:12-54`) is
  computed before naming by the same predicate, moved into the owner.

**Guarantees.**

- **Entry stability.** Phase E sees only entry declarations and runs today's
  algorithms in today's order. Every entry type name and variable therefore
  equals what an entry-only import allocates. Documents without introduced
  declarations are byte-identical.
- **Cross-namespace safety.**
  - A schema variable is always `"Imported"+GoName`, and X1 chooses only
    GoNames whose variable is free.
  - Scheme and clone variables are allocated by `uniqueName` against all of V.
  - No two declarations share a type name or a variable.
  - In the reviewed pair, the entry keeps `ImportedFooSecurity`. The external
    schema becomes type `FooSecurity2`, with variable `ImportedFooSecurity2`.
- **Determinism.** Phase E uses raw-name order, as today. Phase X uses the
  §3 portable total order. Suffixes use `uniqueName`'s first-free sequence.

**Scope of the owner.**

- **Repeated calls.** Selection pruning (`selection.go:345`) and analysis
  (`analyze.go:180-204`) call the same owner on their current document. It is a
  pure function of that document, so repeated calls are consistent.
- **Other component kinds.** Parameter, request-body, response, header and
  example keys need no design names here. Their raw keys come from the raw-key
  step.

**Evidence.** Model M4 (`bundle_naming`) covers schema types, fixed schema
variables and scheme variables across every external order. It checks entry
stability and the absence of duplicates. It reproduces the today and
revision 4 counterexamples on the reviewed pair. Error clones are not modeled;
obligation P4 covers them.

**Shared and cyclic targets.**

- Shared targets stay shared.
- Cycles become component cycles and use the existing recursive-reference
  plan. An initializer-only cycle still yields
  `schema-reference-initialization`.
- Reference-occurrence `summary` and `description` stay with each occurrence,
  under the existing Reference Object overlay rules.
- Byte-identical content at distinct keys stays distinct.
- After bundling, the existing local rules apply. That includes the rejection
  of cycles made only of aliases.

### 4. Path Item references

**Invariant.** Each `paths/<key>` occurrence imports exactly the operations of
its effective Path Item, with:

- its own bundled path
- the origin of each inlined node

A shared Path Item target never merges two path keys. A failed or undefined
Path Item never contributes operations.

```text
effective(key, node, doc):
  parts := []; seen := {}; cur := (node, doc)
  while cur.node has "$ref":
      e := resolve pathItem edge (cur.node.$ref against base(cur.doc))
      if e.failure            -> blocker(path-reference, "#/paths/<key>", Origin=e.at, cause)
      if e.target in seen     -> blocker(path-reference, ..., "Path Item reference cycle")   // no finite meaning
      if hops > 64            -> fatal budget
      seen += e.target; parts.append(fields(cur.node) - "$ref" in cur.doc); cur = node(e.target)
  parts.append(fields(cur.node) in cur.doc)
  if a field name occurs in two parts -> blocker(path-reference, ..., "field <f> appears in both the
                                          referencing and referenced Path Item; OAS 3.2 §4.9.1 leaves this undefined")
  return union(parts)   // each field keeps its own document for base and origin
bundled tree: "#/paths/<key>" := effective, with nested references in each field rewritten
              against that field's own document; every inlined node keeps its Origin
```

- **Scope.** Local and external Path Item references use this same owner. The
  separate `path-reference` rejection at `selection.go:154-156` is deleted.
- **Fields.** "Field" includes every Path Item fixed field and every `x-*`
  extension.
- **Existing diagnostics still apply** at the bundled path, with origin:
  - `path-metadata`
  - `servers`
  - unsupported operations
- **Security requirements and tags** inside inlined operations follow §5 and
  §7: names come from the entry, and URIs resolve against the base of the
  operation's own document.
- **Operation identity.**
  - Operations reached through two path keys are two operations.
  - If they carry the same `operationId`, the analyzer reports `operation-id`
    at each duplicate operation's bundled path. Two operations claiming one ID
    violate the OAS 3.2 `operationId` MUST, and the exporter rejects them in
    `ValidateOperationIdentities`.
  - The check runs in the analyzer over the selected operation set, so it
    applies to every input.
  - **Compatibility effect.** A single document with authored duplicate
    operation IDs used to produce a design that `loom gen` rejected. It now
    fails at import.
  - In partial mode, every operation in a duplicate group is skipped and
    listed.
- **Entry `components.pathItems`.** Targets there resolve like any other target.
  The declarations themselves keep `component-kind` until #962 defines how
  their identity is preserved.

  #962 consumes the resolver's `pathItem` edges and their targets. It neither
  re-resolves nor loads.

### 5. Discriminator mapping edges (consumed by #1005)

**Invariant.** Every `mapping` value and every `defaultMapping` value in an
admitted document becomes exactly one typed edge. The edge has a deterministic
outcome. That outcome does not depend on whether #1005 admits the
discriminator.

```text
classify(v):  name if v matches ^[A-Za-z0-9._-]+$ (§4.25.3 recommendation), else uri
              ("./Cat" is a uri; "Cat" is a name even if a file named Cat exists)
outcome(edge in document D):
  name : entry components/schemas has v      -> entry(v)                 // G.3: entry document
         otherwise                          -> failed("no entry schema component <v>")
  uri  : t := resolve(v, base(D)); fixed-point lookup as for kind schema
         loaded                             -> bundled(transplantKey(t))  // joins the transplant set
         absolute, non-file, not loadable   -> external(emit canonical absolute URI)   // never fetched
         otherwise                          -> failed(cause)
bundled tree: uri-form values that resolve become "#/components/schemas/<key>";
              an external value keeps its authored spelling only if it resolves identically
              against the regenerated base (§7 rule), else becomes the canonical absolute URI;
              name-form values are unchanged (entry keys never change)
```

**The resolver records outcomes and never blocks on them.** #1005 owns:

- admission
- blocking
- implicit mapping
- the closure edges that keep mapped components selected

#1005 admits discriminators from these outcomes. Bundled branches are local
components, so a mapping URI to a loaded alternative binds to that branch. The
admission rules are in
[Discriminators](../docs/openapi-import-coverage.md#discriminators).

### 6. Failure representation and selection

**Invariant.**

- A failed occurrence blocks exactly the imports whose selected closure reaches
  it.
- A failed Path Item blocks every selection that might include it.
- No failure prevents the model from being built, and no failure renders as a
  widened schema.

```text
for each failed component occurrence o (kind k):
  bundled tree: replace o's Reference Object with $ref "#/components/<k>/<blockedKey(o)>"
  add component <blockedKey(o)> with a minimal kind-valid body:
      schema {} | parameter {name, in: query, schema: {}} | response {description}
      requestBody {content: {}} | header {schema: {}} | example {}
      | securityScheme {type: apiKey, name: x-loom-unresolved, in: header}
  record blocker: Diagnostic{Code: "external-reference",
                             Path: "#/components/<k>/<blockedKey(o)>", Origin: o.at, Message: cause}
blockedKey(o) = first free key "x-loom-unresolved-<n>" in that namespace; n follows the occurrence order
                (portableKey(o.at.doc) with the entry as class -1, then the occurrence's pointer tokens)
for each failed Path Item occurrence at key p:
  bundled tree: "#/paths/<p>" := {} (no operations); record blocker{Code: "path-reference", Path: "#/paths/<p>", Origin}
  SelectionReport.UnresolvedPaths += p
```

**Component blockers.**

- **One blocked component per occurrence.** Each failed occurrence has its own
  identity and message. Occurrences are never merged by target.
- **Always fatal.** Neither `external-reference` nor `path-reference` is in
  `lossyDiagnosticCodes`.
- **Backstop.** The shared planner refuses to render any document whose
  closure contains a blocked key. Blocked components never claim Go names.
- **Closure coverage.** `componentClosure` gains examples and security schemes.
  Selection and partial import must satisfy the reachability invariant for
  every supported component kind.

  Model controls show the gap with today's five kinds:
  - `fiveKinds` drops a reachable example blocker.
  - `unfilteredExamples` (the current filter) retains an unreachable one.

**Unknown Path Item membership.** The operations and tags of an unresolved
Path Item are unknowable, so its blocker is kept or dropped as follows:

- **No selection:** kept.
- **`--path` or `--path-prefix`:** kept exactly when its path key matches. This
  is decided precisely.
- **Any `--tag`:** always kept. Membership cannot be decided, so it fails
  closed. Model control `assumeUnselected` shows that dropping it is unsound.

**Security-key failures.**

- The resolver records one diagnostic at the requirement key's bundled path.
- `securityRequirements` consumes that recorded failure *instead of* emitting
  its own "not a supported component" message.

**Whole-graph failures.** These still return a nil document with diagnostics,
as `analyze.go:58-60` does today:

- an entry parse error
- an invalid entry `$self`
- an identity conflict
- a budget breach

**Modes.**

| Mode | Behavior |
|---|---|
| Strict (no selection) | Every blocker is kept. The import is blocked, and nothing is written. |
| `--tag`, `--path-prefix`, `--path` | `pruneComponents` closure, then `filterSelectionDiagnostics` on the bundled `Path`. Component blockers are kept exactly when reachable. Path Item blockers follow the rules above. |
| `--skip-unrenderable` (`AnalyzePartial`) | Per-operation closure. Operations that reach a blocker are skipped and listed. Unresolved Path Item keys that might be selected are listed as skipped. The other operations are written. |
| `--report` | Prints every blocker with its origin and bundled path, and then the provenance table (§8). |
| `--list-tags` | Succeeds from the bundled model. It prints `UnresolvedPaths` and marks tag counts as lower bounds when any Path Item is unresolved. It still fails on whole-graph failures. |

### 7. Security requirement URIs (OpenAPI 3.2)

**Owner.** One import-side normalizer. It must stay compatible with the
exporter's **scheme-global** rewrite (`openapi32.go:279-317`).

**Resolving a key in a requirement contained in document D** (the entry, or the
document of an inlined operation):

```text
if key is an entry security scheme component name -> that component        (§4.30 precedence; G.3 entry)
else key is a URI reference: t := resolve(key, base(D)); t must be a Security Scheme Object
case t in the entry (any spelling, including absolute self-URIs):
    key → entry component name; no URI metadata
case t external, canonical document is not file:
    transplant the scheme; Meta("openapi:security:uri", v) where
      v = the single authored spelling, if every key that targets this scheme
          uses that one spelling AND it resolves to the same target against
          the regenerated base (the spelling is absolute, or entry $self is absolute and
          the spelling resolves identically from the entry);
      v = canonical absolute URI (canonical doc + "#" + emit(pointer)), otherwise
case t external, canonical document is file:
    transplant as a local component; keys → allocated component name; no URI metadata
```

**Name keys are never resolved against D's own components.** A name that
matches no entry component fails, with a hint to use a URI key. Appendix
G.3.1's example resolves `MySecurity` to the entry's scheme.

**Rules within one Security Requirement Object.** §4.30 says that "all schemes
MUST be satisfied" within one object, and each value lists the scopes, or for
other scheme types the roles, "required for the execution".

Two keys in one object that resolve to the same scheme are aliases of one
conjunct. The request must satisfy both, with the same credential, so it needs
every scope or role either key lists:

```text
normalize(requirement R):
  for each scheme s identified by at least one key of R (in first-key order):
      emit one key for s, with scopes = sort(dedupe(⋃ scopes of every key of R that identifies s))
```

- **Order.** The emitted key takes the position of s's first key. This matches
  today's sorted scope lists (`security.go:255-268`).
- **No conflict cases.** No scope combination is rejected.
- **Separate objects stay separate.** They remain OR alternatives, in their
  original order, with their own merged scopes. Alternatives are never merged
  with each other.

Model M3 checks union, and its controls show that the alternatives are wrong.
On `{absH: [a], relH: [b]}`, with both keys identifying one scheme:

| Mode | Regenerated key | Effect |
|---|---|---|
| union (adopted) | `absH: [a, b]` | Exact |
| `dropAlias` | `absH: [a]` | Loses `b` |
| `intersect` | `absH: []` | Widens access |
| Revision 2 rule (`rejectUnequal`) | Valid requirement rejected | — |

In 3.0 and 3.1 documents, URI keys keep the existing `security-requirement`
diagnostic.

**Equivalence rule.** For every requirement object in the regenerated 3.2
document:

- It requires the same set of schemes as the source object. Every key resolves,
  through the regenerated base and §4.30 precedence, to a scheme equivalent to
  the source target. That is the same external URI, or a copied definition.
- Each scheme's scope set equals the union of the scope sets of the source keys
  that identify it.
- The order of requirement objects is preserved.

**Key bytes.**

- Key bytes stay identical only in the single-spelling case.
- Elsewhere, key spellings are canonicalized, following the precedent for
  local references (`docs/openapi-import-coverage.md:258-262`).
- Preserving each occurrence's spelling would require the exporter to own
  per-requirement URI keys. That would be a separate exporter change, outside
  this ticket.

**Model corrections.** Model M3 fixed three rules in this section:

- The containing-document base (control `entryBase`).
- The ban on URI metadata for entry schemes (control `keepMixed`).
- Merging aliases by union in place of the revision 2 rejection (controls
  `rejectUnequal`, `dropAlias` and `intersect`).

### 8. Distribution and provenance (decided policy)

**Bundled output.** Regenerated OpenAPI is a single bundled document. This is
the accepted policy: contract equivalence plus retained provenance. In the
coverage ledger, external references are *Conditional*: semantics are
preserved, and layout is normalized. Path Items are inlined until #962 adds
reusable Path Item output.

**Retained provenance.** The analysis result keeps the full resolver graph:

- documents, identities and aliases
- every edge with its authored spelling, source location and outcome
- the mapping from every bundled node to its source location

That graph drives three outputs:

- every diagnostic's `Origin`
- the `--report` provenance table. For each transplanted component, inlined
  Path Item and rewritten security or mapping value, the table lists:
  - the bundled pointer
  - the portable origin
  - every referencing occurrence
- the consumers that #962, #1005 and #1022 build on

**Public projection and operation identity.** Retain the exact bundled operation
source address when discovering an operation, including the `additionalOperations`
container and authored key case. All security, closure, planner, promotion and
partial-filter decisions consume that fact independently of the uppercase wire
method. A candidate operation's explicit `security: []` removes inherited root
protection and its irrelevant root failure; unresolved inherited or local
protection remains fatal. Strict and partial import must never emit an operation
whose required security was dropped during projection.

Internal diagnostic paths and provenance keys remain exact until all matching
finishes. Public diagnostic fields, locations, error text and CLI argument-error
wrappers then receive a detached projection that removes URI userinfo, including
malformed URI and duplicate-key errors. Nonsecret locations remain unchanged.
The public owner first attaches origins to the final diagnostic collection,
including planner errors added after analysis, then projects detached fields.
Generic JSON Pointer tokens and all prose/quoted errors stay token-bounded.
Whole-URI projection requires explicit authored URI metadata or a grammar-owned
URI-key position; a scheme-looking prefix never supplies that type. Source
mapping errors return only a safe validation reason, never the rejected composite
argument; local PATH values may contain spaces, `@` and `=`. Duplicate key errors
use grammar roles after alias validation. Grammar collections and canonicalization
share effective YAML mapping enumeration: explicit members retain source order,
inherited members follow in sorted order, and whole-value override/first-source
merge precedence preserves raw value-node identity. Entry syntax and supported
version are validated before opening supplied document roots, and the resolver
precedes libopenapi formatting. These boundaries feed one userinfo-removal primitive; a later nonsecret email or
pointer segment cannot extend a preceding URI authority.
Origin attachment compares both document and pointer: an inlined local Path Item
retains its original pointer even though its document is still `entry`.

**The design file carries no provenance metadata.** Nothing in Loom would read
it, and omitting it keeps a hand-bundled document byte-identical to its
multi-file equivalent.

**Transports.** Import emits HTTP designs. Bundled components are ordinary
types, so later gRPC or JSON-RPC exposure is unchanged. There is no wire or
runtime change.

## Accepted and Rejected Examples

| Input | Outcome |
|---|---|
| Local `#/components/schemas/Pet/properties/id`, `Pet/oneOf/0`, or an inline response Schema | Promote one canonical target component; ordinary references and URI mappings share it and preserve authored source identity. |
| Authored `foo_bar` with a local promoted `foo-bar` | The authored type keeps `FooBar`; the introduced type becomes `FooBar2`. |
| A local `properties` dictionary, `required` array, Operation or example-data object | Reject as a non-Schema target; promotion does not widen grammar admission. |
| `./models.yaml#/components/schemas/Pet`, a file under the root | Component `Pet`. The design is byte-identical to a hand-bundled entry. |
| `models.yaml#/X` and `./sub/../models.yaml#/X` | One key, one component |
| `#/components/schemas/Item%7E1Name` and `#/components/schemas/Item~1Name` inside `ext.yaml` | One key: `ext.yaml`'s `Item/Name` |
| `#/components/schemas/a%2Fb` | Pointer tokens `components, schemas, a, b`. Distinct from `a~1b`. |
| The entry `$self: https://e.test/api/openapi.json` plus a reference back through its retrieval spelling | Resolves to the entry component. No copy is made. |
| The entry `$self: https://e.test/b.yaml`, with `--document https://e.test/=./` and a reference to `https://e.test/b.yaml` | Fatal `document-identity`. `b.yaml` is loaded despite the alias. |
| `b.yaml` (no `$self`) reached both as `file:` and through `--document https://e.test/=./` | Fatal `document-identity`: two bases |
| The same `b.yaml` with `$self: https://e.test/b.yaml` | One document with two aliases |
| Distinct `b.yaml` and `c.yaml` both claiming `$self: https://e.test/shared.yaml` | Fatal `document-identity`, even if their bytes match |
| A reference to X before `c.yaml` (with `$self: X`) is loaded | Resolves to `c.yaml` (fixed point) |
| Entry `Pet` plus an external `Pet` | Entry `Pet` keeps its raw key and Go name. The external raw key is `Pet_2`, which Goifies to `Pet2`. Pass 2 allocates `Pet2`, unless an entry already took it. |
| Entry `pet_store` plus external `PetStore` | Raw keys stay distinct. The entry keeps Go name `PetStore`, and the external becomes `PetStore2`. Today's single pass would swap them (M4). |
| Entry security scheme `Foo` plus external schema `FooSecurity` | The entry keeps variable `ImportedFooSecurity`. The external schema becomes type `FooSecurity2`, with variable `ImportedFooSecurity2`. Today's code and the revision 4 per-namespace passes would rename the entry to `ImportedFooSecurity2` (M4). |
| `--document https://a.test/=./a/` and `--document https://b.test/=./b/`, each holding `models.yaml#/components/schemas/Pet` | Two targets ordered by `(1, "https://a.test/models.yaml")` < `(1, "https://b.test/models.yaml")`. Raw keys are `Pet` and `Pet_2` (or `Pet_2` and `Pet_3` if the entry has a `Pet`). The result is the same in any checkout or CLI order. |
| Byte-only `Analyze` without `$self`; `#/components/schemas/Pet` and `models.yaml#/X` | `Pet` resolves to the entry, unchanged. `models.yaml#/X` fails with "the entry has no base URI". |
| Byte-only `Analyze` with `$self: openapi.json` (relative); a reference `openapi.json#/components/schemas/Pet` | `$self` is projected verbatim. The reference fails with "no base URI". It is not treated as a self-reference. |
| `x.yaml#/A` → `y.yaml#/B` → `x.yaml#/A` | Two components and a recursive plan |
| `paths: {/pets: {$ref: ./paths/pets.yaml}}` with a Path Item root file | `/pets` operations are imported. Their diagnostics carry origin `paths/pets.yaml#/get/...`. |
| `/a` and `/b` both `$ref` one Path Item without `operationId` | Two independent operation sets |
| The same, where the Path Item has `operationId: list` | `operation-id` at both bundled operation paths |
| `/a: {$ref: x.yaml, summary: S}` where `x.yaml` has no `summary` | Fields are unioned. `path-metadata` (lossy) is reported at `#/paths/~1a`. |
| `/a: {$ref: x.yaml, get: {...}}` where `x.yaml` also has `get` | `path-reference`: undefined field overlap |
| `/a` → `p1.yaml` → `p2.yaml` → `p1.yaml` | `path-reference`: cycle |
| An unresolved Path Item `/c` under `--tag t1` | Blocker kept; `/c` listed under unresolved paths |
| An unresolved Path Item `/c` under `--path-prefix /a` | Blocker dropped |
| One node referenced as a Schema and as a Path Item | `reference-context` at every occurrence |
| The issue probe | Imports, with `Meta("openapi:self", ...)` |
| The probe plus a `models.json` reference and no mapping | A blocker for `https://example.test/api/models.json` |
| A missing reference used only by an unselected tag | The selected import succeeds |
| `%2e%2e/secret.yaml` under a prefix mapping | Rejected by the segment rules |
| A symlink out of a mapping root | Rejected by `os.Root` |
| `https://u:p@h/x.yaml` | Rejected; the userinfo is never printed |
| A `$ref` key inside `example:` data | Left as data |
| A 3.2 requirement `{K: [a], "./k.yaml#/components/securitySchemes/K": [b]}` resolving to two different schemes | Both kept |
| The same requirement, with both keys resolving to one scheme | One key for that scheme, with scopes `[a, b]` |
| Alternatives `[{K: [a]}, {"./k.yaml#…K": [b]}]`, where both keys are the entry `K` | Two alternatives kept in order: `{K: [a]}` OR `{K: [b]}` |
| An inlined `paths/x.yaml` operation requiring `h.yaml#/components/securitySchemes/S`, where the entry has no `$self` | Resolved against `paths/x.yaml`'s base. The key becomes the canonical absolute URI if the scheme is non-file. |
| An inlined operation requiring `MySecurity`, defined only in its own document | `security-requirement`: not an entry component (G.3) |
| `mapping: {cat: ./pets.yaml#/components/schemas/Cat}` | Edge `bundled(Cat)`, rewritten to `#/components/schemas/Cat` for #1005 |
| `defaultMapping: https://other.test/s.json#/Fallback`, not supplied | Edge `external(...)`, never fetched |
| `$self: "https://e.test/a.json#top"` | Accepted. The base is `https://e.test/a.json`. |
| A target schema with `$id` | `schema-resource` (#1022) |

## Contract Matrix

| Capability | Supported | Rejected | Deferred / owned elsewhere | Default | Source | Placement | Routing | Naming | Equivalence | Proof |
|---|---|---|---|---|---|---|---|---|---|---|
| Identity and aliases | `$self` and retrieval aliases; fixed-point lookup; load before alias | Identity or base conflicts; invalid `$self` | `$id` identities (#1022) | Retrieval URI | `$self` | `openapi:self` | Resolver | Authored `$self` | One key per document and pointer | M1, P1, P2 |
| Fragments | RFC 6901 §6 keys; canonical emission | Invalid pointers | — | — | Authored fragment | Target keys and emitted URIs | Resolver | `emit` | One key per decoded pointer | Fragment model, P1 |
| No-base entry | Fragment-only and absolute refs; relative `$self` projected | Relative refs without a base | Application default base (not adopted) | Byte-only has no base | `$self` | Private entry `sourceID` | Resolver | — | Unchanged output without non-fragment refs | M1, P1 |
| File and mapped documents | Reference root; exact and prefix mappings | Escapes, queries, bad segments, budget breaches | Hard-link unification | INPUT's directory | Disk and CLI | Bundled components | Resolver → analyzer | Raw keys: entry first, then `order`. Design names: complete entry inventory first, then external declarations by joint type/variable choice. | Bundle equivalence; entry Go names stable | M4, P3, P4 |
| Network | None (explicit policy) | Every unsupplied remote document | Opt-in fetching (new ticket) | Off | CLI | — | — | — | — | P5 |
| Path Item references | Local and external refs, chains, non-overlapping siblings, per-key inlining | Cycles, undefined field overlap, unresolved targets, duplicate `operationId` | Reusable Path Item output and entry `components.pathItems` (#962) | — | Path Item `$ref` | Inlined at `#/paths/<key>` | Resolver → analyzer | Path key | Same operations per key | M2, P8 |
| Discriminator mapping edges | Typed name or URI outcomes | — (consumer policy) | Admission (#1005) | — | `mapping`, `defaultMapping` | Rewritten values plus edge list | Resolver → #1005 | Bundled key or canonical URI | Same target per value | P9 |
| Failures and selection | Per-occurrence blockers; closure over every kind; Path Item fail-closed under tags | Rendering a blocked key | — | Strict blocks | Resolver | Bundled `Path` plus `Origin` | Extended shared closure | `x-loom-unresolved-n` | Reachable iff blocking | M2, P6 |
| Security URIs (3.2) | External non-file URIs; self-URIs normalized; containing-document base; alias scope union within one object | URI keys before 3.2; non-entry names; unresolved keys | Exporter-owned per-occurrence spellings | Component name wins | Requirement key | `openapi:security:uri` or a local component | Credential plan | Single spelling or canonical URI | Same schemes per object, scope sets equal to the source union | M3, P7 |
| Provenance | Graph retained; `Origin`; report table | — | — | Always on | Resolver graph | Analysis result and report | — | Portable keys | Every bundled node maps to a source | P6, P10 |

## Alternatives Considered

**Resolver and loading**

- **libopenapi rolodex.** Rejected. It ignores `$self`, scans whole trees, loses
  provenance and bypasses the source guards.
- **Resolving an alias without loading the mapped URI.** Rejected. It hides
  conflicts, so the outcome depends on order (control `aliasShortcut`).
- **Failing early, before the model is built, as today.** Rejected. It makes
  selection, partial import and `--list-tags` impossible.

**Representation**

- **Inlining at each occurrence.** Rejected for components, because shared
  identity and cycles are lost. Accepted for Path Items, which are not
  reusable Loom declarations until #962.
- **Name-keyed merging across documents.** Rejected. It collapses distinct
  targets.
- **Spelling-keyed targets built before alias resolution.** Rejected. It
  duplicates declarations (control `spellingKeys`).
- **A placeholder `{}` without a blocker.** Rejected. It widens the schema
  silently (control `placeholder`).

**Path Items**

- **Merging overlapping Path Item fields** (referenced wins, or referencing
  wins). Rejected. OAS 3.2 §4.9.1 leaves the overlap undefined.
- **Treating unresolved Path Items as unselected under tag selection.**
  Rejected. It is unsound (control `assumeUnselected`).

**Security keys**

- **A per-scheme URI meta claiming identical keys.** Rejected. The exporter
  rewrites globally (controls `authoredCollapse` and `keepMixed`).
- **Resolving names against the containing document.** Rejected. G.3
  recommends the entry, and a same-named scheme would silently change.
- **Resolving security URIs against the entry base.** Rejected (control
  `entryBase`).
- **Rejecting aliases with unequal scopes** (the revision 2 rule). Rejected.
  The requirements are valid, and union is their exact meaning (control
  `rejectUnequal`).
- **Keeping one alias's scopes, or intersecting them.** Rejected. Both change
  authorization, and the intersection widens access (controls `dropAlias` and
  `intersect`).

**Identity and naming**

- **Inventing a default base for a byte-only entry.** Rejected. It would
  resolve references that the source cannot resolve, and it would register a
  made-up identity (control `inventBase`).
- **URI-based document keys.** Rejected. A document without a base has no URI,
  so its fragment-only references would have no stable key.
- **Ordering external targets by path relative to each mapping root.** Rejected.
  Two roots can contain the same relative path, so that order is not total.
- **One-pass Go-name allocation over the bundled union.** Rejected. An external
  raw name that sorts earlier takes the entry's Go name (control `singlePass`).
- **Two passes per namespace** (revision 4). Rejected. Schema variables are
  fixed and are reserved before scheme variables, so an external schema still
  renames an entry security scheme (control `perNamespace`).

**Distribution and provenance**

- **Exporter support for foreign components and external `$ref`.** Rejected.
  This would be new exporter ownership, and bundled regeneration with retained
  provenance satisfies the ticket.
- **Provenance metadata in the generated design.** Rejected. Nothing reads it,
  and it would break bundle byte-equivalence.

## Formal Models (run; explicit, opt-in, never CI)

The models live under `internal/openapiimport/tla/`, each with a README that
records commands, toolchain, counts, assumptions and seams. They were run on
2026-11-05 with Corretto 25.0.5 and `tla2tools` 1.7.4.

- **M1 — `resolution_identity/ResolutionIdentity.tla`.** The worklist is
  compared with a declarative least fixed point. Target keys use internal
  document keys.
  - The entry is either retrieved from a file or byte-only. Its `$self` is
    absent, absolute or relative.
  - The checked configuration passes `FatalMatches`, `OutcomeMatches`,
    `OneKeyPerTarget`, `TransplantComplete`, `NoInventedIdentity` and
    `Termination`: 3,835,998 distinct states.
  - Each control violates its target:
    - `spellingKeys` and `noBaseCheck` → `OneKeyPerTarget`. `noBaseCheck` means
      per-retrieval documents, not an absent base.
    - `earlyFail` and `inventBase` → `OutcomeMatches`
    - `canonicalDedupe` and `aliasShortcut` → `FatalMatches`
- **M1f — `resolution_identity/FragmentPointer.tla`.**
  - The checked configuration passes `RoundTrip`, `KeyStable` and
    `PercentInvariant`: 1,350,723 distinct states.
  - `splitFirst` → `PercentInvariant`.
  - `escapeOrder` → `RoundTrip`.
- **M2 — `blocked_selection/BlockedSelection.tla`.**
  - The checked configuration passes every retention, rendering, partial and
    Path Item invariant, plus `Termination`: 263,616 distinct states.
  - Each control violates its target:
    - `placeholder` → `NoBlockedRendered`
    - `originFilter` and `unfilteredExamples` → `BlockerKeptIffImported`
    - `fiveKinds` → `PartialSound`
    - `assumeUnselected` → `PathItemSound`
- **M3 — `security_projection/SecurityProjection.tla`.** The oracle uses the
  §4.30 meaning: each requirement object maps its schemes to the union of their
  keys' scope sets.
  - The checked configuration passes `Equivalent`, `SpellingPreserved` and
    `NoFalseRejection`: 1,572,864 distinct states.
  - Each control violates its target:
    - `authoredCollapse`, `keepMixed`, `entryBase`, `dropAlias` and
      `intersect` → `Equivalent`
    - `rejectUnequal` → `NoFalseRejection`
  - On `{absH: [a], relH: [b]}`:
    - The union witness regenerates `absH: [a, b]`.
    - `dropAlias` gives `[a]` and `intersect` gives `[]`. Both fail
      `EquivalentOnAB`.
- **M4 — `bundle_naming/BundleNaming.tla`.**
  - Covers schema types, fixed schema variables and scheme variables.
  - The complete-inventory allocator passes `EntryStable` and `NoDuplicate`
    for every order of external declarations: 1,858 distinct states.
  - Today's code and the revision 4 per-namespace rule both violate
    `EntryStable` on entry security `Foo` against external schema
    `FooSecurity`.
  - Today's type allocation alone violates `EntryTypesStable` on entry
    `Foo_Security` against external `FooSecurity`.
  - Error clones are not modeled.

These bounded checks establish their stated properties only within each
model's abstractions. They do not prove URI parsing, `os.Root`, libopenapi or
the Go implementation.

## Implementation Proof Obligations (unproved in Go)

**P1. Identity and fragments**

- RFC 3986 resolution and normalization vectors produce single keys.
- An absolute `$self` re-bases relative references away from the filesystem.
- **No-base entries.** A byte-only entry without an absolute `$self`:
  - resolves fragment-only references in its private document key
  - fails every relative non-fragment reference with "no base URI"
  - registers no identity
  - is unchanged for documents without non-fragment references
- Pointer keys follow RFC 6901 §6. This includes `%2F` versus `~1`, `%7E`
  versus `~`, and round trips of `emit`.

**P2. `$self` output**

- `$self` round-trips byte-equal in 3.2 output and is absent from 3.1 output.
- A `$self` with a fragment is projected verbatim, and the fragment is stripped
  for the base.

**P3. Bundle equivalence**

- Moving a non-colliding component into an external file gives a byte-identical
  design.
- Documents in the unchanged class match the base commit byte for byte.
- Exact numeric tokens and the effective values of YAML merges survive the
  copy.

**P4. Confinement and budgets**

- Confinement holds for each mapping and for the reference root. The cases are
  `..`, encoded separators, absolute paths and symlinks.
- Every budget is enforced.
- Load-before-alias conflicts are detected.
- **Machine independence.** Names are the same across process-isolated runs
  from different checkout paths, and across permuted `--document` order. The
  only exception is an absolute `file:` URI that the author wrote into a
  `$self` or a mapping.
- **Same relative path under two roots.** Two mapping roots holding the same
  relative path produce distinct, ordered raw keys.
- **Entry design names are stable.** Every entry type name, schema variable,
  security scheme variable and entry-origin error-clone type and variable
  equals the name an entry-only import allocates. Pairs to cover:
  - `pet_store`/`PetStore`
  - `Foo_Security`/`FooSecurity`
  - entry security `Foo` against external schema `FooSecurity`
  - an entry error clone `ImportedXError` against an external schema `XError`
- **No duplicates.** No two declarations share a type name or a design variable.

**P5. No network I/O for any input**

**P6. Mode matrix**

- Strict, selected, partial, report and list-tags modes behave as §6 states,
  including Path Item fail-closed rules and every M2 control input.
- Every transplanted or inlined diagnostic maps back to its exact origin.
- Data-position `$ref` keys are untouched.

**P7. Exporter symmetry**

- Importing exporter `openapi:security:uri` output, with the scheme document
  supplied, and then regenerating, satisfies the §7 equivalence rule.
- Keys are byte-identical in the single-spelling case.
- Every M3 control input regenerates correctly.
- **Merging aliases.** Aliases of one scheme within one object merge into one
  key with the sorted, deduplicated union of their scopes or roles.
- **Alternatives.** Separate objects stay separate alternatives, in order.

**P8. Path Items**

- Local, external and chained references import with per-key operations and
  origins.
- Overlap, cycles and unresolved targets block with exact paths.
- Duplicate `operationId` blocks at import.
- Shared Path Items regenerate operation-equivalent output.

**P9. Discriminator edges**

- Name and URI classification, entry name lookup, URI rewriting and external
  canonicalization produce the outcomes in §5.
- Discriminators remain rejected until #1005.

**P10. Provenance**

- The report lists every bundled node with a non-entry origin, together with
  its referencing occurrences.

**Existing gates**

- Validate rendered specs with libopenapi and Vacuum.
- Generated code must compile.
- Extend the exporter-symmetry fixture.
- Update the field-level coverage ledger:
  - `$self` becomes Preserved.
  - External references and Path Item references become Conditional (bundled
    layout).

## Acceptance Coverage (#957)

| Ticket item | Design | Evidence |
|---|---|---|
| Scope field `$self` | §1 projection | P2 |
| Scope field: external and relative `$ref` | §1–§4 for every admitted position, including Path Items and no-base entries | M1, M2, M4, P1, P3, P4, P8 |
| Scope field: URI-based security references | §7 | M3, P7 |
| Resolve document identity and reference bases without flattening occurrence identity | §1, §3, §4, §6, §8 | M1, M2, P6, P10 |
| Local files | §2 reference root | P4 |
| Supplied external documents | §2 mappings, §1 registration | M1, P4 |
| Relative references | §1 RFC 3986 bases, including containing-document bases | M1, M3, P1 |
| Cycles | §3 component cycles; §4 Path Item cycles rejected (no finite meaning); visited-set termination | M1 `Termination`, P3, P8 |
| Fragment escaping | §1 `pointer` and `emit` | M1f, P1 |
| Explicit network policy; no implicit fetching | §2 | P5 |
| Focused regression cases and coverage ledger | Obligations P1–P10 | Implementation |
| Existing OpenAPI validation and generated-code checks | Existing gates | Implementation |

## Cross-Ticket Boundaries

- **#962 (reusable Path Items).** #962 consumes the `pathItem` edges, adds
  `components.pathItems` declarations, and adds reusable Path Item output.
  Until then, references are inlined.
- **#1005 (discriminator import).** #1005 consumes the `discriminatorMapping`
  edges (§5) and owns admission and closure.
- **#1022 and #1021 (schema resources and dialects).** These tickets register
  schema-resource identities in the same registry and lift `schema-resource`
  and dialect rejections. #957 fails closed on them.

## Open Questions

None at the ticket-scope level.

**Decisions taken in this revision:**

- bundled regeneration with retained provenance
- full Path Item reference support
- no network access
- entry-document resolution of names
- containing-document base for URIs
- the global `operation-id` import check
- internal document keys separate from optional URI bases; no invented base
- portable-key total order, and reservation of the complete entry design-name
  inventory before external declarations
- union of alias scopes within one security requirement object

The parent may reject any of these during contract review.
