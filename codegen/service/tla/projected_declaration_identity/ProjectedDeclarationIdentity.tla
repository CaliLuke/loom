---------------- MODULE ProjectedDeclarationIdentity ----------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode
Types == {"ordinary", "intrinsic"}
Occurrences == 1..4
Owner(i) == IF i \in {1, 2} THEN "ordinary" ELSE "intrinsic"
Intrinsic(id) == id = "intrinsic"
OccurrenceNullable(i) == i \in {2, 4}
Nullable(i) == Intrinsic(Owner(i)) \/ OccurrenceNullable(i)
Allocated(id) == IF id = "ordinary" THEN "ChildView" ELSE "ChildView2"
PhysicalName(i) == IF Mode = "old" /\ Nullable(i)
                   THEN "loom.Nullable[" \o Allocated(Owner(i)) \o "]"
                   ELSE Allocated(Owner(i))
Parameter(i) == IF Mode \in {"old", "names-only"} THEN Nullable(i)
                ELSE IF Mode = "erase-presence" THEN FALSE
                ELSE Intrinsic(Owner(i))
UseNullable(i) == IF Mode = "erase-presence" THEN FALSE ELSE Nullable(i)
VARIABLES remaining, declared, names, parameters, uses
vars == <<remaining, declared, names, parameters, uses>>
Init == /\ remaining = Occurrences /\ declared = {}
        /\ names = [id \in Types |-> ""]
        /\ parameters = [id \in Types |-> FALSE]
        /\ uses = [i \in Occurrences |-> FALSE]
Visit(i) == LET id == Owner(i) IN
           /\ i \in remaining
           /\ remaining' = remaining \ {i}
           /\ declared' = declared \cup {id}
           /\ names' = IF id \in declared THEN names
                        ELSE [names EXCEPT ![id] = PhysicalName(i)]
           /\ parameters' = IF id \in declared THEN parameters
                             ELSE [parameters EXCEPT ![id] = Parameter(i)]
           /\ uses' = [uses EXCEPT ![i] = UseNullable(i)]
Next == (\E i \in remaining: Visit(i)) \/ (remaining = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
DeclarationIdentity == \A id \in declared: names[id] = Allocated(id)
ParameterAgreement == \A id \in declared: parameters[id] = Intrinsic(id)
OccurrencePreservation == \A i \in Occurrences \ remaining: uses[i] = Nullable(i)
Terminates == <>(remaining = {})
=====================================================================
