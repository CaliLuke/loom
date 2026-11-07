# File-local union validation imports

An untagged union emits validation in two places: named validation methods are
owned by their branch declaration's file, and the JSON matcher embeds each
bound `union.JSON.Branches[].Validate` fragment in the union's file. Shared
package membership does not share Go imports between those files. The old
collector followed only named methods, so a custom-package union with a string
length constraint called `utf8.RuneCountInString` without importing unicode/utf8.
The same POST/body-only design fails on committed c86613180fbba9eb232f9969f95d18a9427ebe72;
removing the custom package makes the ordinary service file compile.

`collectUnionValidationUTF8Paths` now collects dependencies from both emitted
fragment owners. It uses the matcher's bound validation, not the field's cached
validation or source schema traversal. A nested named call contributes its
UTF-8 import to the validator file, not the caller. Existing imports, allocation
and location decisions are unchanged. Consumers of affected generated code must
regenerate after updating; the authored DSL and wire validation contract do not
change.

## Opt-in model

`ValidationImports.tla` varies both fragments' file ownership and whether each
needs the import. `Coverage` requires every use to have a file-local import;
`Absence` rejects unconditional publication to unrelated files. The model is
about the relevant validation dependency after ordinary unused-import cleanup;
it does not model other fixed codec imports, parsing, identifier aliases,
arbitrary call graphs, string-analysis precision or actual generated Go.

Run explicitly with the installed TLA+ tools, outside ordinary gates:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -config checked.cfg -metadir "$TMPDIR/loom-validation-imports" ValidationImports.tla
```

TLC 2.19 / TLA+ tools 1.7.4 checks 16 distinct assignments. `old.cfg` fails
Coverage for a matcher-only dependency; `blanket.cfg` fails Absence; `checked.cfg`
passes both. All three configurations were executed before the production fix.
No checker invocation, installation or proof-derived gate is part of CI.

`TestUnionValidationImportOwnership` checks the actual descriptor-to-file seam,
including both owners, absence and unused field-cache controls.
`TestUnionMatcherValidationImports` renders custom object/primitive/nested and
ordinary declarations. `TestUnionValidationImportsGenerated` compiles generated
service packages and checks Unicode minimum/maximum length rejection, accepted
branch identity and JSON round trips, including an unconstrained control.
Bounded CLI generation additionally covers the real POST body and both HTTP
sides; the sole intended generated-byte difference is the missing import in
custom choice.go. Ordinary and unconstrained custom output stays byte-identical.
