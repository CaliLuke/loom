# Framework capability designs

This index links capability contracts to their durable acceptance
records. Remaining framework work is tracked in [the roadmap](ROADMAP.md).

| Issue | Contract and evidence | Status |
| --- | --- | --- |
| [#968](https://github.com/CaliLuke/loom/issues/968) | [Parameter serialization](../docs/http-guide.md), [acceptance](../internal/httpencoding/parameter_codecs_validation.md) | Implemented; acceptance and historical dependency findings are retained with the codec owner. |
| [#1014](https://github.com/CaliLuke/loom/issues/1014) | [Untagged union contract](../codegen/unionjson/README.md), [import coverage](../docs/openapi-import-coverage.md#schema-coverage), [acceptance](../internal/openapiimport/oneof_import_validation.md) | Implemented and published to canonical main in `15786bca94813a3ffb7b46c6c97fe6a403be7985`; bounded compilation and determinism checks pass. |
| [#957](https://github.com/CaliLuke/loom/issues/957) | [External reference graphs](issue-957-external-references-design.md), [acceptance](../internal/openapiimport/external_references_validation.md) | Implemented in parent `3d386572c2c6a5b41596f121327e129d67b2704d`; publication pending the normal batch push. Contract retained for #962. |
| [#1005](https://github.com/CaliLuke/loom/issues/1005) | [Discriminator import](../docs/openapi-import-coverage.md#discriminators), [hint metadata](../docs/dsl-reference.md#openapi-32-features), [acceptance](../internal/openapiimport/discriminator_import_validation.md) | Implemented in commit `cadba28e0e551401dd94015822f6b9d467e6ffb3`; publication pending the normal batch push. |
| [#976](https://github.com/CaliLuke/loom/issues/976) | [Status selection](../docs/http-guide.md#status-selected-responses), [DSL](../docs/dsl-reference.md#response-selection), [informational responses](../docs/http-guide.md#informational-responses), [SSE handshakes](../docs/http-guide.md#sse-handshake-status-and-completion), [design](../http/codegen/internal/transportir/response_selection_design.md), [acceptance](../http/codegen/internal/transportir/response_selection_validation.md) | Implemented. |
| [#1024](https://github.com/CaliLuke/loom/issues/1024) | [Durable workflow roles](../execution/selvedge/tla/workflow_roles/design.md), [acceptance](../execution/selvedge/workflow_roles_validation.md), [Selvedge prerequisite](../execution/selvedge/tla/workflow_roles/selvedge-v081-prerequisite-check.md) | Implemented; the native prerequisite and consumer checks are recorded with the owner. |
