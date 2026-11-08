# Root server representation

The strict importer retains plain root Server Objects in source order. Each
accepted absolute HTTP or HTTPS URL becomes an explicit Server/Host/URI design
entry, retaining the URL text and description. This avoids relying on a generated
default host. Root servers apply to all imported services through the existing
server finalizer.

OpenAPI 3.2 requires a nonempty authored server name in this bounded subset.
OpenAPI 3.0/3.1 unnamed entries use deterministic internal names, which their
export format omits. Names are documentation values, not Go identifiers; repeated
names retain separate ordered entries. Description presence is retained in the
normalized model; empty descriptions follow the existing exporter omission rule.

Variables, userinfo, query, fragment, relative or non-HTTP URLs, extensions and
path/operation overrides remain unsupported. Strict Render repeats the shared
admission decision for normalized Documents. Partial import uses that same
owner to omit unsupported root entries explicitly while retaining supported
entries and operations. It does not convert an invalid URL into a default URL.

# Primitive contributor construction

A single inline non-null string, integer, number or boolean contributor in allOf
is retained as a semantic Bases edge. Its assertions and examples remain on the
contributor; outer assertions and examples remain local. Type and format provide
carrier context without flattening those layers or creating a semantic Ref.
Multiple contributors, nullable contributors, object/array contributors and
changes to the contributor's type or Go carrier format remain outside this subset.

Native Type construction uses an explicit construction-only NamedSchema pointing
to the exact retained contributor node. Selection and initialization planning
follow that recorded identity. Only these constructor declarations omit canonical
OpenAPI name metadata; authored and bundled declarations retain their identity.
Reserved async schemas participate only in complete graph comparison, never in
constructor allocation.

SSE success occurrences must agree on their complete normalized event contracts,
including both layers' assertions, examples and presence. Distinct authored
references cannot collapse, even when an inline occurrence precedes them. A
reference and an inline contract use the existing one-sided reference comparison.
Only after this check does an operation prepare one common event for its successes.
An authored declaration occurrence supplies the representative whenever present,
so an earlier inline occurrence cannot erase its identity before reserved graph
comparison. Status/header occurrences and their source paths remain independent.

The maintained producer_primitive.json is the unchanged native producer witness
with minimum 0 on its contributor, local minimum 5, distinct examples, 200/201
successes, Header examples, root server, dialect and reserved async declaration.
Direct tests exercise normalization, native DSL evaluation, selection, rejected
compositions and unequal event contracts. Shared example compatibility remains
owned by the common complete-graph validator. Actual exporter first/second rounds
require the composed producer/importer source and are a separate qualification.
