# Response status source bindings (#976)

Design evidence for the `StatusField` source in the accepted #976
[response-selection design](../../../http/codegen/internal/transportir/response_selection_design.md). The correspondence below names
the Go regressions. The existing `../http_wire_bindings` model is unchanged.
The modeled binding plan is the response-owned plan; the request
`ensureWireBindings` plan never receives response sources.

`StatusBindings.tla` explores one status-selected success pair: exact `200`
(body member `ok`, header `etag`, both branch-required) and `2XX` (body
member `accepted`), with one shared `StatusField` carrying `Minimum(200)`.
Transitions: the shared binding plan admits or rejects an authored competing
header/body mapping of the StatusField; a Loom server selects the branch from
the field value, checks domain and branch-required members, and publishes; a
foreign server sends any status, members and competing copies; the client
selects once, reconstructs members in order body, headers, status, then
validates.

## Reproduce

Same toolchain as the status selection model (Corretto 25.0.5, TLA+ tools
1.7.4, SHA-256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
for cfg in checked competing implicit-body body-last validate-first exact-unbound copy-all; do
  "$JAVA" -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
    -config "$cfg.cfg" -metadir "/tmp/loom-976-tlc/sb-$cfg" StatusBindings.tla
done
```

## Recorded results (2026-11-05, source `f0c7a2b8c`)

| Configuration | Initial | Generated / distinct | Result |
| --- | --- | --- | --- |
| `checked` | 240 | 16,120 / 10,800 | pass: StatusIsolation, Agreement, BranchRequired, BranchDisjoint, NoDroppedBinding, NoStatusInBody |
| `competing` | 240 | 15,975 / 15,975 | NoDroppedBinding: an admitted `Header` mapping of the StatusField is silently overwritten |
| `implicit-body` | 240 | 2,189 / 2,029 | NoStatusInBody: the field stays an implicit body member |
| `body-last` | 240 | 5,815 / 5,655 | StatusIsolation: a body copy applied after the status wins |
| `validate-first` | 240 | 7,349 / 7,189 | Agreement: validation sees zero before the status is assigned |
| `exact-unbound` | 240 | 7,349 / 7,189 | Agreement: the exact branch leaves the required field unreconstructed |
| `copy-all` | 240 | 5,823 / 5,663 | BranchDisjoint: members of the other branch reach the result |

An earlier draft of `competing` and `body-last` passed because status was
applied last; the model was corrected to state the real defects (a dropped
binding and a leaked body member) rather than retain vacuous controls.

## Limits and correspondence

Statuses are `{0, 200, 201, 204, 404}`; competing copies carry 404 or nothing.
Error responses, views, media variants, cookies and Tag selection are not
modeled. Go correspondence: `TestStatusFieldWireBinding` covers implicit body
exclusion, competing mappings, wire-local requirements, view survival, media
variants and per-key error identity in the response plan. The generated
`TestFinalStatusRoundTrip` and `TestFinalStatusNativeDecodeConstraint` check
that the client assigns `resp.StatusCode` before validation, and
`TestSelectedFinalPreparation` checks branch-required members before
publication. OpenAPI body schemas derive from the same response plan;
`TestInformationalResponsesAreDocumented` checks that an informational status
field is not documented as a header. These textual correspondences do not
assert a new model run.
