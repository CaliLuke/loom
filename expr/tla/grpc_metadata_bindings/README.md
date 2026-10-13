# gRPC metadata wire bindings — #705

This bounded model compares framing, collision admission and one shared binding
across encoder, decoder and required-key contracts. One or two fields have binary
or text storage, one of three names (including a case variant), one of two
independent channels, and authored or implicit credential origin. Distinct source
fields start with different exact authored names within their channel. Physical
collision comparison folds case, as the gRPC runtime does.

Run from this directory with Java25 and TLC1.7.4 (jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-705-state-derived-owned \
  -config derived-owned.cfg MetadataBindings.tla
```

Replace both configuration names for other variants. Negative checks exit12;
positive checks exit0. Deadlock checking remains enabled; completed resolution
self-loops. Checked on the readiness parent71a0758f6:

| Variant | Result | Meaning |
| --- | --- | --- |
| current | BinaryFraming fails; 509 distinct states | Plain binary keys reach ASCII metadata framing |
| encode-only | ConsumerAgreement fails; 509 distinct | Encoder-only repair leaves decoder/required lookup inconsistent |
| derived-unchecked | DistinctBindings fails; 613 distinct | Normalized derived keys merge distinct source fields |
| auth-late | DistinctBindings fails; 614 distinct | Validating only authored fields misses later credential bindings |
| derived-owned | All invariants pass; 1512 generated/1008 distinct | Admit complete normalized channel bindings before use |
| explicit-owned | All invariants pass; 1512 generated/1008 distinct | Historical alternative requiring explicit binary keys |

The user selected derived-owned. Explicit-owned remains rejected design evidence,
not a compatibility mode. `NoOverRejection` requires the selected owner to accept
noncolliding bindings, including equal names in different channels. The checker
establishes these bounded properties; it does not decide product policy.

## Implementation correspondence

`GRPCEndpointExpr.ensureMetadataBindings` resolves actual service occurrences and
complete implicit security placement before gRPC metadata validation. All
consumers read its canonical mapped names: `extractMetadata`, request codecs,
response codecs and response required-key contracts. `auth-late` represents the
current placement timing if only authored mappings were moved into that owner.
`MethodExpr.EffectiveRequirements` supplies inherited requirements before
method finalization. Reuse existing security-placement rules, rather than
reimplementing them in collision validation. The cached owner publishes fresh mapped attributes without mutating service
fields or retaining stale reverse mappings; finalization reuses that publication.

Direct tests `TestGRPCMetadataEvaluatedKeys`,
`TestGRPCMetadataRejectsCanonicalCollisions`,
`TestGRPCMetadataImplicitCredentialCollision`,
`TestGRPCMetadataChannelsRemainIndependent` and
`TestGRPCMetadataInheritedErrorUsesMappingSource` connect the model to expression
admission and error-source ownership. `TestBinaryMetadataRoundTrip` includes
protobuf-only TCP peers, required absence, binary and empty values, alongside
six generated round-trip specimens. The #690 model continues to own default and
required-before-default admission.

## Limits

The model omits base64, runtime I/O, arbitrary channel/name counts, alias/array
resolution, defaults, custom Go storage and mutation after evaluation. One header
channel represents header/trailer separation; direct tests cover all three actual
placements. Implicit origin abstracts credential placement without modeling
scheme negotiation. It assumes resolution uses validated source types and no
unadmitted binding appears afterward. Generated tests must establish that link.

The separate named-element representation repair #877 is delivered.
`TestBinaryMetadataAggregate` covers named Bytes and a named outer array with
named Bytes elements, repeated empty values, defaults and supplied emptiness.
`TestBinaryMetadataConsumerAgreement` and `TestBinaryMetadataErrorContractAgreement`
check canonical codec and required-contract names; the error control does not
claim that designed-error metadata is delivered at runtime.
Keep sources/configurations and concise findings here; remove checker state and
raw logs after independent review.
