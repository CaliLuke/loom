# Framework capability designs

This index links capability contracts to their durable acceptance
records. Remaining framework work is tracked in [the roadmap](ROADMAP.md).

| Issue | Contract and evidence | Status |
| --- | --- | --- |
| [#1014](https://github.com/CaliLuke/loom/issues/1014) | [Expanded oneOf import](issue-1014-oneof-import-design.md), [acceptance](../internal/openapiimport/oneof_import_validation.md) | Implemented and published to canonical main in `15786bca94813a3ffb7b46c6c97fe6a403be7985`; bounded compilation and determinism checks pass. |
| [#957](https://github.com/CaliLuke/loom/issues/957) | [External reference graphs](issue-957-external-references-design.md), [acceptance](../internal/openapiimport/external_references_validation.md) | Implemented by this atomic change against parent `b9e92c7c98b2bbc1f082a95bb76de363b55edf27`; publication pending the normal batch push. Contract retained for #1005 and #962. |
