# Framework capability designs

This index links capability contracts to their durable acceptance
records. Remaining framework work is tracked in [the roadmap](ROADMAP.md).

| Issue | Contract and evidence | Status |
| --- | --- | --- |
| [#1014](https://github.com/CaliLuke/loom/issues/1014) | [Untagged union contract](../codegen/unionjson/README.md), [import coverage](../docs/openapi-import-coverage.md#schema-coverage), [acceptance](../internal/openapiimport/oneof_import_validation.md) | Implemented and published to canonical main in `15786bca94813a3ffb7b46c6c97fe6a403be7985`; bounded compilation and determinism checks pass. |
| [#957](https://github.com/CaliLuke/loom/issues/957) | [External reference graphs](issue-957-external-references-design.md), [acceptance](../internal/openapiimport/external_references_validation.md) | Implemented in parent `3d386572c2c6a5b41596f121327e129d67b2704d`; publication pending the normal batch push. Contract retained for #962. |
| [#1005](https://github.com/CaliLuke/loom/issues/1005) | [Discriminator import](../docs/openapi-import-coverage.md#discriminators), [hint metadata](../docs/dsl-reference.md#openapi-32-features), [acceptance](../internal/openapiimport/discriminator_import_validation.md) | Implemented by this atomic change against parent `3d386572c2c6a5b41596f121327e129d67b2704d`; publication pending the normal batch push. |

This delivery does not authorize or complete #976.
