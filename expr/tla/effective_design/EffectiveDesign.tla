------------------------ MODULE EffectiveDesign ------------------------
EXTENDS Naturals, FiniteSets, TLC

CONSTANT Behavior
VARIABLES scenario, extendFirst, explicitDefault, dslDefault, stage,
          predicates, fields, admitted, selectedDefault, pointers, heap, mutated

vars == <<scenario, extendFirst, explicitDefault, dslDefault, stage,
          predicates, fields, admitted, selectedDefault, pointers, heap, mutated>>
Uses == {1, 2}
Cells == {0, 1, 2}
Scenarios == {"refinement", "collision", "peer-defaults"}
CompletePredicates == {"base-minimum", "local-maximum"}
ExpectedDefault == IF explicitDefault THEN 5 ELSE 1
Tag(field) == IF field = "n" THEN 2 ELSE 1
UniqueTags(members) == Cardinality({Tag(field): field \in members}) = Cardinality(members)
InheritedDefaults == IF scenario = "peer-defaults" THEN {1, 2} ELSE {1}
ResolvableDefaults(local) == local \/ Cardinality(InheritedDefaults) = 1
ShouldAdmit == scenario = "refinement" \/
               (scenario = "peer-defaults" /\ explicitDefault)

Init == /\ scenario \in Scenarios
        /\ extendFirst \in BOOLEAN
        /\ explicitDefault \in BOOLEAN
        /\ dslDefault = IF explicitDefault THEN 5
                         ELSE IF extendFirst THEN 1 ELSE 0
        /\ stage = [u \in Uses |-> "authored"]
        /\ predicates = [u \in Uses |-> {"local-maximum"}]
        /\ fields = [u \in Uses |-> IF scenario = "collision" THEN {"c"} ELSE {"n"}]
        /\ admitted = [u \in Uses |-> FALSE]
        /\ selectedDefault = [u \in Uses |-> 0]
        /\ pointers = [u \in Uses |-> u]
        /\ heap = [c \in Cells |-> "base"]
        /\ mutated = {}

\* Current evaluation validates only local fields before composition.
EarlyValidate(u) ==
    /\ Behavior = "late-validation"
    /\ stage[u] = "authored"
    /\ stage' = [stage EXCEPT ![u] = "early-validated"]
    /\ admitted' = [admitted EXCEPT ![u] = UniqueTags(fields[u])]
    /\ UNCHANGED <<scenario, extendFirst, explicitDefault, dslDefault,
                   predicates, fields, selectedDefault, pointers, heap, mutated>>

Compose(u) ==
    /\ IF Behavior = "late-validation"
       THEN stage[u] = "early-validated" ELSE stage[u] = "authored"
    /\ predicates' = [predicates EXCEPT ![u] =
                         IF Behavior = "overwrite" THEN {"base-minimum"}
                         ELSE CompletePredicates]
    /\ fields' = [fields EXCEPT ![u] = IF scenario = "collision"
                                            THEN @ \cup {"a"} ELSE @]
    /\ selectedDefault' = [selectedDefault EXCEPT ![u] =
          IF Behavior = "overwrite" THEN
              IF scenario = "peer-defaults" THEN 2 ELSE 1
          ELSE IF Behavior = "copied-overlay" /\ dslDefault # 0
               THEN dslDefault ELSE ExpectedDefault]
    /\ pointers' = [pointers EXCEPT ![u] =
                       IF Behavior = "shared-base" THEN 0 ELSE u]
    /\ stage' = [stage EXCEPT ![u] =
                   IF Behavior = "late-validation" THEN "validated"
                   ELSE "composed"]
    /\ UNCHANGED <<scenario, extendFirst, explicitDefault, dslDefault,
                   admitted, heap, mutated>>

Validate(u) ==
    /\ stage[u] = "composed"
    /\ admitted' = [admitted EXCEPT ![u] =
          UniqueTags(fields[u]) /\ ResolvableDefaults(
              IF Behavior = "copied-overlay" THEN dslDefault # 0 ELSE explicitDefault)]
    /\ stage' = [stage EXCEPT ![u] = "validated"]
    /\ UNCHANGED <<scenario, extendFirst, explicitDefault, dslDefault,
                   predicates, fields, selectedDefault, pointers, heap, mutated>>

Finish(u) ==
    /\ stage[u] = "validated"
    /\ stage' = [stage EXCEPT ![u] = IF admitted[u] THEN "emitted" ELSE "rejected"]
    /\ UNCHANGED <<scenario, extendFirst, explicitDefault, dslDefault,
                   predicates, fields, admitted, selectedDefault, pointers, heap, mutated>>

\* A later consumer owns mutable annotations on its completed occurrence.
Mutate(u) ==
    /\ stage[u] = "emitted"
    /\ u \notin mutated
    /\ heap' = [heap EXCEPT ![pointers[u]] = "changed"]
    /\ mutated' = mutated \cup {u}
    /\ UNCHANGED <<scenario, extendFirst, explicitDefault, dslDefault,
                   stage, predicates, fields, admitted, selectedDefault, pointers>>

Next == \E u \in Uses:
            EarlyValidate(u) \/ Compose(u) \/ Validate(u) \/ Finish(u) \/ Mutate(u)
Spec == Init /\ [][Next]_vars

AuthoredRefinementRetained == \A u \in Uses:
    stage[u] = "emitted" => predicates[u] = CompletePredicates
AdmissionMatchesCompleteContract == \A u \in Uses:
    stage[u] \in {"emitted", "rejected"} => admitted[u] = ShouldAdmit
DefaultUsesAuthoredOverlay == \A u \in Uses:
    stage[u] = "emitted" => selectedDefault[u] = ExpectedDefault
SourceUnchanged == heap[0] = "base"
OtherOccurrenceUnchanged == \A u \in Uses:
    (stage[u] = "emitted" /\ u \notin mutated) => heap[pointers[u]] = "base"
=============================================================================
