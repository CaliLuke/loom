-------------------------- MODULE UnionDefaults --------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANTS Mode, Scenario
VARIABLES sample, accepted, materialized, output, phase
vars == <<sample, accepted, materialized, output, phase>>

\* The expression value owner has already resolved each nested branch. Names
\* and physical slots come from the declaration owners, not from source JSON.
Node(path, branch, primary, storage, payload) ==
    [path |-> path, branch |-> branch, primary |-> primary,
     storage |-> storage, payload |-> payload]
Case(label, rootUnion, knownEarly, nodes) ==
    [label |-> label, rootUnion |-> rootUnion, knownEarly |-> knownEarly,
     nodes |-> nodes]
Cases == {
    Case("direct-known", TRUE, TRUE, {}),
    Case("direct-late", TRUE, FALSE, {}),
    Case("empty-map", FALSE, TRUE, {}),
    Case("omitted-field", FALSE, TRUE, {}),
    Case("map", FALSE, TRUE, {Node("[a]", "right", "Choice", "value", 0)}),
    Case("array", FALSE, TRUE, {Node("[0]", "left", "Choice2", "value", 1)}),
    Case("optional-field", FALSE, TRUE, {Node(".choice", "right", "Choice", "pointer", 0)}),
    Case("required-field", FALSE, TRUE, {Node(".choice", "left", "Choice", "value", 1)}),
    Case("nullable-slot", FALSE, TRUE, {Node("[a]", "right", "Choice", "nullable-value", 0)}),
    Case("recursive-container", FALSE, TRUE,
        {Node(".outer", "left", "Outer", "value", 1),
         Node(".outer.inner", "right", "Choice", "pointer", 0)})
}
SelectedCases == IF Scenario = "all" THEN Cases
    ELSE {c \in Cases : c.label = Scenario}
Allows(c) == IF Mode = "current" THEN ~(c.rootUnion /\ c.knownEarly)
             ELSE IF Mode = "reject-containers" THEN FALSE ELSE ~c.rootUnion
Branch(n) == IF Mode = "reselect" THEN "left" ELSE n.branch
Primary(n) == IF Mode = "unqualified" THEN "Choice" ELSE n.primary
Storage(n) == IF Mode = "value-only" THEN "value" ELSE n.storage
Rendered(n) == [path |-> n.path, branch |-> Branch(n), primary |-> Primary(n),
                storage |-> Storage(n), payload |-> n.payload]
Init == /\ sample = CHOOSE c \in Cases : c.label = "empty-map"
        /\ accepted = FALSE /\ materialized = FALSE /\ output = {}
        /\ phase = "choose"
Choose == /\ phase = "choose" /\ sample' \in SelectedCases
          /\ phase' = "finalized"
          /\ UNCHANGED <<accepted, materialized, output>>
Validate == /\ phase = "finalized" /\ accepted' = Allows(sample)
            /\ phase' = "render" /\ UNCHANGED <<sample, materialized, output>>
Render == /\ phase = "render"
          /\ materialized' = (accepted /\ ~(Mode = "current" /\ sample.nodes # {}))
          /\ output' = IF accepted /\ ~(Mode = "current" /\ sample.nodes # {})
                       THEN {Rendered(n) : n \in sample.nodes} ELSE {}
          /\ phase' = "done" /\ UNCHANGED <<sample, accepted>>
Next == Choose \/ Validate \/ Render
Spec == Init /\ [][Next]_vars
FinalizedAdmission == phase \in {"render", "done"} => accepted = ~sample.rootUnion
NativeRepresentationTotal == phase = "done" /\ accepted => materialized
RetainedSelection == phase = "done" /\ accepted =>
    \A n \in sample.nodes : \E o \in output :
        o.path = n.path /\ o.branch = n.branch /\ o.primary = n.primary
PhysicalStorage == phase = "done" /\ accepted =>
    \A n \in sample.nodes : \E o \in output : o.path = n.path /\ o.storage = n.storage
NoSilentLoss == phase = "done" /\ accepted =>
    Cardinality(output) = Cardinality(sample.nodes) /\
    \A n \in sample.nodes : \E o \in output : o.path = n.path /\ o.payload = n.payload
=============================================================================
