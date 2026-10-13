# Imported name occupancy — #688

1. [x] Reproduce the original duplicate component declaration on main43edbf9fe with a test-only overlay; trace every uniqueName consumer.
2. [x] Independently review shared owner and bounded acceptance before implementation.
3. [x] Make uniqueName claim every emitted case-folded candidate and skip occupied suffixes; preserve default Value and deterministic traversal/spelling for noncolliding inputs.
4. [x] Exercise all consumers directly; compare ordinary and collision generated modules against actual parent and fresh-process repeat, build/vet, and validate rendered specs.
5. [ ] Independent exact final review, atomic delivery and cleanup.

## Owner

The existing importer-local uniqueName is already shared by component names,
operation methods, security declarations, security credential fields, request
parameters, response header/body fields and cloned error declarations. Its map
counts occurrences of bases but never records emitted suffix candidates. Correct
that occupancy contract here; do not patch individual consumers or introduce a
new public allocator. Case-folded identity is existing behavior and stays intact.

All callers must continue receiving a recorded actual identity; original wire
names and component-reference identity remain separate. Existing first-claim
ordering remains: a later authored numbered base may itself receive a suffix if
already occupied. Pre-reserving future names is unnecessary for uniqueness and
would change more ordinary naming choices. Preserve no-collision output exactly.

The existing NameScope loop and package-symbol formal model establish the intended
candidate selection/occupancy invariant. Read that model and retain its bounded
limits. A direct adversarial permutation matrix will expose both prior-base and
prior-generated suffix occupancy, plus case folding and empty fallback. The small owning
`internal/openapiimport/tla/name_occupancy` refinement models the precise old
counter-only selection and candidate occupancy across all request orders; no
general naming rewrite or exhaustive namespace audit.

## Acceptance

Use one compact nontrivial importer specimen for legal component names Pet/Pet2/pet,
three colliding operation IDs, and transport field suffixes, keeping original
source identities and bindings. Direct consumer checks cover security declarations
and credentials plus response-body and error-clone reservation interactions.
The existing ordinary supported fixture must retain exact parent bytes. Collision parent must fail design
compilation or actual generation; corrected design and generated module must
compile/vet and validate JSON/YAML contracts. Repeat independent CLI generation
byte for byte. Reuse importer helpers and only unique modules.

#789 fixed-property Go field collisions are an independent missing field-name
planning boundary. Correct allocator occupancy does not by itself fix those
unallocated fields; leave789open. URI identity891, schema annotations783 and service
package interceptor roles770 are likewise separate owners.

Original red: /tmp/loom-688-readiness.log. The issue reproduction stops at its first
require failure (two ImportedPet2 declarations); it does not establish later
operation assertions. Add independent direct operation checks before final review.

Actual generator reproduction also fails while compiling the design with duplicate
ImportedPet2 declarations: `/tmp/loom-688-compile-readiness.log`. A bounded
candidate-occupancy table adds prior-base, prior-generated, case-folded and empty
fallback witnesses plus an ordinary control. Independent readiness approved the shared occupancy correction before implementation.

TLC1.7.4/Corretto25 readiness: counter fails Distinct (11 generated/9 distinct),
occupied passes (18 generated/12 distinct). Direct table has four expected failing
suffix cases and one ordinary passing case; original component declaration
witness fails independently. Proposed owner change is confined to uniqueName.

## Final acceptance

The correction is confined to `uniqueName`: remember each emitted candidate and
advance past occupied suffixes. Every consumer already retains the returned
identity, so no downstream renaming or renderer exception was added. Direct
checks exercise all consumer families, six request permutations, empty fallback,
case folding and retained rejection of colliding path variables. The original
component/operation specimen now also covers colliding query fields and a
response body beside numbered headers. Source wire names remain unchanged.

On archived parent `e6673300f`, the actual collision specimen fails to compile
with duplicate `ImportedPet2`. Its candidate generates, compiles and vets.
The existing supported fixture has all 16 generated artifacts byte-identical
between actual parent and candidate. Both candidate modules repeat all 32
artifacts across independent CLI processes; four JSON/YAML specifications pass
libopenapi and pinned Vacuum. No checked-in generated fixture changes are
intended. Direct/generated race checks pass (6.588s), and affected existing
name/security/error-clone controls pass under race (20.463s). Scoped lint is clean
after preallocating two test slices; this mechanical edit does not change DSL or
generation inputs. Model source/configuration is unchanged from approved
readiness, so its bounded evidence is reused.

The final manifest `/tmp/loom-688-final-manifest.json` freezes source, parent,
inputs, evidence and artifact comparisons. Final independent review, commit,
push, closure and temporary-output cleanup remain pending.
