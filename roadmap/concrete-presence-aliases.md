# Concrete presence through named aliases — #892

1. [x] Reproduce the two nullable-reference importer CI failures and the direct
   depth-two/depth-three helper failures. Independently confirm the shared owner.
2. [x] Model shallow clearing, whole-graph clearing and alias-spine clearing at
   the existing validation-carrier model boundary. Preserve nullable children.
3. [x] Correct `codegen.concretePresenceAttribute` once for validation,
   transformation, default representation and generated helper naming. Copy only
   the outer alias spine; retain declaration identities and structural children.
4. [x] Check source immutability, scalar/object/collection aliases, recursive
   children and ordinary nonnullable controls. Run affected package tests and
   the actual importer generation failures. Compare a bounded ordinary and
   nullable specimen against the archived parent and isolated repeat; compile
   outputs and validate their specs. Record intended differences.
5. [ ] Independent exact-diff review, atomic commit, push, close and cleanup.

The expression model already retains the correct nullable contract. The helper
consuming an extracted presence carrier removes nullability from only one named
level, while `expr.IsNullable` inherits through the complete alias chain. The
result can be incorrectly treated as another nullable carrier by conversion.
Neither HTTP body cloning nor importer rendering owns this correction.

Nullability on an object member, array element, map entry or recursive child is
a separate occurrence contract. Clearing a deep copy of the entire structural
graph would erase those contracts. The corrected operation follows only named
alias edges, retains names and metadata, and stops before structural children.
This adds no DSL policy, transport exception or compatibility mode.

Readiness: direct alias depths 0/1 pass and 2/3 fail on parent `e67f359e2`;
the existing two importer tests fail during response transformation. An expr
probe confirms effective nullability survives service/body preparation. The
independent owner assessment agrees with spine-only copying. Recursive error
identity assertions and explicit-body value-plan ancestry are separate CI
investigations; neither is included in this repair.

Acceptance on the candidate: complete codegen race suite passes (26.157s),
both actual nullable-import regressions pass (16.971s), scoped lint is clean.
The final direct tests fail on the archived parent for deeper object, scalar,
array and map aliases and the recursive-child specimen; ordinary controls pass.
The bounded model rejects shallow clearing in 7 states and whole-graph clearing
in 5; spine clearing passes all 8 states. Source preservation is an explicit
model assumption checked separately against actual Go declarations.

Actual-parent generation still fails for the nullable reference specimen; the
candidate compiles and vets. The existing supported importer fixture compiles
and vets on both sides, and all 16 ordinary generated artifacts are byte-identical.
Both candidate modules repeat all 32 artifacts exactly in separate CLI processes;
all four JSON/YAML specs pass libopenapi and pinned Vacuum. No checked-in golden
changes are intended or needed. The complete codegen tests include its checked-in
goldens. Only the previously failing alias conversion is newly generated.

Evidence is frozen under `/tmp/loom-892-final-manifest.json`. The first temporary
capture setup used an unsupported Python tar-filter argument, and its parent
overlay initially used `/tmp` instead of the resolved `/private/tmp` path. Both
setup mistakes were corrected before the recorded actual-parent artifact check;
no failed setup run establishes output parity.
