# Untagged oneOf acceptance (#1014)

Initial implementation parent: `ba1c91f8bdbdd780f1f2b95860d048849e7f8fb0`.
The #968 prerequisite is `f9f86095e7c75a8be47bc38686f876ffac709031`. The initial
parent comparison below is historical evidence for those inputs. Integration
now includes subsequent main fixes; its source baseline is
`2386e219f8a79220a6553cc00ee81815d57b4b1a` (the final candidate
tree is recorded in the exact-diff review packet). Do not infer current output equivalence from the
initial parent: the focused integration packets explicitly identify their
baseline and candidate sources.
Working checkout: `/Users/luca/.codex/worktrees/oneof-import/loom`, initially clean.
Scratch: `/tmp/loom-1014-prep`. Go 1.27.3; all Go checks use the shared task cache
and generated modules select this checkout explicitly with LOOM_DIR. No dependency
change is planned. Reuse the already-checked text, presence, closure, recursive-null
and normalization models within their recorded assumptions; rerun only changed
models. Proof tools stay opt-in and outside all delivery gates.

## Workflow recital

1. M1: read the accepted design/plan; verify #968's exact public seams; record
   source inventory, parent generated output, six import controls and the behavior
   table's baseline reports before framework edits.
2. M2: move the two-domain counting owner, preserve JSON behavior, then add
   evaluated annotations and closure after selection plus text entry points.
3. M3: implement the root grammar, branch constructor and codec-owned location
   admission, preserving null carriers and gRPC rejection boundaries.
4. M4: lower and generate JSON/text/form/multipart/CLI/JSON-RPC adapters with
   transactional selection and encode attribution; mounted wire tests prove them.
5. M5: project union closure, nullable branches and captured field codecs through
   OpenAPI 3.1/3.2 and inline JSON Schema.
6. M6: normalize import branches and siblings, preserve null semantics, promote
   only required objects, and route every rejected case to its actual owner.
7. M7: compare parent/current output, compile the same bounded controls, test
   import/export symmetry, process determinism, libopenapi and pinned Vacuum.
8. M8: finish public/internal docs and model correspondence; complete affected
   gates; obtain independent exact-diff review; deliver one atomic commit on main.

## Prerequisite evidence

`TestTextUnionCodecPrerequisite` passes (0.480s). It verifies effective scalar/list
Form explode normalization and one literal claim, alternative residual maps as
one claim, sibling residual rejection, branch-indexed deepObject rejection,
independent string/integer schema membership, int32/int64 private conversion,
and raw comma splitting before percent decoding.

The consumed signatures are:

- `httpencoding.ResolveUnion(Location, string, []BranchShape, Serialization) ([]BranchCodec, KeySet, error)`;
- `httpencoding.CheckNamespaces(Location, map[string]KeySet) error`;
- `http.TextBranch.Schema() (bool, error)` and `Decode() (bool, error)`;
- `http.ParamValueJSON(ParamValue, ParamBranchShape) (jsontext.Value, bool, error)`;
- `http.ParseRawPairs(string) (RawPairs, error)` and
  `DecodePairsParam(RawPairs, string, ParamCodec, ParamKind, ParamNamespace) (ParamValue, error)`;
- `ParseRawFormWithLimit` owns the bounded raw form source. `TextBranch.View`,
  `Member` and `Assign` connect that source to schema/typed branch callbacks.

#968 supplies the seam; #1014 adds selection and generated union adapters. Main
also contains #1013 typed AdditionalAttributes. Its object-owner and physical
member APIs must remain authoritative when adding evaluated annotations; text
and form typed-record rejection remains at the codec boundary.

## Parent evidence

The unchanged generated HTTP controls pass (15.189s): untagged arrays, request
body contracts and tagged form/OAuth. JSON-RPC untagged arrays pass (3.519s).
The parent determinism dump completed (7.453s), SHA-256:
`2b11c7b8faf48daf575675b840e6e412c746b89cd5ab7ea4ffd8fc7433ef15ba`.
The current #968 harness includes 23 designs; M7 compares all unchanged inputs,
rather than assuming the plan's historical six-design count.

All six baseline imports succeed:

| Fixture | SHA-256 |
| --- | --- |
| supported.yaml | `194abfd06b8388663fa764a50cf9a9d116a8f14d669863931fc7844c9ab5aaac` |
| supported.json | `194abfd06b8388663fa764a50cf9a9d116a8f14d669863931fc7844c9ab5aaac` |
| symmetry.yaml | `d41db18b91d1e108cc32372330f759a78a6566edef8e884dce8b9532b098554f` |
| encoded_component_reference.yaml | `00509de63ef35f88adcbe023034f8fa393cfd10668103e8b4959127461989b4c` |
| shared_status_different_schemas.yaml | `3b2e2e1c810ba84ce507af8b0564d65d737d1d3963d053d57eedfd011aef51c8` |
| object_field_names.yaml | `7d9ebb03b06a7d0b9caee70f597e1bbc4325b3995bd4b25c23ed4f468c79fc71` |

The 43 design-table rows and a typed-additional-member integration case each
have a complete source specimen and base CLI report in scratch. They are input
and diagnostic baselines; mounted tests still must establish their wire behavior.

## Inventory dispositions

The search recorded 781 matches in 181 files before framework edits. Every
matched parent line is enumerated below. Milestone dispositions identify the
owning task, not an instruction to rewrite already-correct logic; unchanged
controls retain their original semantic intent. Review any newly discovered
cross-owner impact against this inventory before editing.

