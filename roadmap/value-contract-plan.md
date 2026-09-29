# Value-contract plan

Resolve values once; transports consume shared plans. Preserve untagged bodies and runtime behavior.

M1–M2 core pushed. Remaining work, in order:

1. **#574:** Fix shared schema ownership; preserve async policies; match byte schemas to decoders.
2. **#571 → #456:** Centralize enums/defaults. Await alias override/intersection decision.
3. **#572:** Remove private OpenAPI synthesis; validate rendered examples.
4. **#565:** Migrate transports/CLI; pass generated runtime and SSE tests.
5. **#434 → #573:** Verify protobuf round trips; remove duplicate interpretation; complete inventory and CI.

Fix shared owners, test sibling cases, update proofs. Review, delivery and cleanup follow [AGENTS.md](../AGENTS.md).

[Design](value-contract-design.md) · [Owners](value-contract-inventory.md) · [Comparisons](../internal/valuecontract/README.md) · [Proofs](../expr/lean/value_projection/correspondence.md) · [#574 evidence](../internal/valuecontract/BYTE_SCHEMA.md)
