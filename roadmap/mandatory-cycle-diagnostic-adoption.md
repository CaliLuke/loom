# Mandatory cycle diagnostic adoption

## Scope and acceptance

The parent direct self-alias case in `TestReferencePlanRejectsMandatoryCycles`
expected the initializer planner's diagnostic after #789 moved its rejection to
earlier shared object-field admission. This is a test contract adoption, not a
change to accepted schemas or the dependency race repair (#820).

`planDocument` admits schema shapes before `renderPlan` calls
`planSchemaReferences`. `objectFields` traverses alias bases and rejects an active
base as `schema`, with the component source path and cyclic-base reason. The
public import coverage promises pure alias rejection by reference resolution;
remaining eager array/map initializer cycles retain
`schema-reference-initialization`. Raw self/mutual alias documents also retain
libopenapi's existing parser rejection.

The regression must check the shared graph planner independently for all three
mandatory constructors, then check full admission at the correct owning stage.
Both paths must remain fatal even under allow-lossy; rendering must reject the
same admitted diagnostic. Existing supported recursion, constructor and diamond
controls remain unchanged. No production, model, generated artifact or public
contract changes are needed.

## Sequence

- [x] Read the shape admission, object-field traversal, reference planner and
  documented subset; reproduce the alias-only assertion failure on current main.
- [x] Record #915 and update the owner-specific assertions.
- [x] Run focused reference admission/constructor/normal controls with the race
  detector and the existing scoped lint gate.
- [x] Freeze the exact test/document diff and evidence for independent review.
- [ ] Commit, push and close after authorization (integration owner).

## Evidence

Parent reproduction: `/tmp/loom-cycle-diagnostic-parent.log`, command
`GOEXPERIMENT=jsonv2 go test ./internal/openapiimport -run
'^TestReferencePlanRejectsMandatoryCycles$' -count=1`. Root also reproduced the
same failure with both production and conditional upstream dependencies in
`/tmp/loom-820-current-pin-cycle-control.log`; the prior full importer run is
retained evidence, not a reason to repeat its nine-minute suite for this change.

The existing schema-reference models describe constructor-edge classification
and initializer acyclicity. This repair keeps those owners unchanged and adds a
direct seam assertion separating graph diagnostics from earlier shape admission;
there is no new state or ordering algorithm to model.

Final focused acceptance passes with the race detector (1.662s):

```sh
GOEXPERIMENT=jsonv2 go test -race ./internal/openapiimport \
  -run '^(TestReferencePlan.*|TestAnalyzeRejectsMandatoryReferenceCycles|TestReferenceComponentsMatchReachability)$' \
  -count=1
GOEXPERIMENT=jsonv2 golangci-lint run ./internal/openapiimport
```

The first command covers the repaired three-constructor matrix, preserved
parallel edges, supported constructor contexts, acyclic diamond/names, raw parser
rejections and the existing 512-graph SCC correspondence. Scoped lint reports
zero issues. Logs are `/tmp/loom-cycle-diagnostic-candidate.log` and
`/tmp/loom-915-lint.log`; exact source/owner/evidence hashes and commands are frozen
in `/tmp/loom-915-final-manifest.json`. Generator output is unchanged because only
test assertions and this record change; no generator rerun is required.