| Parent file | Matched lines | Disposition |
| --- | --- | --- |
| `.agents/skills/loom-framework/SKILL.md` | 114, 164, 165, 323, 498, 851, 1276, 1362, 1510, 1515, 1517, 1605, 2021, 2388, 2555 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `.agents/skills/loom-framework/references/formal-models.md` | 126, 151 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `.agents/skills/loom-framework/references/repo-map.md` | 51 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `.agents/skills/loom/SKILL.md` | 420, 561, 567, 593, 891, 994, 1002, 1047, 1048, 1150, 1151, 1152, 1261 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `cmd/loom/import_comparison_input_controls_test.go` | 59 | M7 contract comparison controls; preserve existing normalized semantics while adding union round trips. |
| `cmd/loom/import_comparison_input_test.go` | 167 | M7 contract comparison controls; preserve existing normalized semantics while adding union round trips. |
| `codegen/bytes_validation_manifest.json` | 10 | Unchanged byte-validation inventory; new union support does not alter byte representation obligations. |
| `codegen/generator/cli_json_artifacts_test.go` | 25, 44 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `codegen/generator/cli_json_union_test.go` | 9, 29, 30, 33, 46 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `codegen/generator/generation_determinism_test.go` | 33 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `codegen/generator/optional_collection_validation_test.go` | 40 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `codegen/service/projected_union_validation_context_test.go` | 58 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `codegen/service/service.go` | 71, 139, 150 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `codegen/service/service_data_types.go` | 192, 206, 210, 251, 265, 277, 278, 431 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `codegen/service/service_data_union_order_test.go` | 131, 146, 221, 226, 236, 237, 347, 358 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `codegen/service/testdata/projected_union_validation_dsl.go` | 25 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `codegen/uniongen/codecs.go` | 35, 109 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `codegen/uniongen/emit.go` | 148, 191, 192, 204 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `codegen/uniongen/model.go` | 25, 27, 31, 50 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `codegen/uniongen/tla/null_admission/README.md` | 32 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `codegen/unionjson/README.md` | 6, 17, 40, 46, 57, 78, 87, 96, 100, 145, 150, 153, 271, 273, 278, 294, 323, 325, 326, 354, 360, 362, 372, 379 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `codegen/unionjson/union.go` | 14, 64, 108, 122, 136, 141, 175 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `codegen/unionjson/union_test.go` | 42, 78, 101, 111 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `docs/code-generation.md` | 309, 315 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `docs/dsl-reference.md` | 105, 156, 245, 258, 277, 433, 479, 488, 582, 599, 993, 1036, 1227, 1229, 1238, 1285, 1287, 1289, 1291, 1292, 1293, 2278, 2312, 2315, 2320, 2420, 2559 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `docs/http-guide.md` | 586, 620, 758, 759, 763, 800 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `docs/http-media-contracts.md` | 89 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `docs/openapi-import-coverage.md` | 318 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `docs/plugin_union_codegen.md` | 4, 6, 15, 19 | M8 documentation: update union grammar, matching and closure claims where present; preserve unrelated contracts. |
| `dsl/additional_attributes.go` | 16 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `dsl/attribute_oneof.go` | 58, 63, 65, 73, 76 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `dsl/http_endpoint_body.go` | 75, 92, 93 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `dsl/meta.go` | 380, 396, 401, 406, 411 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `dsl/object_default_readiness_test.go` | 27 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `dsl/oneof_occurrence_test.go` | 12, 14, 18, 22, 25, 33, 36, 37, 50, 52, 59, 60, 61, 62 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `dsl/oneof_test.go` | 193, 195, 198, 206, 210, 212, 217, 220, 223, 230, 233, 235, 238, 241, 244, 247, 253, 256, 297, 301, 311, 314 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `dsl/union_physical_identity_test.go` | 63, 85, 98 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `dsl/untagged_maps_test.go` | 12, 42, 50, 62, 71, 113 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `dsl/untagged_nested_test.go` | 12, 22, 31, 38, 48, 58, 67, 75 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/additional_attributes_test.go` | 47, 49, 67, 341, 436, 446, 664, 672 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/attribute.go` | 33 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/attribute_schema_validation_test.go` | 16, 53 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/attribute_validate.go` | 339, 341, 356, 358, 367, 370 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/attribute_wire_names_test.go` | 42, 43, 46, 54, 65 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/dup.go` | 140 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/equal.go` | 111 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/example_canonicalization.go` | 116 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/example_canonicalization_test.go` | 123, 125, 295, 297, 345 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/grpc_endpoint_test.go` | 23, 24, 57, 58, 71, 72, 93, 94 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/hasher.go` | 118, 119, 333 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/hasher_recursive_test.go` | 232, 234 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_body_element_names_test.go` | 174, 187, 314, 328 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_documentation_union_test.go` | 49, 53, 71, 72, 73 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_empty_response_body_test.go` | 31 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_endpoint.go` | 168 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_endpoint_test.go` | 480, 491, 502, 503, 504, 508, 525, 537, 538, 539 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_endpoint_validate.go` | 119, 134, 150, 153, 154, 402, 403, 408, 409, 498 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_file_response_test.go` | 88 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_media_encoding.go` | 15, 16, 40, 41 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_parameter_serialization.go` | 66 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/http_parameter_serialization_test.go` | 39 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/json_schema_inline.go` | 272 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/json_schema_inline_test.go` | 543, 547, 548, 550 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/lean/value_projection/README.md` | 305 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `expr/lean/value_projection/correspondence.md` | 153, 195, 407 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `expr/map_key_collision_validation_test.go` | 121, 167, 179, 537 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/object_openness.go` | 27 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/object_openness_test.go` | 25, 26, 34, 60, 107, 131, 141, 144, 149, 152, 155, 162, 170, 177, 184, 221, 235 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/presence.go` | 95, 96, 99, 104, 108, 120, 124, 130, 133, 134 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/testdata/endpoint_dsls_01.go` | 912 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/testdata/endpoint_dsls_05.go` | 86, 89 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/testdata/grpc_message_dsls.go` | 24, 25, 99, 100 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/tla/documentation_schema/README.md` | 4 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `expr/tla/grpc_field_tags/README.md` | 27 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `expr/tla/object_openness/README.md` | 73 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `expr/tla/status_error_identity/README.md` | 46 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `expr/tla/union_branch_admission/README.md` | 9, 46 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `expr/types.go` | 61, 63 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/union_branch_admission_test.go` | 23, 24, 51, 89, 103, 104 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/untagged_array_test.go` | 12, 26, 35, 39, 57, 66, 97, 107, 122 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/untagged_validation.go` | 10, 15, 19, 21, 26, 31, 33, 38, 43, 44, 46, 50, 58, 61, 65, 66, 71, 73, 75, 78, 82, 92, 94, 95, 97, 98, 101, 107, 118 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `expr/value_declared.go` | 12, 162 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_declared_test.go` | 118, 266, 275 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_occurrence.go` | 72, 416 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_plan.go` | 117 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_plan_build.go` | 251 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_plan_declaration_test.go` | 78, 109, 132, 133 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_plan_json_shape.go` | 20, 21, 92 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_plan_json_shape_test.go` | 17, 59, 103 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_plan_numeric_precision_test.go` | 40 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_projection_build.go` | 150, 379 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_projection_decode.go` | 204 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_projection_schema.go` | 140 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `expr/value_projection_test.go` | 11, 71, 132 | M4 prepared/lowered JSON shape correspondence, including occurrence closure and typed additional members. |
| `grpc/codegen/proto_constructor_oneof_test.go` | 135, 141 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `grpc/codegen/untagged_array_test.go` | 12, 13, 26, 27, 39, 40, 41, 42, 43 | M3 branch grammar, constructor, location/codec admission and null carriers; preserve unrelated declared identity and transport constraints. |
| `http/codegen/internal/representation/value_plans.go` | 106, 127, 129, 144, 206, 209 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `http/codegen/internal/representation/value_target_graph.go` | 11, 65 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `http/codegen/internal/transportir/model.go` | 164 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `http/codegen/internal/transportir/tla/response_order/README.md` | 10 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `http/codegen/mapped_names_test.go` | 95 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/openapi/internal/ir/analyzer.go` | 445, 462 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/analyzer_baseline_test.go` | 89, 91, 124 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/analyzer_element_names_test.go` | 73, 74, 76, 85, 90 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/analyzer_hash.go` | 277 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/analyzer_hash_test.go` | 67, 72, 74 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/async_baseline_authority_test.go` | 95, 96 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/async_materialization_test.go` | 21, 241, 242, 259, 316, 361 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/async_schema_examples.go` | 76, 133 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/async_schema_inline.go` | 74, 166 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/document_examples.go` | 145, 146, 253, 254, 432, 434, 550, 559 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/document_test.go` | 76, 105, 134, 228, 245, 311, 332, 368, 394 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/example_validation.go` | 15, 18, 25, 53, 80, 81 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/example_validation_test.go` | 48, 49, 80, 81 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/model.go` | 268, 343, 394 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/nested_union_examples_test.go` | 129, 134, 137 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/render.go` | 64 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/representation_equivalence_test.go` | 117, 126 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/representation_registry.go` | 256 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/reusable_components_helpers.go` | 337 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/internal/ir/tla/byte_examples/README.md` | 4 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `http/codegen/openapi/internal/ir/untagged_byte_examples_test.go` | 12, 22, 54, 67, 81, 91, 109, 120, 132, 143 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/schema.go` | 78, 79 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/additional_attributes_test.go` | 221 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/files_test.go` | 843, 863 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/ir_adapter.go` | 364 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/object_openness_test.go` | 49, 112, 156, 191 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/openapi31_filter.go` | 282 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/schema_refs.go` | 122, 197, 256 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/types_dedup_test.go` | 234 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/untagged_arrays_test.go` | 15, 24 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/untagged_byte_examples_test.go` | 19, 36 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/openapi/v3/version_test.go` | 205, 221, 225, 240, 258 | M5 union schema and closure projection; preserve reference identity, examples and version controls. |
| `http/codegen/service_data_body_registry.go` | 26 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `http/codegen/service_data_direct_builder_test.go` | 19, 21, 25, 29, 30, 32 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/service_data_request_body_validator_test.go` | 118, 119, 141 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/service_data_typesupport.go` | 94, 100, 221, 228, 242 | M4 union codec and transport generation; preserve branch identity and existing non-closure output. |
| `http/codegen/service_data_union_order_test.go` | 94, 96, 101, 114, 117 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/inline_body_dsls.go` | 13 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/mapped_names_dsls.go` | 11, 91, 95, 104, 108 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/openapi_dsls_14.go` | 10, 16 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/openapi_dsls_15.go` | 26, 29 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/openapi_dsls_16.go` | 19, 25 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/openapi_dsls_17.go` | 17, 52 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/streaming_dsls.go` | 111, 115 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/untagged_arrays_dsl.go` | 5, 7, 24, 31, 45, 80, 87, 90, 95, 106, 113, 116 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/untagged_arrays_harness.go` | 3, 4 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/testdata/viewed_omitted_union_dsl.go` | 16, 17, 35, 73, 75, 85 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/tla/union_declarations/README.md` | 83, 85, 108, 112, 114, 122 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `http/codegen/union_response_reuse_test.go` | 99 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/untagged_arrays_test.go` | 13, 14, 17 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/value_contract_cli_test.go` | 13, 45 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/value_plan_test.go` | 119, 129, 237 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `http/codegen/view_union_declarations_test.go` | 45, 47, 71, 113, 114, 117 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `internal/openapiimport/analyze.go` | 248, 584, 586, 601 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/analyze_schema_composition.go` | 75, 81, 94, 102 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/analyze_schema_grammar.go` | 75 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/empty_object_test.go` | 386 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/grammar_coverage_test.go` | 112, 635 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/issue_302_test.go` | 12, 51, 55, 70, 103, 149, 161, 171, 187, 210, 258, 268, 323, 330, 345, 386, 392, 404, 413, 441, 450, 467, 475, 502, 505, 528, 531, 556 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/issue_350_test.go` | 55 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/lean/schema_normalization/README.md` | 142, 182, 259, 260 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `internal/openapiimport/media_contract.go` | 163 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/parameter_shape_agreement_test.go` | 48 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/plan.go` | 166, 167, 196, 197, 243, 244, 281, 282, 311, 312, 317, 334, 337, 342, 346, 350, 411, 417, 420, 421, 493, 494, 497, 512, 515, 522 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/plan_references_test.go` | 47, 226, 228 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/reference_consumers_test.go` | 57 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/render_reference_context_test.go` | 183 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/render_schema.go` | 248, 318, 390, 392 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/render_test.go` | 38 | M6 normalization/admission/rendering and owner-routed diagnostics; retain reference, nullable and naming controls. |
| `internal/openapiimport/tla/oneof_null/README.md` | 1, 53 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |
| `internal/testdatacompile/expectations.json` | 284, 290, 308, 326 | M7 generated compilation expectations; preserve existing accepted designs and owner-routed rejections. |
| `jsonrpc/codegen/mapped_names_test.go` | 21, 94, 97 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `jsonrpc/codegen/untagged_arrays_test.go` | 13, 28, 46, 63, 68 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `jsonrpc/codegen/value_contract_cli_test.go` | 13, 50 | Unchanged accepted generator controls unless M4 reveals an intended union boundary change; compare and compile in M7. |
| `pkg/json_shape.go` | 27 | M2 matcher and evaluated-member owner; preserve exact predicates and existing nested mismatch controls. |
| `pkg/json_shape_numeric_precision_test.go` | 44, 54, 58, 66 | M2 matcher and evaluated-member owner; preserve exact predicates and existing nested mismatch controls. |
| `pkg/json_union.go` | 14, 22, 24, 29, 32, 38, 43, 63, 66, 69, 74, 78, 83, 91 | M2 matcher and evaluated-member owner; preserve exact predicates and existing nested mismatch controls. |
| `pkg/json_union_nested_test.go` | 14, 30, 40, 51, 54, 62 | M2 matcher and evaluated-member owner; preserve exact predicates and existing nested mismatch controls. |
| `pkg/json_union_test.go` | 22, 27, 50, 53, 59, 76, 82, 85, 89, 99, 108, 111, 168, 176 | M2 matcher and evaluated-member owner; preserve exact predicates and existing nested mismatch controls. |
| `pkg/tla/untagged_json/README.md` | 1, 3, 13, 14, 36, 39, 76, 124, 129, 131, 164, 173, 177, 181, 206, 238, 270, 315 | Unchanged formal evidence; retain stated bounds. M8 updates correspondence only if an owning implementation seam changes. |


## M2 shared counting owner

The new direct selection tests first fail against a compile-safe stub
(`m2-core-red.log`): unique selection, both ambiguity domains, zero matches,
identity disagreement, full visitation and fatal propagation all expose missing
behavior. Moving the existing counting logic into `internal/unionmatch.Select`
and adapting the JSON boundary preserves all existing JSON matcher behavior.
The direct owner, full pkg and unionjson packages pass (0.438s / 0.527s / 0.418s).
The ordinary mismatch type now lives beside the shared selector; JSON nested
mismatch classification consumes it. The subsequent closure and text work is recorded below.


## M2 closure, text and typed additional-member integration

Compile-safe baseline implementations reproduced the evaluated-member, closure
and text-selection failures (`m2-closure-red.log`). The runtime now shares
selection and post-selection member checking in `internal/unionmatch`.
`JSONShape.Evaluated` distinguishes absent and present additional-member policy,
returns only the selected nested branch annotations, and returns sorted member
names. Closed JSON and text entry points apply those annotations after both
selection domains agree. Empty form roots remain values supplied by the media
owner; the selector does not infer parameter presence from key count.

Typed-object integration exposed two missing owner facts before JSON union
admission: extra-value membership and synthesized required members were absent
from prepared/lowered schema matching (`m2-additional-red.log`); the independent
prepared decoder also accepted an int32 overflow in an extra member
(`m2-additional-decoder-red.log`). Captured value plans now carry the additional
child, required names and omitted names. Both schema and decoder check presence
and omitted names; each uses its own extra-value predicate. Lowered descriptors
retain these facts and explicit open policy. The unionjson policy walk visits
additional values, so field spelling and API closure apply to nested extra
objects, and its renderer shares additional descriptor nodes correctly.

The initial omitted-member test used bare Delete, which intentionally does not
record a transport omission. Correcting the fixture to call the actual
omitMember/Delete sequence made it represent the intended contract; no change to
Delete was needed. A nested additional-object regression first failed because
unionjson did not visit the additional child (`m2-additional-render-red.log`).

Final affected packages at this point (`m3-current.log`, Go 1.27.3): dsl 0.478s,
expr 0.920s, pkg 0.176s, http 3.785s, unionjson 0.215s, unionmatch 0.212s, all pass.
Generated compilation and exact-output comparisons remain M4/M7 obligations.
The earlier full build passed immediately after the counting-only extraction.
The existing text/presence/closure model transitions are unchanged; these tests
are additional implementation correspondence, not new proof claims.

## M3 current implementation boundary

The sole-function OneOf constructor now retains named branch attributes and
constraints, and runtime admission accepts scalar/Any/alias/array/map/nested and
nullable roots. Typed and explicit-open named objects are admitted through the
same recursive JSON shape validator. Tagged branches, anonymous object leaves,
opaque codecs and non-string map keys remain rejected. The old scalar and typed
object rejection tests are replaced by acceptance controls; schema-only versus
runtime validation still uses an actual opaque codec as its rejected case.

The new root grammar and constructor tests first failed at their runtime owner
(`m3-roots-red.log`), then passed. Unused top-level types are not runtime
validation roots, so the DSL tests explicitly attach each type to a method.
Text-location tests first failed at the blanket union ban (`m3-text-red.log`).
Removing that ban only for untagged unions lets existing Resolve/CheckNamespaces
own branch admission and combined claims. Path/query/header/cookie/response
header cases pass, while scalar deepObject and sibling residual claims fail with
the codec owner's diagnostic. Existing tagged-union errors now say "tagged".
Form/multipart, closure metadata and generated transport adapters are pending.


## M4 first generated JSON root specimen

The new bounded root fixture initially failed in service analysis because
unionBranchValidationCode asserted every branch was a UserType. Generalizing
that seam exposed a named scalar pattern compilation error; passing the existing
alias flag to the validation owner fixes the conversion. HTTP analysis now emits
inline validation for the new root kinds while retaining existing named
object/collection validator calls. No hand-written generated files were changed.

The compiled mounted fixture then exposed top-level null rejection before the
union matcher: AllowsNull did not inspect untagged alternatives. Its shared fact
now counts null-admitting alternatives recursively, including enum-constrained
Any and named aliases, instead of adding a transport exception. Direct tests
first reproduced the missing fact (`m3-null-red.log`). Nullable carriers still
come from NameScope, and the shape gate precedes their decode.

The fixture verifies JSON roots for scalar, numeric overlap, Any, aliases, maps,
nested unions, nullable branches, width splits and typed additional members.
For failures it checks no service invocation, HTTP status/code/detail, and
separately the service codec's exact matching error. HTTP retains its established
sanitized body errors, as clarified in the design. It also verifies ambiguous
encoding rejection and unchanged destination on rejected decoding. The initial
root generation run after null correction passed in 2.382s; the expanded
service-codec error assertions passed in 2.738s (`m4-roots-green.log`). The gRPC
root projection test passed in 2.969s: scalar, alias, Any, array, map and nested
unions compile, and nullable results retain the established rejection. The
focused DSL/expr/service owner checks passed (`m4-owners.log`). Closure/text/media adapters
and full importer/exporter scope remain pending.

## Occurrence closure implementation

The occurrence-identity model under `expr/tla/union_closure` confirms that shared
mutation and policy-erasing declaration identity are invalid alternatives. Its
checked mode passes every occurrence ordering; the README records both negative
controls and the model's limits. The end-of-root finalizer now allocates private
closure variants after service/transport finalization. It reserves authored names
and IDs first, keeps repeated policies together and leaves finalized branch
occurrences shared. The identity regression failed before implementation and now
passes with reversed fields, generated-name collisions, recursive open references
and copying (`closure-identity-red.log`, `closure-identity-owners.log`). The latter
full DSL/expr/pkg/unionjson run passes in 0.609s/1.184s/0.326s/0.297s.

`JSONShape.UnevaluatedClosed` carries closure independently from object openness.
The matcher counts original alternatives first, then checks selected evaluated
members; aliases retain local closure without mutating their children. Direct
tests cover maps, Any, explicit open/typed additional objects, nested unions,
ambiguous open records and malformed closure descriptions. They first failed
behaviorally (`closure-shape-behavior-red.log`) and passed with existing matching
controls (`closure-shape-green.log`, 0.405s).

The expression owner admits only false on unions. Prepared plans capture local
or inherited policy, and lowering retains it. Decoder closure follows the
decoder's retained branch identity rather than recounting schema matches.
Initial prepared/lowered tests first failed (`closure-plan-red.log`), then passed
(`closure-plan-progress.log`, 0.629s); the width-split, immutable-capture and tagged
envelope controls also pass. The full affected packages pass in
`closure-owners.log`: expr 0.980s, pkg 0.126s and unionjson 0.207s. The width-split
case retains schema ambiguity while a unique wide Go decoder passes closure;
an extra member rejects the decoder independently. Capture survives subsequent
mutation of the source metadata. Tagged envelopes use their authored key names.

The extended root fixture compiles separate local open/closed declarations and
mounts both tagged and untagged HTTP endpoints. Untagged closure first failed
behaviorally (extra field accepted), then passed in 2.600s. Tagged closure first
failed the same way after the fixture placed its custom keys on the branch
constructor. Shared union generation now selects RejectUnknownMembers only for
closed tagged envelopes, and the complete fixture passes in 2.655s
(`closure-tagged-green.log`). It checks response values, status/code/detail,
service invocation and unchanged selected union on rejection.

Before adding the new root fixture to the process-determinism table, the existing
23-design generated section/file dump remained byte-identical to the parent
baseline (`closure-control-determinism.txt`, comparison exit 0; generator 5.550s).
This establishes no output changes for existing non-closure controls at this
stage. Full ticket comparison and final-source checks remain M7 obligations.

## Closure review corrections

The independent closure-slice review found three issues before ticket review:
high-severity recursive rematching cost; medium-severity constructor policy loss;
and medium-severity merging of differently localized declarations. All have
regression inputs and implementation corrections. Independent focused re-review
confirmed all three fixes with no new findings. This clears only the closure
slice; full-ticket review remains pending.

- `ClosureWork.tla` reproduces repeated matching and supports retaining
  annotations during the original traversal. The local depth-20 probe falls
  from 2.262s to 36.458 microseconds. Runtime tests cover depth 64 and reject a
  deep unevaluated member without contributing failed-branch annotations.
- `CaptureUnionClosure` checks the temporary constructor attribute and returns
  a detached raw union with its policy. Constructor false survives naming/copy;
  constructor true fails admission (`closure-constructor-red/green.log`).
- The identity model now distinguishes finalized contracts, with a nominal-only
  negative control. Cache buckets compare complete contract graphs before
  sharing. DSL tests retain different authored envelope keys and share truly
  identical occurrences. Direct recursive comparisons also distinguish branch
  constraints, nullability, member metadata and additional schemas
  (`closure-contract-red.log`, `closure-contract-matrix.log`).

The full DSL/expr/pkg/unionjson packages pass after these fixes (0.661s/0.772s/
0.193s/0.201s, `closure-review-owners.log`). Mounted HTTP and JSON-RPC root
specimens, including nullable objects/maps and captured service state, pass in
4.003s/3.600s (`closure-rereview-generated.log`). Five process-isolated generations
pass with the complete current root fixture in 13.425s
(`closure-rereview-determinism.log`). Proof checks remain explicit design tools.

## Text adapter progress (M4, not final acceptance)

The shared text matcher now drives query, path, header, cookie, response-header
and CLI adapters. Leaf codecs come from `internal/httpencoding`; each candidate
receives its own raw view and private typed destination. Encoding builds private
wire data, repeats matching and checks branch identity before publication.
The mounted specimen covers scalar ambiguity, Go-width/schema disagreement,
delimited lists, flattened objects, residual maps with sibling keys, deepObject,
optional absence/empty collection behavior, and raw text Bytes.

`ParamBytes` uses base64 only for its canonical JSON schema-predicate projection;
the actual location codec keeps raw text. Direct scalar/list/object/map regressions
first failed, then passed; the generated Bytes enum/length specimen accepts raw
`é` and rejects base64 wire text. Text object property names now come from the
parameter owner independently of authored JSON tags. Full unionjson controls pass.

The CLI regression exposed truncation of `hello&peer=1` into `hello`.
`RawPairsFromValue` retains an already bounded raw value, validates percent escapes
and preserves encoded collection delimiters. Bindings with flattened, prefixed or
residual ownership instead parse a complete query fragment; this choice comes
from the resolved key set. The generated scalar and flattened-object CLI cases,
plus direct delimiter/empty/invalid-escape tests, pass in
`text-cli-boundary-green.log` (http 0.514s, codegen 2.653s).

Full ticket acceptance remains pending, including nested text composition,
closure for text/form objects, media adapters, import/export work, final output
comparison and independent review. These partial checks do not authorize a
partial ticket commit.

The object response-header specimen initially failed compilation because the
result constructor lost the argument's native type context and emitted missing
conversion helpers. Retaining that context fixes the owning constructor input;
the object header round trip passes (`text-response-object-green.log`, 5.950s).

Nested text generation initially panicked on the Union parameter shape.
`ComposeTextUnion` now retains each nested predicate result, counts nested schema
membership independently, and treats ordinary inner contract failures as candidate
mismatches. The generator emits a tree and keeps the entire selection path for
outbound identity checks. The mounted fixture accepts a distinct outer string
when an inner width union is ambiguous, rejects outer ambiguity, and prevents
an invalid selected inner string from being silently encoded as its boolean
peer (`text-nested-identity.log`, 4.804s). A named string-pattern branch exposed
a missing alias flag in validation emission; that fixture now compiles too.
The full http/unionmatch/unionjson packages pass (4.037s/0.079s/0.428s).

The 23 original designs still have byte-identical rendered output. Of their
767 raw/rendered sections, 88 raw sections differ only by adding (87) or moving
(1) the jsontext import candidate; import finalization produces unchanged files.
The two new root/location fixtures account for the additional output. See
`text-baseline-comparison.txt`; final M7 comparison must be refreshed after
subsequent generator changes. Union defaults remain rejected by the existing
DSL contract (`dsl/attribute.go`, `oneof_default_and_forward_test.go`); M4 does
not introduce defaults for union values.

The focused text-slice review found two medium CLI carrier issues: exploded
query lists needed repeated pairs, and exploded cookie maps needed a complete
cookie fragment. Both now derive grammar from ownership and multiplicity,
recursing through nested codecs. Generated regressions cover repeated query and
cookie values, nested repeated queries, sibling exclusion and malformed cookies.
The corrected fixture passes in 7.010s; five-process determinism passes in
31.397s. Independent re-review confirmed both fixes with no new findings for
snapshot `5a63cd822acb4782b4bf0f070362f3dd610aec52a54f941cd0bcf666e8be40cf`.
This is a focused review only; the ticket still needs full final review.
Temporary nested-model checker directories and the review reproduction module
were removed after review; durable models and summarized evidence remain.

## Export progress (M5)

Inline JSON Schema originally dropped the union closure keyword. Direct tests
cover raw and named occurrences, local and inherited nullability, keyword
placement inside the value arm, unchanged open occurrences and immutable source
declarations. These tests failed before projection was added and now pass with
all inline-schema controls (`inline-closure-green.log`, 0.637s).

OpenAPI schema analysis now projects the same owner policy before nullable
wrapping, including a closure refinement next to a named schema reference.
The direct raw/named/nullable/open matrix passes (`export-closure-green.log`,
0.654s). Both root and location fixtures build through libopenapi as JSON and
YAML on 3.2.0 and 3.1.1 (`untagged-targets.log`, 0.848s). The 3.1 location variant
uses explicit non-exploded form cookies, including its sibling control cookie;
raw cookie serialization stays on 3.2 under #968's existing contract. The same
four cases are registered with the maintained pinned Vacuum contract suite.
Form-field encoding and remaining exporter matrix rows are still pending.

The raw inline-constrained branch, named object reference and nullable scalar
branch projections pass direct IR assertions, as do query/path/header/cookie
parameter schemas and response-header union schemas with their resolved
style/explode values (`export-shapes-locations.log`, 0.520s). The pinned Vacuum
v0.36.2 rules accept JSON and YAML for all four root/location target specimens
(`untagged-vacuum.log`, 32.091s). A temporary scoped probe reused the maintained
installer, renderer and lint helpers; it has been removed. The normal contract
suite retains the four specimen cases.

## Fresh-agent audit corrections

The fresh review found distinct ownership defects in the in-progress ticket.
These corrections remain part of the atomic #1014 change; slice approvals do not
replace its final exact-diff review.

- Constructor directives: `OneOf(func)` now rejects root directives that cannot
  survive a DataType return. Accepted union metadata and branch DSL remain.
  Direct red/green, generated HTTP/RPC roots and 862 unchanged output sections
  passed; independent slice review approved (`constructor-contract` evidence).
- Additional-member runtime plans: policy traversal and target graph copying now
  include additional containers with occurrence-specific layout. Four direct
  missing-policy failures and the shared-context alias failure pass after the
  correction; generated HTTP/RPC additional-composite modules compile and run.
  Main's A14 projection correction will own these graph edges after integration.
- Native union presence: an implicit null-admitting union field keeps its native
  union carrier, with empty kind for absence and a selected null branch for
  supplied null. Explicit outer Nullable keeps its wrapper. Shared declarations,
  validation, transforms, authorization/interceptors and protobuf derive this
  from `expr.HasNativeJSONPresence`; WrapNullable was removed. Generated HTTP,
  JSON-RPC and gRPC null matrices and existing controls pass. Null-free output
  is byte-identical; 22 null-fixture files change as intended from noncompiling
  output to native declarations, omission, Kind validation and conversion.
  Independent slice review approved. The attempted broad gRPC sweep produced no
  completion result; it is not counted as passing evidence.
- Bytes text projection: raw octets become base64 JSON before schema membership;
  UTF-8 validation remains specific to String. Binary scalar/list/object/map
  controls and generated query/client cases pass. The prior test incorrectly
  rejecting binary Bytes was removed from the UTF-8 String rejection matrix.
- Header domain: integrated main's reviewed scalar-edge-whitespace rejection.
  Generated request and response union encoders reject leading/trailing SP/HTAB
  before publication; interior tab remains valid. Runtime parameter and generated
  location tests pass (0.791s / 7.700s).
- Form body identity: removed the one-field shape heuristic, FlatFormUnion
  metadata and generated synthetic wrapper methods. Authored objects retain
  named codecs; flat OAuth uses an actual root union. Maintained one-field
  list/map/object and multi-field JSON controls pass, along with flat OAuth and
  an independently compiled/mounted tagged named-field specimen. The stale
  render assertion/golden was corrected and re-reviewed. Bounded comparison has
  103 sections: 82 identical, 21 intended changes in field decoding, wrapper
  removal, CLI JSON-field identities and explicit OpenAPI field encoding.
  Five process-isolated generations agree. Kind model passes 24/16 states;
  old shape inference violates the invariant. Independent slice review approved.
- Union schema identity: canonical hashes retain effective occurrence closure;
  named occurrences use the shared component projection. Anonymous bodies in
  both declaration orders and aliases pass. Of 862 output sections, 858 remain
  identical; four root-spec sections add distinct closed-body schemas. HTTP/RPC
  generated roots, libopenapi, Vacuum and isolated determinism pass. Independent
  slice review approved.

Evidence lives under `/tmp/loom-fresh-audit/` in `constructor-contract`,
`additional-plan`, `union-presence`, `bytes-domain`, `header-domain`,
`form-body-identity` and `union-schema-identity`. Tests and model sources remain
in the repository; task-owned checker state has been removed after review.
Go checks use Go1.27.3, `GOCACHE=/tmp/loom-957-build-cache.r6al1q`, and local
`LOOM_DIR` for generated modules. Root form/multipart generation, text closure,
full import normalization and final ticket acceptance remain unfinished.

## Recursive field codecs and physical carrier integration

This section supersedes the earlier pending statements for source/application
capture, field Encoding Object export, and the generated multipart specimen.
It does not assert final ticket acceptance or delivery. Main integration now also
includes reviewed correctness and dependency updates beyond the original parent;
final output comparison must identify its actual source inputs rather than rely
on the initial production-source equivalence statement above.

The shared field codec retains source type before import normalization and keeps
whole-value encoding distinct from per-item encoding. Explicit style settings
select text serialization even when a content type is also authored. The common
media classifier handles MIME parameters, structured JSON suffixes and case;
forms and multipart parts use the same classification. Nil native collections
remain absent, while nonnil empty collections remain supplied. Direct owner and
runtime tests and the independent source/application review pass.

Exporter capture retains branch-local defaults and authored media labels. A
single physical field requires one Encoding Object that agrees with every
alternative; incompatible explicit alternatives are rejected. Source type
assertions are validated against the admitted value domain, including nullable
and finite-enum refinements. Independent expr/IR/v3 checks, two checked models,
libopenapi/Vacuum specimens and process-isolated artifact comparisons pass.
Ordinary form/style JSON and YAML match the recorded main controls byte for byte.
The imported EncodingSource design has also been regenerated and its complete
service/server/client/CLI module compiles after the shared carrier corrections.

Physical storage follows captured codec roles. Form and multipart roots and
text leaves retain native storage; JSON fields retain decoder presence in both
directions because outgoing validation replays typed decoding. Private projected
field declarations require private owning body declarations. Media variants are
rebound after their enclosing body is copied. Alias flattening retains effective
occurrence-first metadata, including imported source-type facts. Direct controls
cover array and absent source types, request/response variants and independent
JSON/native branches.

Collection validation resolves the actual element reference, independently of
parent extraction. Inline object elements are pointers; union elements are
values; presence wrappers do not change the contained object's storage. The
shared NameScope collection reference and definition use the same rule, and
nullable inline objects receive one wrapper. A compiled array/map matrix checks
reference/declaration agreement. Conversion now independently captures source
and target presence, including named and nested collections. Selected nil nested
unions preserve the outer selection without dereferencing nil; independent
compiled stale-target controls and the model confirm this correction.

CLI conversion and validation failures return a zero value of the declared
builder result type. The shared owner no longer assumes every nonprimitive
schema has a nilable result. All 21 affected CLI golden differences are only
local typed-zero declarations and returns. Shared CLI tests, compiled HTTP/gRPC
controls and independent-process gRPC output comparison pass. The independent
review also compiles six result carriers with ordinary and zero-named locals
through malformed, validation-error and valid paths. No findings remain for this
slice.

The maintained multipart fixture passes mounted checks for optional and required
roots, nullable JSON values, selected Body absence, ordered mixed file/text
parts, unknown empty-name parts, recursive JSON and media variants. The complete
form/Content fixture passes required-key, mixed source-array, nested item,
request/response media and outgoing selection checks. Full codegen passes after
the collection initializer correction. Independent review of those complete
implementation slices and the importer map-context correction has completed
without unresolved findings. Final contract symmetry, repository gates,
documentation/roadmap finalization and the exact atomic ticket review remain
separate obligations; none of the slice approvals authorizes the whole commit.

Integration evidence: `/tmp/loom-1014-export-encoding`,
`/tmp/loom-cli-zero-owner`, `/tmp/loom-1014-child-carrier`,
`/tmp/loom-collection-transform-presence`, `/tmp/loom-form-content-union`, and
`/tmp/loom-multipart-generation`. Their manifests identify bounded source inputs;
review reports separately identify the approved exact diff. These temporary
packets support review; permanent regression tests and model correspondence
remain beside their owning implementation.


## Integration corrections and independent review

Dynamic form maps now consume the captured `Element` codec for each literal
physical key. Ordinary, untagged and multi-media roots share that emitter.
JSON items, objects, nested collections, source-array unions, text alternatives,
empty and bracketed keys, deterministic replay, malformed input and unrepresentable
empty entries have mounted coverage. Form admission follows the media kind at
every root context; ordinary single-media multipart custom/raw handling remains
separate. The shared `AttributeContext.CollectionElementRef` owns physical slot
references for transforms, validation, defaults and these dynamic map adapters.
The 36-file independent review and its comment/import-only re-review pass.
Of 140 before/candidate sections, eight intentionally change map adapter calls;
all eight compared modules compile and isolated output matches.

Collection defaults retain the shared `DefaultForField` admission policy while
materializing the destination's native, Optional or Nullable storage. Compiled
regressions cover 72 representation/policy cases and nested collection literal
slots, named layouts, absence, supplied zeros/empty values, disabled defaults and
explicit null. The review's raw Optional assignment and omitted native-pointer
default findings are fixed and independently confirmed. The bounded generated
controls remain byte-identical (138 sections). Its separately exposed transport
package-reference defect is closed by the traversal correction below.

Detached graph mutations now use source-node identity at `expr.walkUserTypes`,
`makeHTTPTypeRecursive` and captured transport IR normalization. Distinct nodes
may share a nominal ID; all require normalization, suffixing and package removal.
Shared and recursive edges are visited once, with registration before mutating
callbacks. This aligns the traversals with `DupAtt` without changing generated
schema/declaration identity. Both traversal orders and immutable source graphs
have direct tests. The SourceTraversal model reproduces the nominal-ID omission
and checks complete, exactly-once, terminating source traversal under its bounds.
Existing HTTP and JSON-RPC ObjectDefaults modules now compile and pass their
mounted race/vet tests. Of 119 compared raw/rendered sections, 20 changes remove
incorrect service-package qualification from local transport references/default
literals and merge the resulting duplicate helpers. All four candidate modules
compile; ordinary form and raw body/package controls are byte-identical. Five
independent generations match exactly. Independent review found no issues.

The eight-file lint integration also has independent approval: 194 generated
sections are unchanged, with four compiled modules and five-process determinism.
The later media-metadata helper relocation addresses the 700-line source gate
without changing its implementation; its 119 compared sections are identical.

The bounded exported-design gate for `UntaggedRootsDSL` and
`UntaggedLocationsDSL` passes (19.609s): generation, `loom example`, module build
and vet. The aggregate `make test` attempt exposed a stale test-only comparator
reading the removed `RequestBody.FormEncoding`; the comparison owner now reads
occurrence `Schema.MediaEncoding`. The complete OpenAPI contract gate passes:
rendered v3 libopenapi/Vacuum specimens (209.703s), existing CLI semantic
round-trip (30.136s), generated HTTP/JSON-RPC byte contracts (26.612s/19.422s),
independent value constraints (10.518s) and IR byte projection (10.236s).
The final CLI comparison owner passes its existing semantic and direct controls
(29.664s). The composed 3.1/3.2 import, generation, compiled runtime, reimport and
regeneration check passes (91.485s), including the full JSON/YAML contract oracle
after both generation passes. Four final exported specimens pass Vacuum; scoped
CLI lint reports zero issues. The independent review's missing second-export
assertion is fixed and confirmed; the reviewer also invokes the final helper on
both retained outputs (1.041s). No production importer change was needed for
reference-plus-enum: the obsolete test adapter caused that rejection by rewriting
it into an unsupported intersection, and has been removed.

Current runtime race coverage passes for pkg/unionmatch (1.676s/1.360s)
and the complete HTTP package (29.588s). The initial HTTP race failure came from
CPU compression consuming a test fixture's network deadline; the reviewed
fixture prepares the same full compressed payload before I/O and still checks
all limit/terminal/close behavior. No runtime policy or timeout changed. Final
lint and whole-repository outcomes remain pending.

Review packets and source manifests: `/tmp/loom-form-map-codec`,
`/tmp/loom-default-presence-review`, `/tmp/loom-source-traversal`,
`/tmp/loom-1014-integration-lint`, and `/tmp/loom-1014-encoding-file-split`.
Corresponding independent reports live under `/tmp/loom-map-independent`,
`/tmp/loom-default-independent`, `/tmp/loom-traversal-independent`, and
`/tmp/loom-lint-independent`. They are review evidence, not permanent build
requirements. Durable regressions and opt-in model inputs stay in the repository.


The main integration audit compares the candidate filesystem with parent
`2386e219f` (including the separately reviewed CLI assertion, rejection
expectations and WebSocket fixture corrections). A three-way audit against its
preceding parent `4cfd16225` found 27 already-integrated paths and 15 textual
overlaps, with no missing nonconflicting additions. Independent review of all
15 overlaps found no lost main fixes; nine affected package regressions pass.
The original named-array additional-value-plan control is retained and passes.
New files copied from main remain present even where this detached checkout's
older index does not track them; final review must compare complete files against
the actual main parent, rather than interpreting an index-only diff as deletions.

Remaining aggregate evidence is explicitly incomplete. The full CLI attempt
failed on a disk-space linker error and a stale atomic-output fixture that still
expects supported `oneOf` input to fail. The full lint attempt reaches test
sources after passing root package lint, then reports nine fixture-function
complexity findings and one whitespace finding. These failures require corrected
fixtures and rerun gates; the composed symmetry slice approval does not cover
them or authorize the final ticket commit.


## Final-suite findings and acceptance audit follow-up

The aggregate test run completed with disk-exhaustion failures as well as
stale assertions and confirmed generator defects. These are tracked separately;
no infrastructure failure is counted as a passing package.

Optional union custom-error fields exposed a constructor ownership defect:
`transportFieldBinding` describes physical field storage, whereas constructor
adaptation must describe the detached transform result. Error constructors now
share `selectBodyField` and `buildResponseResultPointer` with success responses.
A union transform already returns a pointer. The strengthened direct regression
and compiled clients reproduce the erroneous double pointer before the fix;
all direct and mounted controls pass afterward (15.341s). Across 385 generated
sections, only the raw/rendered copies of four error constructors change from
`Choice: &v` to `Choice: v`. Both baseline control modules and all three candidate
modules compile; the baseline union module fails as expected. Five independent
processes match (10.711s), scoped lint reports zero issues, and exact-diff review
approves the correction. Evidence: `/tmp/loom-error-union-constructor`.

The stale atomic-output fixture now rejects unsupported `anyOf`, while keeping
all output-atomicity assertions; all three cases pass (0.748s), with independent
approval. The nine fixture helper extractions preserve 396 generated sections
across ten designs exactly, compile all ten modules, and pass five-process
comparison (67.412s). Independent review confirms evaluation order and captures;
its two misplaced harness comments are corrected and confirmed separately.
The optional-form and Any diagnostic assertion refresh retains presence,
reconstruction and rejection checks: direct cases pass (0.372s), and the existing
mounted optional-object group passes (4.671s). Its exact diff is approved.

Two JSON-RPC golden headers add the shared `jsontext` import only; the non-update
checks pass (0.273s), and independent review approves their unchanged bodies.
Twenty-eight HTTP goldens reflect already-reviewed codec/storage corrections:
explicit `FieldValue`, native form carrier and validator names, and preservation
of converted map keys with typed zero values for missing Optional slots. The
291 original golden inputs were frozen; the 28 changed files have no other
changes or removed validations. Normal checks pass (0.812s), with separate
independent approval. These expectation updates introduce no generator changes.

A fresh requirement-by-requirement audit confirms the six historical import
controls' exact hashes and the completed two-export composed oracle. It also
identifies unfinished plan obligations: generation of every accepted specimen,
full null/overlap semantic comparisons, precise per-row rendered/diagnostic
facts, importer closure-profile checks, owner-routed diagnostic wording,
isolated regeneration of the symmetry fixture, and the current 44-row CLI
report comparison. Mounted closure, nested-three ambiguity and error-envelope
coverage is being expanded. These obligations must be closed explicitly before
final ticket review; broad package success cannot replace them.

The subsequent nullable/source-type IR correction, multipart fixture updates,
new mounted cases, final repository gates and exact whole-ticket review are
still in progress. None of the slice approvals above authorizes committing the
whole ticket. The main push attempt also terminated unsuccessfully after build
storage exhaustion; publication remains pending.

## Acceptance closure after disk recovery

The 13-file nullable/source-type correction has final independent approval for
its exact frozen delta (`6884936f`). The full IR suite passes (43.021s), generated
nullable/reference controls pass (84.591s), and scoped lint reports zero issues.
All 34 freshly regenerated ordinary/nullable artifacts match the retained
accepted outputs exactly. Evidence and final review are under
`/tmp/loom-importer-final-failures-independent`.

The seven-file multipart fixture correction also has final independent approval
(`89ffaf83`). Its six-design comparison retains 75 identical files and explains
76 changed or added files; three isolated generations produce the same 151
files byte for byte. The final example and collision controls compile and pass,
the shared JSON-RPC credential race tests pass (6.034s), and scoped lint reports
zero issues. Evidence: `/tmp/loom-multipart-integration-tests`.

The mounted closure/nested/error-envelope expansion is independently approved
(`b10da274`). Its eight closure profiles preserve match counts before closure;
the nested ambiguity counts three matches; selected error bodies reconstruct
their envelopes. Generated HTTP and JSON-RPC controls, five-process determinism,
both OpenAPI targets, four pinned Vacuum specimens and scoped lint pass. Its
88-section comparison preserves every JSON-RPC section, all 15 previous HTTP
paths and their components. Evidence: `/tmp/loom-root-closure-coverage`.

All 44 current CLI reports now match the design-table outcomes: 32 accepted,
12 rejected with the expected diagnostic owner. Each source is byte-identical
to the original baseline and checked-in specimen. Current and original reports
are paired in `/tmp/loom-1014-final-reports/comparison.json`; direct matrix tests
separately assert diagnostic paths and wording. The stale sibling-default design
row is reconciled with the explicit constructor contract and existing DSL
regressions: branch defaults remain supported; whole-union defaults are rejected.

Generation of the complete accepted matrix exposed a shared naming defect:
a branch named `Kind` collides with the generated `Kind()` method. The correction
is complete and independently approved (`729d507b`). `codegen/unionnames.Resolve`
allocates storage fields and method suffixes once, including fixed-method,
sanitized-name and authored-suffix collisions. Service/view emitters, transforms,
validation, HTTP and gRPC consume that allocation. Ordinary outputs remain byte
identical; collision controls compile and run through HTTP, gRPC and the public
union plugin. Five-process comparison, scoped lint and independent review pass.
The public `uniongen.Field.MethodName` contract and regeneration guidance are
recorded in the maintained documentation. Evidence: `/tmp/loom-union-member-names`.

The empty-form emitter correction is independently approved (`e92a8eef`). Shared
field emitters receive their target explicitly, so genuinely empty objects need
no unused local alias. Across 180 generated sections, 138 remain identical and
42 have intended binding changes. Six modules compile; direct, mounted and
five-process determinism controls pass. Evidence: `/tmp/loom-empty-form-fields`.

Two separately reviewed owner corrections close adversarial outgoing-body cases:

- Named-object conversion helpers preserve nil from their resolved pointer
  parameter, independently of whether the first occurrence is required or
  optional. This prevents both empty-object fabrication and nil dereference.
  Full codegen tests pass (32.414s); the direct compiled matrix passes (1.987s).
  Across 180 sections, 168 are identical and the remaining 12 contain only nil
  guard insertions. All six base/candidate modules compile; five-process
  comparison (11.040s) and scoped lint pass. The bounded model reproduces the
  original failure and checks presence/requiredness independence. Exact six-file
  review approves `b0921b4f`. Evidence: `/tmp/loom-object-helper-presence`.
- Request-body presence is checked before conversion from the canonical service
  carrier. Empty native union Kind is absent, including behind a nonnil pointer;
  required bodies reject it and optional bodies omit it. Selected-body mappings
  retain other query/header fields. Nullable, custom codecs, raw JSON and the
  required Any member's explicit null contract remain distinct. Across 268
  sections, 258 are identical and only ten client encoder sections change.
  Ten base/candidate modules compile; full body/form controls (10.362s),
  multipart/null controls (14.719s), JSON-RPC selected-body controls (8.114s),
  native-null controls (2.863s), custom conversions (4.646s), five-process
  determinism (13.876s) and lint pass. Exact 19-file review approves `4e577618`.
  Evidence: `/tmp/loom-request-union-presence`.

All 32 accepted import rows now have successful generation, compilation and
source/export acceptance evidence. The original 30 successes are supplemented
by the final empty-form rows 37/38 (7.684s). Six independently identified weak
semantic oracles were strengthened (rows 02/20/25/31/35/44): explicit source
membership tables and six bad-export mutations establish that the assertions
are not vacuous. Those six generated cases pass (49.509s), and the correction
has independent approval (`21e41d80`). All eight closure profiles pass for source
and exported documents (16 subtests, 4.301s); JSON/YAML parsing and compilation
also pass. The symmetry fixture's independent regeneration passes (18.84s).
All 3,752 captured source/input hashes remain unchanged across the final focused
runs. Evidence: `/tmp/loom-1014-m6-final-focused/outcome.md`. An interrupted run
with an overly broad selector is retained as unsuccessful evidence and is not
counted toward acceptance.

Final aggregate gates and exact whole-ticket review remain required. These slice
approvals do not authorize the ticket commit.


## Planned test names and maintained coverage

The implementation consolidates several prospective test names. Independent
source inspection maps each obligation to maintained tests; names alone are not
used as evidence of completion:

| Planned obligation | Maintained coverage |
| --- | --- |
| Import locations and error envelope | `TestOneOfImportBehaviorTable`, `TestOneOfImportedDesignGenerates`, `TestOneOfReferencedErrorEnvelope`, and the mounted `UntaggedClosureProfilesHarness`. All 44 rows use their actual schema location. |
| Null normalization and outer siblings | `TestOneOfNormalizedFacts`, `TestOneOfSemanticPreservation`, `TestOneOfSiblingAdmission`, `TestOneOfSiblingForwardUnionReference`, and `TestOneOfDefaultBoundary`. |
| Captured field codecs and media context | `TestOneOfCapturedCodecThroughRootReference`, `TestOneOfFormFieldCodecs`, `TestOneOfRootEncodingContext*`, and `TestOneOfRootEncodingGenerates`. |
| Promotion and deterministic naming | Matrix rows 03/19/20, `TestUntaggedOneOfPromotesNestedObjectBranch`, shared name-occupancy tests and independent symmetry regeneration. |
| New-root lowering | `TestUntaggedRootGrammar`, prepared/lowered JSON-shape tests and compiled `TestRootWireContracts` in the maintained roots harness. |
| Untagged and tagged occurrence closure | `TestUnionClosurePreparedAndLowered`, `TestUnionClosurePreservesIndependentDecoderSelection`, `TestUnionClosureTaggedEnvelope`, compiled `TestOccurrenceClosure` and JSON-RPC closed-envelope rejection. |
| Closure metadata and form/multipart admission | `TestUnionClosureMetadata`, `TestHTTPFormUnionAdmission`, `TestHTTPMultipartUnionAdmission`, encoding-contract tests and compiled form/multipart fixtures. |
| CLI and JSON-RPC roots | `TestCLIParsesTextUnionFlag` is declared inside the generated locations harness; the JSON-RPC entry point is `TestJSONRPCUntaggedRoots`. |
| Exported unions | `TestUntaggedUnionExportClosure`, `TestUntaggedUnionExportShapes`, `TestUntaggedUnionExportLocations`, and captured form Encoding Object tests. |

The former scalar-rejection test is deliberately replaced by
`TestUntaggedAcceptsRootKinds`. `TestOneOf` and `TestUntagged` in the execution
commands are selection prefixes, not missing function declarations. The full
source map is retained with the whole-ticket review evidence; this mapping does
not replace terminal test results or authorize a commit.


## Explicit inert closure and the optional Vacuum probe

The extra `UntaggedFormDSL` before/candidate probe reports the same Vacuum
`oas-schema-check` diagnostic for `FormContentNested`: its outer union explicitly
authors `unevaluatedProperties: false` and declares source type `array`. JSON
Schema permits that keyword and makes it inert on array instances; Vacuum
v0.39.3 deliberately rejects object-only keywords on an explicitly array schema.
Its upstream `TestSchemaType_NewConstraints_ObjectKeywordsOnArray` requires this
behavior. Independent 3.1/3.2 reproductions and controls confirm the policy; this
is not a confirmed dependency bug or an unfiled upstream defect.

The fixture in `http/codegen/testdata/untagged_form_content_dsl.go` is the durable
input. `expr.CaptureUnionClosure` records its authored union policy, the OpenAPI
analyzer projects it onto that same union, and `schema_encoding_source.go`
retains its authored source type. The generated branches and array item schemas
do not acquire closure. Runtime matching applies the closure check only to
object values, as the accepted design requires. Moving it into record items
would incorrectly close those objects; deleting it would erase explicit policy
from the structural export contract. Neither change is made to appease lint.

Both probe results remain recorded as lint errors, not passes. No lint exclusion,
waiver, dependency patch or default contract-gate change is introduced. The
required contract gate and its invalid-spec controls remain separate acceptance
obligations. Independent analysis and standalone controls are retained under
`/tmp/loom-vacuum-contentnested`; the actual fixture preserves reproduction after
temporary probe cleanup.


## Aggregate verification on the integrated candidate

The aggregate run uses Go 1.27.3, the same shared task cache and explicit local
source mode. `/tmp/loom-1014-aggregate-final/source-input-hashes.json` captures
5,432 build inputs; its initial recheck found no changed captured input. The
corrections below subsequently changed that candidate, so these results retain
their original source scope rather than proving the corrected final tree.
`make fmt` passes. Its only retained ticket change is import grouping in
`cmd/loom/import_comparison_input_controls_test.go`; 79 formatter-only edits to
files identical to the intended main parent are restored byte for byte, keeping
unrelated changes outside the ticket. `git diff --check` passes.

- `make lint`: PASS (192.433s), including testdata and integration-module lint.
- `go test -race ./pkg ./http ./internal/unionmatch -count=1`: PASS (28.012s).
- `make openapi-contract`: PASS (297.327s), including rendered specs, pinned
  Vacuum and invalid-spec controls, semantic import round trips, generated
  HTTP/JSON-RPC byte schemas and independent projection checks.
- The plan's bounded `UntaggedRootsDSL` and `UntaggedLocationsDSL` compile tier:
  PASS (40.073s package time; 41.928s command). Both designs run generation,
  example generation, module tidy, complete build and vet against this checkout.
  Neither has an expected-failure entry. This covers the final closure-profile,
  naming and presence inputs, without expanding into the full design corpus.
- Documentation lint after the acceptance updates: PASS (2.677s). The planned
  stub audit finds only intended generated-application placeholders and their
  tests, plus an explanatory Lean comment; no unfinished framework path remains
  in those matches.
- The explicit importer/OpenAPI/CLI run passes OpenAPI (0.458s), IR (44.152s),
  versioned output (43.287s) and CLI (337.902s). The importer reaches Go's default
  ten-minute package timeout while its current generated-form test has run for
  eight seconds; no assertion failure precedes the timeout. The same full
  importer package also ran in `make test` with the repository's forty-minute
  limit. That aggregate run is now terminal and failed; the timeout result is
  not used as evidence of importer correctness.
- `make test`: FAIL (2,076.298s). Its failures include stale service/HTTP output
  expectations, a multipart import-collision fixture that still expected a
  custom codec, missing reservation of the generated local name `val`, an
  incorrect native constructor target for selected built-in Problem fields,
  recursive OpenAPI component export, and JSON-RPC presence expectations.
  Final aggregate acceptance remains outstanding.

The service nil-guard goldens, HTTP presence/helper expectations and multipart
fixture corrections have each passed focused validation and independent review.
The shared HTTP/JSON-RPC import allocator now reserves `val`; the existing AST
inventory, ordinary-name and colliding-name generated applications, byte
comparison and independent generation checks passed. Its one-file correction
was independently approved. These changes preserve the central presence and
name-allocation contracts rather than adding renderer-specific exceptions.

The selected Problem correction makes response analysis return its conversion
and actual native target together. The built-in adapter already returns a
complete service error, so its constructor must not wrap that error in the
selected public wire field. All ten selected fields, custom selected-field and
union controls, generated compilation and runtime checks, independent-process
determinism and scoped lint passed. Its exact eight-file correction was
independently approved in
`/tmp/loom-selected-problem-target/independent-review.md`.

The five JSON-RPC presence expectation files now assert the shared absence
predicates for optional and required carriers. Their existing DSL and runtime
harnesses are unchanged; six direct groups (0.795s), six generated
compile/vet/race harnesses (24.172s), including required `Any` nil as JSON null,
and scoped lint pass. Independent review approved the exact five-file change
in `/tmp/loom-jsonrpc-presence-expectations/independent-review.md`.

The recursive exporter correction acquires a canonical comparison reference
before traversing the candidate and compares complete schema bodies at the
outer analysis boundary. It preserves the existing equality relation, explicit
name conflicts and the separate representation-allocation owner. Errors and
panics remove only provisional registrations. The earlier graph-copy hypothesis
was rejected: evaluated recursive declarations can already have equivalent
finite unfoldings with different raw fingerprints.

Focused validation passes: complete IR package (13.122s), rendered source-type
and nullable parser controls (0.831s), and five maintained recursive import
cases with compilation, vet, parsing and independent regeneration (58.107s).
Ordinary and nullable controls each preserve all 16 generated files byte for
byte against the prior implementation and across independent candidate runs.
Pinned Vacuum passes all six retained JSON/YAML specimens. The exact recursive
schema assertion preserves its accepted source-type wrapper, null alternative,
references and array/map constraints. Bounded models distinguish unregistered
recursion and premature comparison from completed acquisition and conflict
handling; their bounds remain explicit in the owning README.

The first aggregate retry found `analyzer.go` four lines above its configured
700-line limit. That retry's test process was deliberately cancelled before
editing source and is not a test result. The lifecycle closure now lives beside
its canonical-comparison owner; a real projection-panic regression confirms
rollback and panic preservation. The file-size gate and scoped lint pass.
Full `make lint` then passes (86.225s). The final source manifest contains 5,435
inputs; both the pre-test and post-test rechecks record zero changes. Its
SHA-256 is `eec892264f20c04099e5ff79aef2d03d03e91d723e06313091554d5da5f8ff94`.
The exact twelve-file recursive correction, including the lifecycle relocation,
is independently approved with no findings in
`/tmp/loom-recursive-identity/review-r2/independent-review.md`.

The pre-review integrated gates on candidate r5 are terminal:

- `make lint`: PASS (86.225s), including repository file-size limits, root,
  testdata and integration-module checks.
- `make openapi-contract`: PASS (86.589s), including current rendered schemas,
  pinned Vacuum and negative controls, semantic round trips and independent
  projection checks.
- `make test`: PASS (1,545.187s), with the repository's forty-minute package
  timeout. Fresh affected package results include CLI (226.746s), shared
  codegen (67.575s), generator (985.696s), HTTP codegen (1,202.754s), IR
  (39.354s), versioned OpenAPI (42.620s), importer (1,506.741s) and JSON-RPC
  codegen (666.534s). This completes the full importer obligation left open by
  the earlier ten-minute explicit invocation. Valid cached package results
  retain their exact source/input dependency scope; the earlier failed and
  deliberately cancelled runs are not counted as passes.

The runtime race and bounded Roots/Locations compile evidence above is retained
for its recorded inputs alongside r5 full-suite checks and the recursive
correction's own output and compiler controls. The subsequent whole-review
findings below change the matcher and ordinary form decoder; r5 passes alone do
not establish acceptance of those corrections. Their affected checks and output
comparisons must cover the corrected inputs. The canonical-main push's exhaustive
corpus is separate; no additional full corpus is claimed as a local ticket
acceptance requirement.

M8 documentation retains the delivered contract and readiness row, removes the
execution plan, and points the roadmap to durable evidence. Whole-ticket
commit authorization belongs to the final exact-diff review packet, including
actual staged-tree equality and preservation of unrelated preparation; slice
approvals alone do not authorize the complete commit.

Each completed gate has its exact command, elapsed time, terminal exit status
and complete log in `/tmp/loom-1014-aggregate-final`. The separate canonical-main
push completed successfully, including its 4,576.258-second exhaustive corpus;
that result covers main `2386e219f`, not this uncommitted candidate.

## Whole-review corrections

The independent review of r5 (`4ded5dcf6f05231c006f99aeca9246852ae90a78`)
found two P2 defects despite the green aggregate gates:

- A singleton retained for occurrence closure crossed inconsistent cardinality
  contracts: compact DSL syntax rejected it, and function-rendered forms reached
  runtime selectors and JSONShape validation that also required two branches.
- Ordinary form decoding inferred absence from decoded key count. A generated
  required empty-object client therefore produced a request its server rejected;
  optional nonempty separators-only input also lost its physical presence.

The corrections belong to the shared matcher/descriptor/import-rendering owners
and the shared form-root decoder respectively. Compact DSL syntax keeps its
separate two-argument contract; nonempty runtime branch sets retain ordinary
selection and closure. All form roots share raw-input presence before validating
the complete value set. The known failures were reproduced in regression tests
and bounded model counterexamples before correction. Exact focused output,
compiler, runtime and model evidence and focused re-reviews are recorded below.
Whole-ticket authorization additionally checks the assembled candidate and actual
staged tree. The full r5 review found no additional confirmed defects.

Ordinary form request decoding now shares raw-input presence with map and union
roots: required zero-byte bodies decode an empty value set, optional zero bytes
remain absent, and nonempty separators-only input remains present. The maintained
form fixture covers 49 required/optional root/member wire cases and generated-client
empty-object replay, with exact status/problem detail, service calls and payload
state. Final affected checks include the unchanged tagged-form reflection and
nullable JSON body-contract controls. Three representative designs produce only
the intended server-decoder changes (261 of 265 raw/rendered sections identical);
all six before/candidate modules compile and five isolated candidate generations
are byte-identical. The existing body-presence model reproduces rejected key-count
admission and checks the raw-media policy; its proof assumptions remain bounded
and commands opt-in. This resolves the whole-review ordinary-form P2 finding;
aggregate gates and final independent review apply to the assembled ticket.

The exact fourteen-file correction is independently approved with no unresolved
findings in `/tmp/loom-ordinary-form-presence/independent-review.md`, patch SHA-256
`6d57aaa0adbe005ea0db90161c6163f4c158c342c38818c85401d498e8bd312c`.
The first refactor omitted the tagged reflection fallback; the existing mounted
body-contract check caught it, and the final retry restores and checks that path.
The failed intermediate runs are not acceptance evidence.

The singleton correction aligns importer construction, shared selection and
JSONShape admission around a nonempty branch set. Function-form DSL preserves
singleton occurrence closure; compact syntax still requires two arguments.
The nullable-pair owner now recognizes an already-normalized OneOf value without
a direct Type or Ref, preserving branch grammar and closure before adding
nullability. The existing exact two-alternative wrapper grammar remains bounded.

The frozen nineteen-file correction and before-image correspondence are in
`/tmp/loom-singleton-closure/review`. Direct selector/descriptor/DSL checks pass;
final maintained import, generated codec, independent closure oracle, parser,
export/reimport/evaluation and process-byte checks pass (15.423s). All 22 ordinary
and 18 previously renderable singleton artifacts are unchanged against r5.
New primitive, promoted-object and null-normalized singleton cases now generate
and run where the baseline failed. Pinned Vacuum passes all six JSON/YAML
specimens. Scoped lint and repository file-size checks pass. The minimum-two
model counterexample fails as intended; five singleton profiles and the ordinary
control pass within their stated bounds. The nullable-pair correspondence note
explicitly does not claim that bare-null oneOf proofs establish anyOf parsing.

The affected complete runtime packages (`internal/unionmatch`, `pkg`, `http`)
pass uncached race tests on the corrected matcher (9.330s). This replaces the r5
runtime evidence for the changed matching contract. The final combined source
freeze and invalidated aggregate lint/OpenAPI checks are recorded below.

The exact nineteen-file singleton correction is independently approved with no
unresolved findings in `/tmp/loom-singleton-closure/review/independent-review.md`,
patch SHA-256 `d008af3cb708e24e022ba2d4f52b81f38fe64605f167332267bc5a021f4e6574`.
Both P2 findings from the whole-ticket review are resolved.

The combined corrected source contains 5,441 recorded inputs, SHA-256
`1be87a104cb14907c35db959fbc59c136b943618de0c7adc40606caed9176b5b`.
Current full `make lint` passes (64.852s), and current full
`make openapi-contract` passes (64.760s). The focused correction packets cover
all changed matcher, importer and form generation seams, including bounded
before/candidate output and actual compiled transport behavior; unchanged
boundaries retain their r5 full-suite evidence. The full exported-design corpus
is not rerun as a ticket acceptance gate. Final input recheck, documentation
check, exact candidate assembly and staged preservation evidence live alongside
the terminal gate records and final review packet.

## Checked-in JSON-RPC fixture synchronization

The first main push passed lint and the complete root race/coverage suite, then
the JSON-RPC integration gate found stale CLI files in the maintained `ticktock`
and `mixedtick` fixtures. The shared CLI generator already returns the declared
payload type's zero value on conversion errors; these two fixtures still used
literal `nil`. Regeneration changes only those two generated CLI files, with
three error-return sites each. The resulting pointer zero remains nil; request,
response and SSE behavior are unchanged. No generator or runtime correction was
needed, and generated files were not edited by hand.

Both fixture modules compile and pass their tests. The complete JSON-RPC
integration suite passes, including independent-process regeneration with exact
fixture-tree comparison and the maintained SSE, GET-listener and mixed-transport
behavior checks (tests package 30.110s). Module manifests return to their original
bytes after `go mod tidy`. The exact correction, before/after tree hashes, failed
push and successful focused logs are retained in
`/tmp/loom-1014-fixture-refresh`. Existing root-suite evidence still covers the
unchanged framework source. The correction and final amended ticket tree require
independent review before amending the unpublished atomic commit; the failed
push is not publication evidence.


## Package declaration ownership correction

The naming correction is qualified against local main
`0b45c8318b9d88d900cd008880433a7596feeb63`, with exact 59-path packet SHA-256
`cc75c9e9b09bf33eb892b057f0c197d7e8bfbcbb7f72fa7231b932d9de6701e6`.
It belongs to the same unpublished #1014 atomic change against parent
`2386e219f8a79220a6553cc00ee81815d57b4b1a`; qualification is not publication.

Valid declarations could previously collide with fixed HTTP names, validators,
union companions or view helpers. A generated viewed result could also reference
`FooBar2` while its views package declared `FooBar`. The shared NameScope now
allocates a nominal declaration and its complete companion family together at
first local reference. Service, view and HTTP owners supply their actual package
inventories. View metadata maps have their own role, and the physical viewed
envelope owns one identity for its declaration, references and validator; naming
an undeclared source-result copy no longer reserves a phantom family.

Consumers must regenerate affected services and clients and use the allocated
Go names. No compatibility aliases retain uncompilable declarations. Authored
wire values, branch tags and logical view labels retain their contracts. Plugin
authors use the shared family inventory described in
[union code generation](../../docs/plugin_union_codegen.md).

The final bounded checks ran on Go 1.27.3 with explicit local source selection:

- Direct service validation passes 41 top-level tests and 271 test nodes,
  including cached/copied identities, nullable views and cross-result helpers.
- Nine representative designs cover fixed and endpoint roles, validator and
  union companions, projected/envelope/map collisions, grouped clients,
  multipart roles and repeated routes, with three ordinary controls.
- The combined generated service/view/HTTP/CLI module compiles. Independent
  candidate processes produce exactly the same 158 files; all three ordinary
  designs also match baseline byte for byte.
- All 44 changed files have classified differences: complete declaration-family
  suffixes and corresponding references/helpers, plus allocated type names in
  diagnostics. The bounded comparison preserves struct tags, route literals,
  branch wire constants and logical view-label sets.

The retained baseline reproduces the declaration/reference failure and the
projected/envelope collision; failed intermediate runs remain failures. The
[declaration-family model](../../codegen/tla/declaration_families/README.md) and
[service symbol models](../../codegen/service/tla/package_symbols/README.md)
record their bounded assumptions and counterexamples. Their checks remain
explicit design aids outside ordinary gates. Direct and compiled tests check
the corresponding implementation; the models do not prove arbitrary Go output.

Exact source, command, output and comparison evidence is retained in
`/tmp/loom-1005-http-package-names/revision-envelope`. Independent review accepted
that exact naming packet without unresolved findings in
`/tmp/loom-generic-names-independent-review/revision-envelope-review/report.md`.
The original whole-ticket, fixture synchronization and file-local validation
import reviews remain evidence for unchanged source; the latter authorized the
tree in `0b45c831` and is recorded in
`/tmp/loom-1014-union-import-independent-review/final-amendment-review.md`.
No new full-suite or exported-design-corpus pass is claimed for the naming
correction. Final composition, documentation checks, exact-parent review and
normal commit/push hooks remain required before delivery. Standalone native
default work and #957/#1005/#976 preparation are outside this amendment.
