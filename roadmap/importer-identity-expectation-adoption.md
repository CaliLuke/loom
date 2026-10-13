# Importer identity expectation adoption (#900)

The shared importer metadata owner preserves schema descriptions, including an
explicitly absent description. Two older path-identity tests require the former
Attribute call without its metadata block and fail before their identity checks
can complete. This task adopts the delivered output without production changes.

## Sequence and acceptance

- [x] Read the rendered output and both existing tests. Reproduce all four matrix
  failures and the snake-case failure on main; retain the parent evidence.
- [x] Update the complete expected Attribute blocks, including exact field name,
  String type, empty Description and schema-description metadata. Retain GET and
  Param spelling checks and the existing evaluated-design/OpenAPI round trip.
- [x] Run both affected tests under race and existing parameter-description
  controls. Check the exact diff and configured lint; no generator corpus is
  needed because production and generated output do not change.
- [ ] Obtain independent exact-diff review, commit and push this atomic adoption,
  close #900, and preserve evidence before cleaning temporary outputs.

The acceptance rule is exact identity with the delivered metadata block, not a
weaker substring that would accept a renamed field or a different type. Existing
metadata models remain applicable; no new semantic decision is introduced.

Both identity tests and the existing three-version parameter-description controls
pass under race, including evaluated designs and generated OpenAPI round trips.
Configured lint reports zero new issues. Only two expected strings change.
