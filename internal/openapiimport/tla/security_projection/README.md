# Security requirement URI projection (#957)

`SecurityProjection.tla` is design evidence for §5 of
[`roadmap/issue-957-external-references-design.md`](../../../../roadmap/issue-957-external-references-design.md).
It composes the import normalizer with the exporter's scheme-global
`openapi:security:uri` rewrite (`applySecuritySchemeURIs`,
`http/codegen/openapi/v3/openapi32.go:279-317`). It then resolves the
regenerated document.

The oracle resolves each source key independently:

- A component name resolves against the entry document's components
  (OAS 3.2 §4.30; Appendix G.3 recommends the entry document).
- A URI resolves against the base of the document that contains it
  (RFC 3986 §5.1). OAS §4.1.2.3 says that implicit connections do not change
  URI resolution.

Requirements may come from the entry document or from an external Path Item
document `D`, whose base is https.

**Authorization meaning.** OAS 3.2 §4.30 says that every scheme named in one
Security Requirement Object must be satisfied, with every scope (or role) it
lists. Two keys that identify the same scheme therefore require the union of
their scope sets. Separate requirement objects stay separate OR alternatives.

The oracle computes each requirement object's meaning as a map: every required
scheme maps to that union. The regenerated document is interpreted the same
way.

## Decisions informed

- **Spelling preservation.** One authored spelling is kept only when it is the
  only spelling for the scheme and resolves to the same scheme against the
  regenerated base. Otherwise the key becomes the canonical absolute URI.
  Counterexample: `authoredCollapse`.
- **Entry schemes never carry URI metadata.** Every spelling that resolves to an
  entry scheme becomes that scheme's component name. Counterexample: `keepMixed`.
  Under it, the exporter deletes the component, and the name-keyed requirement
  no longer resolves.
- **Containing-document base.** URI keys inside external documents resolve
  against that document's base, not the entry's base. Counterexample:
  `entryBase`. This rule was added to the design because the model exposed it.
  The earlier draft resolved every key against the entry base.
- **Aliases merge by union.** Aliases of one scheme within one requirement object
  merge into one key whose scope set is the union of theirs. The earlier V1 rule
  rejected unequal scope sets. Control `rejectUnequal` shows that this rejects
  valid requirements.

  Keeping only one alias's scopes, or intersecting them, changes the
  authorization contract. On `{absH: [a], relH: [b]}`:

  | Mode | Regenerated key |
  |---|---|
  | union (checked) | `absH: [a, b]` |
  | `dropAlias` | `absH: [a]` |
  | `intersect` | `absH: []`, which widens access |

## Commands and results

The toolchain is the same as in `../resolution_identity/README.md`. To run the
check:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 4 -metadir "$(mktemp -d)" -config checked.cfg SecurityProjection.tla
```

Observed results on 2026-11-05 (revision with union scope semantics,
`-workers 8`):

| Configuration | Expected | Observed |
|---|---|---|
| `checked.cfg` | `Equivalent`, `SpellingPreserved` and `NoFalseRejection` hold | Pass: 2,097,152 generated, 1,572,864 distinct, depth 3, exit 0 |
| `authored-collapse.cfg` | `Equivalent` fails | Fails, exit 12. Input: `D` requires `h.yaml#…: [b]`, and the entry has no `$self`. The metadata keeps the relative key, which does not resolve in the regenerated document. |
| `keep-mixed.cfg` | `Equivalent` fails | Fails, exit 12. Input: the entry has `$self`, and the key is `K`'s absolute self-URI. The exporter deletes `K`, so the key no longer resolves. |
| `entry-base.cfg` | `Equivalent` fails | Fails, exit 12. Input: `D`'s `h.yaml#…` is resolved against the entry's `file:` base. The file scheme is copied instead of the https scheme. |
| `reject-unequal.cfg` | `NoFalseRejection` fails | Fails, exit 12. Input: `D` requires `{absH: [], relH: [a]}`, a valid requirement for scheme S with `[a]`. The old rule rejected it. |
| `drop-alias.cfg` | `Equivalent` fails | Fails, exit 12. Input: `{absH: [], relH: [a]}` regenerates as `absH: []`. |
| `intersect.cfg` | `Equivalent` fails | Fails, exit 12. Input: `{absH: [], relH: [b]}` regenerates as `absH: []`. |
| `witness-union.cfg` | Expected witness: `UnionWitness` is "violated" | Exit 12. `{absH: [a], relH: [b]}` regenerates as `absH: [a, b]`. |
| `drop-alias-ab.cfg` | `EquivalentOnAB` fails | Fails, exit 12. The same input regenerates as `absH: [a]`. |
| `intersect-ab.cfg` | `EquivalentOnAB` fails | Fails, exit 12. The same input regenerates as `absH: []`. |

## Assumptions and limits

The model is bounded:

- Two requirement objects.
- Up to four keys, using the spellings `K`, `absH`, `relH` and `absSelfK`.
- Two containing documents.
- The entry `$self` is either present or absent.

Each key lists one of the scope sets `[]`, `[a]` or `[b]`. The set `[a, b]` can
only arise from merging. Requirement objects are maps, so property names are
unique. Scopes and roles follow the same rule, so the model does not
distinguish scheme types.

**Equivalence.** A scheme is "equivalent" when it is the same target: either an
unchanged external URI, or a copied local definition. The model does not
compare scheme bodies.

**Out of scope.** The model does not prove:

- URI parsing
- 3.0/3.1 rejection
- credential planning
- exporter serialization

**Go correspondence.** Exporter-symmetry fixtures and import tests below
cover each control as a concrete document, including byte-identical keys in the
single-spelling case.

The model is an explicit design tool and is never a CI gate.

## Implementation correspondence

`resolve_security.go` projects keys by resolved target, unions alias scopes
and selects stable export URI metadata. `resolve_bundle.go` records occurrence
bases; `security.go` retains credential validation. `TestSecurityURIRequirements`,
`TestSecurityURIContainingDocument` and `TestSecurityURINamePrecedenceAndVersion`
exercise the controls and version boundary. The `identity` subtest of
`TestOpenAPIImportExporterSymmetryRoundTrip` generates and compiles a service and
checks exact `$self` and security key bytes.

The bounded assumptions and recorded checker results above remain unchanged.
